package main

import (
	"flag"
	"fmt"
	"math"
	"math/big"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/wheelcomplex/preinit/bigcounter"
	"github.com/wheelcomplex/preinit/cmtp"
	"github.com/wheelcomplex/preinit/misc"
)

// globe size of byteHash
var globeHashSize int

// globe size of chunk
var globeChunkSize int

// newByteChunk new [][]byte for sync.Pool.New
func newByteChunk() interface{} {
	bc := make([][]byte, globeChunkSize)
	for i := 0; i < globeChunkSize; i++ {
		bc[i] = make([]byte, globeHashSize)
	}
	return bc
}

// newHashChunk new []uint32 for sync.Pool.New
func newHashChunk() interface{} {
	return make([]uint32, globeChunkSize)
}

// newHashSizeChunk new []uint32 with size limit
func newHashSizeChunk(size int) []uint32 {
	return make([]uint32, size)
}

// one worker for one thread
type hashWorker struct {
	index     int                   // index of worker
	generator bigcounter.BigCounter // standalone in worker
	checksum  cmtp.Checksum         // standalone in worker
	hashPool  *sync.Pool            // []uint32 pool, share with all worker
	counterCh chan []uint32         // result out, []uint32
	bytePool  *sync.Pool            // [][]byte pool, standalone in worker
	m         sync.Mutex            // locker for worker
	closing   chan struct{}         // closing signal
	closed    chan struct{}         // closed signal
	finished  bool                  // finish signal for closer
	destroyed bool                  // destroyed flag
}

// newHashWorker return new hashWorker
// index, generator, checksum should be standalone for worker
// hashPool, counterCh share with all worker
func newHashWorker(index int,
	generator bigcounter.BigCounter,
	checksum cmtp.Checksum,
	hashPool *sync.Pool,
	counterCh chan []uint32) *hashWorker {

	hw := &hashWorker{
		index:     index,
		generator: generator,
		checksum:  checksum,
		bytePool: &sync.Pool{
			New: newByteChunk,
		},
		hashPool:  hashPool,
		counterCh: counterCh,
		closing:   make(chan struct{}, 128),
		closed:    make(chan struct{}, 128),
	}

	go hw.closer()
	return hw
}

// Destroy free all internal resource
// all call to destroyed hashWorker may case panic
func (hw *hashWorker) Destroy() {
	//println("Destroy enter", hw.index)
	hw.m.Lock()
	defer hw.m.Unlock()
	if hw.destroyed {
		// already destroyed
		return
	}
	//println("Destroy try", hw.index)
	select {
	case <-hw.closing:
	default:
		close(hw.closing)
	}
	// waitting for worker close
	//println("Destroy wait", hw.index)
	<-hw.closed
	hw.destroyed = true
	hw.generator = nil
	hw.checksum = nil
	hw.bytePool = nil
	//println("Destroy done", hw.index)
}

// Run blocking run hash test
func (hw *hashWorker) Run() {
	defer func() {
		close(hw.closed)
		hw.Destroy()
		//println("hashWorker", hw.index, "exited")
	}()

	var bchunk [][]byte
	var hchunk []uint32
	// TODO: compare select closing check and bool closing check

	for hw.finished == false {

		bchunk = hw.bytePool.Get().([][]byte)
		hchunk = hw.hashPool.Get().([]uint32)

		for idx, _ := range bchunk {
			// generate hash buffer
			hw.generator.FillExpBytes(bchunk[idx])

			// hash and save result
			hchunk[idx] = hw.checksum.Checksum32(bchunk[idx])

			// generator ++
			if err := hw.generator.Plus(); err != nil {
				hchunk = hchunk[:idx+1]
				break
			}
		}
		hw.bytePool.Put(bchunk)

		hw.counterCh <- hchunk

		//select {
		//case hw.counterCh <- hchunk:
		//default:
		//	println("hw.counterCh <- hchunk blocking")
		//	hw.counterCh <- hchunk
		//}
	}
}

// closer run hash test in goroutine
func (hw *hashWorker) closer() {
	<-hw.closing
	hw.finished = true
}

//
type hashTester struct {
	size       int                   //
	groupsize  int                   //
	generator  bigcounter.BigCounter // standalone in worker
	checksum   cmtp.Checksum         // standalone in worker
	hashPool   *sync.Pool            // []uint32 pool, share with all worker
	counterCh  chan []uint32         // result out, []uint32
	count      bigcounter.BigCounter // counter for qps compute
	workers    []*hashWorker         //
	maxcnt     *big.Int              //
	bigSecond  *big.Int              //
	results    []uint16              // all hash info
	distr      []uint16              // index by math.MaxUint32 % groupsize
	collided   []uint16              // index by math.MaxUint32 % groupsize
	startts    time.Time             //
	endts      time.Time             //
	esp        *big.Int              //
	qps        *big.Int              //
	limit      *time.Timer           //
	closed     chan struct{}         //
	closing    chan struct{}         //
	maxworker  int                   //
	lockThread bool                  //
	finished   bool                  //
	m          sync.Mutex            //
	rm         sync.Mutex            // result locker
}

//
func NewhashTester(size int,
	generator bigcounter.BigCounter,
	checksum cmtp.Checksum,
	groupsize int,
	limit int64,
	lockThread bool) *hashTester {

	// -1, reserve one cpu for commond task and counter
	maxworker := runtime.GOMAXPROCS(-1) - 1
	if maxworker < 1 {
		maxworker = 1
	}

	if size < 1 {
		size = 1
	}

	// set globe globeHashSize for pool new
	globeHashSize = size

	// larger globeChunkSize use more memory and save more cpu
	globeChunkSize = 512
	countersize := maxworker * 2048 * 2048
	// 3168 last better

	if groupsize <= 0 {
		groupsize = 4096
	}

	if limit <= 0 {
		limit = math.MaxUint32
	}

	maxcnt, _ := big.NewInt(0).SetString(generator.Max(), 10)
	//fmt.Printf("generator.Size() %d, generator.Max() = %s || %x => %s || %x\n", generator.Size(), generator.Max(), generator.Bytes(), maxcnt.String(), maxcnt.Bytes())

	ht := &hashTester{
		size:      size,
		groupsize: groupsize,
		generator: generator,
		checksum:  checksum,
		hashPool: &sync.Pool{
			New: newHashChunk,
		},
		counterCh:  make(chan []uint32, countersize),
		count:      generator.New(),
		workers:    make([]*hashWorker, maxworker),
		maxcnt:     maxcnt,
		bigSecond:  big.NewInt(int64(time.Second)),
		results:    make([]uint16, math.MaxUint32+1),
		distr:      make([]uint16, groupsize),
		collided:   make([]uint16, groupsize),
		startts:    time.Now(),
		endts:      time.Now(),
		qps:        big.NewInt(0),
		closed:     make(chan struct{}, 128),
		closing:    make(chan struct{}, 128),
		limit:      time.NewTimer(time.Duration(limit) * time.Second),
		maxworker:  maxworker,
		lockThread: lockThread,
	}
	//

	// initial stat
	ht.Stat()
	go ht.closer()
	go ht.counter()
	go ht.run()
	return ht
}

//
func (ht *hashTester) run() {
	defer func() {
		// all done, close
		close(ht.counterCh)
		//misc.Tpf(fmt.Sprintln(ht.maxworker, "worker exited"))
	}()
	var wg sync.WaitGroup
	//misc.Tpf(fmt.Sprintln("lauch", ht.maxworker, "worker"))

	// initial generator step
	genstep := big.NewInt(0)
	genstep = genstep.Div(ht.maxcnt, big.NewInt(int64(ht.maxworker)))
	genptr := big.NewInt(0)

	for i := 0; i < ht.maxworker; i++ {
		// initial worker
		generator := ht.generator.New()
		checksum := ht.checksum.New(0)
		generator.SetInit(genptr.Bytes())
		endptr := generator.New()
		endptr.FromBigInt(big.NewInt(1).Mul(genstep, big.NewInt(int64(i+1))))
		endptr.Mimus()
		if i == ht.maxworker-1 {
			generator.SetMax(ht.maxcnt.Bytes())
		} else {
			generator.SetMax(endptr.Bytes())
		}
		//println("generator#", i, "start from", generator.String(), "end at", generator.Max())
		genptr = genptr.Add(genptr, genstep)

		ht.workers[i] = newHashWorker(i, generator, checksum, ht.hashPool, ht.counterCh)

		wg.Add(1)
		go ht.runWorker(ht.workers[i], &wg)
		//time.Sleep(2)
	}
	ht.startts = time.Now()
	ht.Stat()
	wg.Wait()
	select {
	case <-ht.closing:
	default:
		close(ht.closing)
	}
	ht.Stat()
}

//
func (ht *hashTester) runWorker(worker *hashWorker, wg *sync.WaitGroup) {
	defer func() {
		//worker.Destroy()
		wg.Done()
	}()
	if ht.lockThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	//misc.Tpf(fmt.Sprintln("worker", worker.index, "running"))
	worker.Run()
	//misc.Tpf(fmt.Sprintln("worker", worker.index, "exited"))
}

func (ht *hashTester) counter() {
	defer func() {
		//println("counter exited")
		ht.Close()
	}()
	if ht.lockThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	for hchunk := range ht.counterCh {
		for idx, _ := range hchunk {
			ht.results[hchunk[idx]]++
		}
		ht.rm.Lock()
		ht.count.AddUint64(uint64(len(hchunk)))
		ht.rm.Unlock()
		ht.hashPool.Put(hchunk)
	}
	// worker exited
	ht.Stat()
	misc.Tpf(fmt.Sprintln("result computing"))
	for idx, _ := range ht.results {
		ridx := idx % ht.groupsize
		//results     []uint16        // all hash info
		//distr       []uint16        // index by math.MaxUint32 % groupsize
		//collided    []uint16        // index by math.MaxUint32 % groupsize
		ht.distr[ridx]++
		if ht.results[ht.results[idx]] > 0 {
			ht.collided[ridx]++
		}
	}
	misc.Tpf(fmt.Sprintln("result computed"))
	close(ht.closed)
}

// show qps
func (ht *hashTester) Stat() (countstr, qps string, esp time.Duration) {
	select {
	case <-ht.closing:
		// already closing, do not update
	case <-ht.closed:
		// already closed, do not update
	default:
		ht.endts = time.Now()
	}
	esp = ht.endts.Sub(ht.startts)
	ht.rm.Lock()
	count := ht.count.ToBigInt()
	ht.rm.Unlock()
	//fmt.Printf("Stat(), esp %v, count %s, %s\n", esp, ht.count.String(), count.String())
	ht.esp = big.NewInt(int64(esp.Nanoseconds()))
	ht.qps = ht.qps.Mul(count, ht.bigSecond)
	ht.qps = ht.qps.Div(ht.qps, ht.esp)
	countstr, qps = count.String(), ht.qps.String()
	return
}

// Size
func (ht *hashTester) Size() int {
	return int(ht.size)
}

//
func (ht *hashTester) closer() {
	defer func() {
		ht.limit.Stop()
		// stop all workers
		for i, _ := range ht.workers {
			go ht.workers[i].Destroy()
		}
	}()
	select {
	case <-ht.limit.C:
		misc.Tpf("%s\n", "stop for reach time limit")
		return
	case <-ht.closing:
		misc.Tpf("%s\n", "stop for closing")
		return
	}
}

// Close free memory
func (ht *hashTester) Close() {
	ht.m.Lock()
	defer ht.m.Unlock()
	if ht.finished == true {
		return
	}
	select {
	case <-ht.closing:
	default:
		close(ht.closing)
	}
	<-ht.closed
	//misc.Tpffmt.Sprintln("closing"))
	ht.results = nil
	ht.hashPool = nil
	//ht.collided = nil
	//ht.distr = nil
	//misc.Tpffmt.Sprintln("GC"))
	runtime.GC()
	//misc.Tpffmt.Sprintln("FreeMemory"))
	debug.FreeOSMemory()
	//misc.Tpffmt.Sprintln("closed"))
	ht.finished = true
}

// Result return distr/collided map
// caller will blocked until finished
func (ht *hashTester) Result() ([]uint16, []uint16) {
	<-ht.closed
	return ht.distr, ht.collided
}

//
func (ht *hashTester) Wait() <-chan struct{} {
	return ht.closed
}

func main() {
	profileport := flag.Int("port", 6060, "profile http port")
	runlimit := flag.Int("time", 60, "run time(seconds) limit for each hash")
	size := flag.Int("size", 256, "block size")
	countsize := flag.Int("counter", 8, "counter size")
	groupsize := flag.Int("groupsize", 2048, "group size")
	cpus := flag.Int("cpu", 0, "cpus")
	lock := flag.Bool("lock", true, "lock os thread")
	stat := flag.Bool("stat", true, "show interval stat")
	flag.Parse()
	fmt.Printf(" go tool pprof http://localhost:%d/debug/pprof/profile\n", *profileport)
	go func() {
		fmt.Println(http.ListenAndServe(fmt.Sprintf("localhost:%d", *profileport), nil))
	}()

	if *cpus <= 0 {
		if runtime.NumCPU() > 1 {
			*cpus = runtime.NumCPU() - 1
		} else {
			*cpus = 1
		}
	}
	runtime.GOMAXPROCS(*cpus)
	if *size < 1 {
		*size = 1
	}

	if *groupsize < 16 {
		*groupsize = 16
	}

	if *runlimit < 10 {
		*runlimit = 10
	}

	//
	allhasher := map[string]cmtp.Checksum{
		//"Murmur3": cmtp.NewMurmur3(0),
		//"noop":    cmtp.NewNoopChecksum(0),
		"xxhash1": cmtp.NewXxhash(0),
		//"xxhash2": cmtp.NewXxhash(0),
		//"xxhash3": cmtp.NewXxhash(0),
	}
	//
	misc.Tpf("testing")
	for idx, _ := range allhasher {
		fmt.Printf(" %s", idx)
	}
	fmt.Printf(" ...\n")
	alldistr := make(map[string][]uint16)
	allcollided := make(map[string][]uint16)
	for idx, onehash := range allhasher {
		misc.Tpf("timelimit %d seconds, counter size %d, size %d, groupsize %d, cpus %d, stat %v, lock os thread %v, start %s test\n", *runlimit, *countsize, *size, *groupsize, runtime.GOMAXPROCS(-1), *stat, *lock, idx)
		alldistr[idx] = make([]uint16, *groupsize)
		allcollided[idx] = make([]uint16, *groupsize)
		ht := NewhashTester(*size, bigcounter.NewAnyBaseCounter(*countsize), onehash, *groupsize, int64(*runlimit), *lock)
		waitCh := ht.Wait()
		if *stat {
			go func() {
				tk := time.NewTicker(5e9)
				defer tk.Stop()
				var preesp time.Duration
				for {
					select {
					case <-waitCh:
						return
					case <-tk.C:
						count, qps, esp := ht.Stat()
						if preesp != esp {
							misc.Tpf("Int %s, size %d, count %s, esp %v, qps %s\n", idx, *size, count, esp, qps)
							preesp = esp
						}
					}
				}
			}()
		}
		<-waitCh
		distr, collided := ht.Result()
		copy(alldistr[idx], distr)
		copy(allcollided[idx], collided)
		count, qps, esp := ht.Stat()
		misc.Tpf("End %s, size %d, count %s, esp %v, qps %s\n", idx, *size, count, esp, qps)
		misc.Tpf("%s test done\n", idx)
	}

	//
	// github.com/wheelcomplex/svgo
	//

	//for name, _ := range alldistr {
	//	fmt.Printf("--- distr %s ---\n", name)
	//	for idx, _ := range alldistr[name] {
	//		fmt.Printf("%d, %d\n", idx, alldistr[name][idx])
	//	}
	//	fmt.Printf("--- collided %s ---\n", name)
	//	for idx, _ := range allcollided[name] {
	//		if allcollided[name][idx] > 0 {
	//			fmt.Printf("%d, %d\n", idx, allcollided[name][idx])
	//		}
	//	}
	//}
	misc.Tpf("all %d done\n", len(allhasher))
}

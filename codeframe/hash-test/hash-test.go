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

	"github.com/wheelcomplex/preinit/cmtp"
	"github.com/wheelcomplex/preinit/misc"
)

//
type byteHash struct {
	num uint32
	buf []byte
}

type hashTester struct {
	size        uint32            //
	groupsize   uint32            //
	generators  []misc.BigCounter //
	pool        []byteHash        //
	poollast    uint64            //
	idleblocks  chan []uint64     //
	procblocks  chan []uint64     //
	countblocks chan []uint64     //
	hashs       []cmtp.Checksum   //
	count       misc.BigCounter   //
	maxcnt      *big.Int          //
	bigSecond   *big.Int          //
	results     []uint16          // all hash info
	distr       []uint16          // index by math.MaxUint32 % groupsize
	collided    []uint16          // index by math.MaxUint32 % groupsize
	startts     time.Time         //
	endts       time.Time         //
	esp         *big.Int          //
	qps         *big.Int          //
	limit       *time.Ticker      //
	closed      chan struct{}     //
	closing     chan struct{}     //
	wg          sync.WaitGroup    //
	maxproc     int               //
	lock        bool              //
	closedflag  bool              //
}

//
func NewhashTester(size int, gen misc.BigCounter, hash cmtp.Checksum, groupsize uint32, limit int64, lock bool) *hashTester {
	if size < 1 {
		size = 1
	}
	if groupsize == 0 {
		groupsize = 4096
	}
	if limit <= 0 {
		limit = math.MaxUint32
	}
	maxproc := runtime.GOMAXPROCS(-1) - 1
	// -1, reserve one cpu for commond task and counter
	if maxproc < 1 {
		maxproc = 1
	}

	maxcnt, _ := big.NewInt(0).SetString(gen.Max(), 10)
	//fmt.Printf("gen.Size() %d, gen.Max() = %s || %x => %s || %x\n", gen.Size(), gen.Max(), gen.Bytes(), maxcnt.String(), maxcnt.Bytes())

	initpoolsize := uint64(size) * 10000
	pool := make([]byteHash, initpoolsize)
	chunksize := initpoolsize / 100
	if chunksize < uint64(size) {
		chunksize = uint64(size)
	}
	println("initpoolsize", initpoolsize, "batch length", chunksize)
	chunkptr := uint64(0)
	var chunk []uint64
	idleblocks := make(chan []uint64, initpoolsize*10)
	for i := uint64(0); i < initpoolsize; i++ {
		pool[i] = byteHash{
			num: 0,
			buf: make([]byte, size),
		}
		if chunkptr == 0 {
			chunk = make([]uint64, chunksize)
		}
		chunk[chunkptr] = i
		chunkptr++
		if chunkptr == chunksize {
			idleblocks <- chunk
			chunkptr = 0
		}
	}
	if chunkptr > 0 {
		idleblocks <- chunk
	}
	//for i := uint64(0); i < initpoolsize; i++ {
	//	idleblocks <- uint64(i)
	//}

	ht := &hashTester{
		size:        uint32(size),
		groupsize:   groupsize,
		pool:        pool,
		poollast:    initpoolsize - 1,
		idleblocks:  idleblocks,
		procblocks:  make(chan []uint64, size*100),
		countblocks: make(chan []uint64, initpoolsize),
		generators:  make([]misc.BigCounter, maxproc),
		hashs:       make([]cmtp.Checksum, maxproc),
		count:       gen.New(),
		maxcnt:      maxcnt,
		bigSecond:   big.NewInt(int64(time.Second)),
		results:     make([]uint16, math.MaxUint32+1),
		distr:       make([]uint16, groupsize),
		collided:    make([]uint16, groupsize),
		startts:     time.Now(),
		endts:       time.Now(),
		qps:         big.NewInt(0),
		closed:      make(chan struct{}, 128),
		closing:     make(chan struct{}, 128),
		limit:       time.NewTicker(time.Duration(limit) * time.Second),
		maxproc:     maxproc,
		lock:        lock,
	}
	//
	genstep := big.NewInt(0)
	genstep = genstep.Div(ht.maxcnt, big.NewInt(int64(ht.maxproc)))
	genptr := big.NewInt(0)
	for i := 0; i < ht.maxproc; i++ {
		ht.hashs[i] = hash.New(0)
		ht.generators[i] = gen.New()
		//ht.generators[i].FromBigInt(genptr)
		ht.generators[i].SetInit(genptr.Bytes())
		endptr := ht.generators[i].New()
		endptr.FromBigInt(big.NewInt(1).Mul(genstep, big.NewInt(int64(i+1))))
		endptr.Mimus()
		ht.generators[i].SetMax(endptr.Bytes())
		//println("generator#", i, "start from", ht.generators[i].String(), "end at", ht.generators[i].Max())
		genptr = genptr.Add(genptr, genstep)
	}
	// update last generator
	i := ht.maxproc - 1
	ht.generators[i].SetMax(ht.maxcnt.Bytes())
	//println("generator#", i, "start from", ht.generators[i].String(), "end at", ht.generators[i].Max())
	go ht.closer()
	//go ht.genbuf()
	go ht.counter()
	go ht.procbuf()
	// initial stat
	ht.Stat()
	return ht
}

//
func (ht *hashTester) closer() {
	select {
	case <-ht.limit.C:
		fmt.Printf("%s\n", "stop for reach time limit")
		ht.closedflag = true
		return
	case <-ht.closing:
		fmt.Printf("%s\n", "stop for closing")
		ht.closedflag = true
		return
	case <-ht.closed:
		fmt.Printf("%s\n", "stop for closed")
		ht.closedflag = true
		return
	}
}

/*
//
func (ht *hashTester) genbuf() {
	defer func() {
		// all done, close
		close(ht.procblocks)
		ht.limit.Stop()
	}()
	if ht.lock {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	var buf []uint64
	for ht.closedflag == false {
		buf = <-ht.idleblocks
		for idx, _ := range buf {
			pidx := buf[idx]
			if closing == true {
				// mark unused item
				ht.pool[pidx].buf = nil
			} else {
				ht.generators[i].FillExpBytes(ht.pool[pidx].buf)
				//fmt.Printf("G: %x\n", ht.pool[pidx].buf)
				// over flow check
				if err := ht.generators[i].Plus(); err != nil {
					//fmt.Printf("stop for %s\n", err)
					ht.closedflag = true
				}
			}
		}

		ht.procblocks <- buf

	}
}
*/

func (ht *hashTester) dohash(i int) {
	defer func() {
		//println("hasher", i, "exited")
		ht.wg.Done()
	}()
	if ht.lock {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	misc.Tpf(fmt.Sprintln("generator#", i, "start from", ht.generators[i].String(), "end at", ht.generators[i].Max(), "\n"))
	var buf []uint64
	var closing bool
	for ht.closedflag == false && closing == false {
		buf = <-ht.idleblocks
		for idx, _ := range buf {
			pidx := buf[idx]
			if ht.closedflag == true || closing == true {
				// mark unused item
				ht.pool[pidx].buf = nil
			} else {
				ht.generators[i].FillExpBytes(ht.pool[pidx].buf)
				ht.pool[pidx].num = ht.hashs[i].Checksum32(ht.pool[pidx].buf)
				//last = ht.generators[i].String()
				//fmt.Printf("G: %s, %d, %x\n", ht.generators[i].String(), ht.pool[pidx].num, ht.pool[pidx].buf)
				if err := ht.generators[i].Plus(); err != nil {
					misc.Tpf(fmt.Sprintln("generator#", i, "exiting, current", ht.generators[i].String(), "end at", ht.generators[i].Max()))
					closing = true
				}
			}
		}

		ht.countblocks <- buf

	}
}

func (ht *hashTester) procbuf() {
	defer func() {
		ht.limit.Stop()
		close(ht.countblocks)
	}()
	//println("lauch", ht.maxproc, "hasher")
	for i := 0; i < ht.maxproc; i++ {
		ht.wg.Add(1)
		go ht.dohash(i)
		time.Sleep(1)
	}
	ht.wg.Wait()
	ht.endts = time.Now()
	//misc.Tpf(fmt.Sprintln("end", ht.maxproc, "hasher"))
}

func (ht *hashTester) counter() {
	defer func() {
		println("counter exited")
		ht.Close()
	}()
	if ht.lock {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	var bcount uint64
	for buf := range ht.countblocks {
		bcount = 0
		for idx, _ := range buf {
			pidx := buf[idx]
			if ht.pool[pidx].buf == nil {
				continue
			}
			//results     []uint16        // all hash info
			//distr       []uint16        // index by math.MaxUint32 % groupsize
			//collided    []uint16        // index by math.MaxUint32 % groupsize
			ridx := uint32(ht.pool[pidx].num) % ht.groupsize
			ht.distr[ridx]++
			if ht.results[ht.pool[pidx].num] > 0 {
				ht.collided[ridx]++
			}
			ht.results[ht.pool[pidx].num]++
			bcount++
		}
		ht.count.AddUint64(bcount)

		ht.idleblocks <- buf

	}
}

// show qps
func (ht *hashTester) Stat() (countstr, qps string, esp time.Duration) {
	select {
	case <-ht.closing:
		// already closing, do not update
	default:
		ht.endts = time.Now()
	}
	esp = ht.endts.Sub(ht.startts)
	count := ht.count.ToBigInt()
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

// Close free memory
func (ht *hashTester) Close() {
	select {
	case <-ht.closed:
		// already closed
		return
	default:
	}
	select {
	case <-ht.closing:
	default:
		close(ht.closing)
	}
	misc.Tpf(fmt.Sprintln("closing"))
	//ht.Result()
	ht.hashs = nil
	ht.results = nil
	ht.pool = nil
	//ht.collided = nil
	//ht.distr = nil
	misc.Tpf(fmt.Sprintln("GC"))
	runtime.GC()
	misc.Tpf(fmt.Sprintln("FreeMemory"))
	debug.FreeOSMemory()
	close(ht.closed)
	misc.Tpf(fmt.Sprintln("closed"))
}

// Result return distr/collided map
func (ht *hashTester) Result() ([]uint16, []uint16) {
	return ht.distr, ht.collided
}

//
func (ht *hashTester) Wait() <-chan struct{} {
	return ht.closed
}

func main() {
	profileport := flag.Int("port", 6060, "profile http port")
	runlimit := flag.Int("time", 60, "run time(seconds) limit for each hash")
	size := flag.Int("size", 128, "block size")
	countsize := flag.Int("counter", 4, "counter size")
	groupsize := flag.Int("groupsize", 2048, "group size")
	cpus := flag.Int("cpu", 0, "cpus")
	lock := flag.Bool("lock", true, "lock os thread")
	stat := flag.Bool("stat", false, "show interval stat")
	flag.Parse()
	fmt.Printf("go tool pprof http://localhost:%d/debug/pprof/profile\n", *profileport)
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
		"Murmur3": cmtp.NewMurmur3(0),
		//"noop":    cmtp.NewNoopChecksum(0),
		//"xxhash": cmtp.NewXxhash(0),
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
		misc.Tpf("timelimit %d seconds, counter size %d, size %d, groupsize %d, cpus %d, lock os thread %v, start %s test\n", *runlimit, *countsize, *size, *groupsize, runtime.GOMAXPROCS(-1), *lock, idx)
		alldistr[idx] = make([]uint16, *groupsize)
		allcollided[idx] = make([]uint16, *groupsize)
		ht := NewhashTester(*size, misc.NewAnyBaseCounter(*countsize), onehash, uint32(*groupsize), int64(*runlimit), *lock)
		waitCh := ht.Wait()
		if *stat {
			go func() {
				tk := time.NewTicker(5e9)
				defer tk.Stop()
				for {
					select {
					case <-waitCh:
						return
					case <-tk.C:
						count, qps, esp := ht.Stat()
						fmt.Printf("Int %s, size %d, count %s, esp %v, qps %s\n", idx, *size, count, esp, qps)
					}
				}
			}()
		}
		<-waitCh
		distr, collided := ht.Result()
		copy(alldistr[idx], distr)
		copy(allcollided[idx], collided)
		count, qps, esp := ht.Stat()
		fmt.Printf("End %s, size %d, count %s, esp %v, qps %s\n", idx, *size, count, esp, qps)
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

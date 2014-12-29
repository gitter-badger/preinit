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
	size        uint32          //
	groupsize   uint32          //
	generator   misc.BigCounter //
	idleblocks  chan *byteHash  //
	procblocks  chan *byteHash  //
	countblocks chan *byteHash  //
	hashs       []cmtp.Checksum //
	count       misc.BigCounter //
	maxcnt      *big.Int        //
	bigSecond   *big.Int        //
	results     []uint16        // all hash info
	distr       []uint16        // index by math.MaxUint32 % groupsize
	collided    []uint16        // index by math.MaxUint32 % groupsize
	startts     time.Time       //
	endts       time.Time       //
	esp         *big.Int        //
	qps         *big.Int        //
	limit       *time.Ticker    //
	closed      chan struct{}   //
	closing     chan struct{}   //
	wg          sync.WaitGroup  //
	maxgo       int             //
	lock        bool
}

//
func NewhashTester(size int, gen misc.BigCounter, hash cmtp.Checksum, groupsize uint32, limit int64, lock bool) *hashTester {
	if size < 1 {
		size = 1
	}
	if groupsize == 0 {
		groupsize = 4096
	}
	maxcnt, _ := big.NewInt(0).SetString(gen.Max(), 10)
	idleblocks := make(chan *byteHash, 4096)
	for i := 0; i < 4096; i++ {
		idleblocks <- &byteHash{
			num: 0,
			buf: make([]byte, size),
		}
	}
	if limit <= 0 {
		limit = math.MaxUint32
	}
	maxgo := runtime.GOMAXPROCS(-1) - 1
	if maxgo < 1 {
		maxgo = 1
	}
	ht := &hashTester{
		size:        uint32(size),
		groupsize:   groupsize,
		generator:   gen.New(),
		idleblocks:  idleblocks,
		procblocks:  make(chan *byteHash, 4096),
		countblocks: make(chan *byteHash, 4096),
		hashs:       make([]cmtp.Checksum, maxgo),
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
		maxgo:       maxgo,
		lock:        lock,
	}
	for i := 0; i < ht.maxgo; i++ {
		ht.hashs[i] = hash.New(0)
	}
	go ht.genbuf()
	go ht.counter()
	go ht.procbuf()
	return ht
}

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
	for {
		buf := <-ht.idleblocks
		ht.generator.FillExpBytes(buf.buf)
		//fmt.Printf("G: %x\n", buf.buf)
		ht.procblocks <- buf
		// over flow check
		if err := ht.generator.Plus(); err != nil {
			//fmt.Printf("stop for %s\n", err)
			return
		}
		select {
		case <-ht.limit.C:
			//fmt.Printf("%s\n", "stop for reach time limit")
			return
		default:
		}
		select {
		case <-ht.closing:
			//fmt.Printf("%s\n", "stop for closing")
			return
		default:
		}
	}
}

func (ht *hashTester) dohash(cnt int) {
	defer ht.wg.Done()
	if ht.lock {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	for buf := range ht.procblocks {
		//fmt.Printf("H: %x\n", buf.buf)
		buf.num = ht.hashs[cnt].Checksum32(buf.buf)
		ht.countblocks <- buf
	}
}

func (ht *hashTester) procbuf() {
	defer func() {
		close(ht.countblocks)
	}()
	maxgo := runtime.GOMAXPROCS(-1) - 2
	if maxgo < 1 {
		maxgo = 1
	}
	//println("lauch", maxgo, "hasher")
	for i := 0; i < maxgo; i++ {
		ht.wg.Add(1)
		ht.dohash(i)
	}
	ht.wg.Wait()
	//println("end", maxgo, "hasher")
}

func (ht *hashTester) counter() {
	defer func() {
		ht.Close()
	}()
	/*
		results     []uint16        // all hash info
		distr       []uint16        // index by math.MaxUint32 % groupsize
		collided    []uint16        // index by math.MaxUint32 % groupsize
	*/
	if ht.lock {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	for buf := range ht.countblocks {
		//fmt.Printf("C: %s, %d:%x\n", ht.count.String(), buf.num, buf.buf)
		idx := buf.num % ht.groupsize
		ht.distr[idx]++
		if ht.results[buf.num] > 0 {
			ht.collided[idx]++
		}
		ht.results[buf.num]++
		//
		ht.count.Plus()
		//fmt.Printf("C: %s, %x, %x\n", ht.count.String(), ht.count.Bytes(), buf)
		//fmt.Printf("C: %s, %x, %x\n", ht.count.String(), ht.count.FillBytes(buf), buf)
		ht.idleblocks <- buf
	}
}

// show qps
func (ht *hashTester) Stat() (countstr, qps string, esp time.Duration) {
	ht.endts = time.Now()
	esp = ht.endts.Sub(ht.startts)
	count := ht.count.ToBigInt()
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
	case <-ht.closing:
	default:
		close(ht.closing)
	}
	ht.hashs = nil
	ht.results = nil
	//ht.collided = nil
	//ht.distr = nil
	runtime.GC()
	debug.FreeOSMemory()
	close(ht.closed)
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

	if *runlimit < 60 {
		*runlimit = 60
	}

	//
	allhasher := map[string]cmtp.Checksum{
		"Murmur3": cmtp.NewMurmur3(0),
		//"noop":    cmtp.NewNoopChecksum(0),
		"xxhash": cmtp.NewXxhash(0),
	}
	//
	alldistr := make(map[string][]uint16)
	allcollided := make(map[string][]uint16)
	for idx, onehash := range allhasher {
		misc.Tpf("timelimit %d seconds, size %d, groupsize %d, cpus %d, lock os thread %v, start %s test\n", *runlimit, *size, *groupsize, runtime.GOMAXPROCS(-1), *lock, idx)
		alldistr[idx] = make([]uint16, *groupsize)
		allcollided[idx] = make([]uint16, *groupsize)
		ht := NewhashTester(*size, misc.NewAnyBaseCounter(8), onehash, uint32(*groupsize), int64(*runlimit), *lock)
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
	misc.Tpf("all %d done", len(allhasher))
}

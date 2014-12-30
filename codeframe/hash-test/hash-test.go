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
	pool        []byteHash      //
	poollast    uint64          //
	idleblocks  chan []uint64   //
	procblocks  chan []uint64   //
	countblocks chan []uint64   //
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
		pool:        pool,
		poollast:    initpoolsize - 1,
		idleblocks:  idleblocks,
		procblocks:  make(chan []uint64, size*100),
		countblocks: make(chan []uint64, initpoolsize),
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
	// initial stat
	ht.Stat()
	return ht
}

//
func (ht *hashTester) genbuf() {
	defer func() {
		// all done, close
		close(ht.procblocks)
		ht.limit.Stop()
	}()
	var closing bool
	go func() {
		select {
		case <-ht.limit.C:
			//fmt.Printf("%s\n", "stop for reach time limit")
			closing = true
			return
		case <-ht.closing:
			//fmt.Printf("%s\n", "stop for closing")
			closing = true
			return
		}
	}()
	if ht.lock {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	//newcnt := 0
	//tk := time.NewTicker(1e9)
	//defer tk.Stop()
	//go func() {
	//	for _ = range tk.C {
	//		if newcnt > 0 {
	//			println("new item qps:", newcnt)
	//			newcnt = 0
	//		}
	//	}
	//}()
	var buf []uint64
	for closing == false {
		buf = <-ht.idleblocks
		//select {
		//case buf = <-ht.idleblocks:
		//	//println("got idle buf", buf)
		//default:
		//	if ht.poollast == math.MaxUint64 {
		//		println("blocked in buf = <-ht.idleblocks for ", ht.poollast)
		//		buf = <-ht.idleblocks
		//	} else {
		//		newitem := byteHash{
		//			num: 0,
		//			buf: make([]byte, ht.size),
		//		}
		//		ht.poollast++
		//		ht.pool = append(ht.pool, newitem)
		//		buf = ht.poollast
		//		//println("add new buf", buf)
		//		newcnt++
		//	}
		//}
		for idx, _ := range buf {
			pidx := buf[idx]
			if closing == true {
				// mark unused item
				ht.pool[pidx].buf = nil
			} else {
				ht.generator.FillExpBytes(ht.pool[pidx].buf)
				//fmt.Printf("G: %x\n", ht.pool[pidx].buf)
				// over flow check
				if err := ht.generator.Plus(); err != nil {
					//fmt.Printf("stop for %s\n", err)
					closing = true
				}
			}
		}

		ht.procblocks <- buf

		//select {
		//case ht.procblocks <- buf:
		//default:
		//	println("ht.procblocks <- buf blocking", buf)
		//	ht.procblocks <- buf
		//}

		//// over flow check
		//if err := ht.generator.Plus(); err != nil {
		//	//fmt.Printf("stop for %s\n", err)
		//	return
		//}
	}
}

func (ht *hashTester) dohash(cnt int) {
	defer ht.wg.Done()
	if ht.lock {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}
	for buf := range ht.procblocks {
		for idx, _ := range buf {
			pidx := buf[idx]
			if ht.pool[pidx].buf == nil {
				continue
			}
			//fmt.Printf("H: %x\n", ht.pool[buf].buf)
			ht.pool[pidx].num = ht.hashs[cnt].Checksum32(ht.pool[pidx].buf)
		}
		ht.countblocks <- buf
		//select {
		//case ht.countblocks <- buf:
		//default:
		//	println("ht.countblocks <- buf blocking", buf)
		//	ht.countblocks <- buf
		//}
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
	ht.endts = time.Now()
	//println("end", maxgo, "hasher")
}

func (ht *hashTester) counter() {
	defer func() {
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
			ht.results[ht.pool[pidx].num]++
			bcount++
		}
		ht.count.Mul(bcount)

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
	ht.Result()
	ht.hashs = nil
	ht.results = nil
	ht.pool = nil
	//ht.collided = nil
	//ht.distr = nil
	runtime.GC()
	debug.FreeOSMemory()
	close(ht.closed)
}

// Result return distr/collided map
func (ht *hashTester) Result() ([]uint16, []uint16) {
	select {
	case <-ht.closed:
		// already closed
		return ht.distr, ht.collided
	default:
	}
	//reset
	for idx, _ := range ht.distr {
		ht.distr[idx] = 0
	}
	for idx, _ := range ht.collided {
		ht.collided[idx] = 0
	}
	/*
		results     []uint16        // all hash info
		distr       []uint16        // index by math.MaxUint32 % groupsize
		collided    []uint16        // index by math.MaxUint32 % groupsize
	*/
	for num, _ := range ht.results {
		idx := uint32(num) % ht.groupsize
		ht.distr[idx]++
		if ht.results[num] > 0 {
			ht.collided[idx]++
		}
	}
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
	misc.Tpf("all %d done", len(allhasher))
}

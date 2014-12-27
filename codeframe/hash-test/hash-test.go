package main

import (
	"flag"
	"fmt"
	"math"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync"
	"time"

	"github.com/wheelcomplex/preinit/misc"
)

func show(size int, count *uint64, start time.Time) {
	totalneed := uint64(math.Pow(2, float64(size*8)))
	esp := time.Now().Sub(start)
	qps := (*count * uint64(time.Second) / uint64(esp.Nanoseconds()))
	if totalneed < *count {
		fmt.Printf("size %d, need %d - count %d = %d, esp %v, qps %d\n", size, totalneed, *count, int64(totalneed-*count), esp, qps)
	} else {
		fmt.Printf("size %d, need %d - count %d = %d, esp %v, qps %d\n", size, totalneed, *count, uint64(totalneed-*count), esp, qps)
	}

}

func genbuf(idleblocks, procblocks chan []byte, size int) {
	defer func() {
		// all done, close
		close(procblocks)
	}()
	ab := misc.NewBigCounter(size)
	for {
		buf := <-idleblocks
		//copy(buf, ab.bytes())
		ab.FillBytes(buf)
		//fmt.Printf("%x\n", buf)
		procblocks <- buf
		// over flow check
		if err := ab.Plus(); err != nil {
			//fmt.Printf("%s\n", err)
			return
		}
	}
}

func procbuf(procblocks, idleblocks chan []byte, count, totalProc *uint64) {
	for buf := range procblocks {
		//fmt.Printf("%x\n", buf)
		*count++
		idleblocks <- buf
	}
	*totalProc = *count
}

func main() {
	profileport := flag.Int("port", 6060, "profile http port")
	size := flag.Int("size", 16, "block size")
	cpus := flag.Int("cpu", 1, "cpus")
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
	procblocks := make(chan []byte, 4096)
	idleblocks := make(chan []byte, 4096)
	for i := 0; i < 2048; i++ {
		idleblocks <- make([]byte, *size)
	}

	var wg sync.WaitGroup

	// gen
	wg.Add(1)
	go func() {
		defer wg.Done()
		genbuf(idleblocks, procblocks, *size)
	}()

	// proc
	wg.Add(1)
	var count, totalProc uint64
	go func() {
		defer wg.Done()
		procbuf(procblocks, idleblocks, &count, &totalProc)
	}()
	// show
	stat := time.NewTicker(2e9)
	defer stat.Stop()
	start := time.Now()
	go func() {
		for range stat.C {
			show(*size, &count, start)
		}
	}()
	wg.Wait()
	show(*size, &totalProc, start)
}

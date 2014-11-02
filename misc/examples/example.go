//
// package misc demo
//

package main

import (
	"flag"
	"runtime"
	"time"

	"github.com/wheelcomplex/chanlogger"
	"github.com/wheelcomplex/misc"
)

var l *chanlogger.Clogger

func init() {
	l = chanlogger.NewLogger()
}

func stackIn(stack *misc.Stack, count chan<- int64) {
	var inc int64
	in := stack.In()
	defer func() {
		recover()
		count <- inc
	}()
	for {
		in <- struct{}{}
		inc++
	}
	return
}

func stackOut(stack *misc.Stack, count chan<- int64) {
	var inc int64
	out := stack.Out()
	for _ = range out {
		inc++
	}
	count <- inc
}

func run1() {
	runtime.GOMAXPROCS(*cpuNum)
	in := make(chan struct{}, 1024)
	count := make(chan int64, 1024)

	var to int64 = 0
	// in
	go func() {
		defer func() {
			recover()
		}()
		for {
			in <- struct{}{}
		}
		return
	}()
	// out
	go func() {
		var inc int64
		for _ = range in {
			inc++
		}
		count <- inc
		return
	}()
	for i := int(0); i < (*timeNum); i++ {
		time.Sleep(1e9)
		if i == (*timeNum)-1 {
			l.Print(i % 10)
		} else {
			l.Print(i%10, "-")
		}
	}
	close(in)
	l.Println("-|")
	to = <-count
	l.Println("run1 total out", to, "out QPS", misc.RoundString(to/int64(*timeNum), 10000))
	time.Sleep(1e9)
	return
}

func run2() {
	runtime.GOMAXPROCS(*cpuNum)
	in := make(chan struct{}, 1024)
	out := make(chan struct{}, 1024)
	var to int64 = 0
	// in
	go func() {
		defer func() {
			recover()
		}()
		for {
			in <- struct{}{}
		}
		return
	}()
	// forward
	go func() {
		for v := range in {
			out <- v
		}
		return
	}()
	// out
	var inc int64
	go func() {
		for _ = range out {
			inc++
		}
		return
	}()
	for i := int(0); i < (*timeNum); i++ {
		time.Sleep(1e9)
		if i == (*timeNum)-1 {
			l.Print(i % 10)
		} else {
			l.Print(i%10, "-")
		}
	}
	close(in)
	l.Println("-|")
	to = inc
	l.Println("run2 total out", to, "out QPS", misc.RoundString(to/int64(*timeNum), 10000))
	time.Sleep(1e9)
	return
}

func runStack() {
	ing := 4
	outg := 1
	runtime.GOMAXPROCS(*cpuNum)

	incount := make(chan int64, 1024)
	outcount := make(chan int64, 1024)
	stack := misc.NewStack(1024, 10240, false)
	for i := 0; i < ing; i++ {
		go stackIn(stack, incount)
	}
	for i := 0; i < outg; i++ {
		go stackOut(stack, outcount)
	}
	var ti int64 = 0
	var to int64 = 0
	for i := int(0); i < (*timeNum); i++ {
		time.Sleep(1e9)
		if i == (*timeNum)-1 {
			l.Print(i % 10)
		} else {
			l.Print(i%10, "-")
		}
	}
	l.Println("-|")
	stack.Close()
	for i := 0; i < ing; i++ {
		v := <-incount
		ti = ti + v
	}
	for i := 0; i < outg; i++ {
		v := <-outcount
		to = to + v
	}
	l.Println("runStack total in", ti, "in QPS", misc.RoundString(ti/int64(*timeNum), 10000), "total out", to, "out QPS", misc.RoundString(to/int64(*timeNum), 10000), "in - out =", ti-to)
	time.Sleep(1e9)
}

func runNumber() {
	strs := []string{"a987", "9a87", "987a", "987", "0987"}
	for _, str := range strs {
		l.Printf("misc.IsNumeric(%s): %v\n", str, misc.IsNumeric(str))
	}
	l.Printf("-----\n")
	i := misc.UUID()
	misc.UNUSED(i)
}

func scanner() {
	runtime.GOMAXPROCS(18)

	path := "/"
	l.Printf("folderScanner: %v\n", path)
	folderScanner := misc.NewFolderScanner()
	var dcounter int64 = 0
	var fcounter int64 = 0
	var ecounter int64 = 0
	//folderScanner.SetOutputFilter(true, "^/.*\\.go$")
	//folderScanner.SetOutputFilter(true, "^/.*\\.txt$")
	//folderScanner.SetOutputFilter(true, "^/.*/data$")
	folderScanner.SetScanFilter(false, "^/mnt/.*")
	folderScanner.SetScanFilter(false, "^/proc/.*")
	folderScanner.Recursive = true
	folderScanner.Target = misc.FOLDER_SCAN_ALL
	folderScanner.SetWorker(16)
	pathchan, err := folderScanner.Scan(path)
	if err != nil {
		l.Printf("folderScanner.Scan: %v\n", err)
		return
	}
	defer folderScanner.Close()
	loop := true
	for loop {
		select {
		case newInfo := <-pathchan:
			if newInfo == nil {
				loop = false
				//l.Printf("folderScannerScan Closed: %v\n", path)
			} else {
				switch {
				case newInfo.Err != nil:
					ecounter++
					//l.Printf("E#%d: %v\n", ecounter, newInfo.Err)
					//folderScanner.Close()
				case newInfo.IsFolder == true:
					dcounter++
					//l.Printf("D#%d: %v\n", dcounter, newInfo.Path)
					//l.Printf("%v\n", newInfo.Path)
				default:
					fcounter++
					//l.Printf("F#%d: %v\n", fcounter, newInfo.Path)
					//l.Printf("%v\n", newInfo.Path)
				}
			}
		}
	}
	l.Printf("folderScannerScan, total dir %v file %v, errs %v\n", dcounter, fcounter, ecounter)
}

var mode *string
var loopNum *int
var cpuNum *int
var timeNum *int

func main() {
	mode = flag.String("m", "stack", "run mode(stack/number/scan)")
	loopNum = flag.Int("l", 2, "loop number")
	cpuNum = flag.Int("c", 1, "loop number")
	timeNum = flag.Int("t", 10, "one loop time")
	flag.Parse()
	if *cpuNum < 1 {
		*cpuNum = 1
	}
	if *loopNum < 1 {
		*loopNum = 1
	}
	switch *mode {
	case "stack":
		for i := 0; i < *loopNum; i++ {
			runStack()
			run1()
			run2()
		}
	case "number":
		runNumber()
	case "scan":
		scanner()
	}
}

//
//
//
//
//
//
//
//
//
//
//
//

//
// preinit demo
//

package main

import (
	"time"

	"github.com/wheelcomplex/preinit"
	"github.com/wheelcomplex/preinit/logger"
)

func main() {

	preinit.Usage()

	for i := 0; i < 1000; i++ {
		// test logger
		preinit.Stdout("dup stdout msg")
	}
	preinit.Stdout("no-dup stdout msg")

	Logger := logger.NewLogger("[pre] ", logger.LogFlag)
	//Logger.Calldepth(3)
	Logger.Stderr("Logger msg")
	Logger.SetPrefix("[new-" + preinit.PIDSTR + "] ")
	Logger.Stderr("Logger new msg")

	time.Sleep(100e9)

	preinit.PreExit()
}

/*
			//l4g.Trace("Received message: %s (%d)", "aaaaa", 5)
			//l4g.Debug("Received message: %s (%d)", "aaaaa", 5)
			//l4g.Error("Received message: %s (%d)", "aaaaa", 5)
			fmt.Println("\np0--------------------:", preinit.CmdString())
			//
			cmdline := "-bool1 -bool2 -a a --bl bl -c 0,1,2,3,4,5 f1 f2 f3 f4"
			//cmdline := "--bh -bj"
			fmt.Println("---------------- parse:", cmdline)
			preinit.ParseString(cmdline)
			fmt.Println("p1--------------------:", preinit.CmdString())
			preinit.SetParserFlag("--ooo", "out1")
			preinit.SetParserFlag("--ooo", "out2")
			preinit.SetParserFlag("--fds", "19,20")
			preinit.SetParserFlag("--", "-")
			preinit.SetParserFlag("f6", "f7")
			preinit.SetParserFlag("--bh", "-bj")
			fmt.Println("p2--------------------:", preinit.CmdString())
			preinit.DelParserFlag("f7", "")
			preinit.DelParserFlag("-c", "0")
			preinit.DelParserFlag("-a", "0")
			preinit.DelParserFlag("--bl", "")
			preinit.DelParserFlag("-bj", "")
			fmt.Println("p3--------------------:", preinit.CmdString())
			preinit.Usage()

		preinit.SetVersion("java version \"%s\"", "1.7.0_60")
		preinit.SetDescription(`Java(TM) SE Runtime Environment (build 1.7.0_60-b19)
	Java HotSpot(TM) 64-Bit Server VM (build 24.60-b09, mixed mode)`)
		preinit.SetNotes("java is come from oracle")
		preinit.SetOption("--threads", "9", "set max running thread, 0 for all number of CPU")
		preinit.SetOptions("--fds", []string{"6", "7", "8", "9"}, "set file description pass to children")
		preinit.SetFlag("--foreground", "run app in foreground, default is run in background(daemon)")
		preinit.SetFlag("-F", "run app in foreground, default is run in background(daemon)")
		preinit.SetFlag("--", "test --, run app in foreground, default is run in background(daemon)")
		preinit.SetNoFlags([]string{"f11", "f42", "f3", "f14"}, "file list to compress")
		fmt.Println("p4--------------------:", preinit.CmdString())
		preinit.Usage()
		fmt.Println("")
		fmt.Println("")
		fmt.Println("lists:", preinit.GetNoFlagString())
		fmt.Println("--threads:", preinit.GetInt("--threads"))
		fmt.Println("--fds:", preinit.GetInts("--fds"))
		// no defined option
		fmt.Println("--nodef:", preinit.GetString("--nodef"))
		fmt.Println("--nodefs:", preinit.GetStringList("--nodefs"))
		fmt.Println("p6--------------------:", preinit.CmdString())
		fmt.Println(os.Args)
			t1 := "M: " + preinit.ArgLine
			preinit.SetProcTitle(t1)
			fmt.Println("p7--------------------")
			fmt.Println(os.Args)
			fmt.Println("p8--------------------")
			t2 := "W: " + preinit.ArgFullLine

	preinit.SetProcTitlePrefix("Master: ")
	fmt.Println("p9--------------------")

	for i := 0; i < 1000; i++ {
		// test logger
		preinit.Stdout("dup stdout msg")
	}
	preinit.Stdout("no-dup stdout msg")

	//Logger := logger.NewLogger("[pre] ", logger.LogFlag)
	////Logger.Calldepth(4)
	//Logger.Stdout("Logger msg")
	//Logger.SetPrefix("[new]")
	//Logger.Stdout("Logger new msg")

	fds := preinit.GetOpenListOfPid(preinit.PID)
	preinit.Errlogf("NumGoroutine %d, opened fd: %v\n", runtime.NumGoroutine(), fds)
	for _, f := range fds {
		preinit.Errlogf("\t%d -> %s\n", f.Fd(), f.Name())
	}

*/

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

//
// preinit demo
//

package main

import (
	"fmt"
	"os"
	"time"

	pre "github.com/wheelcomplex/preinit"
	"github.com/wheelcomplex/preinit/misc"
	//log "github.com/wheelcomplex/preinit/logger"

	//l4g "github.com/wheelcomplex/preinit/log4go"
)

func main() {
	/*
		//l4g.Trace("Received message: %s (%d)", "aaaaa", 5)
		//l4g.Debug("Received message: %s (%d)", "aaaaa", 5)
		//l4g.Error("Received message: %s (%d)", "aaaaa", 5)
		fmt.Println("\np0--------------------:", pre.CmdString())
		//
		cmdline := "-bool1 -bool2 -a a --bl bl -c 0,1,2,3,4,5 f1 f2 f3 f4"
		//cmdline := "--bh -bj"
		fmt.Println("---------------- parse:", cmdline)
		pre.ParseString(cmdline)
		fmt.Println("p1--------------------:", pre.CmdString())
		pre.SetParserFlag("--ooo", "out1")
		pre.SetParserFlag("--ooo", "out2")
		pre.SetParserFlag("--fds", "19,20")
		pre.SetParserFlag("--", "-")
		pre.SetParserFlag("f6", "f7")
		pre.SetParserFlag("--bh", "-bj")
		fmt.Println("p2--------------------:", pre.CmdString())
		pre.DelParserFlag("f7", "")
		pre.DelParserFlag("-c", "0")
		pre.DelParserFlag("-a", "0")
		pre.DelParserFlag("--bl", "")
		pre.DelParserFlag("-bj", "")
		fmt.Println("p3--------------------:", pre.CmdString())
		pre.Usage()
	*/
	pre.SetVersion("java version \"%s\"", "1.7.0_60")
	pre.SetDescription(`Java(TM) SE Runtime Environment (build 1.7.0_60-b19)
Java HotSpot(TM) 64-Bit Server VM (build 24.60-b09, mixed mode)`)
	pre.SetNotes("java is come from oracle")
	pre.SetOption("--thread", "9", "set max running thread, 0 for all number of CPU")
	pre.SetOptions("--fds", []string{"6", "7", "8", "9"}, "set file description pass to children")
	pre.SetFlag("--foreground", "run app in foreground, default is run in background(daemon)")
	pre.SetFlag("-F", "run app in foreground, default is run in background(daemon)")
	pre.SetFlag("--", "test --, run app in foreground, default is run in background(daemon)")
	//pre.SetNoFlags([]string{"f11", "f2", "f3", "f14"}, "file list to compress")
	//fmt.Println("p4--------------------:", pre.CmdString())
	pre.Usage()
	fmt.Println("")
	fmt.Println("")
	fmt.Println("lists:", pre.GetNoFlagString())
	fmt.Println("--thread:", pre.GetInt("--thread"))
	fmt.Println("--fds:", pre.GetInts("--fds"))
	// no defined option
	fmt.Println("--nodef:", pre.GetString("--nodef"))
	fmt.Println("--nodefs:", pre.GetStringList("--nodefs"))
	fmt.Println("p6--------------------:", pre.CmdString())
	fmt.Println(os.Args)
	/*
		t1 := "M: " + pre.ArgLine
		pre.SetProcTitle(t1)
		fmt.Println("p7--------------------")
		fmt.Println(os.Args)
		fmt.Println("p8--------------------")
		t2 := "W: " + pre.ArgFullLine
	*/
	pre.SetProcTitlePrefix("Master: ")
	fmt.Println("p9--------------------")
	// Mon Jan 2 15:04:05 -0700 MST 2006
	// 2006-01-02-15-04-MST
	format := "2006-XXX-01"
	format = ""
	starts := time.Time{}
	nt := misc.TimeFormatNext(format, starts)
	fmt.Println("start from", starts, "next is", nt, "for", format)

	time.Sleep(100e9)
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

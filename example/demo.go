//
// preinit demo
//

package main

import (
	"fmt"

	opt "github.com/wheelcomplex/preinit/options"
	//l4g "github.com/wheelcomplex/preinit/log4go"
)

func main() {
	//l4g.Trace("Received message: %s (%d)", "aaaaa", 5)
	//l4g.Debug("Received message: %s (%d)", "aaaaa", 5)
	//l4g.Error("Received message: %s (%d)", "aaaaa", 5)
	fmt.Println("\np0--------------------:", opt.String())
	//
	cmdline := "-bool1 -bool2 -a a --bl bl -c 0,1,2,3,4,5 f1 f2 f3 f4"
	//cmdline := "--bh -bj"
	fmt.Println("---------------- parse:", cmdline)
	opt.ParseString(cmdline)
	fmt.Println("p1--------------------:", opt.String())
	opt.SetParserFlag("--ooo", "out1")
	opt.SetParserFlag("--ooo", "out2")
	opt.SetParserFlag("--fds", "19,20")
	opt.SetParserFlag("--", "-")
	opt.SetParserFlag("f6", "f7")
	opt.SetParserFlag("--bh", "-bj")
	fmt.Println("p2--------------------:", opt.String())
	opt.DelParserFlag("f7", "")
	opt.DelParserFlag("-c", "0")
	opt.DelParserFlag("-a", "0")
	opt.DelParserFlag("--bl", "")
	opt.DelParserFlag("-bj", "")
	fmt.Println("p3--------------------:", opt.String())
	opt.Usage()
	opt.SetVersion("java version \"%s\"", "1.7.0_60")
	opt.SetDescription(`Java(TM) SE Runtime Environment (build 1.7.0_60-b19)
Java HotSpot(TM) 64-Bit Server VM (build 24.60-b09, mixed mode)`)
	opt.SetNotes("java is come from oracle")
	opt.SetOption("--thread", "9", "set max running thread, 0 for all number of CPU")
	opt.SetOptions("--fds", []string{"6", "7", "8", "9"}, "set file description pass to children")
	opt.SetFlag("--foreground", "run app in foreground, default is run in background(daemon)")
	opt.SetFlag("-F", "run app in foreground, default is run in background(daemon)")
	opt.SetFlag("--", "test --, run app in foreground, default is run in background(daemon)")
	opt.SetNoFlags([]string{"f11", "f2", "f3", "f14"}, "file list to compress")
	fmt.Println("p4--------------------:", opt.String())
	opt.Usage()
	fmt.Println("p5--------------------")
	fmt.Println("lists:", opt.GetNoFlagString())
	fmt.Println("--thread:", opt.GetInt("--thread"))
	fmt.Println("--fds:", opt.GetInts("--fds"))
	// no defined option
	fmt.Println("--nodef:", opt.GetString("--nodef"))
	fmt.Println("--nodefs:", opt.GetStringList("--nodefs"))
	fmt.Println("p6--------------------:", opt.String())
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

// Package preinit provides base utils for go daemon programing.
/*
3. listen master(fork+fd pass)
*/

package preinit

/*

#include "setproctitle.h"

*/
import "C"

// C.spt_init1 defined in setproctitle.h
import (
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"github.com/wheelcomplex/preinit/logger"
	"github.com/wheelcomplex/preinit/options"
)

// SetGoMaxCPUs set runtime.GOMAXPROCS, -1 for use all cpus, 0 for use cpus - 1, other for use N cpus
// at less one cpu
// return final using cpus
func SetGoMaxCPUs(n int) int {
	switch {
	case n <= -1:
		runtime.GOMAXPROCS(runtime.NumCPU())
	case n == 0:
		if runtime.NumCPU() > 1 {
			runtime.GOMAXPROCS(runtime.NumCPU() - 1)
		} else {
			runtime.GOMAXPROCS(1)
		}
	default:
		runtime.GOMAXPROCS(n)
	}
	return runtime.GOMAXPROCS(-1)
}

//// VARS ////

// default logger for preinit
var Logger *logger.LoggerT

//
func loggerInit() {
	Logger = logger.NewLogger("[pre-"+strconv.Itoa(os.Getpid())+"] ", logger.LogFlag)
	// set to 5, all call by wrapper
	Logger.Calldepth(5)
}

// copy of os.Args for default arguments parser
var Args []string

// convert Args([]string) to line string
var ArgLine string

// command line args with default flags/options
var ArgFullLine string

// convert Args[0] to absolute file path
var ExecFile string

// default opt Parser
// default opt Parser
// do not include ExecFile
var opts = options.NewOpts(os.Args[1:])

// initial default command line parser
func argsInit() {
	Args = make([]string, 0, 0)
	Args = append(Args, os.Args...)
	ExecFile = options.GetExecFileByPid(os.Getpid())
	ArgLine = options.ArgsToSpLine(Args)
	ArgFullLine = options.CleanArgLine(os.Args[0] + " " + opts.String())
	//
}

/// end of command line parser

// PreExit prepare to exit
// will close log channel
func PreExit() {
	Logger.Close()
}

// CleanExit close all know fd/socket and sync, and exit
func CleanExit(code int) {
	os.Stdout.Sync()
	os.Stderr.Sync()
	PreExit()
	os.Exit(code)
}

// autoAppDir return dir string base on prefix or executing file
func autoAppDir(prefix, suffix string) string {
	var dir string
	suffix = SafeFileName(suffix)
	prefix = SafeFileName(prefix)
	if prefix == "" {
		fpath := options.GetExecFileByPid(os.Getpid())
		pwd := fpath
		fpath = path.Base(path.Dir(strings.TrimRight(pwd, "/")))
		if fpath == "bin" || fpath == "sbin" {
			dir = pwd + "/../" + suffix + "/"
		} else {
			dir = pwd + "/./" + suffix + "/"
		}
	} else {
		dir = prefix + "/./" + suffix + "/"
	}
	return path.Clean(dir)
}

// GetForkState return current state of daemon fork(forkState)
func GetForkState() ForkStateT {
	return forkState
}

// SetForkState set forkState and return old state of daemon fork(forkState)
func SetForkState(state ForkStateT) ForkStateT {
	old := forkState
	forkState = state
	return old
}

// type of fork state
type ForkStateT int

// state of daemon fork
const (
	FORK_UNSET ForkStateT = iota
	FORK_CHROOT
	FORK_PARENT
	FORK_INTERNAL
	FORK_DISPATCHER
	FORK_WORKER
	FORK_UTIL
)

// name of ForkStateT
var forkStrings = map[ForkStateT]string{
	FORK_UNSET:      "unset",
	FORK_CHROOT:     "chroot",
	FORK_PARENT:     "parent",
	FORK_INTERNAL:   "internal",
	FORK_DISPATCHER: "dispatcher",
	FORK_WORKER:     "worker",
	FORK_UTIL:       "util",
}

// String return name of for state
// return empty string for invalid state
func (f *ForkStateT) String() string {
	if _, ok := forkStrings[*f]; ok {
		return forkStrings[*f]
	}
	return ""
}

// PID of proc
var PID int

// PID string of proc
var PIDSTR string

// internal state of daemon
var forkState ForkStateT

//
// TODO: convert fd to net listener
// https://groups.google.com/forum/#!msg/golang-nuts/Ws09uN64I80/JBs7753OkWsJ
// http://stackoverflow.com/questions/17193086/how-to-pass-net-listeners-fd-to-child-process-safely/
// http://grisha.org/blog/2014/06/03/graceful-restart-in-golang/

/*

3. daemon+UID/GID
4. signal
5. fdpass+drop privileges
6. children monitor
7. pidfile // lockfile
8. add/del env var

http://golang.org/pkg/runtime/#LockOSThread

http://golang.org/pkg/runtime/#UnlockOSThread

https://code.google.com/p/go/source/detail?r=a25343ee3016

http://golang.org/pkg/os/#ProcAttr

https://github.com/vbatts/go-cgroup
http://blog.chinaunix.net/uid-20164485-id-3253720.html
cat /cgroup/cpu/daenmons/http/tasks   //受控的PID列表

https://groups.google.com/forum/#!topic/golang-nuts/ZHzaQvjH4TA

Your post inspired me to rewrite my "nschroot" tool in Go and it works fine. I found most of what I needed by sniffing around in the syscall package source.

https://github.com/tobert/lnxns

I'm not sure if the syscall.ForkLock.Lock() is necessary, but from reading syscall/exec_unix.go, it sounded like a good idea. http://golang.org/src/pkg/syscall/exec_unix.go?s=6845:6910#L180

I couldn't find any good information on the dangers of running after fork() in go. In my call to clone() I'm careful not to share any more of the process than is necessary so it should be fairly safe to continue doing things, but I haven't tried it yet since nschroot execs right away. It's likely the GC/CoW interactions will use up a little extra memory, but that's normal for fork(). All the usual rules of fork() apply.

Putting children into cgroups can be done by a double fork. Fork once, add that pid to the cgroup's tasks file, then fork your real work inside the namespace with CLONE_PARENT and let the middle child exit.


1. patch go source code to enable/implate setuid/setgid/seteuid/setegid
2. parrent create cgroup for child and fork to --forkhelper
3. forkhelper LockOSThread
4. forkhelper setuid and call os/exec/startproc
5. forkhelper os.Exit(0)
6. parrent get all pid from cgroup

The easiest way is to pass the listener in the ExtraFiles field of exec.Cmd.

Example of parent:

var l *net.TCPListener
cmd := exec.Command(...)
f, err := l.File()
cmd.ExtraFiles = []*os.File{f}

Example of child:

l, err := net.FileListener(os.NewFile(3, "listener"))

You may also want to generalize this and have the child accept PROGRAMNAME_LISTENER_FD as an
environment variable. Then the parent would set the environment variable to 3 before starting the child.

// use net.FilePacketConn for UDPConn
// http://golang.org/pkg/net/#FilePacketConn

*/

// pre defined dirs
// proc working directory will be "root" dir
var preDirs = map[string]string{
	"root": "./",
	"var":  "./var",
	"log":  "./log",
	"run":  "./run",
	"tmp":  "./tmp",
	"data": "./data",
}

// updatePreDirs set key,value to preDirs
func updatePreDirs(key, value string) {
	if key == "" || value == "" {
		return
	}
	preDirs[key] = value
}

// SetProcTitle

const (
	// These values must match the return values for spt_init1() used in C.
	HaveNone        = 0
	HaveNative      = 1
	HaveReplacement = 2
)

var (
	HaveSetProcTitle int
)

// orig proc title
var OrigProcTitle string

//
func setproctitle_init() {
	if len(OrigProcTitle) == 0 {
		OrigProcTitle = CleanArgLine(os.Args[0] + " " + opts.String())
	}
	HaveSetProcTitle = int(C.spt_init1())

	if HaveSetProcTitle == HaveReplacement {
		newArgs := make([]string, len(os.Args))
		for i, s := range os.Args {
			// Use cgo to force go to make copies of the strings.
			cs := C.CString(s)
			newArgs[i] = C.GoString(cs)
			C.free(unsafe.Pointer(cs))
		}
		os.Args = newArgs

		env := os.Environ()
		for _, kv := range env {
			skv := strings.SplitN(kv, "=", 2)
			os.Setenv(skv[0], skv[1])
		}

		argc := C.int(len(os.Args))
		arg0 := C.CString(os.Args[0])
		defer C.free(unsafe.Pointer(arg0))

		C.spt_init2(argc, arg0)

		// Restore the original title.
		SetProcTitle(os.Args[0])
	}
}

func SetProcTitle(title string) {
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	C.spt_setproctitle(cs)
}

func SetProcTitlePrefix(prefix string) {
	title := prefix + OrigProcTitle
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	C.spt_setproctitle(cs)
}

func SetProcTitleSuffix(prefix string) {
	title := OrigProcTitle + prefix
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	C.spt_setproctitle(cs)
}

/*
nginx SetProcTitle:

root      1861  0.0  0.0  90220  1492 ?        Ss   Oct20   0:00 nginx: master process /usr/sbin/nginx -c /etc/nginx/nginx.conf
www-data  1862  0.0  0.0  90500  2312 ?        S    Oct20   0:00 nginx: worker process
www-data  1863  0.0  0.0  90500  2056 ?        S    Oct20   0:07 nginx: worker process
www-data  1864  0.0  0.0  90500  2056 ?        S    Oct20   0:07 nginx: worker process
www-data  1866  0.0  0.0  90500  2056 ?        S    Oct20   0:07 nginx: worker process

*/

// end of SetProcTitle

var preCfg = &preCfgT{}

func init() {
	setproctitle_init()
	SetForkState(FORK_PARENT)
	PID = os.Getpid()
	PIDSTR = strconv.Itoa(PID)
	loggerInit()
	argsInit()
	//
	// TODO: proc args and add debug/app logfile, fdpass(tcp/udp)
	/*
		// autoAppDir
		// log/ var/ run/ tmp/
		--chroot pah/to/chroot/dir
		--user username/id
		--group groupname/id
		--rootdir pah/to/root/dir
		--vardir pah/to/var/dir
		--logdir pah/to/log/dir
		--rundir pah/to/run/dir
		--tmpdir pah/to/tmp/dir
		--datadir pah/to/data/dir
		--errlogfile suffix log/ if no abs path
		--applogfile suffix log/ if no abs path
		--debuglogfile suffix log/ if no abs path
		--logrotation default 10
		--logmaxsize default 0
		--logmaxline default 0
		--ident prefix/string/to/proctitle/lock
		--threads runtime.GOMAXPROCS(n int)/runtime.NumCPU()
		--daemon run in background, default foreground
		--forkstate chroot/internal/dispatcher/worker/util
		--respawn respawn worker if aborted, respawndelay=5, respawnmax=0
		--listens :8080,:1918,udp:eth0:53,raw:eth1,unix:/tmp/socket.pipe for worker
		--fds name/list/of/fds,from --listens

	*/

	type preCfgT struct {
		chroot       string
		user         string
		group        string
		rootdir      string
		vardir       string
		rundir       string
		tmpdir       string
		datadir      string
		logdir       string
		errlogfile   string
		applogfile   string
		debuglogfile string
		logrotation  int
		logmaxsize   int
		logmaxline   int
		ident        string
		threads      string
		respawn      bool
		respawndelay int
		respawnmax   int
		forkstate    string
		listens      string
		fds          int
		daemon       bool
		help         bool
	}
	// Opts
	opts.SetVersion("Go lang package preinit, version \"%s\"", "0.0.1")
	opts.SetDescription(`Provides utils for go daemon programing.
such as daemonize, proc respawn, drop privileges of proc, pass FDs to child proc.`)

	opts.SetOption("--pr-chroot", "", "(available for root only)set proc chroot directory, proc will chroot befor do any thing, default: no chroot")
	opts.SetOption("--pr-user", "www-data", "(available for root only)set dispatcher/worker running user name or user id, empty to run as current user")
	opts.SetOption("--pr-group", "www-data", "(available for root only)set dispatcher/worker running group name or group id, empty to run as group of --user")
	opts.SetOption("--pr-rootdir", "", "set proc working directory, by default, log/ var/ run/ tmp/ base on this directory, default: current directory")
	opts.SetOption("--pr-vardir", "", "set proc var directory, default: --rootdir + /var/")
	opts.SetOption("--pr-rundir", "", "set proc run directory, default: --rootdir + /run/")
	opts.SetOption("--pr-tmpdir", "", "set proc data directory, default: --rootdir + /tmp/")
	opts.SetOption("--pr-datadir", "", "set proc data directory, default: --rootdir + /data/")
	opts.SetOption("--pr-logdir", "", "set proc log directory, default: --rootdir + /log/")
	opts.SetOption("--pr-errlogfile", "", "set proc error log filename, if path is not absolute, file will be --logdir + logfile, default: disable error logging")
	opts.SetOption("--pr-applogfile", "", "set proc app log file name, if path is not absolute, file will be --logdir + logfile, default: disable app logging")
	opts.SetOption("--pr-debuglogfile", "", "set proc debug log file name, if path is not absolute, file will be --logdir + logfile, default: disable debug logging")
	opts.SetOption("--pr-logrotation", "10", "set proc logging rotation, existed logfile will be overwrited")
	opts.SetOption("--pr-logmaxsize", "2G", "set proc max logfile size, K/M/G suffix is identifyed, zero to disable file size rotation")
	opts.SetOption("--pr-logmaxline", "2G", "set proc max logfile line, K/M/G suffix is identifyed, zero to disable file line rotation")

	opts.SetOption("--pr-ident", "", "set prefix to proctitle, new title will be ident: orig-title, default: disable title prefix")
	opts.SetOption("--pr-threads", "0", "set max running thread(GOMAXPROCS), -1 for all number of CPUs, 0 for CPUs - 1(at less 1)")

	opts.SetOption("--pr-respawn", "true", "respawning for dispatcher/worker, default: true")
	opts.SetOption("--pr-respawndelay", "5", "delay seconds befor respawn dispatcher/worker, at less one second")
	opts.SetOption("--pr-respawnmax", "0", "max time of respawn dispatcher/worker, zero for always respawn")
	opts.SetOption("--pr-forkstate", "", "state of proc fork, default: parent")
	opts.SetOption("--pr-listens", "", "pre-listen list for dispatcher/worker, format: [proto:][addr/nic:]port/path, default proto is tcp, proto raw for rawsocket, default addr is any, multi-listen split by ',', eg,. :8080,udp:eth0:53,raw:eth1,unix:/tmp/socket.pipe, default: no pre-listen")
	opts.SetOption("--pr-fds", "0", "number of pre-listen FDs pass from parent to dispatcher/worker")

	opts.SetFlag("--pr-daemon", "run proc as daemon")
	opts.SetFlag("--pr-help", "show help of preinit options")

	opts.SetNotes("this is internal command line args to contorl Go lang proc")
	//
	// TODO: here
	//println("opts.init() end.")

	/*
		// remote run ? push local binary to remote host which running a exec-agent, run proc in jail, tier all stdin/stdout/stderr/errlog/applog/debuglog back to contorller by contorl connection(s)
	*/
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
//

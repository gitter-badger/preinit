// Package preinit provides utils for go daemon programing.
/*

1. log -- wrapper
2. args -- wrapper
*/

package preinit

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/wheelcomplex/preinit/logger"
	"github.com/wheelcomplex/preinit/options"
)

// misc functions

// http://godoc.org/github.com/sluu99/uuid fromstr

// TimeFormatNext find next time.Time of format
// if from == time.Time{}, from = time.Now()
// return next time.Time or time.Time{} for no next avaible
func TimeFormatNext(format string, from time.Time) time.Time {
	var nextT time.Time
	if format == "" {
		return nextT
	}
	// Mon Jan 2 15:04:05 -0700 MST 2006
	// 2006-01-02-15-04-MST
	/*
		"Nanosecond",
		"Microsecond",
		"Millisecond",
		"Second",
		"Minute",
		"Hour",
		"Day",
		"Week",
		"Month1",
		"Month2",
		"Month3",
		"Month4",
		"year1",
		"year2",
	*/
	//
	timeSteps := []time.Duration{
		time.Nanosecond,
		time.Microsecond,
		time.Millisecond,
		time.Second,
		time.Minute,
		time.Hour,
		time.Hour * 24,
		time.Hour * 24 * 7,
		time.Hour * 24 * 28,
		time.Hour * 24 * 29,
		time.Hour * 24 * 30,
		time.Hour * 24 * 31,
		time.Hour * 24 * 365,
		time.Hour * 24 * 366,
	}
	if from.Equal(time.Time{}) {
		from = time.Now()
	}
	// cut to current format ts
	nowts, err := time.Parse(format, from.Format(format))
	//fmt.Printf("FORMAT: %v, FROM: %v || %v, CUT: %v || %v\n", format, from.Format(format), from, nowts.Format(format), nowts)
	if err != nil {
		// invalid format
		//fmt.Fprintf(os.Stderr, "TimeFormatNext: invalid format: %s\n", format)
		return nextT
	}
	nowstr := nowts.Format(format)
	for _, val := range timeSteps {
		nextT = nowts.Add(val)
		if nowstr != nextT.Format(format) {
			return nextT
		}
	}
	return nextT
}

// SetGoMaxCPUs set runtime.GOMAXPROCS, -1 for use all cpus, 0 for use cpus - 1, other for use N cpus
// at using less one cpu
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

/// multi-channel logger wrapper

// NewRotFile create new WriteCloser for logger
func NewRotFile(filename string, mode os.FileMode, max int, size int, line int, format string) *logger.RotFile_t {
	return logger.NewRotFile(filename, mode, max, size, line, format)

}

// NewTrash ceate dummy WriteCloser for logger
func NewTrash() *logger.Trash_t {
	return logger.NewTrash()
}

func AddListlog(name string, output io.Writer) {
	Logger.AddListlog(name, output)
}

func AddListlogCloser(name string, output io.WriteCloser) {
	Logger.AddListlogCloser(name, output)
}

func GetListLog() map[string]struct{} {
	return Logger.GetListLog()
}

func RemoveListLog(list []string) {
	Logger.RemoveListLog(list)
}

func Applog(v ...interface{}) {
	Logger.Applog(v...)
}

func Applogf(format string, v ...interface{}) {
	Logger.Applogf(format, v...)
}

func Applogln(v ...interface{}) {
	Logger.Applogln(v...)
}

func CloseLogger() {
	Logger.Close()
}

func CloseLogChannel(name string) {
	Logger.CloseLogChannel(name)
}

func Errlog(v ...interface{}) {
	Logger.Errlog(v...)
}

func Errlogf(format string, v ...interface{}) {
	Logger.Errlogf(format, v...)
}

func Errlogln(v ...interface{}) {
	Logger.Errlogln(v...)
}

func Fatal(v ...interface{}) {
	Logger.Fatal(v...)
}

func Fatalf(format string, v ...interface{}) {
	Logger.Fatalf(format, v...)
}

func Fatalln(v ...interface{}) {
	Logger.Fatalln(v...)
}

func Listlog(v ...interface{}) {
	Logger.Listlog(v...)
}

func Listlogf(format string, v ...interface{}) {
	Logger.Listlogf(format, v...)
}

func Listlogln(v ...interface{}) {
	Logger.Listlogln(v...)
}

func Panic(v ...interface{}) {
	Logger.Panic(v...)
}

func Panicf(format string, v ...interface{}) {
	Logger.Panicf(format, v...)
}

func Panicln(v ...interface{}) {
	Logger.Panicln(v...)
}

func Prefix() string {
	return Logger.Prefix()
}

func Print(v ...interface{}) {
	Logger.Print(v...)
}

func Printf(format string, v ...interface{}) {
	Logger.Printf(format, v...)
}

func Println(v ...interface{}) {
	Logger.Println(v...)
}

func SetFlags(flag int) {
	Logger.SetFlags(flag)
}

func SetPrefix(prefix string) {
	Logger.SetPrefix("[" + prefix + "-" + strconv.Itoa(os.Getpid()) + "] ")
}

func SetWriteCloser(name string, output io.WriteCloser) {
	Logger.SetWriteCloser(name, output)
}

func SetWriter(name string, output io.Writer) {
	Logger.SetWriter(name, output)
}

func Stderr(v ...interface{}) {
	Logger.Stderr(v...)
}

func Stderrf(format string, v ...interface{}) {
	Logger.Stderrf(format, v...)
}

func Stderrln(v ...interface{}) {
	Logger.Stderrln(v...)
}

func Stdout(v ...interface{}) {
	Logger.Stdout(v...)
}

func Stdoutf(format string, v ...interface{}) {
	Logger.Stdoutf(format, v...)
}

func Stdoutln(v ...interface{}) {
	Logger.Stdoutln(v...)
}

func Syslog(v ...interface{}) {
	Logger.Syslog(v...)
}

func Syslogf(format string, v ...interface{}) {
	Logger.Syslogf(format, v...)
}

func Syslogln(v ...interface{}) {
	Logger.Syslogln(v...)
}

// Debug write msg to debug
func Debug(v ...interface{}) {
	Logger.Debug(v...)
}

func Debugf(format string, v ...interface{}) {
	Logger.Debugf(format, v...)
}

func Debugln(v ...interface{}) {
	Logger.Debugln(v...)
}

func WriteToList(names []string, v string) {
	Logger.WriteToList(names, v)
}

/// end of multi-channel logger

/// command line parser

// wrapper of options.func
func SetProcTitle(title string) {
	options.SetProcTitle(title)
}

// wrapper of options.func
func SetProcTitlePrefix(prefix string) {
	options.SetProcTitlePrefix(prefix)
}

// wrapper of options.func
func SetProcTitleSuffix(suffix string) {
	options.SetProcTitleSuffix(suffix)
}

// wrapper of options.CmdString()
func CmdString() string {
	return opts.CmdString()
}

// wrapper of options.func
func ArgsString() string {
	return opts.ArgsString()
}

// wrapper of options.func
func SetVersion(format string, a ...interface{}) string {
	return opts.SetVersion(format, a...)
}

// wrapper of options.func
func SetDescription(format string, a ...interface{}) string {
	return opts.SetDescription(format, a...)
}

// wrapper of options.func
func SetNotes(format string, a ...interface{}) string {
	return opts.SetNotes(format, a...)
}

// Powered set powered string of usage
// empty val to return current string
func Powered(val string) string {
	return opts.Powered(val)
}

// wrapper of options.func
func SetOption(long string, defstring string, format string, a ...interface{}) string {
	return opts.SetOption(long, defstring, format, a...)
}

// wrapper of options.func
func SetOptions(long string, defval []string, format string, a ...interface{}) string {
	return opts.SetOptions(long, defval, format, a...)
}

// wrapper of options.func
func SetFlag(long string, format string, a ...interface{}) string {
	return opts.SetFlag(long, format, a...)
}

// wrapper of options.func
func SetNoFlags(defval []string, format string, a ...interface{}) string {
	return opts.SetNoFlags(defval, format, a...)
}

// wrapper of options.func
func VersionString() string {
	return opts.VersionString()
}

// wrapper of options.func
func DescriptionString() string {
	return opts.DescriptionString()
}

// wrapper of options.func
func NoteString() string {
	return opts.NoteString()
}

// wrapper of options.func
func OptionString() string {
	return opts.OptionString()
}

// wrapper of options.func
func FlagString() string {
	return opts.FlagString()
}

// wrapper of options.func
func NoFlagString() string {
	return opts.NoFlagString()
}

// wrapper of options.func
func CommandString() string {
	return opts.CommandString()
}

// wrapper of options.func
func UsageString() string {
	return opts.UsageString()
}

// wrapper of options.func
func Usage() {
	// TODO: show only user defined option when command line without --preinit
	// split use opt from --preinit opt
	// use --pre prefix for all pre option
	opts.Usage()
}

// wrapper of opts.Parse
func Parse(args []string) {
	opts.Parse(args)
}

// wrapper of opts.ParseString
func ParseString(line string) {
	opts.ParseString(line)
}

// wrapper of options.func
func GetParserNoFlags() []string {
	return opts.GetParserNoFlags()
}

// wrapper of options.func
func GetParserNoFlagString() string {
	return opts.GetParserNoFlagString()
}

// wrapper of options.func
func GetNoFlags() []string {
	return opts.GetNoFlags()
}

// wrapper of options.func
func GetNoFlagString() string {
	return opts.GetNoFlagString()
}

// wrapper of options.func
func GetStringList(flag string) []string {
	return opts.GetStringList(flag)
}

// wrapper of options.func
func GetString(flag string) string {
	return opts.GetString(flag)
}

// wrapper of options.func
func GetStrings(flag string) string {
	return opts.GetStrings(flag)
}

// wrapper of options.func
func GetInt(flag string) int {
	return opts.GetInt(flag)
}

// wrapper of options.func
func GetInts(flag string) []int {
	return opts.GetInts(flag)
}

// wrapper of options.func
func GetBool(flag string) bool {
	return opts.GetBool(flag)
}

// wrapper of options.func
func GetFlag(flag string) bool {
	return opts.GetFlag(flag)
}

// wrapper of options.func
func DelParserFlag(key, value string) {
	opts.DelParserFlag(key, value)
}

// wrapper of options.func
func SetParserFlag(key, value string) {
	opts.SetParserFlag(key, value)
}

/// end of command line parser

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

// orig proc title
var OrigProcTitle string

// default opt Parser
var opts *options.OptParser_t

// initial default command line parser
func argsInit() {
	Args = make([]string, 0, 0)
	Args = append(Args, os.Args...)
	ExecFile = options.GetExecFileByPid(os.Getpid())
	// default opt Parser
	// do not include ExecFile
	opts = options.NewOptParser(Args[1:])
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

// GetOpenListOfPid return opened file list of pid
// return empty map if no file opened
func GetOpenListOfPid(pid int) []*os.File {
	var err error
	var file *os.File
	var filelist []string

	fds := make([]*os.File, 0, 0)

	file, err = os.Open("/proc/" + strconv.Itoa(pid) + "/fd/")
	if err != nil {
		Logger.Errlogf("ERROR: %s\n", err)
		return fds
	}
	defer file.Close()

	filelist, err = file.Readdirnames(1024)
	if err != nil {
		if err == io.EOF {
			Logger.Errlogf("read dir end: %s, %v\n", err, filelist)
		} else {
			Logger.Errlogf("ERROR: %s\n", err)
			return fds
		}
	}
	/*
		rhinofly@rhinofly-Y570:~/data/liteide/build$ ls -l /proc/self/fd/
		total 0
		lrwx------ 1 rhinofly rhinofly 64 Nov  4 09:47 0 -> /dev/pts/16
		lrwx------ 1 rhinofly rhinofly 64 Nov  4 09:47 1 -> /dev/pts/16
		lrwx------ 1 rhinofly rhinofly 64 Nov  4 09:47 2 -> /dev/pts/16
		lr-x------ 1 rhinofly rhinofly 64 Nov  4 09:47 3 -> /proc/29484/fd
	*/
	tmpid := strconv.Itoa(int(file.Fd()))
	if len(filelist) > 0 {
		for idx := range filelist {
			//func NewFile(fd uintptr, name string) *File
			link, _ := os.Readlink("/proc/" + strconv.Itoa(pid) + "/fd/" + filelist[idx])
			if filelist[idx] == tmpid {
				//Logger.Errlogf("file in %d dir: %d, %v, link %s, is me %v\n", pid, idx, filelist[idx], link, file.Name())
				continue
			}
			//Logger.Errlogf("file in %d dir: %d, %v -> %s\n", pid, idx, filelist[idx], link)
			fd, err := strconv.Atoi(filelist[idx])
			if err != nil {
				Logger.Errlogf("strconv.Atoi(%v): %s\n", filelist[idx], err)
				continue
			}
			fds = append(fds, os.NewFile(uintptr(fd), link))
		}
	}
	return fds
}

// base command line args process

// SafeFileName replace invalid char with _
// valid char is . 0-9 _ - A-Z a-Z /
func SafeFileName(name string) string {
	name = filepath.Clean(name)
	newname := make([]byte, 0, len(name))
	for _, val := range name {
		if (val >= '0' && val <= '9') || (val >= 'A' && val <= 'Z') || (val >= 'a' && val <= 'z') || val == '_' || val == '-' || val == '/' {
			newname = append(newname, byte(val))
		} else {
			newname = append(newname, '_')
		}
	}
	return string(newname)
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

func init() {
	SetForkState(FORK_PARENT)
	PID = os.Getpid()
	PIDSTR = strconv.Itoa(PID)
	loggerInit()
	argsInit()
	OrigProcTitle = options.OrigProcTitle
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
	// Opts
	opts.SetVersion("Go lang package preinit, version \"%s\"", "0.0.1")
	opts.SetDescription(`Provides utils for go daemon programing.
such as daemonize, proc respawn, drop privileges of proc, pass FDs to child proc.`)

	opts.SetOption("--chroot", "", "(available for root only)set proc chroot directory, proc will chroot befor do any thing, default: no chroot")
	opts.SetOption("--user", "www-data", "(available for root only)set dispatcher/worker running user name or user id, empty to run as current user")
	opts.SetOption("--group", "www-data", "(available for root only)set dispatcher/worker running group name or group id, empty to run as group of --user")
	opts.SetOption("--rootdir", "", "set proc working directory, by default, log/ var/ run/ tmp/ base on this directory, default: current directory")
	opts.SetOption("--vardir", "", "set proc var directory, default: --rootdir + /var/")
	opts.SetOption("--rundir", "", "set proc run directory, default: --rootdir + /run/")
	opts.SetOption("--tmpdir", "", "set proc data directory, default: --rootdir + /tmp/")
	opts.SetOption("--datadir", "", "set proc data directory, default: --rootdir + /data/")
	opts.SetOption("--logdir", "", "set proc log directory, default: --rootdir + /log/")
	opts.SetOption("--errlogfile", "", "set proc error log filename, if path is not absolute, file will be --logdir + logfile, default: disable error logging")
	opts.SetOption("--applogfile", "", "set proc app log file name, if path is not absolute, file will be --logdir + logfile, default: disable app logging")
	opts.SetOption("--debuglogfile", "", "set proc debug log file name, if path is not absolute, file will be --logdir + logfile, default: disable debug logging")
	opts.SetOption("--logrotation", "10", "set proc logging rotation, existed logfile will be overwrited")
	opts.SetOption("--logmaxsize", "0", "set proc max logfile size, K/M/G suffix is identifyed, zero to disable file size rotation")
	opts.SetOption("--logmaxline", "0", "set proc max logfile line, K/M/G suffix is identifyed, zero to disable file line rotation")

	opts.SetOption("--ident", "", "set prefix to proctitle, new title will be ident: orig-title, default: disable title prefix")
	opts.SetOption("--threads", "0", "set max running thread(GOMAXPROCS), -1 for all number of CPUs, 0 for CPUs - 1(at less 1)")

	opts.SetFlag("--daemon", "run proc as daemon")
	opts.SetFlag("--norespawn", "disable respawning for dispatcher/worker")
	opts.SetOption("--respawndelay", "5", "delay seconds befor respawn dispatcher/worker, at less one second")
	opts.SetOption("--respawnmax", "0", "max time of respawn dispatcher/worker, zero for always respawn")
	opts.SetOption("--forkstate", "", "state of proc fork, default: parent")
	opts.SetOption("--listens", "", "pre-listen list for dispatcher/worker, format: [proto:][addr/nic:]port/path, default proto is tcp, proto raw for rawsocket, default addr is any, multi-listen split by ',', eg,. :8080,udp:eth0:53,raw:eth1,unix:/tmp/socket.pipe, default: no pre-listen")
	opts.SetOption("--fds", "0", "number of pre-listen FDs pass from parent to dispatcher/worker")
	opts.SetNotes("this is internal command line args to contorl Go lang proc")
	//
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

// Package logger provides multi-channel logging for go daemon programing
// this package is no for high-performance logging

package logger

import (
	"errors"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/wheelcomplex/preinit/misc"
)

//// base logging support ////
/*
0. five logging file: stdout,stderr,debuglogfile, applogfile, errlogfile + syslog
0.1 send stdout to applogfile(if enabled), stderr to errlogfile(if enabled) after daemon
0.2 no level support
0.3 if debug enabled, applog will send to debuglog too
4. output file rotation by size
4. default logger contorl by commandline args(--errlogfile, --applogfile, --debuglogfile, --logrotation, --logmaxsize)
5. no Drop-in compatibility with code using the standard log package
6. fixed output format
7. dup line reduce
8. no thread safed
*/

// OpenFile flags
const (
	O_RDONLY int = syscall.O_RDONLY // open the file read-only.
	O_WRONLY int = syscall.O_WRONLY // open the file write-only.
	O_RDWR   int = syscall.O_RDWR   // open the file read-write.
	O_APPEND int = syscall.O_APPEND // append data to the file when writing.
	O_CREATE int = syscall.O_CREAT  // create a new file if none exists.
	O_EXCL   int = syscall.O_EXCL   // used with O_CREATE, file must not exist
	O_SYNC   int = syscall.O_SYNC   // open for synchronous I/O.
	O_TRUNC  int = syscall.O_TRUNC  // if possible, truncate file when opened.
)

// logger flag
type LogFlag int

const (
	LOGFLAG_DEBUG  LogFlag = log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile
	LOGFLAG_INFO   LogFlag = log.Ldate | log.Ltime | log.Lmicroseconds
	LOGFLAG_NOTICE LogFlag = log.Ldate | log.Ltime | log.Lmicroseconds
	LOGFLAG_APP    LogFlag = log.Ldate | log.Ltime | log.Lmicroseconds
	LOGFLAG_NONE   LogFlag = 0
)

// six logging file: stdout,stderr,debuglogfile, applogfile, errlogfile, syslog

// preinit logger
// set to DummyOut if one log channel disabled
type LoggerT struct {
	mu        sync.Mutex                // write lock
	writing   bool                      // write flag
	calldepth int                       // call depth
	prefix    string                    // prefix for Logger
	flag      int                       // flag for Logger
	logChs    map[string]*log.Logger    // Loggers
	closers   map[string]io.WriteCloser // records for SetWriteCloser
	list      map[string]struct{}       // default list for ListLog
	dupCount  int                       // dup msg counter
	preCount  int                       // dup msg counter
	last      []byte                    // dup buffer
	dupHint   []byte                    // dup hint buffer
	hint      bool                      //
	dedup     bool                      // is we dedup msg
	curTime   time.Time                 // update time.Now()
	dupTime   time.Time                 // max dup time
	dedups    map[string]bool           //is this channel need dedup
	writeOnce map[string]bool           // is channel writed
	closed    map[string]bool           // is channel closed
	// Logger for stdout
	// Logger for stderr
	// Logger for debug
	// Logger for app
	// Logger for err
	// Logger for syslog
}

// NewLogger create a new LoggerT and initial to default
// flag default to syslog.LOG_DAEMON
func NewLogger(prefix string, flag LogFlag) *LoggerT {
	if flag <= 0 {
		flag = LOGFLAG_APP
	}
	if len(prefix) == 0 {
		prefix = "loggerDefault"
	}
	sysl, err := syslog.NewLogger(syslog.LOG_NOTICE, int(LOGFLAG_NONE))
	if err != nil {
		panic("NewLogger(syslog) failed: " + err.Error())
	}
	l := &LoggerT{
		dedup:     true,
		calldepth: 4,
		prefix:    prefix,
		flag:      int(flag),
		logChs: map[string]*log.Logger{
			"stdout": log.New(os.Stdout, prefix, int(flag)),
			"stderr": log.New(os.Stderr, prefix, int(flag)),
			"debug":  log.New(DummyOut, prefix, int(LOGFLAG_DEBUG)),
			"app":    log.New(DummyOut, prefix, int(flag)),
			"err":    log.New(DummyOut, prefix, int(flag)),
			"sys":    sysl,
		},
		closers: make(map[string]io.WriteCloser),
		list:    make(map[string]struct{}),
		last:    make([]byte, 0, 256),
		dupHint: make([]byte, 0, 256),
		curTime: time.Now(),
		dedups: map[string]bool{
			"stdout": true,
			"stderr": true,
			"debug":  false,
			"app":    false,
			"err":    true,
			"sys":    true,
		},
		closed: map[string]bool{
			"stdout": false,
			"stderr": false,
			"debug":  false,
			"app":    false,
			"err":    false,
			"sys":    false,
		},
		writeOnce: map[string]bool{
			"stdout": false,
			"stderr": false,
			"debug":  false,
			"app":    false,
			"err":    false,
			"sys":    false,
		},
	}
	l.dupTime = l.curTime.Add(5e9)
	return l
}

// SetCalldepth set call depth for logger
// calldepth == -1 to return current call depth
// default is 4
func (l *LoggerT) SetCalldepth(calldepth int) int {
	old := l.calldepth
	if calldepth > 0 {
		l.calldepth = calldepth
	}
	return old
}

// LogChannelList return list of log channel
func (l *LoggerT) LogChannelList() map[string]*log.Logger {
	return l.logChs
}

// SetDedup control simple dedup of logger
// default is no-dedup for channel
func (l *LoggerT) SetChannelDedup(name string, dedup bool) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	old := false
	if _, ok := l.logChs[name]; ok {
		old = l.dedups[name]
		l.dedups[name] = dedup
	} else {
		return old
	}
	// update
	l.dedup = false
	for name, _ = range l.dedups {
		if l.closed[name] {
			continue
		}
		if l.dedups[name] {
			l.dedup = true
			break
		}
	}
	return old
}

// SetWriter set io.Writer of log write channel
// io.Writer will not closed when logger close
func (l *LoggerT) SetWriter(name string, output io.Writer) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.logChs[name]; ok == false {
		return errors.New("logger channel no exited")
	}
	l.CloseLogChannel(name)
	l.logChs[name] = log.New(output, l.prefix, l.flag)
	l.closed[name] = false
	return nil
}

// SetWriteCloser set io.WriteCloser of log write channel
// io.WriteCloser will closed when logger close
func (l *LoggerT) SetWriteCloser(name string, output io.WriteCloser) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.logChs[name]; ok == false {
		return errors.New("logger channel no exited")
	}
	l.CloseLogChannel(name)
	l.closers[name] = output
	l.logChs[name] = log.New(output, l.prefix, l.flag)
	l.closed[name] = false
	return nil
}

// closeLogChannel close io.WriteCloser of log write channel
func (l *LoggerT) closeLogChannel(name string) {
	if _, ok := l.logChs[name]; ok == false {
		return
	}
	if l.closed[name] {
		return
	}
	if _, ok := l.closers[name]; ok {
		l.closers[name].Close()
		delete(l.closers, name)
	}
	l.closed[name] = true
	return
}

// CloseLogChannel close io.WriteCloser of log write channel
func (l *LoggerT) CloseLogChannel(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closeLogChannel(name)
	return
}

// Close close All log write channel
func (l *LoggerT) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for name, _ := range l.logChs {
		l.closeLogChannel(name)
	}
	return
}

// dupcheck check dup msg and return true for dup
// dup hint msg --- last message repleat %d times --- will be output befor next no-dup msg
func (l *LoggerT) dupcheck(s string) bool {
	if misc.CompareByteString(l.last, s) == 0 {
		l.dupCount++
		l.preCount = 0
		if l.dupTime.Before(l.curTime) {
			l.preCount = l.dupCount
		} else if l.dupCount > 199 {
			l.preCount = l.dupCount
		}
		if l.preCount > 0 {
			// insert to output
			l.dupHint = l.dupHint[:0]
			l.dupHint = append(l.dupHint, []byte(fmt.Sprintf("last message repleat %d times", l.dupCount))...)
			l.hint = true
			l.dupCount = 0
			l.dupTime = l.curTime.Add(5e9)
		}
		return true
	}
	if l.dupCount > 0 {
		// insert to output
		l.dupHint = l.dupHint[:0]
		l.dupHint = append(l.dupHint, []byte(fmt.Sprintf("last message repleat %d times", l.dupCount))...)
		l.hint = true
		l.dupCount = 0
		l.dupTime = l.curTime.Add(5e9)
	}
	l.last = l.last[:0]
	l.last = append(l.last, []byte(s)...)
	return false
}

// WriteToList write msg to list of log write channel
func (l *LoggerT) WriteToList(names []string, v string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var isDup bool
	l.curTime = time.Now()
	if l.dedup {
		isDup = l.dupcheck(v)
	}
	for name, _ := range l.writeOnce {
		l.writeOnce[name] = false
	}
	if l.hint {
		// write hint to dedup-enabled channel
		for _, name := range names {
			if l.closed[name] {
				continue
			}
			if l.dedups[name] && l.writeOnce[name] == false {
				l.logChs[name].Output(l.calldepth, string(l.dupHint))
				l.writeOnce[name] = true
			}
		}
		l.hint = false
		for name, _ := range l.writeOnce {
			l.writeOnce[name] = false
		}
	}
	for _, name := range names {
		if l.closed[name] {
			continue
		}
		if isDup && l.dedups[name] {
			//fmt.Printf("isDup %v, l.dedups[%s] %v, l.closed[%s] %v: %s\n", isDup, name, l.dedups[name], name, l.closed[name], v)
			continue
		}
		if l.writeOnce[name] == false {
			l.logChs[name].Output(l.calldepth, v)
			l.writeOnce[name] = true
		}
	}
}

// wrappers

// Fatal write msg to err logger and call to os.Exit(1)
func (l *LoggerT) Fatal(v ...interface{}) {
	l.Errlog(v...)
	os.Exit(1)
}

func (l *LoggerT) Fatalf(format string, v ...interface{}) {
	l.Errlog(fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *LoggerT) Fatalln(v ...interface{}) {
	l.Errlog(fmt.Sprintln(v...))
	os.Exit(1)
}

// Panic write msg to err logger and call to panic().
func (l *LoggerT) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.Errlog(s)
	panic(s)

}
func (l *LoggerT) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.Errlog(s)
	panic(s)
}
func (l *LoggerT) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.Errlog(s)
	panic(s)
}

// Print write msg to debug+stdout
func (l *LoggerT) Print(v ...interface{}) {
	l.WriteToList([]string{"debug", "stdout"}, fmt.Sprint(v...))
}

func (l *LoggerT) Printf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "stdout"}, fmt.Sprintf(format, v...))
}

func (l *LoggerT) Println(v ...interface{}) {
	l.WriteToList([]string{"debug", "stdout"}, fmt.Sprintln(v...))
}

// Stdout write msg to debug+stdout
func (l *LoggerT) Stdout(v ...interface{}) {
	l.WriteToList([]string{"debug", "stdout"}, fmt.Sprint(v...))
}

func (l *LoggerT) Stdoutf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "stdout"}, fmt.Sprintf(format, v...))
}

func (l *LoggerT) Stdoutln(v ...interface{}) {
	l.WriteToList([]string{"debug", "stdout"}, fmt.Sprintln(v...))
}

// Stderr write msg to debug+stderr
func (l *LoggerT) Stderr(v ...interface{}) {
	l.WriteToList([]string{"debug", "stderr"}, fmt.Sprint(v...))
}

func (l *LoggerT) Stderrf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "stderr"}, fmt.Sprintf(format, v...))
}

func (l *LoggerT) Stderrln(v ...interface{}) {
	l.WriteToList([]string{"debug", "stderr"}, fmt.Sprintln(v...))
}

func (l *LoggerT) Prefix() string {
	return l.prefix
}

func (l *LoggerT) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
	for name, _ := range l.logChs {
		l.logChs[name].SetPrefix(prefix)
	}
}

// Applog write msg to app+debug
func (l *LoggerT) Applog(v ...interface{}) {
	l.WriteToList([]string{"debug", "app"}, fmt.Sprint(v...))
}

func (l *LoggerT) Applogf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "app"}, fmt.Sprintf(format, v...))
}

func (l *LoggerT) Applogln(v ...interface{}) {
	l.WriteToList([]string{"debug", "app"}, fmt.Sprintln(v...))
}

// Errlog write msg to err+debug+syslog+stderr
func (l *LoggerT) Errlog(v ...interface{}) {
	l.WriteToList([]string{"debug", "err", "sys", "stderr"}, fmt.Sprint(v...))
}

func (l *LoggerT) Errlogf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "err", "sys", "stderr"}, fmt.Sprintf(format, v...))
}

func (l *LoggerT) Errlogln(v ...interface{}) {
	l.WriteToList([]string{"debug", "err", "sys", "stderr"}, fmt.Sprintln(v...))
}

// Syslog write msg to debug+syslog
func (l *LoggerT) Syslog(v ...interface{}) {
	l.WriteToList([]string{"debug", "sys"}, fmt.Sprint(v...))
}

func (l *LoggerT) Syslogf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "sys"}, fmt.Sprintf(format, v...))
}

func (l *LoggerT) Syslogln(v ...interface{}) {
	l.WriteToList([]string{"debug", "sys"}, fmt.Sprintln(v...))
}

// Debug write msg to debug
func (l *LoggerT) Debug(v ...interface{}) {
	l.WriteToList([]string{"debug"}, "[DEBUG] "+fmt.Sprint(v...))
}

func (l *LoggerT) Debugf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug"}, "[DEBUG] "+fmt.Sprintf(format, v...))
}

func (l *LoggerT) Debugln(v ...interface{}) {
	l.WriteToList([]string{"debug"}, "[DEBUG] "+fmt.Sprintln(v...))
}

// AddListlog add ListLog write channel with io.Writer
// io.Writer will not closed when logger close
func (l *LoggerT) AddListlog(name string, output io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.logChs[name]; ok {
		l.closeLogChannel(name)
	}
	l.logChs[name] = log.New(output, l.prefix, l.flag)
	l.list[name] = struct{}{}
	l.dedups[name] = false
	l.closed[name] = false
	l.writeOnce[name] = false
	return
}

// AddListlogCloser add ListLog write channel with io.WriteCloser
// io.WriteCloser will closed when logger close
func (l *LoggerT) AddListlogCloser(name string, output io.WriteCloser) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.logChs[name]; ok {
		l.closeLogChannel(name)
	}
	l.closers[name] = output
	l.logChs[name] = log.New(output, l.prefix, l.flag)
	l.list[name] = struct{}{}
	l.dedups[name] = false
	l.closed[name] = false
	l.writeOnce[name] = false
	return
}

// GetListLog return current list of ListLog write channel
func (l *LoggerT) GetListLog() map[string]struct{} {
	return l.list
}

// RemoveListLog delete list of ListLog write channel
func (l *LoggerT) RemoveListLog(list []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, name := range list {
		delete(l.list, name)
		l.closeLogChannel(name)
	}
	return
}

// Listlog write msg to listed write channel
func (l *LoggerT) Listlog(v ...interface{}) {
	l.WriteToList(misc.ListToSlice(l.list), fmt.Sprint(v...))
}

func (l *LoggerT) Listlogf(format string, v ...interface{}) {
	l.WriteToList(misc.ListToSlice(l.list), fmt.Sprintf(format, v...))
}

func (l *LoggerT) Listlogln(v ...interface{}) {
	l.WriteToList(misc.ListToSlice(l.list), fmt.Sprintln(v...))
}

//

// dummy writer, like io/ioutil.Discard, but without syscall
type Trash_t struct{}

// NewTrash create a new Trash
func NewTrash() *Trash_t {
	return &Trash_t{}
}

/*
   71	// Writer is the interface that wraps the basic Write method.
   72	//
   73	// Write writes len(p) bytes from p to the underlying data stream.
   74	// It returns the number of bytes written from p (0 <= n <= len(p))
   75	// and any error encountered that caused the write to stop early.
   76	// Write must return a non-nil error if it returns n < len(p).
   77	// Write must not modify the slice data, even temporarily.
   78	type Writer interface {
   79		Write(p []byte) (n int, err error)
   80	}
*/

// Write accept []byte and do nothing
func (t *Trash_t) Write(p []byte) (n int, err error) {
	return len(p), nil
}

/*
   82	// Closer is the interface that wraps the basic Close method.
   83	//
   84	// The behavior of Close after the first call is undefined.
   85	// Specific implementations may document their own behavior.
   86	type Closer interface {
   87		Close() error
   88	}
*/

// Close do nothing
func (t *Trash_t) Close() error {
	return nil
}

/*

// This log writer sends output to a file
type FileLogWriter struct {
	rec chan *LogRecord
	rot chan bool

	// The opened file
	filename string
	file     *os.File

	// The logging format
	format string

	// File header/trailer
	header, trailer string

	// Rotate at linecount
	maxlines          int
	maxlines_curlines int

	// Rotate at size
	maxsize         int
	maxsize_cursize int

	// Rotate daily
	daily          bool
	daily_opendate int

	// Keep old logfiles (.001, .002, etc)
	rotate bool
}

*/

// file WriteCloser, with rotation, copy from log4go
// using mutx,no go routine
type RotFile_t struct {
	mu         sync.Mutex  // Writer mutex
	filename   string      // file name with full path
	curFile    string      // current file to write
	mode       os.FileMode // mode of new log file
	file       *os.File    // os.File of curFile
	num        int         // max file for rotation, <= 0 for no rotation
	curNum     int         //
	size       int         // max file size for rotation
	curSize    int         //
	line       int         // max line for rotation
	curLine    int         //
	format     string      // date string format to insert into filename
	nextTime   time.Time   //
	errDummy   bool        // drop all msg if writer error
	openNext   bool        // should we open next file
	msgSize    int         // size of one message
	errTryTime time.Time   // retry when io error
}

// NewRotFile create a new RotFile WriteCloser
// date format chars must inside 0-9 A-Z a-z _ - /, invalid char will replace by _
func NewRotFile(filename string, mode os.FileMode, max int, size int, line int, format string) *RotFile_t {
	var err error
	filename, err = filepath.Abs(filepath.Clean(filename))
	if err != nil {
		// give up if Abs faileds
		filename = filepath.Clean(filename)
	}
	r := &RotFile_t{
		filename: misc.SafeFileName(filename),
		mode:     mode,
		num:      max,
		size:     size,
		line:     line,
		format:   misc.SafeFileName(format),
	}
	r.openFile()
	return r
}

// reset flush buffer and close opened file
func (r *RotFile_t) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file != nil {
		r.file.Close()
		r.file = nil
	}
	r.curLine = 0
	r.curNum = 0
	r.curSize = 0
	r.nextTime = time.Time{}
	r.curFile = ""
}

// logFilename try to find next/rotation filename for logging
//
func (r *RotFile_t) logFilename() {
	// only call by openFile
	// check date format
	base := path.Dir(r.filename)
	name := path.Base(r.filename)
	prefix := ""
	rot := ""
	// update next time
	r.nextTime = misc.TimeFormatNext(r.format, time.Time{})
	// use: http://golang.org/pkg/time/#Time.Before at logwrite
	if r.nextTime.Equal(time.Time{}) == false {
		// setup next date string, use date string as filename prefix
		prefix = time.Now().Format(r.format) + "."
	} else if len(r.format) > 0 {
		prefix = r.format + "."
	}
	if r.num > 0 {
		if r.curNum >= r.num {
			r.curNum = 0
		}
		// start from  zero
		rot = "." + strconv.Itoa(r.curNum)
		r.curNum++
	}
	r.curFile = filepath.Clean(base + "/" + prefix + name + rot)
}

// openFile open current file
//
func (r *RotFile_t) openFile() error {
	// no threadsafe, call by Write or NewRotFile, caller is threadsafe
	// curFile closed
	r.reset()
	r.logFilename()
	// try to open current file
	var err error
	r.file, err = os.OpenFile(r.curFile, O_CREATE|O_APPEND|O_WRONLY, r.mode)
	if err != nil {
		err = fmt.Errorf("open %s failed: %s", r.curFile, err.Error())
		// debug
		fmt.Printf("%s\n", err)
		r.errTryTime = time.Now().Add(6e10)
		r.errDummy = true
		return err
	}
	fmt.Printf("open %s ok\n", r.curFile)
	r.curLine = 0
	r.curSize = 0
	r.errDummy = false
	r.openNext = false
	return err
}

// Write write msg to logfile
func (r *RotFile_t) Write(p []byte) (n int, err error) {
	// thread safe
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgSize = len(p)
	if r.errDummy {
		if r.errTryTime.After(time.Now()) {
			return r.msgSize, nil
		}
		// retry write
		r.errDummy = false
		r.openNext = true
	}
	// check date re-open
	if r.nextTime.Equal(time.Time{}) == false {
		// time.Now() >= r.nextTime
		if time.Now().Before(r.nextTime) == false {
			r.openNext = true
		}
	}
	// check file size, check file line
	if (r.size > 0 && r.curSize >= r.size) || (r.line > 0 && r.curLine >= r.line) {
		r.openNext = true
	}
	if r.openNext {
		r.openFile()
		// open failed
		if r.errDummy {
			return r.msgSize, nil
		}
	}
	// udate counter
	r.curSize = r.msgSize + r.curSize
	r.curLine++
	n, err = r.file.Write(p)
	if err != nil {
		r.errTryTime = time.Now().Add(6e10)
		r.errDummy = true
		err = fmt.Errorf("write disabled for write %s failed: %s", r.curFile, err.Error())
		// debug
		fmt.Printf("%s\n", err)
	}
	return n, err
}

// Close flush buffer and close opened file
func (r *RotFile_t) Close() error {
	r.reset()
	r.errDummy = true
	return nil
}

// dummy WriteCloser for logging
var DummyOut = NewTrash()

// default logger
var L = NewLogger("["+strconv.Itoa(os.Getpid())+"]", LOGFLAG_NOTICE)

//

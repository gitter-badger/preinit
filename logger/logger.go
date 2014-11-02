// Package logger provides multi-channel logging for go daemon programing
// this package is no for high-performance logging

package logger

import (
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

// CompareByteString in zero copy
// return 0 for equal, -1 for p less, 1 for p larger
func CompareByteString(p []byte, s string) int {
	pl := len(p)
	sl := len(s)
	if pl > sl {
		return 1
	}
	if pl < sl {
		return -1
	}
	for idx := 0; idx < pl; idx++ {
		if p[idx] != s[idx] {
			if p[idx] > s[idx] {
				return 1
			}
			if p[idx] < s[idx] {
				return -1
			}
		}
	}
	return 0
}

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

// SafeFileName replace invalid char with _
// valid char is . 0-9 _ - A-Z a-Z
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
	date       string      // date string format to insert into filename
	nextTime   time.Time   //
	errDummy   bool        // drop all msg if writer error
	openNext   bool        // should we open next file
	msgSize    int         // size of one message
	errTryTime time.Time   // retry when io error
}

// NewRotFile create a new RotFile WriteCloser
// date format chars must inside 0-9 A-Z a-z _ - /, invalid char will replace by _
func NewRotFile(filename string, mode os.FileMode, max int, size int, line int, date string) *RotFile_t {
	var err error
	filename, err = filepath.Abs(filepath.Clean(filename))
	if err != nil {
		// give up if Abs faileds
		filename = filepath.Clean(filename)
	}
	r := &RotFile_t{
		filename: SafeFileName(filename),
		mode:     mode,
		num:      max,
		size:     size,
		line:     line,
		date:     SafeFileName(date),
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
	r.nextTime = TimeFormatNext(r.date, time.Time{})
	// use: http://golang.org/pkg/time/#Time.Before at logwrite
	if r.nextTime.Equal(time.Time{}) == false {
		// setup next date string, use date string as filename prefix
		prefix = time.Now().Format(r.date) + "."
	}
	if r.num > 0 {
		if r.curNum >= r.num {
			r.curNum = 0
		}
		r.curNum++
		rot = "." + strconv.Itoa(r.curNum)
	}
	r.curFile = filepath.Clean(base + "/" + prefix + name + rot)
}

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

// openFile open current file
//
func (r *RotFile_t) openFile() error {
	// no threadsafe, call by Write or NewRotFile, caller is threadsafe
	// curFile closed
	r.reset()
	if r.nextTime.Equal(time.Time{}) == false {
		r.logFilename()
	}
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
	if r.curSize >= r.size || r.curLine >= r.line {
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
	return nil
}

// six logging file: stdout,stderr,debuglogfile, applogfile, errlogfile, syslog

// ListToSlice convert map[string]struct{} to []string
func ListToSlice(list map[string]struct{}) []string {
	s := make([]string, 0, len(list))
	for name, _ := range list {
		s = append(s, name)
	}
	return s
}

// preinit logger
// set to DummyOut if one log channel disabled
type Logger_t struct {
	prefix   string                    // prefix for Logger
	flag     int                       // flag for Logger
	Logs     map[string]*log.Logger    // Loggers
	closers  map[string]io.WriteCloser // records for SetWriteCloser
	list     map[string]struct{}       // default list for ListWrite
	dupCount int                       // dup msg counter
	last     []byte                    // dup buffer
	curTime  time.Time                 // update time.Now()
	dupTime  time.Time                 // max dup time
	// Logger for stdout
	// Logger for stderr
	// Logger for debug
	// Logger for app
	// Logger for err
	// Logger for syslog
}

// NewLogger create a new Logger_t and initial to default
// flag default to syslog.LOG_DAEMON
func NewLogger(prefix string, flag int) *Logger_t {
	if flag <= 0 {
		flag = int(syslog.LOG_DAEMON)
	}
	if len(prefix) == 0 {
		prefix = "loggerDefault"
	}
	sysl, err := syslog.NewLogger(syslog.LOG_NOTICE, flag)
	if err != nil {
		panic("NewLogger failed for " + err.Error())
	}
	l := &Logger_t{
		prefix: prefix,
		flag:   flag,
		Logs: map[string]*log.Logger{
			"sdtout": log.New(os.Stdout, prefix, flag),
			"stderr": log.New(os.Stderr, prefix, flag),
			"debug":  log.New(DummyOut, prefix, flag),
			"app":    log.New(DummyOut, prefix, flag),
			"err":    log.New(DummyOut, prefix, flag),
			"sys":    sysl,
		},
		closers: make(map[string]io.WriteCloser),
		list:    make(map[string]struct{}),
		last:    make([]byte, 256),
		curTime: time.Now(),
	}
	l.dupTime = l.curTime.Add(30e9)
	return l
}

// dedup check dup msg and return true for dup
func (l *Logger_t) dedup(s string) (string, bool) {
	if CompareByteString(l.last, s) == 0 {
		l.dupCount++
		preCount := 0
		if l.dupTime.Before(l.curTime) {
			preCount = l.dupCount
		} else if l.dupCount > 256 {
			preCount = l.dupCount
		}
		if preCount > 0 {
			l.dupCount = 0
			l.dupTime = l.curTime.Add(30e9)
			return fmt.Sprintf("--- last message repleat %d times ---"), true
		} else {
			return "", true
		}
	}
	l.last = l.last[:0]
	l.last = append(l.last, []byte(s)...)
	l.dupCount = 0
	return "", false
}

// SetWriter set io.Writer of log write channel
// io.Writer will not closed when logger close
func (l *Logger_t) SetWriter(name string, output io.Writer) {
	if _, ok := l.Logs[name]; ok == false {
		return
	}
	l.CloseChannel(name)
	l.Logs[name] = log.New(output, l.prefix, l.flag)
	return
}

// SetWriteCloser set io.WriteCloser of log write channel
// io.WriteCloser will closed when logger close
func (l *Logger_t) SetWriteCloser(name string, output io.WriteCloser) {
	if _, ok := l.Logs[name]; ok == false {
		return
	}
	l.CloseChannel(name)
	l.closers[name] = output
	l.Logs[name] = log.New(output, l.prefix, l.flag)
	return
}

// CloseChannel close io.WriteCloser of log write channel
func (l *Logger_t) CloseChannel(name string) {
	if _, ok := l.Logs[name]; ok == false {
		return
	}
	if _, ok := l.closers[name]; ok {
		l.closers[name].Close()
		delete(l.closers, name)
	}
	l.SetWriter(name, DummyOut)
	return
}

// Close close All log write channel
func (l *Logger_t) Close() {
	for name, _ := range l.Logs {
		l.CloseChannel(name)
	}
	return
}

// WriteTo write msg to one log write channel
func (l *Logger_t) WriteTo(name string, v string) {
	if _, ok := l.Logs[name]; ok == false {
		return
	}
	l.Logs[name].Print(v)
}

// WriteToList write msg to list of log write channel
func (l *Logger_t) WriteToList(names []string, v string) {
	for _, name := range names {
		if _, ok := l.Logs[name]; ok == false {
			continue
		}
		l.WriteTo(name, v)
	}
}

// Write write msg to all log write channel
func (l *Logger_t) Write(v string) {
	for name, _ := range l.Logs {
		l.WriteTo(name, v)
	}
}

// wrappers

// Fatal write msg to err logger and call to os.Exit(1)
func (l *Logger_t) Fatal(v ...interface{}) {
	l.Errlog(v...)
	os.Exit(1)
}

func (l *Logger_t) Fatalf(format string, v ...interface{}) {
	l.Errlog(fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *Logger_t) Fatalln(v ...interface{}) {
	l.Errlog(fmt.Sprintln(v...))
	os.Exit(1)
}

func (l *Logger_t) Flags() int {
	return l.flag
}

// Panic write msg to err logger and call to panic().
func (l *Logger_t) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.Errlog(s)
	panic(s)

}
func (l *Logger_t) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.Errlog(s)
	panic(s)
}
func (l *Logger_t) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.Errlog(s)
	panic(s)
}

// Print write msg to debug+stdout
func (l *Logger_t) Print(v ...interface{}) {
	s := fmt.Sprint(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "stdout"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "stdout"}, s)
}

func (l *Logger_t) Printf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "stdout"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "stdout"}, s)
}

func (l *Logger_t) Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "stdout"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "stdout"}, s)
}

// Stdout write msg to debug+stdout
func (l *Logger_t) Stdout(v ...interface{}) {
	s := fmt.Sprint(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "stdout"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "stdout"}, s)
}

func (l *Logger_t) Stdoutf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "stdout"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "stdout"}, s)
}

func (l *Logger_t) Stdoutln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "stdout"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "stdout"}, s)
}

// Stderr write msg to debug+stderr
func (l *Logger_t) Stderr(v ...interface{}) {
	s := fmt.Sprint(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "stderr"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "stderr"}, s)
}

func (l *Logger_t) Stderrf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "stderr"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "stderr"}, s)
}

func (l *Logger_t) Stderrln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "stderr"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "stderr"}, s)
}

//
func (l *Logger_t) SetFlags(flag int) {
	l.flag = flag
}

func (l *Logger_t) Prefix() string {
	return l.prefix
}

func (l *Logger_t) SetPrefix(prefix string) {
	l.prefix = prefix
}

// Applog write msg to app+debug
func (l *Logger_t) Applog(v ...interface{}) {
	l.WriteToList([]string{"debug", "app"}, fmt.Sprint(v...))
}

func (l *Logger_t) Applogf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "app"}, fmt.Sprintf(format, v...))
}

func (l *Logger_t) Applogln(v ...interface{}) {
	l.WriteToList([]string{"debug", "app"}, fmt.Sprintln(v...))
}

// Errlog write msg to err+debug+syslog+stderr
func (l *Logger_t) Errlog(v ...interface{}) {
	s := fmt.Sprint(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "err", "sys", "stderr"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "err", "sys", "stderr"}, s)
}

func (l *Logger_t) Errlogf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "err", "sys", "stderr"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "err", "sys", "stderr"}, s)
}

func (l *Logger_t) Errlogln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "err", "sys", "stderr"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "err", "sys", "stderr"}, s)
}

// Syslog write msg to debug+syslog
func (l *Logger_t) Syslog(v ...interface{}) {
	s := fmt.Sprint(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "sys"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "sys"}, s)
}

func (l *Logger_t) Syslogf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "sys"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "sys"}, s)
}

func (l *Logger_t) Syslogln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	if rep, dup := l.dedup(s); dup {
		if rep != "" {
			l.WriteToList([]string{"debug", "sys"}, rep)
		}
		return
	}
	l.WriteToList([]string{"debug", "sys"}, s)
}

// AddList set list of log write channel for ListWrite
func (l *Logger_t) AddList(list []string) {
	for _, name := range list {
		l.list[name] = struct{}{}
	}
	return
}

// RemoveList delete list of log write channel for ListWrite
func (l *Logger_t) RemoveList(list []string) {
	for _, name := range list {
		delete(l.list, name)
	}
	return
}

// Listlog write msg to listed write channel
func (l *Logger_t) Listlog(v ...interface{}) {
	l.WriteToList(ListToSlice(l.list), fmt.Sprint(v...))
}

func (l *Logger_t) Listlogf(format string, v ...interface{}) {
	l.WriteToList(ListToSlice(l.list), fmt.Sprintf(format, v...))
}

func (l *Logger_t) Listlogln(v ...interface{}) {
	l.WriteToList(ListToSlice(l.list), fmt.Sprintln(v...))
}

//
var DummyOut io.WriteCloser

var LogFlag int

//
func logger_init() {
	LogFlag = int(syslog.LOG_DAEMON)
	DummyOut = NewTrash()
}

//
func init() {
	logger_init()
}

//

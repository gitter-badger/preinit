package logger

import (
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
	//"strconv"
	//"strings"
	//"unsafe"
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

// six logging file: stdout,stderr,debuglogfile, applogfile, errlogfile, syslog

// ListToSlice convert map[string]struct{} to []string
func ListToSlice(list map[string]struct{}) []string {
	s := make([]string, 0, len(list))
	for name, _ := range list {
		s = append(s, name)
	}
	return s
}

// simple logger
// set to DummyOut if one log channel disabled
type Logger_t struct {
	prefix  string                    // prefix for Logger
	flag    int                       // flag for Logger
	Logs    map[string]*log.Logger    // Loggers
	closers map[string]io.WriteCloser // records for SetWriteCloser
	list    map[string]struct{}       // default list for ListWrite
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
	}
	return l
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

// Close close All log write channel
func (l *Logger_t) Close() {
	for name, _ := range l.Logs {
		l.CloseChannel(name)
	}
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
	l.Logs[name] = log.New(DummyOut, l.prefix, l.flag)
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
	l.WriteToList([]string{"debug", "stdout"}, fmt.Sprint(v...))
}

func (l *Logger_t) Printf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "stdout"}, fmt.Sprintf(format, v...))
}

func (l *Logger_t) Println(v ...interface{}) {
	l.WriteToList([]string{"debug", "stdout"}, fmt.Sprintln(v...))
}

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

// Errlog write msg to err+app+debug+syslog+stderr
func (l *Logger_t) Errlog(v ...interface{}) {
	l.WriteToList([]string{"debug", "app", "err", "sys", "stderr"}, fmt.Sprint(v...))
}

func (l *Logger_t) Errlogf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "app", "err", "sys", "stderr"}, fmt.Sprintf(format, v...))
}

func (l *Logger_t) Errlogln(v ...interface{}) {
	l.WriteToList([]string{"debug", "app", "err", "sys", "stderr"}, fmt.Sprintln(v...))
}

// Syslog write msg to debug+syslog
func (l *Logger_t) Syslog(v ...interface{}) {
	l.WriteToList([]string{"debug", "sys"}, fmt.Sprint(v...))
}

func (l *Logger_t) Syslogf(format string, v ...interface{}) {
	l.WriteToList([]string{"debug", "sys"}, fmt.Sprintf(format, v...))
}

func (l *Logger_t) Syslogln(v ...interface{}) {
	l.WriteToList([]string{"debug", "sys"}, fmt.Sprintln(v...))
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

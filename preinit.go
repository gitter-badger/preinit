/*

1. log
2. args
3. daemon+UID/GID
4. signal
5. fdpass+drop privileges
6. children monitor

*/

/*
	https://code.google.com/p/log4go/
*/

package preinit

import (
	//	"fmt"
	"io"
	"os"
	//	"strconv"
	//	"strings"
	//	"unsafe"

	opt "github.com/wheelcomplex/preinit/options"
)

// copy of os.Args for default arguments parser
var Args []string

// convert Args([]string) to line string
var ArgLine string

// command line args with default flags/options
var ArgFullLine string

// convert Args[0] to absolute file path
var ExecFile string

var OrigProcTitle string

// default opt Parser
var Opts *opt.OptParser_t

// initial default command line parser
func args_init() {
	Args = make([]string, 0, 0)
	Args = append(Args, os.Args...)
	ExecFile = opt.GetExecFileByPid(os.Getpid())
	// default opt Parser
	// do not include ExecFile
	Opts = opt.NewOptParser(Args[1:])
	ArgLine = opt.ArgsToSpLine(Args)
	ArgFullLine = opt.CleanArgLine(os.Args[0] + " " + Opts.String())
	//
}

//// wraps

// wrapper of options.func
func SetProcTitle(title string) {
	opt.SetProcTitle(title)
}

// wrapper of options.func
func SetProcTitlePrefix(prefix string) {
	opt.SetProcTitlePrefix(prefix)
}

// wrapper of options.String()
func CmdString() string {
	return Opts.String()
}

// wrapper of options.func
func ArgsString() string {
	return Opts.ArgsString()
}

// wrapper of options.func
func SetVersion(format string, a ...interface{}) string {
	return Opts.SetVersion(format, a...)
}

// wrapper of options.func
func SetDescription(format string, a ...interface{}) string {
	return Opts.SetDescription(format, a...)
}

// wrapper of options.func
func SetNotes(format string, a ...interface{}) string {
	return Opts.SetNotes(format, a...)
}

// wrapper of options.func
func SetOption(long string, defstring string, format string, a ...interface{}) string {
	return Opts.SetOption(long, defstring, format, a...)
}

// wrapper of options.func
func SetOptions(long string, defval []string, format string, a ...interface{}) string {
	return Opts.SetOptions(long, defval, format, a...)
}

// wrapper of options.func
func SetFlag(long string, format string, a ...interface{}) string {
	return Opts.SetFlag(long, format, a...)
}

// wrapper of options.func
func SetNoFlags(defval []string, format string, a ...interface{}) string {
	return Opts.SetNoFlags(defval, format, a...)
}

// wrapper of options.func
func VersionString() string {
	return Opts.VersionString()
}

// wrapper of options.func
func DescriptionString() string {
	return Opts.DescriptionString()
}

// wrapper of options.func
func NoteString() string {
	return Opts.NoteString()
}

// wrapper of options.func
func OptionString() string {
	return Opts.OptionString()
}

// wrapper of options.func
func FlagString() string {
	return Opts.FlagString()
}

// wrapper of options.func
func NoFlagString() string {
	return Opts.NoFlagString()
}

// wrapper of options.func
func CommandString() string {
	return Opts.CommandString()
}

// wrapper of options.func
func UsageString() string {
	return Opts.UsageString()
}

// wrapper of options.func
func Usage() {
	Opts.Usage()
}

// wrapper of Opts.Parse
func Parse(args []string) {
	Opts.Parse(args)
}

// wrapper of Opts.ParseString
func ParseString(line string) {
	Opts.ParseString(line)
}

// wrapper of options.func
func GetParserNoFlags() []string {
	return Opts.GetParserNoFlags()
}

// wrapper of options.func
func GetParserNoFlagString() string {
	return Opts.GetParserNoFlagString()
}

// wrapper of options.func
func GetNoFlags() []string {
	return Opts.GetNoFlags()
}

// wrapper of options.func
func GetNoFlagString() string {
	return Opts.GetNoFlagString()
}

// wrapper of options.func
func GetStringList(flag string) []string {
	return Opts.GetStringList(flag)
}

// wrapper of options.func
func GetString(flag string) string {
	return Opts.GetString(flag)
}

// wrapper of options.func
func GetStrings(flag string) string {
	return Opts.GetStrings(flag)
}

// wrapper of options.func
func GetInt(flag string) int {
	return Opts.GetInt(flag)
}

// wrapper of options.func
func GetInts(flag string) []int {
	return Opts.GetInts(flag)
}

// wrapper of options.func
func GetBool(flag string) bool {
	return Opts.GetBool(flag)
}

// wrapper of options.func
func DelParserFlag(key, value string) {
	Opts.DelParserFlag(key, value)
}

// wrapper of options.func
func SetParserFlag(key, value string) {
	Opts.SetParserFlag(key, value)
}

/// end of command line parser

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

*/

// dummy writer, like io/ioutil.Discard, but without syscall
type Trash_t struct{}

// NewTrash create a new Trash
func NewTrash() *Trash_t {
	return &Trash_t{}
}

// NewTrashWriter create a new Trash and return Writer of it
func NewTrashWriter() func(p []byte) (n int, err error) {
	t := &Trash_t{}
	return t.Write
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

//
type logWriter_t struct {
	disabled bool           // is writer disabled
	writer   io.WriteCloser // writer for log writting
}

// five logging file: stdout,stderr,debuglogfile, applogfile, errlogfile + syslog

type Prelog_t struct {
	stdout io.WriteCloser // Writer for os.Stdout
}

//// VARS ////

func init() {
	args_init()
	OrigProcTitle = opt.OrigProcTitle
	//println("preinit.init() end.")
}

//

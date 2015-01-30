/*
	Package getopt help for get command options
*/

package getopt

/*

#include "setproctitle.h"

*/
import "C"

//
import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

// C.spt_init1 defined in setproctitle.h

const (
	// These values must match the return values for spt_init1() used in C.
	HaveNone        = 0
	HaveNative      = 1
	HaveReplacement = 2
)

var (
	HaveSetProcTitle int
)

var OrigProcTitle string

func setproctitle_init() {
	Opts = NewOptParser(os.Args[1:])
	if len(OrigProcTitle) == 0 {
		OrigProcTitle = CleanArgLine(os.Args[0] + " " + Opts.String())
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

// end of SetProcTitle

// argsIndex return index of flag in args, if no found return -1
func argsIndex(args []string, flag string) int {
	var index int = -1
	for idx, val := range args {
		if val == flag {
			index = idx
			break
		}
	}
	return index
}

// CleanArgLine
func CleanArgLine(line string) string {
	var oldline string
	for {
		oldline = line
		line = strings.Replace(line, "  ", " ", -1)
		if oldline == line {
			break
		}
	}
	return strings.Trim(line, ", ")
}

// CleanSplitLine
func CleanSplitLine(line string) string {
	var oldline string
	for {
		oldline = line
		line = strings.Replace(line, "  ", " ", -1)
		if oldline == line {
			break
		}
	}
	for {
		oldline = line
		line = strings.Replace(line, " ", ",", -1)
		if oldline == line {
			break
		}
	}
	for {
		oldline = line
		line = strings.Replace(line, ",,", ",", -1)
		if oldline == line {
			break
		}
	}
	return strings.Trim(line, ", ")
}

// GetExecFileByPid return absolute execute file path of running pid
// return empty for error
// support linux only
func GetExecFileByPid(pid int) string {
	file, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
	if err != nil {
		return ""
	}
	return file
}

// ArgsToSpLine convert []string to string line, split by space
func ArgsToSpLine(list []string) string {
	tmpArr := make([]string, 0, len(list))
	for _, val := range list {
		val = CleanArgLine(val)
		// val = strings.Replace(val, " ", ",", -1)
		tmpArr = append(tmpArr, val)
	}
	return strings.Trim(strings.Join(tmpArr, " "), ", ")
}

// argsToList convert []string to string list, split by ,
func argsToList(list []string) string {
	return strings.Trim(strings.Join(list, ","), ", ")
}

// lineToArgs convert string to []string
func lineToArgs(line string) []string {
	return strings.Split(CleanSplitLine(line), ",")
}

// option_t save option data
type option_t struct {
	long    string   // long option
	defval  []string // default value
	desc    string   // description of this option
	sestion string   // sestion of this option
}

// String of option_t
// example: -t/--timeout
func (o *option_t) String() string {
	if o.long == "" {
		return ""
	}
	var line string = ""
	if o.long != "" && strings.HasPrefix(o.long, "__") == false {
		line = o.long
	}
	if o.sestion == "options" {
		line = line + " [value,...], "
	} else if line != "" {
		line = line + ", "
	}
	line = line + o.desc
	if len(o.defval) > 0 {
		cnt := 0
		defstr := ""
		for _, val := range o.defval {
			val = CleanArgLine(val)
			if val == "" || val == " " {
				continue
			}
			defstr = defstr + "," + val
			cnt++
		}
		if cnt > 0 {
			line = line + ", default: " + CleanArgLine(defstr)
		}
	} else if o.sestion == "flags" {
		// flags default to false
		line = line + ", default: false"
	}
	return CleanArgLine(line)
}

// opt paser struct
type Opts_t struct {
	longKeys           []string                        // list of --flag
	longArr            map[string][]string             // list for '--flag' options
	noFlagList         []string                        // list for '/path/filename /path/file2 /path/file3'
	sestions           map[string]map[string]*option_t // sestion list, default include: __version, __desc, options, flags, lists, __notes
	sestionKeys        map[string][]string             // list of --options in order
	maxSestionTitleLen int                             // prefix lenght for usage format
	powered            string                          // powered string
}

// NewOptParserString parsed args and return opt paser struct
func NewOptParserString(line string) *Opts_t {
	return NewOptParser(lineToArgs(line))
}

// NewOptParser parsed args and return opt paser struct
func NewOptParser(args []string) *Opts_t {
	op := new(Opts_t)
	op.reset()
	op.Parse(args)
	return op
}

// getOption return one option_t by sestion, flag
// return nil if no exist
func (op *Opts_t) getOption(sestion, flag string) *option_t {
	if _, ok := op.sestions[sestion]; ok == false {
		return nil
	}
	if _, ok := op.sestions[sestion][flag]; ok == false {
		return nil
	}
	return op.sestions[sestion][flag]
}

// setOption setup one option by sestion, short flag, long flag, default values, description text
// will overwrite old data if exist
func (op *Opts_t) setOption(sestion, long string, defval []string, format string, a ...interface{}) string {
	if sestion == "" {
		return ""
	}
	if long == "--" || long == "-" {
		return ""
	}
	if strings.HasPrefix(sestion, "__") == false && op.maxSestionTitleLen < len(sestion) {
		op.maxSestionTitleLen = len(sestion)
	}
	if _, ok := op.sestions[sestion]; ok == false {
		op.sestions[sestion] = make(map[string]*option_t)
	}
	// overwrite
	op.sestions[sestion][long] = &option_t{
		long:    long,
		desc:    fmt.Sprintf(format, a...),
		sestion: sestion,
		defval:  defval,
	}
	if _, ok := op.sestionKeys[sestion]; ok == false {
		op.sestionKeys[sestion] = make([]string, 0, 0)
	}
	for idx, val := range op.sestionKeys[sestion] {
		if val == long {
			//fmt.Println("sestionKeys", sestion, long, "existed")
			op.sestionKeys[sestion][idx] = ""
		}
	}
	op.sestionKeys[sestion] = append(op.sestionKeys[sestion], long)
	return op.sestions[sestion][long].String()
}

// SetVersion set version text for apps
// version text will show in first line of usage
func (op *Opts_t) SetVersion(format string, a ...interface{}) string {
	return op.setOption("__version", "__version", []string{}, format, a...)
}

// SetDescription set description text for apps
// description text will show after version
func (op *Opts_t) SetDescription(format string, a ...interface{}) string {
	return op.setOption("__desc", "__desc", []string{}, format, a...)
}

// SetNotes set notes text for apps
// notes text will show after flags
func (op *Opts_t) SetNotes(format string, a ...interface{}) string {
	return op.setOption("__notes", "__notes", []string{}, format, a...)
}

// SetOption set option(--key value) for apps
func (op *Opts_t) SetOption(long string, defstring string, format string, a ...interface{}) string {
	return op.setOption("options", long, lineToArgs(defstring), format, a...)
}

// SetOptions set option(--key value) for apps
func (op *Opts_t) SetOptions(long string, defval []string, format string, a ...interface{}) string {
	return op.setOption("options", long, defval, format, a...)
}

// SetFlag set flag(--flag) for apps
func (op *Opts_t) SetFlag(long string, format string, a ...interface{}) string {
	return op.setOption("flags", long, []string{}, format, a...)
}

// SetNoFlags set no flags item for apps
// option key is lists
func (op *Opts_t) SetNoFlags(defval []string, format string, a ...interface{}) string {
	return op.setOption("lists", "__lists", defval, format, a...)
}

// sestionString return sestion text in string for UsageString
// sestion text end with \n
// if sestion no exist return empty
// if no option inside this sestion return sestion title only(end with \n)
func (op *Opts_t) sestionString(sestion string) string {
	var text string
	if _, ok := op.sestions[sestion]; ok == false {
		return text
	}
	if len(op.sestions[sestion]) == 0 {
		return text
	}
	if strings.HasPrefix(sestion, "__") == false {
		padlen := op.maxSestionTitleLen - len(sestion)
		if padlen > 0 {
			text = strings.Repeat(" ", padlen) + sestion + ":\n"
		} else {
			text = sestion + ":\n"
		}
	}
	// option line
	//fmt.Printf("op.sestionKeys[%s]: %v\n", sestion, op.sestionKeys[sestion])
	//fmt.Printf("op.sestions[%s]: %v\n", sestion, op.sestions[sestion])
	for _, idx := range op.sestionKeys[sestion] {
		if idx == "" {
			continue
		}
		//fmt.Printf("op.sestions[%s]: %v => %v\n", sestion, idx, op.sestions[sestion][idx])
		text = text + "  " + op.sestions[sestion][idx].String() + "\n"
	}
	return strings.Trim(text, " ")
}

///// __version, __desc, options, flags, lists, __notes

// VersionString return version text in string
func (op *Opts_t) VersionString() string {
	return op.sestionString("__version")
}

// DescriptionString return description text in string
func (op *Opts_t) DescriptionString() string {
	return op.sestionString("__desc")
}

// NoteString return notes text in string
func (op *Opts_t) NoteString() string {
	if op.powered == "" {
		return op.sestionString("__notes")
	} else {
		return op.sestionString("__notes") + op.powered
	}
}

// OptionString return options text in string
func (op *Opts_t) OptionString() string {
	return op.sestionString("options")
}

// FlagString return flags text in string
func (op *Opts_t) FlagString() string {
	return op.sestionString("flags")
}

// NoFlagString return noFlags text in string
func (op *Opts_t) NoFlagString() string {
	return op.sestionString("lists")
}

// CommandString return command line template in string
func (op *Opts_t) CommandString() string {
	longopt := ""
	if _, ok := op.sestions["options"]; ok {
		for idx, _ := range op.sestions["options"] {
			if op.sestions["options"][idx].long != "" {
				longopt = "[--options value,...]"
				break
			}
		}
	}
	longflag := ""
	if _, ok := op.sestions["flags"]; ok {
		for idx, _ := range op.sestions["flags"] {
			if op.sestions["flags"][idx].long != "" {
				longflag = "[--flag]"
				break
			}
		}
	}
	noflag := ""
	if _, ok := op.sestions["lists"]; ok {
		if len(op.sestions["lists"]) > 0 {
			noflag = "[f1 f2 f3 ...]"
		}
	}
	text := CleanArgLine(longflag + " " + longopt + " " + noflag)
	if len(text) == 0 || text == " " {
		return "\n"
	}
	return GetExecFileByPid(os.Getpid()) + " " + text + "\n"
}

// UsageString return usage text in string
func (op *Opts_t) UsageString() string {
	return strings.Trim(op.VersionString()+"\n"+op.DescriptionString()+"\nUSAGE:\n"+op.CommandString()+op.OptionString()+op.FlagString()+op.NoFlagString()+op.NoteString(), "\n") + "\n"
}

// Usage output usage text to stderr
func (op *Opts_t) Usage() {
	fmt.Fprintf(os.Stderr, "%s", op.UsageString())
}

// parserReset parser for reuses
// old value discardeds
func (op *Opts_t) parserReset() {
	op.longKeys = make([]string, 0, 0)
	op.longArr = make(map[string][]string)
	op.noFlagList = make([]string, 0, 0)
}

// reset parser for reuses
// old value discardeds
func (op *Opts_t) reset() {
	op.parserReset()
	op.sestions = make(map[string]map[string]*option_t)
	op.sestions["options"] = make(map[string]*option_t)
	op.sestions["flags"] = make(map[string]*option_t)
	op.sestionKeys = make(map[string][]string)
	op.maxSestionTitleLen = 0
	op.powered = "Powered by Go"
}

// Powered set powered string of usage
// empty val to return current string
func (op *Opts_t) Powered(val string) string {
	old := op.powered
	if val != "" {
		op.powered = val
	}
	return old
}

// String convert opt paser struct to strings, include default values
func (op *Opts_t) String() string {
	var shortflag, shortoption, longflag, longoption string
	passed := make(map[string]struct{})
	// default flags/options
	for k1, _ := range op.sestions["flags"] {
		// for short flag
		if _, ok := passed[k1]; ok {
			continue
		}
		kval := op.GetStrings(k1)
		if strings.HasPrefix(k1, "--") == false && len(kval) == 0 {
			shortflag = shortflag + " " + k1 + " " + kval
			passed[k1] = struct{}{}
		}
	}
	for k1, _ := range op.sestions["options"] {
		// for short option
		if _, ok := passed[k1]; ok {
			continue
		}
		kval := op.GetStrings(k1)
		if strings.HasPrefix(k1, "--") == false && len(kval) > 0 {
			shortoption = shortoption + " " + k1 + " " + kval
			passed[k1] = struct{}{}
		}
	}
	for k1, _ := range op.sestions["flags"] {
		// for long flag
		if _, ok := passed[k1]; ok {
			continue
		}
		kval := op.GetStrings(k1)
		if strings.HasPrefix(k1, "--") == true && len(kval) == 0 {
			longflag = longflag + " " + k1 + " " + kval
			passed[k1] = struct{}{}
		}
	}
	for k1, _ := range op.sestions["options"] {
		// for long option
		if _, ok := passed[k1]; ok {
			continue
		}
		kval := op.GetStrings(k1)
		if strings.HasPrefix(k1, "--") == true && len(kval) > 0 {
			longoption = longoption + " " + k1 + " " + kval
			passed[k1] = struct{}{}
		}
	}
	// command lne flags/options
	for _, k1 := range op.longKeys {
		// for short flag
		if _, ok := passed[k1]; ok {
			continue
		}
		kval := op.GetStrings(k1)
		if strings.HasPrefix(k1, "--") == false && len(kval) == 0 {
			shortflag = shortflag + " " + k1 + " " + kval
			passed[k1] = struct{}{}
		}
	}
	for _, k1 := range op.longKeys {
		// for short option
		if _, ok := passed[k1]; ok {
			continue
		}
		kval := op.GetStrings(k1)
		if strings.HasPrefix(k1, "--") == false && len(kval) > 0 {
			shortoption = shortoption + " " + k1 + " " + kval
			passed[k1] = struct{}{}
		}
	}
	for _, k1 := range op.longKeys {
		// for long flag
		if _, ok := passed[k1]; ok {
			continue
		}
		kval := op.GetStrings(k1)
		if strings.HasPrefix(k1, "--") == true && len(kval) == 0 {
			longflag = longflag + " " + k1 + " " + kval
			passed[k1] = struct{}{}
		}
	}
	for _, k1 := range op.longKeys {
		// for long option
		if _, ok := passed[k1]; ok {
			continue
		}
		kval := op.GetStrings(k1)
		if strings.HasPrefix(k1, "--") == true && len(kval) > 0 {
			longoption = longoption + " " + k1 + " " + kval
			passed[k1] = struct{}{}
		}
	}
	return CleanArgLine(shortflag + " " + longflag + " " + shortoption + " " + longoption + " " + op.GetNoFlagString())
}

// CmdString convert opt paser struct to strings, include default values
func (op *Opts_t) CmdString() string {
	return op.String()
}

// ArgsString convert opt paser struct to strings, do not include default values
func (op *Opts_t) ArgsString() string {
	var shortflag, shortoption, longflag, longoption string
	for _, k1 := range op.longKeys {
		// for short flag
		kval := strings.Trim(strings.Join(op.longArr[k1], ","), ", ")
		if strings.HasPrefix(k1, "--") == false && len(kval) == 0 {
			shortflag = shortflag + " " + k1 + " " + kval
		}
	}
	for _, k1 := range op.longKeys {
		// for short option
		kval := strings.Trim(strings.Join(op.longArr[k1], ","), ", ")
		if strings.HasPrefix(k1, "--") == false && len(kval) > 0 {
			shortoption = shortoption + " " + k1 + " " + kval
		}
	}
	for _, k1 := range op.longKeys {
		// for long flag
		kval := strings.Trim(strings.Join(op.longArr[k1], ","), ", ")
		if strings.HasPrefix(k1, "--") == true && len(kval) == 0 {
			longflag = longflag + " " + k1 + " " + kval
		}
	}
	for _, k1 := range op.longKeys {
		// for long option
		kval := strings.Trim(strings.Join(op.longArr[k1], ","), ", ")
		if strings.HasPrefix(k1, "--") == true && len(kval) > 0 {
			longoption = longoption + " " + k1 + " " + kval
		}
	}
	return CleanArgLine(shortflag + " " + longflag + " " + shortoption + " " + longoption + " " + ArgsToSpLine(op.noFlagList))
}

// ParseString get opt paser struct ready to use
func (op *Opts_t) ParseString(line string) {
	op.Parse(lineToArgs(line))
}

// ParseMap get opt paser struct ready to use
func (op *Opts_t) Parse(args []string) {
	// reset
	op.parserReset()
	tmpList := make([]string, 0, len(args)+1)
	tmpList = append(tmpList, args...)
	tmpList = append(tmpList, "_PARSER_LAST_HOLDER_")
	// parse
	var newFlag string
	for _, val := range tmpList {
		//println(longFlag, "newFlag", newFlag, "next", val)
		if val == "-" || val == "--" {
			continue
		}
		if val == "_PARSER_LAST_HOLDER_" {
			val = ""
		}
		val = strings.Replace(val, " ", ",", -1)
		if newFlag != "" {
			if len(val) > 0 && strings.HasPrefix(val, "-") == false {
				// this is value for newFlags
				if argsIndex(op.longKeys, newFlag) == -1 {
					op.longKeys = append(op.longKeys, newFlag)
				}
				// overwrite exist --flags v1,v2,v3
				valist := strings.Split(val, ",")
				op.longArr[newFlag] = make([]string, 0, len(valist))
				op.longArr[newFlag] = append(op.longArr[newFlag], valist...)
				newFlag = ""
				continue
			} else {
				// no value for newFlags, it's bool flag
				if argsIndex(op.longKeys, newFlag) == -1 {
					op.longKeys = append(op.longKeys, newFlag)
				}
				// overwrite exist --flags
				op.longArr[newFlag] = make([]string, 0, 1)
				op.longArr[newFlag] = append(op.longArr[newFlag], "")
			}
			newFlag = ""
		}
		if len(val) == 0 {
			continue
		}
		if strings.HasPrefix(val, "-") {
			newFlag = val
		} else {
			op.noFlagList = append(op.noFlagList, val)
			newFlag = ""
		}
	}
}

// GetParserNoFlags return no-flag list in []string
func (op *Opts_t) GetParserNoFlags() []string {
	return op.noFlagList
}

// GetParserNoFlags return no-flag list in string
func (op *Opts_t) GetParserNoFlagString() string {
	return ArgsToSpLine(op.GetParserNoFlags())
}

// getString return first value of this keys
func (op *Opts_t) getString(key string) string {
	if key == "" {
		return ""
	}
	if _, ok := op.longArr[key]; ok {
		// return first value of this key
		return op.longArr[key][0]
	}
	return ""
}

// getStringList return list value of this keys
// if no exist, return empty []string
func (op *Opts_t) getStringList(key string) []string {
	if key == "" {
		return make([]string, 0, 0)
	}
	if _, ok := op.longArr[key]; ok {
		// return list value of this key
		return op.longArr[key]
	}
	return make([]string, 0, 0)
}

///////// export GetOpt*

// GetNoFlags return no flag list in []string
// if option no exist, return defval(if no default defined return nil)
func (op *Opts_t) GetNoFlags() []string {
	val := op.GetParserNoFlags()
	if len(val) == 0 {
		// try defaut value
		if opt := op.getOption("lists", "__lists"); opt != nil {
			val = opt.defval
		}
	}
	return val
}

// GetNoFlagString return no flag list in string
// if option no exist, return defval(if no default defined return nil)
func (op *Opts_t) GetNoFlagString() string {
	return ArgsToSpLine(op.GetNoFlags())
}

// GetStringList return list value of option
// if option no exist, return defval(if no default defined return empty list)
func (op *Opts_t) GetStringList(flag string) []string {
	val := op.getStringList(flag)
	if len(val) == 0 {
		// try defaut value
		if opt := op.getOption("options", flag); opt != nil {
			val = opt.defval
		} else if opt := op.getOption("flags", flag); opt != nil {
			val = opt.defval
		}
	}
	return val
}

// GetString return first value of option
// if option no exist, return defval(if no default defined return empty)
func (op *Opts_t) GetString(flag string) string {
	if list := op.GetStringList(flag); len(list) > 0 {
		return list[0]
	}
	return ""
}

// GetStrings return all value of option
// if option no exist, return defval(if no default defined return empty)
func (op *Opts_t) GetStrings(flag string) string {
	return argsToList(op.GetStringList(flag))
}

/// Get Int/Ints Bool

// GetInt return first value of option
// if option no exist, return defval(if no default defined return -1)
func (op *Opts_t) GetInt(flag string) int {
	if list := op.GetStringList(flag); len(list) > 0 {
		ival, err := strconv.Atoi(list[0])
		if err != nil {
			return -1
		}
		return ival
	}
	return -1
}

// GetInts return all value of option
// if option no exist, return defval(if no default defined return empty []int)
func (op *Opts_t) GetInts(flag string) []int {
	ilist := make([]int, 0, 0)
	if list := op.GetStringList(flag); len(list) > 0 {
		for idx, _ := range list {
			ival, err := strconv.Atoi(list[idx])
			if err != nil {
				ilist = append(ilist, -1)
			} else {
				ilist = append(ilist, ival)
			}
		}
	}
	return ilist
}

// GetBool return true if option exist, otherwise return false
// if option no exist, return defval(if no default defined return -1)
func (op *Opts_t) GetBool(flag string) bool {
	if list := op.GetStringList(flag); len(list) > 0 {
		return true
	}
	return false
}

// GetFlag is alias of GetBool
func (op *Opts_t) GetFlag(flag string) bool {
	return op.GetBool(flag)
}

//// modify options of Opts_t

// DelParserFlag modify Opts_t to match commandLine removed "key value"
// if key is flag, value == "" will remove all value of key, otherwise remove only flag match "key value"
func (op *Opts_t) DelParserFlag(key, value string) {
	newop := NewOptParserString(op.String())
	delop := NewOptParserString(CleanArgLine(key) + " " + CleanArgLine(value))
	//println("DelParserFlag In", key, "||", value, "||", "=>", CleanArgLine(key)+" "+CleanArgLine(value), "||", newop.String())
	op.parserReset()
	// sync short flags
	var match bool
	// sync long flags
	for _, k1 := range newop.longKeys {
		if _, ok := delop.longArr[k1]; ok {
			match = true
			if value == "" {
				// remove this flag
				continue
			}
		} else {
			match = false
		}
		if argsIndex(op.longKeys, k1) == -1 {
			// new flag
			op.longKeys = append(op.longKeys, k1)
		}
		// overwrite/reset longArr
		op.longArr[k1] = make([]string, 0, 0)
		for _, v2 := range newop.longArr[k1] {
			if match && v2 == value {
				// remove this key-value
				continue
			}
			op.longArr[k1] = append(op.longArr[k1], v2)
		}
	}
	// sync standalone value
	for _, val := range newop.noFlagList {
		if argsIndex(delop.noFlagList, val) != -1 {
			// remove this standalone value
			continue
		}
		op.noFlagList = append(op.noFlagList, val)
	}
	//println("DelParserFlag Out", key, "||", value, "||", op.String())
}

// SetParserFlag modify Opts_t to match commandLine "key value"
// empty string will be ignored
// string will be trimmed befor save to Opts_t
// if key is flag(start with - or --) old value of this flag will be overwrited
func (op *Opts_t) SetParserFlag(key, value string) {
	newop := NewOptParserString(CleanArgLine(key) + " " + CleanArgLine(value))
	//println("SetParserFlagIn", key, "||", value, "||", "=>", CleanArgLine(key)+" "+CleanArgLine(value), "||", newop.String())
	// sync long flags
	for _, k1 := range newop.longKeys {
		if argsIndex(op.longKeys, k1) == -1 {
			// new flag
			op.longKeys = append(op.longKeys, k1)
		}
		// overwrite/reset longArr
		op.longArr[k1] = make([]string, 0, 0)
		for _, v2 := range newop.longArr[k1] {
			op.longArr[k1] = append(op.longArr[k1], v2)
		}
	}
	// sync standalone value
	op.noFlagList = append(op.noFlagList, newop.noFlagList...)
	//println("SetParserFlagOut", key, "||", value, "||", op.String())
}

//

// default opt Parser
// default opt Parser
// do not include ExecFile
var opts = NewOptParser(os.Args[1:])

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

//
//
func init() {
	setproctitle_init()
}

//

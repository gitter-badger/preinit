/*
	package options provide commandLine or custom string parse and command option usage helper
*/

//// command line arguments support, flag and usage help ////
/*
. less extenal package used(os/fmt/strings)
. parser struct
. short + long arguments
. value and bool
. no-dash arguments list
. int, string, bool
. array supported: --aa 1 --aa 2 --aa 3 => aa = {1,2,3}
. value return only
. space between flag and value
. osargs = pr.Parse(), parse os.Args.String()
. strargs = pr.StringParse(args string), parse args
. strargs.GetInt("-d", "--debuglevel", 2, "debug level, 0-7, higher for more debug info"), return one int
. strargs.GetIntList("-d", "--debuglevel", []int{1, 2, 3}, "debug level, 0-7, higher for more debug info"), return []int
. strargs.GetParserNoFlags(), return []string of no-flag options
. no exit by invalid value

.
*/
package options

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// argsToLine convert []string to string line, split by space
func argsToLine(list []string) string {
	return strings.Trim(strings.Join(list, " "), " ")
}

// argsToList convert []string to string list, split by ,
func argsToList(list []string) string {
	return strings.Trim(strings.Join(list, ","), ", ")
}

// lineToArgs convert string to []string
func lineToArgs(line string) []string {
	return strings.Split(cleanArgLine(line), " ")
}

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

// cleanArgLine
func cleanArgLine(line string) string {
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

// getExecFileByPid return absolute execute file path of running pid
// return empty for error
// support linux only
func getExecFileByPid(pid int) string {
	file, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
	if err != nil {
		return ""
	}
	return file
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
		line = line + " [value1,value2,value3], "
	} else if line != "" {
		line = line + ", "
	}
	line = line + o.desc
	if len(o.defval) > 0 {
		cnt := 0
		defstr := ""
		for _, val := range o.defval {
			val = cleanArgLine(val)
			if val == "" || val == " " {
				continue
			}
			defstr = defstr + "," + val
			cnt++
		}
		if cnt > 0 {
			line = line + ", default: " + cleanArgLine(defstr)
		}
	} else if o.sestion == "flags" {
		// flags default to false
		line = line + ", default: false"
	}
	return cleanArgLine(line)
}

// opt paser struct
type optParser_t struct {
	longKeys           []string                        // list of --flag
	longArr            map[string][]string             // list for '--flag' options
	noFlagList         []string                        // list for '/path/filename /path/file2 /path/file3'
	sestions           map[string]map[string]*option_t // sestion list, default include: __version, __desc, options, flags, lists, __notes
	maxSestionTitleLen int                             // prefix lenght for usage format
}

// NewOptParserString parsed args and return opt paser struct
func NewOptParserString(line string) *optParser_t {
	return NewOptParser(lineToArgs(line))
}

// NewOptParser parsed args and return opt paser struct
func NewOptParser(args []string) *optParser_t {
	op := new(optParser_t)
	op.reset()
	op.Parse(args)
	return op
}

// getOption return one option_t by sestion, flag
// return nil if no exist
func (op *optParser_t) getOption(sestion, flag string) *option_t {
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
func (op *optParser_t) setOption(sestion, long string, defval []string, format string, a ...interface{}) string {
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
	return op.sestions[sestion][long].String()
}

// SetVersion set version text for apps
// version text will show in first line of usage
func (op *optParser_t) SetVersion(format string, a ...interface{}) string {
	return op.setOption("__version", "__version", []string{}, format, a...)
}

// SetDescription set description text for apps
// description text will show after version
func (op *optParser_t) SetDescription(format string, a ...interface{}) string {
	return op.setOption("__desc", "__desc", []string{}, format, a...)
}

// SetNotes set notes text for apps
// notes text will show after flags
func (op *optParser_t) SetNotes(format string, a ...interface{}) string {
	return op.setOption("__notes", "__notes", []string{}, format, a...)
}

// SetOption set option(--key value) for apps
func (op *optParser_t) SetOption(long string, defval string, format string, a ...interface{}) string {
	return op.setOption("options", long, []string{defval}, format, a...)
}

// SetOptions set option(--key value) for apps
func (op *optParser_t) SetOptions(long string, defval []string, format string, a ...interface{}) string {
	return op.setOption("options", long, defval, format, a...)
}

// SetFlag set flag(--flag) for apps
func (op *optParser_t) SetFlag(long string, format string, a ...interface{}) string {
	return op.setOption("flags", long, []string{}, format, a...)
}

// SetNoFlags set no flags item for apps
// option key is lists
func (op *optParser_t) SetNoFlags(defval []string, format string, a ...interface{}) string {
	return op.setOption("lists", "__lists", defval, format, a...)
}

// sestionString return sestion text in string for UsageString
// sestion text end with \n
// if sestion no exist return empty
// if no option inside this sestion return sestion title only(end with \n)
func (op *optParser_t) sestionString(sestion string) string {
	var text string
	if _, ok := op.sestions[sestion]; ok == false {
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
	if len(op.sestions[sestion]) == 0 {
		return text
	}
	// option line
	for idx, _ := range op.sestions[sestion] {
		text = text + "  " + op.sestions[sestion][idx].String() + "\n"
	}
	return strings.Trim(text, " ")
}

///// __version, __desc, options, flags, lists, __notes

// VersionString return version text in string
func (op *optParser_t) VersionString() string {
	return op.sestionString("__version")
}

// DescriptionString return description text in string
func (op *optParser_t) DescriptionString() string {
	return op.sestionString("__desc")
}

// NoteString return notes text in string
func (op *optParser_t) NoteString() string {
	return op.sestionString("__notes")
}

// OptionString return options text in string
func (op *optParser_t) OptionString() string {
	return op.sestionString("options")
}

// FlagString return flags text in string
func (op *optParser_t) FlagString() string {
	return op.sestionString("flags")
}

// NoFlagString return noFlags text in string
func (op *optParser_t) NoFlagString() string {
	return op.sestionString("lists")
}

// CommandString return command line template in string
func (op *optParser_t) CommandString() string {
	longopt := ""
	if _, ok := op.sestions["options"]; ok {
		for idx, _ := range op.sestions["options"] {
			if op.sestions["options"][idx].long != "" {
				longopt = "[--options value1,value2,value3...]"
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
	text := cleanArgLine(longflag + " " + longopt + " " + noflag)
	if len(text) == 0 || text == " " {
		return "\n"
	}
	return getExecFileByPid(os.Getpid()) + " " + text + "\n"
}

// UsageString return usage text in string
func (op *optParser_t) UsageString() string {
	return op.VersionString() + op.DescriptionString() + "\n" + op.CommandString() + op.OptionString() + op.FlagString() + op.NoFlagString() + op.NoteString()
}

// Usage output usage text to stderr
func (op *optParser_t) Usage() {
	fmt.Fprintf(os.Stderr, "%s", op.UsageString())
}

// parserReset parser for reuses
// old value discardeds
func (op *optParser_t) parserReset() {
	op.longKeys = make([]string, 0, 0)
	op.longArr = make(map[string][]string)
	op.noFlagList = make([]string, 0, 0)
}

// reset parser for reuses
// old value discardeds
func (op *optParser_t) reset() {
	op.parserReset()
	op.sestions = make(map[string]map[string]*option_t)
	op.sestions["options"] = make(map[string]*option_t)
	op.sestions["flags"] = make(map[string]*option_t)
	op.maxSestionTitleLen = 0
}

// String convert opt paser struct to strings, include default values
func (op *optParser_t) String() string {
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
	return cleanArgLine(shortflag + " " + longflag + " " + shortoption + " " + longoption + " " + op.GetNoFlagString())
}

// ArgsString convert opt paser struct to strings
func (op *optParser_t) ArgsString() string {
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
	return cleanArgLine(shortflag + " " + longflag + " " + shortoption + " " + longoption + " " + argsToLine(op.noFlagList))
}

// ParseString get opt paser struct ready to use
func (op *optParser_t) ParseString(line string) {
	op.Parse(lineToArgs(line))
}

// ParseMap get opt paser struct ready to use
func (op *optParser_t) Parse(args []string) {
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
func (op *optParser_t) GetParserNoFlags() []string {
	return op.noFlagList
}

// GetParserNoFlags return no-flag list in string
func (op *optParser_t) GetParserNoFlagString() string {
	return argsToLine(op.GetParserNoFlags())
}

// getString return first value of this keys
func (op *optParser_t) getString(key string) string {
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
func (op *optParser_t) getStringList(key string) []string {
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
func (op *optParser_t) GetNoFlags() []string {
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
func (op *optParser_t) GetNoFlagString() string {
	return argsToLine(op.GetNoFlags())
}

// GetStringList return list value of option
// if option no exist, return defval(if no default defined return empty list)
func (op *optParser_t) GetStringList(flag string) []string {
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
func (op *optParser_t) GetString(flag string) string {
	if list := op.GetStringList(flag); len(list) > 0 {
		return list[0]
	}
	return ""
}

// GetStrings return all value of option
// if option no exist, return defval(if no default defined return empty)
func (op *optParser_t) GetStrings(flag string) string {
	return argsToList(op.GetStringList(flag))
}

/// Get Int/Ints Bool

// GetInt return first value of option
// if option no exist, return defval(if no default defined return -1)
func (op *optParser_t) GetInt(flag string) int {
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
func (op *optParser_t) GetInts(flag string) []int {
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
func (op *optParser_t) GetBool(flag string) bool {
	if list := op.GetStringList(flag); len(list) > 0 {
		return true
	}
	return false
}

//// modify options of optParser_t

// DelParserFlag modify optParser_t to match commandLine removed "key value"
// if key is flag, value == "" will remove all value of key, otherwise remove only flag match "key value"
func (op *optParser_t) DelParserFlag(key, value string) {
	newop := NewOptParserString(op.String())
	delop := NewOptParserString(cleanArgLine(key) + " " + cleanArgLine(value))
	//println("DelParserFlag In", key, "||", value, "||", "=>", cleanArgLine(key)+" "+cleanArgLine(value), "||", newop.String())
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

// SetParserFlag modify optParser_t to match commandLine "key value"
// empty string will be ignored
// string will be trimmed befor save to optParser_t
// if key is flag(start with - or --) old value of this flag will be overwrited
func (op *optParser_t) SetParserFlag(key, value string) {
	newop := NewOptParserString(cleanArgLine(key) + " " + cleanArgLine(value))
	//println("SetParserFlagIn", key, "||", value, "||", "=>", cleanArgLine(key)+" "+cleanArgLine(value), "||", newop.String())
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

// copy of os.Args for default arguments parser
var Args []string

// convert Args([]string) to line string
var ArgLine string

// convert Args[0] to absolute file path
var ExecFile string

// default opt Parser
var Opts *optParser_t

// initial default command line parser
func args_init() {
	Args = make([]string, 0, 0)
	Args = append(Args, os.Args...)
	ExecFile = getExecFileByPid(os.Getpid())
	// default opt Parser
	// do not include ExecFile
	Opts = NewOptParser(Args[1:])
	ArgLine = Opts.String()
}

//// wraps

// wrap
func String() string {
	return Opts.String()
}

// wrap
func ArgsString() string {
	return Opts.ArgsString()
}

// wrap
func SetVersion(format string, a ...interface{}) string {
	return Opts.SetVersion(format, a...)
}

// wrap
func SetDescription(format string, a ...interface{}) string {
	return Opts.SetDescription(format, a...)
}

// wrap
func SetNotes(format string, a ...interface{}) string {
	return Opts.SetNotes(format, a...)
}

// wrap
func SetOption(long string, defval string, format string, a ...interface{}) string {
	return Opts.SetOption(long, defval, format, a...)
}

// wrap
func SetOptions(long string, defval []string, format string, a ...interface{}) string {
	return Opts.SetOptions(long, defval, format, a...)
}

// wrap
func SetFlag(long string, format string, a ...interface{}) string {
	return Opts.SetFlag(long, format, a...)
}

// wrap
func SetNoFlags(defval []string, format string, a ...interface{}) string {
	return Opts.SetNoFlags(defval, format, a...)
}

// wrap
func VersionString() string {
	return Opts.VersionString()
}

// wrap
func DescriptionString() string {
	return Opts.DescriptionString()
}

// wrap
func NoteString() string {
	return Opts.NoteString()
}

// wrap
func OptionString() string {
	return Opts.OptionString()
}

// wrap
func FlagString() string {
	return Opts.FlagString()
}

// wrap
func NoFlagString() string {
	return Opts.NoFlagString()
}

// wrap
func CommandString() string {
	return Opts.CommandString()
}

// wrap
func UsageString() string {
	return Opts.UsageString()
}

// wrap
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

// wrap
func GetParserNoFlags() []string {
	return Opts.GetParserNoFlags()
}

// wrap
func GetParserNoFlagString() string {
	return Opts.GetParserNoFlagString()
}

// wrap
func GetNoFlags() []string {
	return Opts.GetNoFlags()
}

// wrap
func GetNoFlagString() string {
	return Opts.GetNoFlagString()
}

// wrap
func GetStringList(flag string) []string {
	return Opts.GetStringList(flag)
}

// wrap
func GetString(flag string) string {
	return Opts.GetString(flag)
}

// wrap
func GetStrings(flag string) string {
	return Opts.GetStrings(flag)
}

// wrap
func GetInt(flag string) int {
	return Opts.GetInt(flag)
}

// wrap
func GetInts(flag string) []int {
	return Opts.GetInts(flag)
}

// wrap
func GetBool(flag string) bool {
	return Opts.GetBool(flag)
}

// wrap
func DelParserFlag(key, value string) {
	Opts.DelParserFlag(key, value)
}

// wrap
func SetParserFlag(key, value string) {
	Opts.SetParserFlag(key, value)
}

func init() {
	args_init()
	println("preinit.init() end.")
}

//// base logging support ////
/*
1. multi-logger in one proc
2. multi-output in one logger
3. output filter by debug level
4. output file rotation by size
4. default logger contorl by commandline args(--debuglevel, --errlogfile, --logfile, --debuglogfile, --logrotation, --logmaxsize)
5. Drop-in compatibility with code using the standard log package

// debug level come from log4go: Finest(7), Fine(6), Debug(5), Trace(4), Info(3), Warning(2), Error(1), Critical(0)

level <= 2 write to --errlogfile

level <= 3 write to --logfile

level <= 7 write to --debuglogfile

stdout() => Info(3)
stderr() => Warning(2)

no output format


*/

type prelog_t struct {
	level int // debug level 0-8, default
}

var debugLevel int

//// VARS ////

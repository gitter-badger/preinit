/*
	Package getopt help for get command options
*/

package getopt

//
import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ArgsIndex return index of flag in args, if no found return -1
func ArgsIndex(args []string, flag string) int {
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

// ExecFileOfPid return absolute execute file path of running pid
// return empty for error
// support linux only
func ExecFileOfPid(pid int) string {
	file, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
	if err != nil {
		return ""
	}
	return file
}

// StringListToSpaceLine convert []string to string line, split by space
func StringListToSpaceLine(list []string) string {
	tmpArr := make([]string, 0, len(list))
	for _, val := range list {
		val = CleanArgLine(val)
		// val = strings.Replace(val, " ", ",", -1)
		tmpArr = append(tmpArr, val)
	}
	return strings.Trim(strings.Join(tmpArr, " "), ", ")
}

// ArgsToList convert []string to string list, split by ,
func ArgsToList(list []string) string {
	return strings.Trim(strings.Join(list, ","), ", ")
}

// LineToArgs convert string to []string
func LineToArgs(line string) []string {
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
type opts_t struct {
	longKeys           []string                        // list of --flag
	longArr            map[string][]string             // list for '--flag' options
	noFlagList         []string                        // list for '/path/filename /path/file2 /path/file3'
	sestions           map[string]map[string]*option_t // sestion list, default include: __version, __desc, options, flags, lists, __notes
	sestionKeys        map[string][]string             // list of --options in order
	maxSestionTitleLen int                             // prefix lenght for usage format
	powered            string                          // powered string
}

// NewOptsFromString parsed args and return opt paser struct
func NewOptsFromString(line string) *opts_t {
	return NewOpts(LineToArgs(line))
}

// NewOpts parsed args and return opt paser struct
func NewOpts(args []string) *opts_t {
	op := new(opts_t)
	op.reset()
	op.Parse(args)
	return op
}

// getOption return one option_t by sestion, flag
// return nil if no exist
func (op *opts_t) getOption(sestion, flag string) *option_t {
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
// return string of this option
func (op *opts_t) setOption(sestion, long string, defval []string, format string, a ...interface{}) string {
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
func (op *opts_t) SetVersion(format string, a ...interface{}) string {
	return op.setOption("__version", "__version", []string{}, format, a...)
}

// SetDescription set description text for apps
// description text will show after version
func (op *opts_t) SetDescription(format string, a ...interface{}) string {
	return op.setOption("__desc", "__desc", []string{}, format, a...)
}

// SetNotes set notes text for apps
// notes text will show after flags
func (op *opts_t) SetNotes(format string, a ...interface{}) string {
	return op.setOption("__notes", "__notes", []string{}, format, a...)
}

// SetOpt set option(--key value) for apps
func (op *opts_t) SetOpt(long string, defstring string, format string, a ...interface{}) string {
	return op.setOption("options", long, LineToArgs(defstring), format, a...)
}

// SetOpts set option(--key value) for apps
func (op *opts_t) SetOpts(long string, defval []string, format string, a ...interface{}) string {
	return op.setOption("options", long, defval, format, a...)
}

// SetBool set flag(--flag) for apps
func (op *opts_t) SetBool(long string, defstring string, format string, a ...interface{}) string {
	return op.setOption("options", long, []string{defstring}, format, a...)
}

// SetNoFlags set no flags item for apps
// option key is lists
func (op *opts_t) SetNoFlags(defval []string, format string, a ...interface{}) string {
	return op.setOption("lists", "__lists", defval, format, a...)
}

// sestionString return sestion text in string for UsageString
// sestion text end with \n
// if sestion no exist return empty
// if no option inside this sestion return sestion title only(end with \n)
func (op *opts_t) sestionString(sestion string) string {
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
func (op *opts_t) VersionString() string {
	return op.sestionString("__version")
}

// DescriptionString return description text in string
func (op *opts_t) DescriptionString() string {
	return op.sestionString("__desc")
}

// NoteString return notes text in string
func (op *opts_t) NoteString() string {
	if op.powered == "" {
		return op.sestionString("__notes")
	} else {
		return op.sestionString("__notes") + op.powered
	}
}

// OptionString return options text in string
func (op *opts_t) OptionString() string {
	return op.sestionString("options")
}

// FlagString return flags text in string
func (op *opts_t) FlagString() string {
	return op.sestionString("flags")
}

// NoFlagString return noFlags text in string
func (op *opts_t) NoFlagString() string {
	return op.sestionString("lists")
}

// CommandLine return command line template in string, include all options
func (op *opts_t) CommandLine() string {
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
	return ExecFileOfPid(os.Getpid()) + " " + text + "\n"
}

// UsageString return usage text in string
func (op *opts_t) UsageString() string {
	return strings.Trim(op.VersionString()+"\n"+op.DescriptionString()+"\nUSAGE:\n"+op.CommandLine()+op.OptionString()+op.FlagString()+op.NoFlagString()+op.NoteString(), "\n") + "\n"
}

// Usage output usage text to stderr
func (op *opts_t) Usage() {
	fmt.Fprintf(os.Stderr, "%s", op.UsageString())
}

// parserReset parser for reuses
// old value discardeds
func (op *opts_t) parserReset() {
	op.longKeys = make([]string, 0, 0)
	op.longArr = make(map[string][]string)
	op.noFlagList = make([]string, 0, 0)
}

// reset parser for reuses
// old value discardeds
func (op *opts_t) reset() {
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
func (op *opts_t) Powered(val string) string {
	old := op.powered
	if val != "" {
		op.powered = val
	}
	return old
}

// String convert opt paser struct to strings, include default values
func (op *opts_t) String() string {
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
	return CleanArgLine(shortflag + " " + longflag + " " + shortoption + " " + longoption + " " + op.OptNoFlagsLine())
}

// OrigString convert opt paser struct to strings, do not include default values
func (op *opts_t) OrigString() string {
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
	return CleanArgLine(shortflag + " " + longflag + " " + shortoption + " " + longoption + " " + StringListToSpaceLine(op.noFlagList))
}

// ParseString get opt paser struct ready to use
func (op *opts_t) ParseString(line string) {
	op.Parse(LineToArgs(line))
}

// ParseMap get opt paser struct ready to use
func (op *opts_t) Parse(args []string) {
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
				if ArgsIndex(op.longKeys, newFlag) == -1 {
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
				if ArgsIndex(op.longKeys, newFlag) == -1 {
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
func (op *opts_t) GetParserNoFlags() []string {
	return op.noFlagList
}

// GetParserNoFlags return no-flag list in string
func (op *opts_t) GetParserNoFlagString() string {
	return StringListToSpaceLine(op.GetParserNoFlags())
}

// getString return first value of this keys
func (op *opts_t) getString(key string) string {
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
func (op *opts_t) getStringList(key string) []string {
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

// OptNoFlags return no flag list in []string
// if option no exist, return defval(if no default defined return nil)
func (op *opts_t) OptNoFlags() []string {
	val := op.GetParserNoFlags()
	if len(val) == 0 {
		// try defaut value
		if opt := op.getOption("lists", "__lists"); opt != nil {
			val = opt.defval
		}
	}
	return val
}

// OptNoFlagsLine return no flag list in string
// if option no exist, return defval(if no default defined return nil)
func (op *opts_t) OptNoFlagsLine() string {
	return StringListToSpaceLine(op.OptNoFlags())
}

// GetStringList return list value of option
// if option no exist, return defval(if no default defined return empty list)
func (op *opts_t) GetStringList(flag string) []string {
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
func (op *opts_t) GetString(flag string) string {
	if list := op.GetStringList(flag); len(list) > 0 {
		return list[0]
	}
	return ""
}

// GetStrings return all value of option
// if option no exist, return defval(if no default defined return empty)
func (op *opts_t) GetStrings(flag string) string {
	return ArgsToList(op.GetStringList(flag))
}

/// Get Int/Ints Bool

// GetInt return first value of option
// if option no exist, return defval(if no default defined return -1)
func (op *opts_t) GetInt(flag string) int {
	if list := op.GetStringList(flag); len(list) > 0 {
		ival, err := strconv.Atoi(list[0])
		if err != nil {
			return -1
		}
		return ival
	}
	return -1
}

// GetIntList return all value of option
// if option no exist, return defval(if no default defined return empty []int)
func (op *opts_t) GetIntList(flag string) []int {
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
// if option no exist, return defval(if no default defined return false)
// if option == false/disable return false
func (op *opts_t) GetBool(flag string) bool {
	if list := op.GetStringList(flag); len(list) > 0 {
		val := strings.ToLower(list[0])
		if val == "false" || val == "disable" || val == "" {
			return false
		}
		return true
	}
	return false
}

//
func (op *opts_t) OptBool(flag *bool, long string, defstring string, format string, a ...interface{}) {
	op.SetOpt(long, defstring, format, a...)
	*flag = op.GetBool(long)
	return
}

//
func (op *opts_t) OptString(flag *string, long string, defstring string, format string, a ...interface{}) {
	op.SetOpt(long, defstring, format, a...)
	*flag = op.GetString(long)
	return
}

//
func (op *opts_t) OptStringList(flag []string, long string, defval []string, format string, a ...interface{}) {
	op.SetOpts(long, defval, format, a...)
	flag = flag[:0]
	flag = append(flag, op.GetStringList(long)...)
	return
}

//
func (op *opts_t) OptStrings(flag *string, long string, defstring string, format string, a ...interface{}) {
	op.SetOpt(long, defstring, format, a...)
	*flag = op.GetStrings(long)
	return
}

//
func (op *opts_t) OptInt(flag *int, long string, defstring string, format string, a ...interface{}) {
	op.SetOpt(long, defstring, format, a...)
	*flag = op.GetInt(long)
	return
}

//
func (op *opts_t) OptIntList(flag []int, long string, defval []string, format string, a ...interface{}) {
	op.SetOpts(long, defval, format, a...)
	flag = flag[:0]
	flag = append(flag, op.GetIntList(long)...)
	return
}

//

// DelKeyValue modify opts_t to match commandLine removed "key value"
// if key is flag, value == "" will remove all value of key, otherwise remove only flag match "key value"
func (op *opts_t) DelKeyValue(key, value string) {
	newop := NewOptsFromString(op.String())
	delop := NewOptsFromString(CleanArgLine(key) + " " + CleanArgLine(value))
	//println("DelKeyValue In", key, "||", value, "||", "=>", CleanArgLine(key)+" "+CleanArgLine(value), "||", newop.String())
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
		if ArgsIndex(op.longKeys, k1) == -1 {
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
		if ArgsIndex(delop.noFlagList, val) != -1 {
			// remove this standalone value
			continue
		}
		op.noFlagList = append(op.noFlagList, val)
	}
	//println("DelKeyValue Out", key, "||", value, "||", op.String())
}

// SetKeyValue modify opts_t to match commandLine "key value"
// empty string will be ignored
// string will be trimmed befor save to opts_t
// if key is flag(start with - or --) old value of this flag will be overwrited
func (op *opts_t) SetKeyValue(key, value string) {
	newop := NewOptsFromString(CleanArgLine(key) + " " + CleanArgLine(value))
	//println("SetKeyValueIn", key, "||", value, "||", "=>", CleanArgLine(key)+" "+CleanArgLine(value), "||", newop.String())
	// sync long flags
	for _, k1 := range newop.longKeys {
		if ArgsIndex(op.longKeys, k1) == -1 {
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
	//println("SetKeyValueOut", key, "||", value, "||", op.String())
}

//

// default opt Parser
// default opt Parser
// do not include ExecFile
var opts = NewOpts(os.Args[1:])

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
func VersionString() string {
	return opts.VersionString()
}

// wrapper of options.func
func CommandLine() string {
	return opts.CommandLine()
}

// wrapper of options.func
func UsageString() string {
	return opts.UsageString()
}

// wrapper of options.func
func Usage() {
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
func SetOpt(long string, defstring string, format string, a ...interface{}) string {
	return opts.SetOpt(long, defstring, format, a...)
}

// wrapper of options.func
func SetOpts(long string, defval []string, format string, a ...interface{}) string {
	return opts.SetOpts(long, defval, format, a...)
}

// wrapper of options.func
func SetBool(long string, defstring string, format string, a ...interface{}) string {
	return opts.SetBool(long, defstring, format, a...)
}

// wrapper of options.func
func SetNoFlags(defval []string, format string, a ...interface{}) string {
	return opts.SetNoFlags(defval, format, a...)
}

// wrapper of options.func
func OptNoFlags() []string {
	return opts.OptNoFlags()
}

// wrapper of options.func
func OptNoFlagsLine() string {
	return opts.OptNoFlagsLine()
}

// wrapper of options.func
func GetString(flag string) string {
	return opts.GetString(flag)
}

// wrapper of options.func
func GetStringList(flag string) []string {
	return opts.GetStringList(flag)
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
func GetIntList(flag string) []int {
	return opts.GetIntList(flag)
}

// wrapper of options.func
func GetBool(flag string) bool {
	return opts.GetBool(flag)
}

//
func OptBool(flag *bool, long string, defstring string, format string, a ...interface{}) {
	opts.OptBool(flag, long, defstring, format, a...)
	return
}

//
func OptString(flag *string, long string, defstring string, format string, a ...interface{}) {
	opts.OptString(flag, long, defstring, format, a...)
	return
}

//
func OptStringList(flag []string, long string, defval []string, format string, a ...interface{}) {
	opts.OptStringList(flag, long, defval, format, a...)
	return
}

//
func OptStrings(flag *string, long string, defstring string, format string, a ...interface{}) {
	opts.OptStrings(flag, long, defstring, format, a...)
	return
}

//
func OptInt(flag *int, long string, defstring string, format string, a ...interface{}) {
	opts.OptInt(flag, long, defstring, format, a...)
	return
}

//
func OptIntList(flag []int, long string, defval []string, format string, a ...interface{}) {
	opts.OptIntList(flag, long, defval, format, a...)
	return
}

//

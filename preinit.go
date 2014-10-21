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
	//"github.com/wheelcomplex/preinit/log4go"
	//"fmt"
	"os"
	"strings"
)

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
. strargs.GetNoFlags(), return []string of no-flag options
. no exit by invalid value

.
*/

// argsToLine convert []string to string line, split by space
func argsToLine(list []string) string {
	return strings.Join(list, " ")
}

// lineToArgs convert string to []string
func lineToArgs(line string) []string {
	return strings.Split(cleanSpaces(line), " ")
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

// cleanSpaces
func cleanSpaces(line string) string {
	var oldline string
	for {
		oldline = line
		line = strings.Replace(line, "  ", " ", -1)
		if oldline == line {
			break
		}
	}
	return strings.Trim(line, " ")
}

// opt paser struct
type optParser_t struct {
	shortKeys  []string            // list of -flag
	shortArr   map[string][]string // list for '-f' options
	longKeys   []string            // list of --flag
	longArr    map[string][]string // list for '--flag' options
	noFlagList []string            // list for '/path/filename /path/file2 /path/file3'
	defaults   map[string][]string // defaults value for flags, no-flag default value use key _PARSER_NOFLAG_INDEX_
}

// NewOptParserString parsed args and return opt paser struct
func NewOptParserString(line string) *optParser_t {
	return NewOptParser(lineToArgs(line))
}

// NewOptParser parsed args and return opt paser struct
func NewOptParser(args []string) *optParser_t {
	op := new(optParser_t)
	op.defaults = make(map[string][]string)
	op.Parse(args)
	return op
}

// reset parser for reuses
// old value discardeds
func (op *optParser_t) reset() {
	op.shortKeys = make([]string, 0, 0)
	op.shortArr = make(map[string][]string)
	op.longKeys = make([]string, 0, 0)
	op.longArr = make(map[string][]string)
	op.noFlagList = make([]string, 0, 0)
	// do not reset usage/default
}

// String convert opt paser struct to strings
func (op *optParser_t) String() string {
	var line string
	// for map
	for _, k1 := range op.shortKeys {
		// for slice
		for _, v2 := range op.shortArr[k1] {
			line = line + " " + k1 + " " + v2
		}
	}
	// for map
	for _, k1 := range op.longKeys {
		// for slice
		for _, v2 := range op.longArr[k1] {
			line = line + " " + k1 + " " + v2
		}
	}
	line = line + " " + argsToLine(op.noFlagList)
	return cleanSpaces(line)
}

// ParseString get opt paser struct ready to use
func (op *optParser_t) ParseString(line string) {
	op.Parse(lineToArgs(line))
}

// ParseMap get opt paser struct ready to use
func (op *optParser_t) Parse(args []string) {
	// reset
	op.reset()
	tmpList := make([]string, 0, len(args)+1)
	tmpList = append(tmpList, args...)
	tmpList = append(tmpList, "_PARSER_LAST_HOLDER_")
	// parse
	var newFlag string
	var longFlag bool
	for _, val := range tmpList {
		//println(longFlag, "newFlag", newFlag, "next", val)
		if val == "_PARSER_LAST_HOLDER_" {
			val = ""
		}
		if newFlag != "" {
			if len(val) > 0 && strings.HasPrefix(val, "--") == false && strings.HasPrefix(val, "-") == false {
				// this is value for newFlags
				if longFlag {
					if argsIndex(op.longKeys, newFlag) == -1 {
						op.longKeys = append(op.longKeys, newFlag)
					}
					if _, ok := op.longArr[newFlag]; ok == false {
						op.longArr[newFlag] = make([]string, 0, 0)
					}
					op.longArr[newFlag] = append(op.longArr[newFlag], val)
				} else {
					if argsIndex(op.shortKeys, newFlag) == -1 {
						op.shortKeys = append(op.shortKeys, newFlag)
					}
					if _, ok := op.shortArr[newFlag]; ok == false {
						op.shortArr[newFlag] = make([]string, 0, 0)
					}
					op.shortArr[newFlag] = append(op.shortArr[newFlag], val)
				}
				newFlag = ""
				longFlag = false
				continue
			} else {
				// no value for newFlags, it's bool flag
				if longFlag {
					if argsIndex(op.longKeys, newFlag) == -1 {
						op.longKeys = append(op.longKeys, newFlag)
					}
					if _, ok := op.longArr[newFlag]; ok == false {
						op.longArr[newFlag] = make([]string, 0, 0)
					}
					op.longArr[newFlag] = append(op.longArr[newFlag], "")
				} else {
					if argsIndex(op.shortKeys, newFlag) == -1 {
						op.shortKeys = append(op.shortKeys, newFlag)
					}
					if _, ok := op.shortArr[newFlag]; ok == false {
						op.shortArr[newFlag] = make([]string, 0, 0)
					}
					op.shortArr[newFlag] = append(op.shortArr[newFlag], "")
				}
			}
			newFlag = ""
			longFlag = false
		}
		if len(val) == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(val, "--"):
			newFlag = val
			longFlag = true
		case strings.HasPrefix(val, "-"):
			newFlag = val
			longFlag = false
		default:
			op.noFlagList = append(op.noFlagList, val)
			newFlag = ""
			longFlag = false
		}
	}
}

// GetNoFlags return no-flag list in []string
func (op *optParser_t) GetNoFlags() []string {
	return op.noFlagList
}

// GetNoFlags return no-flag list in string
func (op *optParser_t) GetNoFlagString() string {
	return ArgsToLine(op.GetNoFlags())
}

// GetString return first value of this keys
func (op *optParser_t) GetString(key string) string {
	if key == "" {
		return ""
	}
	if _, ok := op.shortKeys[key]; ok {
		// return first value of this key
		return op.shortArr[key][0]
	}
	if _, ok := op.longKeys[key]; ok {
		// return first value of this key
		return op.longArr[key][0]
	}
	return ""
}

// GetStringList return list value of this keys
// if no exist, return empty []string
func (op *optParser_t) GetStringList(key string) []string {
	if key == "" {
		return make([]string, 0, 0)
	}
	if _, ok := op.shortKeys[key]; ok {
		// return list value of this key
		return op.shortArr[key]
	}
	if _, ok := op.longKeys[key]; ok {
		// return list value of this key
		return op.longArr[key]
	}
	return make([]string, 0, 0)
}

// GetOptNoFlags return no flag list in []string
// if option no exist, return defval(if no default defined return nil)
func (op *optParser_t) GetOptNoFlags() []string {
	val := op.GetNoFlags()
	if len(val) == 0 {
		// try defaut value
		if _, ok := op.defaults["_PARSER_NOFLAG_INDEX_"]; ok {
			// use first value of this flag
			val = op.defaults["_PARSER_NOFLAG_INDEX_"]
		}
	}
	return val
}

// GetOptNoFlagString return no flag list in string
// if option no exist, return defval(if no default defined return nil)
func (op *optParser_t) GetOptNoFlagString() string {
	return argsToLine(op.GetOptNoFlags())
}

// GetOptString return first value of option
// if option no exist, return defval(if no default defined return empty)
func (op *optParser_t) GetOptString(flag string) string {
	val := op.GetString(flag)
	if val == "" {
		// try defaut value
		if _, ok := op.defaults[flag]; ok {
			// use first value of this flag
			val = op.defaults[flag][0]
		}
	}
	return val
}

// DelFlag modify optParser_t to match commandLine removed "key value"
// if key is flag, value == "" will remove all value of key, otherwise remove only flag match "key value"
func (op *optParser_t) DelFlag(key, value string) {
	newop := NewOptParserString(op.String())
	delop := NewOptParserString(cleanSpaces(key) + " " + cleanSpaces(value))
	//println("DelFlag In", key, "||", value, "||", "=>", cleanSpaces(key)+" "+cleanSpaces(value), "||", newop.String())
	op.reset()
	// sync short flags
	var match bool
	for _, k1 := range newop.shortKeys {
		if _, ok := delop.shortArr[k1]; ok {
			match = true
			if value == "" {
				// remove this flag
				continue
			}
		} else {
			match = false
		}
		if argsIndex(op.shortKeys, k1) == -1 {
			// new flag
			op.shortKeys = append(op.shortKeys, k1)
		}
		// overwrite/reset shortArr
		op.shortArr[k1] = make([]string, 0, 0)
		for _, v2 := range newop.shortArr[k1] {
			if match && v2 == value {
				// remove this key-value
				continue
			}
			op.shortArr[k1] = append(op.shortArr[k1], v2)
		}
	}
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
	//println("DelFlag Out", key, "||", value, "||", op.String())
}

// SetFlag modify optParser_t to match commandLine "key value"
// empty string will be ignored
// string will be trimmed befor save to optParser_t
// if key is flag(start with - or --) old value of this flag will be overwrited
func (op *optParser_t) SetFlag(key, value string) {
	newop := NewOptParserString(cleanSpaces(key) + " " + cleanSpaces(value))
	//println("SetFlagIn", key, "||", value, "||", "=>", cleanSpaces(key)+" "+cleanSpaces(value), "||", newop.String())
	// sync short flags
	for _, k1 := range newop.shortKeys {
		if argsIndex(op.shortKeys, k1) == -1 {
			// new flag
			op.shortKeys = append(op.shortKeys, k1)
		}
		// overwrite/reset shortArr
		op.shortArr[k1] = make([]string, 0, 0)
		for _, v2 := range newop.shortArr[k1] {
			op.shortArr[k1] = append(op.shortArr[k1], v2)
		}
	}
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
	//println("SetFlagOut", key, "||", value, "||", op.String())
}

// copy of os.Args for default arguments parser
var Args []string

// convert Args([]string) to line string
var ArgLine string

// default opt Parser
var Opts *optParser_t

// initial default command line parser
func args_init() {
	Args = make([]string, 0, 0)
	Args = append(Args, os.Args...)
	// default opt Parser
	Opts = NewOptParser(Args)
	ArgLine = Opts.String()
}

// wrapper of Opts.Parse
func Parse(args []string) {
	Opts.Parse(args)
}

// wrapper of Opts.ParseString
func ParseString(line string) {
	Opts.ParseString(line)
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

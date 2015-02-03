/*
	Package misc provides util functions for general programing
*/

package misc

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// misc functions

// http://godoc.org/github.com/sluu99/uuid fromstr

// utils for package getopt

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

// GetOpenListOfPid return opened file list of pid
// return empty map if no file opened
func GetOpenListOfPid(pid int) []*os.File {
	var err error
	var file *os.File
	var filelist []string

	fds := make([]*os.File, 0, 0)

	file, err = os.Open("/proc/" + strconv.Itoa(pid) + "/fd/")
	if err != nil {
		//Logger.Errlogf("ERROR: %s\n", err)
		return fds
	}
	defer file.Close()

	filelist, err = file.Readdirnames(1024)
	if err != nil {
		if err == io.EOF {
			//Logger.Errlogf("read dir end: %s, %v\n", err, filelist)
		} else {
			//Logger.Errlogf("ERROR: %s\n", err)
			return fds
		}
	}
	/*
		ls -l /proc/self/fd/
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
				//Logger.Errlogf("strconv.Atoi(%v): %s\n", filelist[idx], err)
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

// ip <=> long
func Ip2long(ipstr string) (ip uint32) {
	r := `^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})`
	reg, err := regexp.Compile(r)
	if err != nil {
		return
	}
	ips := reg.FindStringSubmatch(ipstr)
	if ips == nil {
		return
	}

	ip1, _ := strconv.Atoi(ips[1])
	ip2, _ := strconv.Atoi(ips[2])
	ip3, _ := strconv.Atoi(ips[3])
	ip4, _ := strconv.Atoi(ips[4])

	if ip1 > 255 || ip2 > 255 || ip3 > 255 || ip4 > 255 {
		return
	}

	ip += uint32(ip1 * 0x1000000)
	ip += uint32(ip2 * 0x10000)
	ip += uint32(ip3 * 0x100)
	ip += uint32(ip4)

	return
}

func Long2ip(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", ip>>24, ip<<8>>24, ip<<16>>24, ip<<24>>24)
}

// Convert uint to net.IP
func inet_ntoa(ipnr int64) net.IP {
	var bytes [4]byte
	bytes[0] = byte(ipnr & 0xFF)
	bytes[1] = byte((ipnr >> 8) & 0xFF)
	bytes[2] = byte((ipnr >> 16) & 0xFF)
	bytes[3] = byte((ipnr >> 24) & 0xFF)

	return net.IPv4(bytes[3], bytes[2], bytes[1], bytes[0])
}

// Convert net.IP to int64
func inet_aton(ipnr net.IP) int64 {
	bits := strings.Split(ipnr.String(), ".")

	b0, _ := strconv.Atoi(bits[0])
	b1, _ := strconv.Atoi(bits[1])
	b2, _ := strconv.Atoi(bits[2])
	b3, _ := strconv.Atoi(bits[3])

	var sum int64

	sum += int64(b0) << 24
	sum += int64(b1) << 16
	sum += int64(b2) << 8
	sum += int64(b3)

	return sum
}

func UNUSED(v interface{}) {
	_ = v
	return
}

// GetXID return fmt.Sprintf("%x", d)
func GetXID(d interface{}) string {
	return fmt.Sprintf("%x", d)
}

// CommError is common error with error code support
type CommError struct {
	Code  int
	Err   error
	Addon interface{}
}

// String return formated string of CommError
func (ce *CommError) String() string {
	return fmt.Sprintf("error code %d, %s", ce.Code, ce.Err.Error())
}

// Error return formated string of CommError
func (ce *CommError) Error() string {
	return ce.String()
}

// IsNumeric returns true if a string contains only ascii digits from 0-9.
func IsNumeric(s string) bool {
	for _, c := range s {
		if int(c) > int('9') || int(c) < int('0') {
			return false
		}
	}
	return true
}

// RoundInt64 return n - (n % base) and (n % base)
func RoundInt64(n int64, base int64) (int64, int64) {
	v := n % base
	return n - v, v
}

func RoundString(n int64, base int64) string {
	v1, v2 := RoundInt64(n, base)
	v2len := len(fmt.Sprintf("%d", base))
	v2str := fmt.Sprintf("%d", v2)
	v2c := len(v2str) + 1
	for i := v2len; i > v2c; i-- {
		v2str = "0" + v2str
	}
	return fmt.Sprintf("%d.%s", v1/base, v2str)
}

// buffer size of uuidChan
const UUIDCHANBUFFSIZE int = 128

// UUIDChan
type UUIDChan struct {
	C      chan int64 // output channel
	m      sync.Mutex //
	closed bool
}

// NewUUIDChan return a new output chan of background UUID generator
// close exit chan will stop the generator
func NewUUIDChan() *UUIDChan {
	out := make(chan int64, UUIDCHANBUFFSIZE)
	var seed int64
	seedstr := strconv.FormatInt(int64(time.Now().UnixNano()), 10) + ":" + strconv.Itoa(os.Getpid()) + ":" + strconv.Itoa(os.Getppid())
	h := fnv.New64a()
	_, err := h.Write([]byte(seedstr))
	if err != nil {
		seed = int64(time.Now().UnixNano())
	} else {
		seed = int64(h.Sum64())
	}
	h.Reset()
	//fmt.Println("NewUUIDChan, UUIDGen seed:", seed)
	r := rand.New(rand.NewSource(seed))
	go func() {
		// handle close
		defer func() {
			recover()
		}()
		for {
			out <- r.Int63()
		}
	}()
	return &UUIDChan{
		C: out,
	}
}

//
func (uc *UUIDChan) Close() {
	uc.m.Lock()
	defer uc.m.Unlock()
	if uc.closed {
		return
	}
	uc.closed = true
	close(uc.C)
	return
}

// UUID use hash/fnv1a64 to generate int64
// base on time.Now() / os.Getpid() / os.Getpid() / runtime.ReadMemStats()
// NOTICE: this function is slow, use UUIDChan for heavy load
func UUID() int64 {
	var seed int64
	seedstr := strconv.FormatInt(int64(time.Now().UnixNano()), 10) + ":" + strconv.Itoa(os.Getpid()) + ":" + strconv.Itoa(os.Getppid())
	h := fnv.New64a()
	_, err := h.Write([]byte(seedstr))
	if err != nil {
		seed = int64(time.Now().UnixNano())
	} else {
		seed = int64(h.Sum64())
	}
	h.Reset()
	r := rand.New(rand.NewSource(seed))
	return r.Int63()
}

// StringUUID use hash/fnv1a64 to generate int64
// base on time.Now() / os.Getpid() / os.Getpid() / runtime.ReadMemStats()
// NOTICE: this function is slow
func StringUUID(s string) int64 {
	h := fnv.New64a()
	_, err := h.Write([]byte(s))
	if err != nil {
		panic(fmt.Sprintf("StringUUID(%s) failed for: %s", s, err.Error()))
	}
	return int64(h.Sum64())
}

// ByteUUID use hash/fnv1a64 to generate int64
// base on time.Now() / os.Getpid() / os.Getpid() / runtime.ReadMemStats()
// NOTICE: this function is slow
func ByteUUID(s []byte) int64 {
	h := fnv.New64a()
	_, err := h.Write(s)
	if err != nil {
		panic(fmt.Sprintf("ByteUUID(%s) failed for: %s", s, err.Error()))
	}
	return int64(h.Sum64())
}

// ByteInt64 convert []byte to int64
func ByteInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

// UUIDByte convert int64 to []byte
func UUIDByte(s int64) []byte {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(s))
	return buf
}

// UUIDByte2 convert int64 to []byte
func UUIDByte2(s int64) []byte {
	in := make([]byte, 0, 8)
	v := uint64(s)
	return append(in, byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// UUIDString convert int64 to string
func UUIDString(v int64) string {
	return fmt.Sprintf("%x%x%x%x%x%x%x%x", byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// UUIDHexBytes convert int64 to hex []byte
func UUIDHexBytes(v int64) []byte {
	return []byte(fmt.Sprintf("%x%x%x%x%x%x%x%x", byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32), byte(v>>24), byte(v>>16), byte(v>>8), byte(v)))
}

// Byte2UUIDByte
func Byte2UUIDByte(buf []byte) []byte {
	return UUIDByte(StringUUID(string(buf)))
}

/// end of UUID

func TimeNowString() string {
	return time.Now().Format(time.RFC1123)
}

func SleepSeconds(delay int) {
	timer := time.NewTimer(time.Duration(delay) * time.Second)
	<-timer.C
	return
}

func SafeStringRepeat(s string, count int) string {
	if count <= 0 {
		return ""
	}
	return strings.Repeat(s, count)
}

func IsPidAlive(pid int) (alive bool) {
	alive = false
	if pid < 1 {
		alive = true
	} else {
		err := syscall.Kill(pid, 0)
		//LogStderr("IsPidAlive %d: %s/%v", pid, err, err)
		if err == nil || err == syscall.EPERM {
			alive = true
		}
	}
	return
}

// UnicodeIndex return Unicode string index
func UnicodeIndex(str, substr string) int {
	// position of substr in str
	result := strings.Index(str, substr)
	if result >= 0 {
		// convert bytes befor substr to []byte
		prefix := []byte(str)[0:result]
		// convert []byte to rune
		rs := []rune(string(prefix))
		// got it
		result = len(rs)
	}
	return result
}

// SubString return substring in Unicode
func SubString(str string, begin, length int) (substr string) {
	// convert to []rune
	rs := []rune(str)
	lth := len(rs)

	// bound check
	if begin < 0 {
		begin = 0
	}
	if begin >= lth {
		begin = lth
	}
	end := begin + length
	if end > lth {
		end = lth
	}

	// got it
	return string(rs[begin:end])
}

// Tpf write msg with time suffix to stdout
func Tpf(format string, v ...interface{}) {
	ts := fmt.Sprintf("[%s] ", time.Now().String())
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s%s", ts, msg)
}

// Tpln write msg with time suffix to stdout
func Tpln(format string, v ...interface{}) {
	ts := fmt.Sprintf("[%s] ", time.Now().String())
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s%s\n", ts, msg)
}

//
type ByteRWCloser struct {
	Bytes []byte
	rptr  int
	wptr  int
}

func NewByteRWCloser(size int) *ByteRWCloser {
	return &ByteRWCloser{
		Bytes: make([]byte, size),
	}
}

func NewBRWC(p []byte) *ByteRWCloser {
	return &ByteRWCloser{
		Bytes: p,
	}
}

// Read Implementations Reader interface
func (brw *ByteRWCloser) Read(p []byte) (n int, err error) {
	if brw.rptr >= len(brw.Bytes) {
		err = io.EOF
		return
	}
	lb := len(brw.Bytes[brw.rptr:])
	lp := len(p)
	if lb > lp {
		n = lp
	} else {
		n = lb
		err = io.EOF
	}
	copy(p, brw.Bytes[brw.rptr:])
	brw.rptr += n
	return
}

// Write Implementations Writer interface
func (brw *ByteRWCloser) Write(p []byte) (n int, err error) {
	if brw.wptr >= len(brw.Bytes) {
		err = io.ErrShortWrite
		return
	}
	lb := len(brw.Bytes[brw.wptr:])
	lp := len(p)
	if lb > lp {
		n = lp
	} else {
		n = lb
		err = io.ErrShortWrite
	}
	copy(brw.Bytes[brw.wptr:], p)
	brw.wptr += n
	return
}

// Close Implementations Closer interface
func (brw *ByteRWCloser) Close() (err error) {
	brw.Reset()
	return
}

// Reset reset internal ptr to new
func (brw *ByteRWCloser) Reset() {
	brw.rptr = 0
	brw.wptr = 0
	return
}

////TODO: /bin/ip monitor
////get ip addr list with mask, key is 'ip/mask', value is ip only
//func GetIpAddrList() (IpAddrList map[string]string) {

//	IpAddrList = make(map[string]string)

//	ipaddrlist, err := net.InterfaceAddrs()
//	if err == nil {
//		for _, value := range ipaddrlist {
//			addrString := value.String()
//			pos := strings.Index(addrString, "/")
//			if pos < 1 {
//				pos = len(addrString)
//			}
//			IpAddrList[addrString] = addrString[:pos]
//		}
//	} else {
//		//Logger.Stderrf("InterfaceAddrs: %v", err)
//		CleanExit(1)
//	}
//	return
//}

//

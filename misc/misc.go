/*
 Package misc provides util functions for general programing
*/

package misc

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func UNUSED(v interface{}) {
	_ = v
	return
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

// uuidChan output UUID from UUID generator
var uuidChan chan int64

// buffer size of uuidChan
const UUIDCHANBUFFSIZE int = 10240

// miscMutex
var miscMutex *sync.Mutex

// GetUUIDChan return output chan of background UUID generator
func GetUUIDChan() chan int64 {
	miscMutex.Lock()
	defer miscMutex.Unlock()
	if uuidChan == nil {
		uuidChan, _ = NewUUIDChan(UUIDCHANBUFFSIZE)
	}
	return uuidChan
}

// NewUUIDChan return a new output chan of background UUID generator
// close exit chan will stop the generator
func NewUUIDChan(buferSize int) (out chan int64, exit chan struct{}) {
	if buferSize < 1 {
		buferSize = 1
	}
	out = make(chan int64, buferSize)
	exit = make(chan struct{}, 1)
	go UUIDGen(uuidChan, exit)
	go UUIDGen(uuidChan, exit)
	return out, exit
}

// UUIDGen generator UUID and output to out channel
// exit if exit channel closed
func UUIDGen(out chan int64, exit chan struct{}) {
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
	fmt.Println("UUIDGen seed:", seed)
	r := rand.New(rand.NewSource(seed))
	loop := true
	for loop {
		select {
		case <-exit:
			loop = false
		default:
			out <- r.Int63()
		}
	}
}

// UUID use hash/fnv1a64 to generate int64
// base on time.Now() / os.Getpid() / os.Getpid() / runtime.ReadMemStats()
// NOTICE: this function is slow
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

//TODO: /bin/ip monitor
//get ip addr list with mask, key is 'ip/mask', value is ip only
func GetIpAddrList() (IpAddrList map[string]string) {

	IpAddrList = make(map[string]string)

	ipaddrlist, err := net.InterfaceAddrs()
	if err == nil {
		for _, value := range ipaddrlist {
			addrString := value.String()
			pos := strings.Index(addrString, "/")
			if pos < 1 {
				pos = len(addrString)
			}
			IpAddrList[addrString] = addrString[:pos]
		}
	} else {
		Logger.Stderrf("InterfaceAddrs: %v", err)
		CleanExit(1)
	}
	return
}

//
func init() {

	// initial to nil, until user call GetUUIDChan
	uuidChan = nil

	//
	miscMutex = &sync.Mutex{}
}

//

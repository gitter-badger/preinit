/*
 Package misc provides util functions for general programing
*/

package misc

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

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
	go func() {
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
		fmt.Println("NewUUIDChan, UUIDGen seed:", seed)
		r := rand.New(rand.NewSource(seed))
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
	if brw.rptr > len(brw.Bytes) {
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
	if brw.rptr > len(brw.Bytes) {
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

// simpe AES encryption/decryption
// using static IV base on key
// The length of the AES key, either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256
const AES_KEYLEN int = 16

/*
// The AES block size in bytes.
const BlockSize = 16
*/

// AES implemented easy to use crypto/aes encryption/decryption
// using PKCS7Padding
type AES struct {
	key          []byte           // AES-128
	iv           []byte           // iv nonce
	block        cipher.Block     //
	blockEncrypt cipher.BlockMode //
	blockDecrypt cipher.BlockMode //
	mutex        sync.Mutex       //
}

// NewAES create new *AES with key and iv map
func NewAES(key []byte) *AES {
	var err error
	ae := &AES{}
	ae.keyinitial(key)
	ae.block, err = aes.NewCipher(ae.key)
	if err != nil {
		panic(fmt.Sprintf("aes.NewCipher failed: %s", err.Error()))
	}
	ae.blockEncrypt = cipher.NewCBCEncrypter(ae.block, ae.iv)
	ae.blockDecrypt = cipher.NewCBCDecrypter(ae.block, ae.iv)
	return ae
}

// keyinitial
func (ae *AES) keyinitial(key []byte) {
	ae.key = make([]byte, 0, AES_KEYLEN)
	ae.iv = make([]byte, 0, aes.BlockSize)
	//
	ae.key = append(ae.key, Byte2UUIDByte(key)...)
	keylen := len(ae.key)
	for keylen < AES_KEYLEN {
		// key is short
		ae.key = append(ae.key, Byte2UUIDByte(ae.key)...)
		keylen = len(ae.key)
	}
	ae.key = ae.key[:AES_KEYLEN]

	//fmt.Printf("NewAES: iv(%d): %x\n", len(ae.iv), ae.iv)
	ae.iv = append(ae.iv, Byte2UUIDByte(ae.key)...)
	ivlen := len(ae.iv)
	for ivlen < aes.BlockSize {
		// iv is short
		ae.iv = append(ae.iv, Byte2UUIDByte(ae.iv)...)
		ivlen = len(ae.iv)
	}
	ae.iv = ae.iv[:aes.BlockSize]
}

// PKCS7Padding
// dst always bigger then src
func (ae *AES) PKCS7Padding(dst, plainText []byte) []byte {
	padding := aes.BlockSize - len(plainText)%aes.BlockSize
	// WARNING: memory copy
	dst = append(dst, plainText...)
	dst = append(dst, bytes.Repeat([]byte{byte(padding)}, padding)...)
	return dst
}

//
func (ae *AES) PKCS7UnPadding(plainText []byte) ([]byte, error) {
	length := len(plainText)
	if length == 0 {
		return nil, fmt.Errorf("PKCS7UnPadding, invalid input: zero length")
	}
	unpadding := int(plainText[length-1])
	offset := (length - unpadding)
	if offset > length || offset <= 0 {
		return nil, fmt.Errorf("PKCS7UnPadding, invalid input: length %d unpadding %d offset %d", length, unpadding, offset)
	}
	return plainText[:offset], nil
}

// EncryptSize return size of encrypted msg with padding
// EncryptSize always bigger then srclen
func (ae *AES) EncryptSize(srclen int) int {
	return srclen + aes.BlockSize - (srclen % aes.BlockSize)
}

// Encrypt encrypt src into []byte, if len(src) is no multi of aes.BlockSize fill it with random bytes
// warning: if len(src) is no multi of aes.BlockSize memcopy occoured,
// lenght of encrypted []byte bigger then len(src)
// if encryptText is too smal to hold encrypted msg, new slice will created
func (ae *AES) Encrypt(encryptText []byte, src []byte) []byte {
	ae.mutex.Lock()
	defer ae.mutex.Unlock()
	encryptText = encryptText[:0]
	// WARNING: memory copy
	encryptText = ae.PKCS7Padding(encryptText, src)
	ae.blockEncrypt.CryptBlocks(encryptText, encryptText)
	return encryptText
}

// Decrypt decrypt src into []byte, if len(src) is no multi of aes.BlockSize return error
// lenght of decrypted []byte small then len(src)
// if decryptText is too smal to hold decrypted msg, new slice will created
func (ae *AES) Decrypt(decryptText []byte, src []byte) ([]byte, error) {
	ae.mutex.Lock()
	defer ae.mutex.Unlock()
	srclen := len(src)
	if srclen%aes.BlockSize != 0 {
		return nil, fmt.Errorf("AES Decrypt, invalid input length, %d % %d = %d(should be zero)", srclen, aes.BlockSize, srclen%aes.BlockSize)
	}
	if len(decryptText) < srclen {
		decryptText = make([]byte, srclen)
	}
	ae.blockDecrypt.CryptBlocks(decryptText, src)
	var err error
	decryptText, err = ae.PKCS7UnPadding(decryptText)
	if err != nil {
		return nil, err
	}
	return decryptText, nil
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
//		Logger.Stderrf("InterfaceAddrs: %v", err)
//		CleanExit(1)
//	}
//	return
//}

//

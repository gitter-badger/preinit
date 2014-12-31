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
	"hash"
	"hash/fnv"
	"io"
	"math"
	"math/big"
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

// Tpf write msg with time suffix to stdout
func Tpf(format string, v ...interface{}) {
	ts := fmt.Sprintf("[%s] ", time.Now().String())
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s%s", ts, msg)
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

//
// TODO: standalone keyaes package
//

// FnvUintExpend use fnv64a to expend uint64
func FnvUintExpend(init uint64, size int) []byte {
	buf := make([]byte, binary.Size(&init))
	binary.BigEndian.PutUint64(buf, uint64(init))
	return FnvExpend(buf, size)
}

// FnvUintFastExpend use fnv64a to expend uint64
func FnvUintFastExpend(init uint64, size int) []byte {
	buf := make([]byte, binary.Size(&init))
	binary.BigEndian.PutUint64(buf, uint64(init))
	return FnvFastExpend(buf, size)
}

// FnvExpend use fnv64a to expend []byte
func FnvExpend(init []byte, size int) []byte {
	exp := make([]byte, 0, size)
	h := fnv.New64a()
	h.Write(init)
	exp = h.Sum(exp)
	keylen := len(exp)
	for keylen < size {
		// key is short
		h.Write(exp)
		exp = h.Sum(exp)
		keylen = len(exp)
	}
	exp = exp[:size]
	return exp
}

// FnvFastExpend use fnv64a and byte shift to expend []byte
func FnvFastExpend(init []byte, size int) []byte {
	exp := make([]byte, 0, size)
	h := fnv.New64a()
	h.Write(init)
	exp = h.Sum(exp)
	keylen := len(exp)
	ptr := keylen
	offset := 0
	for keylen < size {
		if ptr == offset {
			ptr = keylen
			offset = ptr - aes.BlockSize - aes.BlockSize - aes.BlockSize
			if offset < 0 {
				offset = 0
			}
			for wptr := int(keylen / 3); wptr > 0; wptr-- {
				exp = append(exp, exp[wptr])
				exp = append(exp, exp[keylen-wptr])
			}
		}
		// key is short
		h.Write(exp[offset:ptr])
		ptr--
		exp = h.Sum(exp)
		keylen = len(exp)
	}
	exp = exp[:size]
	return exp
}

// The length of the AES key, either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256
const AES_KEYLEN int = 16

// IVSCOUNT size of iv map
const IVSCOUNT int = 1024 * 1024

// AES implemented crypto/aes encryption/decryption with nonce iv
// using PKCS7Padding
type AES struct {
	key          []byte           // AES-128
	ivs          []byte           // N * iv map
	nonce        uint64           // index of iv
	encryptsum   uint64           // checksum of plaintext
	decryptsum   uint64           // checksum of plaintext
	hdrlen       int              // length of binary nonce
	uuid         *UUIDChan        // uuid output for nonce
	eniv         []byte           // iv for encrypt
	deiv         []byte           // iv for decrypt
	block        cipher.Block     //
	blockEncrypt cipher.BlockMode //
	blockDecrypt cipher.BlockMode //
	mutex        sync.Mutex       //
	fnv64a       hash.Hash64      //
}

// NewAES create new *AES with key and iv map
func NewAES(key []byte) *AES {
	var err error
	ae := &AES{
		fnv64a: fnv.New64a(),
	}
	ae.hdrlen = binary.Size(ae.nonce)
	ae.uuid = NewUUIDChan()
	ae.keyinitial(key)
	ae.block, err = aes.NewCipher(ae.key)
	if err != nil {
		panic(fmt.Sprintf("aes.NewCipher failed: %s", err.Error()))
	}
	ae.blockEncrypt = cipher.NewCBCEncrypter(ae.block, ae.eniv)
	ae.blockDecrypt = cipher.NewCBCDecrypter(ae.block, ae.deiv)
	return ae
}

// sum64a return fnv64a sum base on AES key
func (ae *AES) sum64a(p []byte, iv []byte) uint64 {
	defer ae.fnv64a.Reset()
	ae.fnv64a.Reset()
	ae.fnv64a.Write(iv)
	ae.fnv64a.Write(p)
	ae.fnv64a.Write(ae.key)
	return ae.fnv64a.Sum64()
}

// keyinitial
func (ae *AES) keyinitial(key []byte) {
	//
	ae.key = FnvExpend(key, AES_KEYLEN)

	//
	ae.eniv = FnvExpend(ae.key, aes.BlockSize)

	//
	ae.deiv = FnvExpend(ae.eniv, aes.BlockSize)

	//
	ae.ivs = FnvFastExpend(ae.key, IVSCOUNT*aes.BlockSize)
}

// ivnonce get nonce iv pair
// if input n == 0, return random nonce + iv pair
func (ae *AES) ivnonce(n uint64, isEncrypt bool) uint64 {
	if n == 0 {
		n = uint64(<-ae.uuid.C)
	}
	ptr := (int(n) % IVSCOUNT) * aes.BlockSize
	if isEncrypt {
		copy(ae.eniv, ae.ivs[ptr:ptr+aes.BlockSize])
		binary.Write(NewBRWC(ae.eniv), binary.BigEndian, &n)
	} else {
		copy(ae.deiv, ae.ivs[ptr:ptr+aes.BlockSize])
		binary.Write(NewBRWC(ae.deiv), binary.BigEndian, &n)
	}
	return n
}

// PackSize return size of pack for srclen, and paddingsize included in packsize
func (ae *AES) PackSize(srclen int) (packsize, paddingsize int) {
	paddingsize = aes.BlockSize - ((srclen + ae.hdrlen) % aes.BlockSize)
	packsize = srclen + ae.hdrlen + paddingsize
	//println("PackSize, src", srclen, "paddingsize", paddingsize, "packsize", packsize)
	return
}

// EncryptSize return size of encrypted msg(included checksum) with padding
// EncryptSize = uint64(nonce)+packsize
func (ae *AES) EncryptSize(srclen int) int {
	packsize, _ := ae.PackSize(srclen)
	//println("EncryptSize, src", srclen, "full size", ae.hdrlen+packsize)
	return ae.hdrlen + packsize
}

// encryptPack implemented PKCS7Padding
// dst always bigger then src
//
// encrypt transport format:
// uint64(nonce)+encryptBlock
// encryptBlock = hash(src)+[]byte(src)
//
func (ae *AES) encryptPack(dst, src []byte) []byte {
	//
	_, padding := ae.PackSize(len(src))
	// WARNING: memory copy
	dst = dst[:ae.hdrlen]
	// write checksum of src
	ae.encryptsum = ae.sum64a(src, ae.eniv)
	binary.Write(NewBRWC(dst), binary.BigEndian, &ae.encryptsum)
	dst = append(dst, src...)
	dst = append(dst, bytes.Repeat([]byte{byte(padding)}, padding)...)
	return dst
}

// decryptUnPack implemented PKCS7Padding
func (ae *AES) decryptUnPack(plainText []byte) ([]byte, error) {
	length := len(plainText)
	if length < ae.hdrlen {
		return nil, fmt.Errorf("decryptUnPack, invalid input: invalid length")
	}
	unpadding := int(plainText[length-1])
	offset := (length - unpadding)
	if offset > length || offset <= 0 {
		//return nil, fmt.Errorf("decryptUnPack, invalid input: length %d unpadding %d offset %d", length, unpadding, offset)
		return nil, fmt.Errorf("decryptUnPack, invalid input: invalid offset")
	}
	if offset < ae.hdrlen {
		return nil, fmt.Errorf("decryptUnPack, invalid input: invalid unpadding length")
	}
	//ae.decryptsum = uint64(ByteUUID(plainText[ae.hdrlen:offset]))
	ae.decryptsum = ae.sum64a(plainText[ae.hdrlen:offset], ae.deiv)
	binary.Read(NewBRWC(plainText[:ae.hdrlen]), binary.BigEndian, &ae.encryptsum)
	if ae.decryptsum != ae.encryptsum {
		return nil, fmt.Errorf("decryptUnPack, invalid input: checksum mismatch, need %x, got %x", ae.decryptsum, ae.encryptsum)
		//return nil, fmt.Errorf("decryptUnPack, invalid input: checksum mismatch")
	}
	return plainText[ae.hdrlen:offset], nil
}

// Encrypt encrypt src into []byte
// lenght of encrypted []byte bigger then len(src)
// if encryptText is too smal to hold encrypted msg, new slice will created
func (ae *AES) Encrypt(encryptText []byte, src []byte) []byte {
	ae.mutex.Lock()
	defer ae.mutex.Unlock()
	dstlen := ae.EncryptSize(len(src))
	if len(encryptText) < dstlen {
		encryptText = make([]byte, dstlen)
	}
	// update nonce encrypt ae.eniv
	ae.nonce = ae.ivnonce(0, true)
	//
	// encrypt transport format: uint64(nonce)+encryptBlock(uint64(hash(src))+[]byte(src))
	//
	ae.encryptPack(encryptText[ae.hdrlen:], src)

	//fmt.Printf("AES, Encrypt IV(%d): %x\n", ae.nonce, ae.eniv)
	ae.blockEncrypt = cipher.NewCBCEncrypter(ae.block, ae.eniv)
	ae.blockEncrypt.CryptBlocks(encryptText[ae.hdrlen:], encryptText[ae.hdrlen:])
	// fill nonce, no error handle
	binary.Write(NewBRWC(encryptText[:ae.hdrlen]), binary.BigEndian, &ae.nonce)
	return encryptText
}

// Decrypt decrypt src into []byte, if len(src) is no multi of aes.BlockSize return error
// lenght of decrypted []byte small then len(src)
// if decryptText is too smal to hold decrypted msg, new slice will created
func (ae *AES) Decrypt(decryptText []byte, src []byte) ([]byte, error) {
	ae.mutex.Lock()
	defer ae.mutex.Unlock()
	srclen := len(src) - ae.hdrlen
	if srclen%aes.BlockSize != 0 {
		return nil, fmt.Errorf("AES Decrypt, invalid input length, %d % %d = %d(should be zero, nonce cut off)", srclen, aes.BlockSize, srclen%aes.BlockSize)
	}
	if len(decryptText) < srclen {
		decryptText = make([]byte, srclen)
	}
	decryptText = decryptText[:srclen]
	// get nonce, no error handle
	binary.Read(NewBRWC(src[:ae.hdrlen]), binary.BigEndian, &ae.nonce)
	// update nonce ae.deiv
	ae.ivnonce(ae.nonce, false)
	//fmt.Printf("AES, Decrypt IV(%d): %x\n", ae.nonce, ae.deiv)
	ae.blockDecrypt = cipher.NewCBCDecrypter(ae.block, ae.deiv)
	ae.blockDecrypt.CryptBlocks(decryptText, src[ae.hdrlen:])
	var err error
	decryptText, err = ae.decryptUnPack(decryptText)
	if err != nil {
		return nil, err
	}
	return decryptText, nil
}

// Close discard all internal resource
func (ae *AES) Close() {
	ae.mutex.Lock()
	defer ae.mutex.Unlock()
	if ae.ivs == nil {
		return
	}
	ae.ivs = nil
	ae.fnv64a.Reset()
	ae.uuid.Close()
}

//
type BigCounter interface {

	//
	New() BigCounter

	//
	String() string

	//
	FillBytes(p []byte) []byte

	//
	FillExpBytes(p []byte) []byte

	//
	Bytes() []byte

	//
	ExpBytes(size int) []byte

	//
	Mimus() error

	// Cmp compares x and y and returns:
	//
	//   -1 if a <  y
	//    0 if a == y
	//   +1 if a >  y
	//
	Cmp(b BigCounter) int

	//
	Mul(m uint64) error

	//
	AddUint64(m uint64) error

	//
	Plus() error

	//
	Touint64() []uint64

	//
	Size() int

	//
	Max() string

	//
	Init() string

	//
	SetMax(p []byte)

	//
	SetInit(p []byte)

	//
	ToBigInt() *big.Int

	//
	FromBigInt(b *big.Int)

	//
	Reset(over bool)
}

//
type AnyBaseCounter struct {
	w BigCounter
}

func NewAnyBaseCounter(size int) *AnyBaseCounter {
	if size < 0 {
		size = -size
	}
	if size == 0 {
		size = 1
	}
	bc := &AnyBaseCounter{}
	if size <= 8 {
		bc.w = Newfix64base(size)
	} else {
		bc.w = NewAny255Base(size)
	}
	//bc.w = NewAny255Base(size)
	return bc
}

//
func (a *AnyBaseCounter) New() BigCounter {
	return a.w.New()
}

//
func (a *AnyBaseCounter) String() string {
	return a.w.String()
}

//
func (a *AnyBaseCounter) ToBigInt() *big.Int {
	return a.w.ToBigInt()
}

//
func (a *AnyBaseCounter) FromBigInt(b *big.Int) {
	a.w.FromBigInt(b)
}

//
func (a *AnyBaseCounter) SetMax(p []byte) {
	a.w.SetMax(p)
}

//
func (a *AnyBaseCounter) SetInit(p []byte) {
	a.w.SetInit(p)
}

//
func (a *AnyBaseCounter) Max() string {
	return a.w.Max()
}

//
func (a *AnyBaseCounter) Init() string {
	return a.w.Init()
}

//
func (a *AnyBaseCounter) FillBytes(p []byte) []byte {
	return a.w.FillBytes(p)
}

//
func (a *AnyBaseCounter) FillExpBytes(p []byte) []byte {
	return a.w.FillExpBytes(p)
}

//
func (a *AnyBaseCounter) Bytes() []byte {
	return a.w.Bytes()
}

//
func (a *AnyBaseCounter) ExpBytes(size int) []byte {
	return a.w.ExpBytes(size)
}

//
func (a *AnyBaseCounter) Mimus() error {
	return a.w.Mimus()
}

func (a *AnyBaseCounter) Cmp(b BigCounter) int {
	return a.w.Cmp(b)
}

//
func (a *AnyBaseCounter) AddUint64(m uint64) error {
	return a.w.AddUint64(m)
}

//
func (a *AnyBaseCounter) Mul(m uint64) error {
	return a.w.Mul(m)
}

//
func (a *AnyBaseCounter) Plus() error {
	return a.w.Plus()
}

//
func (a *AnyBaseCounter) Touint64() []uint64 {
	return a.w.Touint64()
}

//
func (a *AnyBaseCounter) Size() int {
	return a.w.Size()
}

//
func (a *AnyBaseCounter) Reset(over bool) {
	a.w.Reset(over)
}

//
type fix64base struct {
	size  int
	last  int
	init  uint64
	max   uint64
	count uint64
	disp  uint64
}

//
func Newfix64base(size int) *fix64base {
	if size < 0 {
		size = -size
	}
	if size == 0 {
		size = 1
	}
	if size > 8 {
		size = 8
	}
	max := uint64(1)
	for i := 0; i < size*8; i++ {
		max = max * 2
	}
	max = max - 1
	return &fix64base{
		size: size,
		last: size - 1,
		init: 0,
		max:  max,
	}
}

//
func (a *fix64base) New() BigCounter {
	return Newfix64base(a.size)
}

//
func (a *fix64base) String() string {
	return fmt.Sprintf("%d", uint64(a.count))
}

// Cmp compares x and y and returns:
//
//   -1 if a <  y
//    0 if a == y
//   +1 if a >  y
//
func (a *fix64base) Cmp(b BigCounter) int {
	if ab, ok := b.(*fix64base); ok {
		return a.cmp(ab)
	}
	a64 := a.Bytes()
	aptr := 0
	for aptr == 0 && aptr < len(a64) {
		aptr = bytes.IndexByte(a64[aptr:], byte(0))
	}
	b64 := b.Bytes()
	bptr := 0
	for bptr == 0 && bptr < len(b64) {
		bptr = bytes.IndexByte(b64[bptr:], byte(0))
	}
	if aptr < bptr {
		return -1
	}
	if aptr > bptr {
		return 1
	}
	for i := 0; i <= aptr; i++ {
		if a64[i] < b64[i] {
			return -1
		}
		if a64[i] > b64[i] {
			return 1
		}
	}
	return 0
}

// Cmp compares x and y and returns:
//
//   -1 if a <  y
//    0 if a == y
//   +1 if a >  y
func (a *fix64base) cmp(b *fix64base) int {
	if a.count < b.count {
		return -1
	}
	if a.count == b.count {
		return 0
	}
	return 1
}

//
func (a *fix64base) FromBigInt(b *big.Int) {
	// Uint64 returns the uint64 representation of x.
	// If x cannot be represented in a uint64, the result is undefined.
	// func (x *Int) Uint64() uint64
	a.count = b.Uint64()
}

//
func (a *fix64base) ToBigInt() *big.Int {
	return big.NewInt(int64(a.count))
}

//
func (a *fix64base) Max() string {
	return fmt.Sprintf("%d", a.max+uint64(1))
}

//
func (a *fix64base) Init() string {
	return fmt.Sprintf("%d", a.init)
}

//
func (a *fix64base) SetInit(p []byte) {
	// a.max is uint64, size == 8
	ptr := binary.Size(uint64(0)) - len(p)
	if ptr < 0 {
		ptr = 0
	}
	//fmt.Printf("(a *fix64base) SetInit %x\n", p)
	newbuf := make([]byte, ptr)
	newbuf = append(newbuf, p...)
	//fmt.Printf("(a *fix64base) SetInit %x\n", newbuf)
	a.init = binary.BigEndian.Uint64(newbuf)
	//fmt.Printf("(a *fix64base) SetInit final %d || %x\n", a.init, a.init)
	a.count = a.init
}

//
func (a *fix64base) SetMax(p []byte) {
	// a.max is uint64, size == 8
	ptr := binary.Size(uint64(0)) - len(p)
	if ptr < 0 {
		ptr = 0
	}
	//fmt.Printf("(a *fix64base) SetMax %x\n", p)
	newbuf := make([]byte, ptr)
	newbuf = append(newbuf, p...)
	//fmt.Printf("(a *fix64base) SetMax %x\n", newbuf)
	a.max = binary.BigEndian.Uint64(newbuf)
	a.max--
	//fmt.Printf("(a *fix64base) SetMax final %d || %x\n", a.max, a.max)
}

//
func (a *fix64base) FillBytes(p []byte) []byte {
	if len(p) == 0 {
		return p
	}
	fptr := len(p) - 1
	for i := a.last; i >= 0 && fptr >= 0; i-- {
		a.disp = a.count
		for j := 0; j < (a.last - i); j++ {
			a.disp = a.disp >> 8
		}
		p[fptr] = uint8(a.disp)
		fptr--
	}
	/*
		p[0] = uint8(a.count >> 8 >> 8 >> 8 >> 8 >> 8 >> 8 >> 8)
		p[1] = uint8(a.count >> 8 >> 8 >> 8 >> 8 >> 8 >> 8)
		p[2] = uint8(a.count >> 8 >> 8 >> 8 >> 8 >> 8)
		p[3] = uint8(a.count >> 8 >> 8 >> 8 >> 8)
		p[4] = uint8(a.count >> 8 >> 8 >> 8)
		p[5] = uint8(a.count >> 8 >> 8)
		p[6] = uint8(a.count >> 8)
		p[7] = uint8(a.count)
	*/
	return p
}

// FillExpBytes fill binary bytes of number
func (a *fix64base) FillExpBytes(p []byte) []byte {
	if len(p) == 0 {
		return p
	}
	fptr := len(p) - 1
	step := fptr / a.size
	for i := a.last; i >= 0 && fptr >= 0; i-- {
		a.disp = a.count
		for j := 0; j < (a.last - i); j++ {
			a.disp = a.disp >> 8
		}
		p[fptr] = uint8(a.disp)
		fptr = fptr - step
	}
	/*
		p[0] = uint8(a.count >> 8 >> 8 >> 8 >> 8 >> 8 >> 8 >> 8)
		p[1] = uint8(a.count >> 8 >> 8 >> 8 >> 8 >> 8 >> 8)
		p[2] = uint8(a.count >> 8 >> 8 >> 8 >> 8 >> 8)
		p[3] = uint8(a.count >> 8 >> 8 >> 8 >> 8)
		p[4] = uint8(a.count >> 8 >> 8 >> 8)
		p[5] = uint8(a.count >> 8 >> 8)
		p[6] = uint8(a.count >> 8)
		p[7] = uint8(a.count)
	*/
	return p
}

//
func (a *fix64base) Bytes() []byte {
	buf := make([]byte, a.size)
	return a.FillBytes(buf)
}

//
func (a *fix64base) ExpBytes(size int) []byte {
	buf := make([]byte, size)
	return a.FillExpBytes(buf)
}

//
func (a *fix64base) Mimus() error {
	if a.count == a.init {
		a.Reset(false)
		return fmt.Errorf("Mimus cross %d, reset to max %d", a.init, a.max)
	}
	a.count--
	return nil
}

//
func (a *fix64base) AddUint64(m uint64) error {
	for i := uint64(0); i < m; i++ {
		if a.count >= a.max {
			a.Reset(true)
			return fmt.Errorf("Plus cross max %d, reset to %d", a.max, a.init)
		}
		a.count++
	}
	return nil
}

//
func (a *fix64base) Mul(m uint64) error {
	// counter == 0
	if a.count == 0 {
		return nil
	}
	pre := a.count
	for i := uint64(0); i < m; i++ {
		if a.count >= a.max {
			a.Reset(true)
			return fmt.Errorf("Plus cross max %d, reset to %d", a.max, a.init)
		}
		a.count += pre
	}
	return nil
}

//
func (a *fix64base) Plus() error {
	if a.count >= a.max {
		a.Reset(true)
		return fmt.Errorf("Plus cross max %d, reset to %d", a.max, a.init)
	}
	a.count++
	return nil
}

//
func (a *fix64base) Touint64() []uint64 {
	return []uint64{uint64(a.count)}
}

// Reset to up flow or down flow
// over == true, fill to zero
// over == false, fill to MAX
func (a *fix64base) Reset(over bool) {
	//fmt.Printf("reset %v when count %d\n", over, a.count)
	if over {
		// up flow
		a.count = a.init
	} else {
		a.count = a.max
	}
}

// Size return size of Bytes()
func (a *fix64base) Size() int {
	return a.size
}

//
type Any255Base struct {
	*fix64base
	ptr   int
	init  []uint8
	max   []uint8
	count []uint8
}

//
func NewAny255Base(size int) *Any255Base {
	if size < 0 {
		size = -size
	}
	if size == 0 {
		size = 1
	}
	a := &Any255Base{
		Newfix64base(0),
		0,
		make([]uint8, size),
		make([]uint8, size),
		make([]uint8, size),
	}
	for i := 0; i < size; i++ {
		a.max[i] = math.MaxUint8
	}
	a.size = size
	a.last = a.size - 1
	a.ptr = a.size - 1
	return a
}

//
func (a *Any255Base) New() BigCounter {
	return NewAny255Base(a.size)
}

//
func (a *Any255Base) String() string {
	bigNum := big.NewInt(0).SetBytes(a.Bytes())
	return bigNum.String()
}

// TODO: fix
/*
 Any255Base max() buf [255 255], 65536
initpoolsize 2560000 batch length 25600
[2014-12-31 15:22:09.496578126 +0800 CST] Any255Base max() buf [170 41], 43562
[2014-12-31 15:22:09.496637779 +0800 CST] generator# 0 start from 0 end at 43562

[2014-12-31 15:22:09.496639219 +0800 CST] Any255Base max() buf [84 84], 21589
[2014-12-31 15:22:09.496651632 +0800 CST] Any255Base max() buf [254 126], 65151
[2014-12-31 15:22:09.496697291 +0800 CST] generator# 1 start from 10922 end at 21589

[2014-12-31 15:22:09.496710886 +0800 CST] generator# 2 start from 21844 end at 65151

[2014-12-31 15:22:09.496709523 +0800 CST] Any255Base max() buf [168 169], 43178
[2014-12-31 15:22:09.496762264 +0800 CST] generator# 3 start from 32766 end at 43178

[2014-12-31 15:22:09.496795273 +0800 CST] Any255Base max() buf [82 212], 21205
[2014-12-31 15:22:09.496837543 +0800 CST] generator# 4 start from 43688 end at 21205

[2014-12-31 15:22:09.496824675 +0800 CST] Any255Base max() buf [1 0], 257
[2014-12-31 15:22:09.496869212 +0800 CST] generator# 5 start from 54610 end at 257

reset true when count [42 170]
[2014-12-31 15:22:09.498135967 +0800 CST] Any255Base max() buf [84 84], 21589
[2014-12-31 15:22:09.498152134 +0800 CST] generator# 1 current 10922 end at 21589
reset true when count [127 254]
[2014-12-31 15:22:09.498190957 +0800 CST] Any255Base max() buf [168 169], 43178
[2014-12-31 15:22:09.498210085 +0800 CST] generator# 3 current 32766 end at 43178
reset true when count [0 0]
[2014-12-31 15:22:09.498467526 +0800 CST] Any255Base max() buf [170 41], 43562
[2014-12-31 15:22:09.498488716 +0800 CST] generator# 0 current 0 end at 43562
reset true when count [85 84]
[2014-12-31 15:22:09.498552124 +0800 CST] Any255Base max() buf [254 126], 65151
[2014-12-31 15:22:09.498572099 +0800 CST] generator# 2 current 21844 end at 65151
reset true when count [170 168]
[2014-12-31 15:22:09.49898484 +0800 CST] Any255Base max() buf [82 212], 21205
[2014-12-31 15:22:09.498999589 +0800 CST] generator# 4 current 43688 end at 21205
reset true when count [213 82]
[2014-12-31 15:22:09.499067052 +0800 CST] Any255Base max() buf [1 0], 257
[2014-12-31 15:22:09.499080341 +0800 CST] generator# 5 current 54610 end at 257

*/

//
func (a *Any255Base) Init() string {
	maxbuf := make([]byte, a.size)
	for i := 0; i < a.size; i++ {
		maxbuf[i] = a.init[i]
	}
	bigNum := big.NewInt(0).SetBytes(maxbuf)
	Tpf("Any255Base Init() buf %v, %s\n", maxbuf, bigNum.String())
	return bigNum.String()
}

//
func (a *Any255Base) Max() string {
	maxbuf := make([]byte, a.size)
	for i := 0; i < a.size; i++ {
		maxbuf[i] = a.max[i]
	}
	bigNum := big.NewInt(0).SetBytes(maxbuf)
	bigNum = bigNum.Add(bigNum, big.NewInt(1))
	Tpf("Any255Base max() buf %v, %s\n", maxbuf, bigNum.String())
	return bigNum.String()
}

//
func (a *Any255Base) FromBigInt(b *big.Int) {
	// Bytes returns the absolute value of x as a big-endian byte slice.
	// func (x *Int) Bytes() []byte
	buf := b.Bytes()
	blen := len(buf)
	/*
		fptr := len(p) - 1
		for ptr := a.last; ptr >= 0 && fptr >= 0; ptr-- {
			p[fptr] = byte(a.count[ptr])
			fptr--
		}
	*/
	ptr := a.last
	for i := 0; i < blen && ptr >= 0; i++ {
		a.count[ptr] = buf[i]
		ptr--
	}
	a.ptr = ptr
}

//
func (a *Any255Base) ToBigInt() *big.Int {
	return big.NewInt(0).SetBytes(a.Bytes())
}

// Cmp compares x and y and returns:
//
//   -1 if a <  y
//    0 if a == y
//   +1 if a >  y
//
func (a *Any255Base) Cmp(b BigCounter) int {
	if ab, ok := b.(*Any255Base); ok {
		return a.cmp(ab)
	}
	a64 := a.Bytes()
	aptr := 0
	for aptr == 0 && aptr < len(a64) {
		aptr = bytes.IndexByte(a64[aptr:], byte(0))
	}
	b64 := b.Bytes()
	bptr := 0
	for bptr == 0 && bptr < len(b64) {
		bptr = bytes.IndexByte(b64[bptr:], byte(0))
	}
	if aptr < bptr {
		return -1
	}
	if aptr > bptr {
		return 1
	}
	for i := 0; i <= aptr; i++ {
		if a64[i] < b64[i] {
			return -1
		}
		if a64[i] > b64[i] {
			return 1
		}
	}
	return 0
}

// Cmp compares x and y and returns:
//
//   -1 if a <  y
//    0 if a == y
//   +1 if a >  y
//
func (a *Any255Base) cmp(b *Any255Base) int {
	// a < b
	if a.ptr > b.ptr {
		return -1
	}
	// a > b
	if a.ptr < b.ptr {
		return 1
	}
	// a.ptr == b.ptr
	for ptr := 0; ptr <= a.ptr; ptr++ {
		if a.count[ptr] < b.count[ptr] {
			return -1
		}
		if a.count[ptr] > b.count[ptr] {
			return 1
		}
	}
	return 0
}

//
func (a *Any255Base) AddUint64(m uint64) error {
	for i := uint64(0); i < m; i++ {
		if err := a.Plus(); err != nil {
			return fmt.Errorf("AddUint64 cross max %d, reset to %d", a.max, a.init)
		}
	}
	return nil
}

// Mul do a*m
func (a *Any255Base) Mul(m uint64) error {
	// counter == 0
	if a.count[a.last] == 0 && a.last == a.ptr {
		return nil
	}
	ptr := a.last
	for i := uint64(0); i < m; i++ {
		for {
			if a.count[ptr] == a.max[ptr] {
				a.count[ptr] = 0
				ptr--
				if ptr == -1 {
					// overflow
					a.Reset(true)
					ptr = a.last
					return fmt.Errorf("Plus cross max %d, reset to zero", a.max)
				}
				continue
			}
			a.count[ptr]++
			if ptr < a.ptr {
				a.ptr = ptr
			}
		}
	}
	return nil
}

//
func (a *Any255Base) SetMax(p []byte) {
	ptr := a.size - len(p)
	if ptr < 0 {
		ptr = 0
	}
	newbuf := make([]byte, ptr)
	newbuf = append(newbuf, p...)
	for i := ptr; i < a.size; i++ {
		a.max[i] = newbuf[i]
	}
}

//
func (a *Any255Base) SetInit(p []byte) {
	ptr := a.size - len(p)
	if ptr < 0 {
		ptr = 0
	}
	newbuf := make([]byte, ptr)
	newbuf = append(newbuf, p...)
	a.ptr = a.last
	for i := ptr; i < a.size; i++ {
		a.init[i] = newbuf[i]
		a.count[i] = a.init[i]
		if a.init[i] != 0x00 && i < a.ptr {
			a.ptr = i
		}
	}
}

// Plus do ++
func (a *Any255Base) Plus() error {
	ptr := a.last
	for {
		if a.count[ptr] == a.max[ptr] {
			a.count[ptr] = a.init[ptr]
			ptr--
			if ptr == -1 {
				// overflow
				a.Reset(true)
				ptr = a.last
				return fmt.Errorf("Plus cross max %d, reset to zero", a.max)
			}
			continue
		}
		a.count[ptr]++
		if ptr < a.ptr {
			a.ptr = ptr
		}
		return nil
	}
}

// Mimus do --
func (a *Any255Base) Mimus() error {
	for {
		if a.count[a.last] == a.init[a.last] {
			if a.ptr == a.last {
				// overflow
				a.Reset(false)
				return fmt.Errorf("Mimus cross zero, reset to max %d", a.max)
			} else {
				// a.ptr < a.last
				a.count[a.last] = a.max[a.last]
				next := a.last
				for {
					next--
					if a.count[next] == a.init[next] {
						a.count[next] = a.max[next] - 1
					} else {
						a.count[next]--
						break
					}
				}
				if a.count[next] == a.init[next] && a.ptr == next {
					a.ptr++
				}
			}
		} else {
			a.count[a.last]--
			return nil
		}
	}
}

func (a *Any255Base) Bytes() []byte {
	buf := make([]byte, a.size)
	return a.FillBytes(buf)
}

//
func (a *Any255Base) ExpBytes(size int) []byte {
	buf := make([]byte, size)
	return a.FillExpBytes(buf)
}

// FillBytes fill binary bytes of number
func (a *Any255Base) FillBytes(p []byte) []byte {
	if len(p) == 0 {
		return p
	}
	fptr := len(p) - 1
	for ptr := a.last; ptr >= 0 && fptr >= 0; ptr-- {
		p[fptr] = byte(a.count[ptr])
		fptr--
	}
	return p
}

// FillExpBytes fill binary bytes of number
func (a *Any255Base) FillExpBytes(p []byte) []byte {
	if len(p) == 0 {
		return p
	}
	fptr := len(p) - 1
	step := fptr / a.size
	if step <= 0 {
		step = 1
	}
	for ptr := a.last; ptr >= 0 && fptr >= 0; ptr-- {
		p[fptr] = byte(a.count[ptr])
		fptr = fptr - step
	}
	return p
}

// Touint64 return uint64 of Any255Base
func (a *Any255Base) Touint64() []uint64 {
	buf := a.Bytes()
	cnt := len(buf) / 8
	if len(buf)%8 != 0 {
		cnt++
	}
	allbytes := make([]byte, cnt*8)
	copy(allbytes, buf)
	u64 := make([]uint64, cnt)
	for i := 0; i <= cnt; i++ {
		u64[i], _ = binary.Uvarint(allbytes[i*8:])
	}
	return u64
}

// Reset to up flow or down flow
// over == true, fill to zero
// over == false, fill to MAX
func (a *Any255Base) Reset(over bool) {
	fmt.Printf("reset %v when count %v\n", over, a.count)
	for ptr := 0; ptr <= a.last; ptr++ {
		if over {
			// up flow
			a.count[ptr] = a.init[ptr]
		} else {
			a.count[ptr] = a.max[ptr]
		}
	}
	if over {
		// up flow
		a.ptr = a.last
	} else {
		// down flow
		a.ptr = 0
	}
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

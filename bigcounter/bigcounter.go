// bigcounter is fast then math/big.Int

package bigcounter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
)

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
	fmt.Printf("Any255Base Init() buf %v, %s\n", maxbuf, bigNum.String())
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
	fmt.Printf("Any255Base max() buf %v, %s\n", maxbuf, bigNum.String())
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

//

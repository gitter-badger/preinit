//
// Codec and Checksum for Common Multiplexing Transport Proxy (CMTP)
//
//

//
package cmtp

import (
	"errors"

	"github.com/wheelcomplex/preinit/misc"
	"github.com/wheelcomplex/preinit/murmur3"
	"github.com/wheelcomplex/preinit/snappy"
	"github.com/wheelcomplex/preinit/xxhash"
)

// Xxhash is a implementation of CMTP Checksum
type Xxhash struct {
	// nothing here
}

// NewXxhash return new *Xxhash
func NewXxhash(seed uint32) *Xxhash {
	return &Xxhash{}
}

// New return new *Xxhash
func (h *Xxhash) New(seed uint32) Checksum {
	return NewXxhash(seed)
}

// Write (via the embedded io.Writer interface) adds more data to the running hash.
// It never returns an error.
func (nc *Xxhash) Write(p []byte) (n int, err error) {
	return
}

// Sum32 returns the sum of data in uint32
func (h *Xxhash) Sum32() uint32 {
	return uint32(9876)
}

// Sum64 returns the sum of data in uint64
func (h *Xxhash) Sum64() uint64 {
	return uint64(9876)
}

// Checksum32 return uint32(9876)
func (nc *Xxhash) Checksum32(data []byte) uint32 {
	return xxhash.Checksum32(data)
}

// Checksum64 return 6789
func (nc *Xxhash) Checksum64(data []byte) uint64 {
	return uint64(xxhash.Checksum32(data))
}

// Murmur3 is a implementation of CMTP Checksum
type Murmur3 struct {
	// nothing here
}

// NewMurmur3 return new *Murmur3
func NewMurmur3(seed uint32) *Murmur3 {
	return &Murmur3{}
}

// New return new *Murmur3
func (h *Murmur3) New(seed uint32) Checksum {
	return NewMurmur3(seed)
}

// Write (via the embedded io.Writer interface) adds more data to the running hash.
// It never returns an error.
func (nc *Murmur3) Write(p []byte) (n int, err error) {
	return
}

// Sum32 returns the sum of data in uint32
func (h *Murmur3) Sum32() uint32 {
	return uint32(9876)
}

// Sum64 returns the sum of data in uint64
func (h *Murmur3) Sum64() uint64 {
	return uint64(9876)
}

// Checksum32 return uint32(9876)
func (nc *Murmur3) Checksum32(data []byte) uint32 {
	return murmur3.Sum32(data)
}

// Checksum64 return 6789
func (nc *Murmur3) Checksum64(data []byte) uint64 {
	return murmur3.Sum64(data)
}

// NoopChecksum implemented Checksum interface with NOOP
type NoopChecksum struct{}

// NewNoopChecksum return new *NoopChecksum
func NewNoopChecksum(seed uint32) *NoopChecksum {
	return &NoopChecksum{}
}

// New return new Checksum
func (nc *NoopChecksum) New(seed uint32) Checksum {
	return NewNoopChecksum(seed)
}

// Sum32 return uint32(9876)
func (nc *NoopChecksum) Sum32() uint32 {
	return uint32(9876)
}

// Sum64 return uint64(9876)
func (nc *NoopChecksum) Sum64() uint64 {
	return uint64(9876)
}

// Checksum32 return uint32(9876)
func (nc *NoopChecksum) Checksum32(data []byte) uint32 {
	return uint32(9876)
}

// Checksum64 return 6789
func (nc *NoopChecksum) Checksum64(data []byte) uint64 {
	return uint64(6789)
}

// Write (via the embedded io.Writer interface) adds more data to the running hash.
// It never returns an error.
func (nc *NoopChecksum) Write(p []byte) (n int, err error) {
	return
}

// Sum appends the current hash to b and returns the resulting slice.
// It does not change the underlying hash state.
func (nc *NoopChecksum) Sum(b []byte) []byte {
	b = append(b, []byte("6789")...)
	return b
}

// Reset resets the Checksum to its initial state.
func (nc *NoopChecksum) Reset() {

}

// Size returns the number of bytes Sum will return.
func (nc *NoopChecksum) Size() int {
	return 4
}

// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.
func (nc *NoopChecksum) BlockSize() int {
	return 8
}

// NoopCodec implemented Codec interface with NOOP
type NoopCodec struct{}

// NewNoopCodec return new *NoopCodec
func NewNoopCodec() *NoopCodec {
	return &NoopCodec{}
}

// New return new Codec
func (nc *NoopCodec) New() Codec {
	return NewNoopCodec()
}

// return max length of decoded []byte when decode src
func (nc *NoopCodec) MaxDecodedLen(src []byte) (int, error) {
	return len(src), nil
}

// Decode return copy of src
func (nc *NoopCodec) Decode(dst, src []byte) (p []byte, err error) {
	if len(dst) < len(src) {
		return nil, &misc.CommError{Code: DECODE_ERR_BUFLEN, Err: errors.New("dst buffer len too small for decode")}
	}
	nw, ew := misc.NewBRWC(dst).Write(src)
	if ew != nil {
		return nil, ew
	}
	return dst[:nw], nil
}

// return max length of encoded []byte when encode src
func (nc *NoopCodec) MaxEncodedLen(srcLen int) int {
	return srcLen
}

// Encode return copy of src
func (nc *NoopCodec) Encode(dst, src []byte) (p []byte, err error) {
	if len(dst) < len(src) {
		return nil, &misc.CommError{Code: ENCODE_ERR_BUFLEN, Err: errors.New("dst buffer len too small for encode")}
	}
	nw, ew := misc.NewBRWC(dst).Write(src)
	if ew != nil {
		return nil, ew
	}
	return dst[:nw], nil
}

// SnappyCodec implemented Codec interface with https://code.google.com/p/snappy-go/
type SnappyCodec struct {
}

// NewSnappyCodec return new SnappyCodec
func NewSnappyCodec() *SnappyCodec {
	return &SnappyCodec{}
}

// New return new *SnappyCodec
func (nc *SnappyCodec) New() Codec {
	return NewSnappyCodec()
}

// return max length of decoded []byte when decode src
func (nc *SnappyCodec) MaxDecodedLen(src []byte) (int, error) {
	return snappy.DecodedLen(src)
}

// Decode return copy of src
func (nc *SnappyCodec) Decode(dst, src []byte) (p []byte, err error) {
	return snappy.Decode(dst, src)
}

// return max length of encoded []byte when encode src
func (nc *SnappyCodec) MaxEncodedLen(srcLen int) int {
	return snappy.MaxEncodedLen(srcLen)
}

// Encode return copy of src
func (nc *SnappyCodec) Encode(dst, src []byte) (p []byte, err error) {
	return snappy.Encode(dst, src)
}

/*
	Package keyaes provides key base aes encrypt/decrypt with nonce iv
	focus on easy to use and fair security
*/

package keyaes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

// The length of the AES key, either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256
const AES_KEYLEN int = 16

// IVSCOUNT size of iv map, 1k * aes.BlockSize
// must > 8
const IVSCOUNT int = 1024

// AES implemented crypto/aes encryption/decryption with nonce iv
// using PKCS7Padding
type AES struct {
	key          []byte           // AES-128
	ivs          []byte           // N * iv map
	ivslast      uint64           // last position of N * iv map
	nonce        uint64           // index of iv
	encryptsum   uint32           // checksum of plaintext
	decryptsum   uint32           // checksum of plaintext
	sumlen       int              // length of binary sum
	hdrlen       int              // length of binary nonce
	uuid         chan uint64      // uuid output for nonce
	eniv         []byte           // iv for encrypt
	deiv         []byte           // iv for decrypt
	block        cipher.Block     //
	blockEncrypt cipher.BlockMode //
	blockDecrypt cipher.BlockMode //
	mutex        sync.Mutex       //
	hash         hash.Hash32      //
}

// NewAES create new *AES with key and iv map
// recommad hash is murmur3.New32()
func NewAES(key []byte, hash hash.Hash32) *AES {
	var err error
	ae := &AES{
		hash: hash,
	}
	//
	ae.sumlen = binary.Size(ae.encryptsum)
	ae.hdrlen = binary.Size(ae.nonce)
	ae.keyinitial(key)
	ae.block, err = aes.NewCipher(ae.key)
	if err != nil {
		panic(fmt.Sprintf("aes.NewCipher failed: %s", err.Error()))
	}
	ae.blockEncrypt = cipher.NewCBCEncrypter(ae.block, ae.eniv)
	ae.blockDecrypt = cipher.NewCBCDecrypter(ae.block, ae.deiv)
	return ae
}

// checksum return hash sum base on AES key
func (ae *AES) checksum(p []byte) (s uint32) {
	ae.hash.Reset()
	//ae.hash.Write(iv)
	ae.hash.Write(p)
	ae.hash.Write(ae.key)
	s = ae.hash.Sum32()
	ae.hash.Reset()
	return
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
	ae.ivslast = uint64(IVSCOUNT*aes.BlockSize - aes.BlockSize)
	ae.ivs = FnvFastExpend(ae.key, IVSCOUNT*aes.BlockSize)
	// println("ae.ivs", len(ae.ivs))
	//
	ae.uuidinitial()
}

func (ae *AES) uuidGen(r *rand.Rand) {
	defer func() {
		// handle close
		recover()
	}()
	for {
		ae.uuid <- uint64(r.Int63())
	}
}

// uuidinitial
func (ae *AES) uuidinitial() {
	if ae.uuid != nil {
		return
	}
	ae.uuid = make(chan uint64, 2048)
	h := fnv.New64a()
	h.Write(ae.key)
	h.Write([]byte(strconv.FormatInt(int64(time.Now().UnixNano()), 10) + ":" + strconv.Itoa(os.Getpid()) + ":" + strconv.Itoa(os.Getppid())))
	r := rand.New(rand.NewSource(int64(h.Sum64())))
	h.Reset()
	go ae.uuidGen(r)
	return
}

// ivnonce get nonce iv pair
// if input n == 0, return random nonce + iv pair
func (ae *AES) ivnonce(n uint64, isEncrypt bool) uint64 {
	if n == 0 {
		n = <-ae.uuid
	}
	ptr := n % ae.ivslast
	//ptr := (int(n) % IVSCOUNT) * aes.BlockSize
	if isEncrypt {
		copy(ae.eniv, ae.ivs[ptr:ptr+aes.BlockSize])
		// overwrite 8 bytes wih nonce
		binary.BigEndian.PutUint64(ae.eniv, uint64(n))
	} else {
		copy(ae.deiv, ae.ivs[ptr:ptr+aes.BlockSize])
		// overwrite 8 bytes wih nonce
		binary.BigEndian.PutUint64(ae.deiv, uint64(n))
	}
	return n
}

// PackSize return size of pack for srclen, and paddingsize included in packsize
func (ae *AES) PackSize(srclen int) (packsize, paddingsize int) {
	paddingsize = aes.BlockSize - ((srclen + ae.sumlen) % aes.BlockSize)
	packsize = ae.sumlen + srclen + paddingsize
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
	dst = dst[:ae.sumlen]
	// write checksum of src
	ae.encryptsum = ae.checksum(src)
	binary.BigEndian.PutUint32(dst, ae.encryptsum)
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
	if offset > aes.BlockSize || offset <= 0 {
		//return nil, fmt.Errorf("decryptUnPack, invalid input: length %d unpadding %d offset %d", length, unpadding, offset)
		return nil, fmt.Errorf("decryptUnPack, invalid input: invalid offset")
	}
	if offset < ae.sumlen {
		return nil, fmt.Errorf("decryptUnPack, invalid input: invalid unpadding length")
	}
	//ae.decryptsum = uint64(ByteUUID(plainText[ae.hdrlen:offset]))
	ae.decryptsum = ae.checksum(plainText[ae.sumlen:offset])
	ae.encryptsum = binary.BigEndian.Uint32(plainText[:ae.sumlen])
	if ae.decryptsum != ae.encryptsum {
		return nil, fmt.Errorf("decryptUnPack, invalid input: checksum mismatch, need %x, got %x", ae.decryptsum, ae.encryptsum)
		//return nil, fmt.Errorf("decryptUnPack, invalid input: checksum mismatch")
	}
	return plainText[ae.sumlen:offset], nil
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
	binary.BigEndian.PutUint64(encryptText[:ae.hdrlen], uint64(ae.nonce))
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
	ae.nonce = binary.BigEndian.Uint64(src[:ae.hdrlen])
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
	ae.hash.Reset()
	close(ae.uuid)
}

//

// FnvUintExpend use hash to expend uint64
func FnvUintExpend(init uint64, size int) []byte {
	buf := make([]byte, binary.Size(&init))
	binary.BigEndian.PutUint64(buf, uint64(init))
	return FnvExpend(buf, size)
}

// FnvUintFastExpend use hash to expend uint64
func FnvUintFastExpend(init uint64, size int) []byte {
	buf := make([]byte, binary.Size(&init))
	binary.BigEndian.PutUint64(buf, uint64(init))
	return FnvFastExpend(buf, size)
}

// FnvExpend use hash to expend []byte
func FnvExpend(init []byte, size int) []byte {
	exp := make([]byte, 0, size)
	h := fnv.New32a()
	h.Write(init)
	buf := h.Sum(nil)
	for idx, _ := range buf {
		exp = append(exp, buf[idx])
		if len(exp) >= size {
			break
		}
	}
	for len(exp) < size {
		// key is short
		h.Write(exp)
		buf := h.Sum(nil)
		for idx, _ := range buf {
			exp = append(exp, buf[idx])
			if len(exp) >= size {
				break
			}
		}
	}
	return exp
}

// FnvFastExpend use hash and byte shift to expend []byte
func FnvFastExpend(init []byte, size int) []byte {
	exp := make([]byte, 0, size)
	h := fnv.New32a()
	h.Write(init)
	buf := h.Sum(nil)
	for idx, _ := range buf {
		exp = append(exp, buf[idx])
		if len(exp) >= size {
			break
		}
	}
	ptr := len(exp)
	offset := 0
	for len(exp) < size {
		if ptr == offset {
			ptr = len(exp)
			offset = ptr - aes.BlockSize - aes.BlockSize - aes.BlockSize
			if offset < 0 {
				offset = 0
			}
			wptr := int(len(exp) / 3)
			for {
				if len(exp) >= size || wptr == 0 {
					break
				}
				exp = append(exp, exp[wptr])
				wptr--
			}
			wptr = int(len(exp) / 3)
			keylen := len(exp)
			for {
				if len(exp) >= size || wptr == 0 {
					break
				}
				exp = append(exp, exp[keylen-wptr])
				wptr--
			}
			continue
		}
		// key is short
		h.Write(exp[offset:ptr])
		ptr--
		buf := h.Sum(nil)
		for idx, _ := range buf {
			exp = append(exp, buf[idx])
			if len(exp) >= size {
				break
			}
		}
	}
	return exp
}

//

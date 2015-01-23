//
// Common Multiplexing Transport Proxy (CMTP)
//
//time cat x.iso |dd | nc 198.19.0.10 9999
//1869824+0 records in
//1869824+0 records out
//957349888 bytes (957 MB) copied, 8.39437 s, 114 MB/s

//real	0m8.405s
//user	0m2.859s
//sys	0m7.158s

//time cat x.iso |dd | nc 127.0.0.1 9999
//1869824+0 records in
//1869824+0 records out
//957349888 bytes (957 MB) copied, 3.05804 s, 313 MB/s

//real	0m3.062s
//user	0m2.294s
//sys	0m4.520s

//
package cmtp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/wheelcomplex/preinit/keyaes"
	"github.com/wheelcomplex/preinit/misc"
)

//
// Common Multiplexing Transport Proxy (CMTP)
//

// use CHANNEL_BUFFER_SIZE channel buffer size for speed
// in benchmark channel buffer size greate then 1024 got max channel forward speed
const CHANNEL_BUFFER_SIZE int = 2048

const CHANNEL_INIT_SIZE int = 16

// max empty io read
const CMTP_MAX_EMPTY_IO int = 16

const CMTP_MAX_UNMARSHAL_ERR int = 16

// initial buffer size for ReadFrom/WriteTo
const CMTP_BUF_INIT int = 4 * 1024

//
const CMTP_SMALL_PKG_SIZE uint64 = 1024

// max buffer size for ReadFrom/WriteTo
const CMTP_BUF_MAX int = 512 * 1024

// uuid source
var uuidCh = misc.NewUUIDChan().C

// following tcp/ip to use binary.BigEndian
var CMTP_ENDIAN = binary.BigEndian

// initial session id
const INIT_SSID uint64 = 1

// initial index id for session
const INIT_INDEX uint64 = 1

// frame type magic code
const (
	FRAME_MAGIC_UNSET uint64 = 0xFF00 + iota
	//
	FRAME_PAYLOADSTART   // stream mark
	FRAME_PAYLOADENCODED // sending stream
	FRAME_PAYLOADPALIN   // sending stream
	FRAME_PAYLOADLAST    // stream mark
	FRAME_MSGSTART       // CMsg mark
	FRAME_MSGERR         // sending CMsg
	FRAME_MSGNEWSSID     // sending CMsg
	FRAME_MSGDELSSID     // sending CMsg
	FRAME_MSGLAST        // CMsg mark
	FRAME_LINKSTART      // link mark
	FRAME_LINKNOOP       // sending CMsg
	FRAME_LINKACK        // sending CMsg
	FRAME_LINKLAST       // link mark
	FRAME_MAGIC_LAST
)

// error code fo Marshal/UnMashal/io
const (
	MARSHAL_ERR_UNSET uint64 = 0xEE00 + iota
	MARSHAL_ERR_EMPTY_WRITEIO
	MARSHAL_ERR_BINARY
	MARSHAL_ERR_INPUTID
	MARSHAL_ERR_HDR
	MARSHAL_ERR_OPTION
	MARSHAL_ERR_BODY
	MARSHAL_ERR_LAST
	//
	UNMARSHAL_ERR_UNSET
	UNMARSHAL_ERR_EMPTY_READIO
	UNMARSHAL_ERR_BINARY
	UNMARSHAL_ERR_BUFLEN
	UNMARSHAL_ERR_INPUTID
	UNMARSHAL_ERR_HDRLEN
	UNMARSHAL_ERR_HDR
	UNMARSHAL_ERR_INVALID
	UNMARSHAL_ERR_OPTIONLEN
	UNMARSHAL_ERR_OPTION
	UNMARSHAL_ERR_PAYLOADLEN
	UNMARSHAL_ERR_PAYLOAD
	UNMARSHAL_ERR_TOOLARGE
	UNMARSHAL_ERR_LAST
	//
	IO_ERR_DIAL
	IO_ERR_READ
	IO_ERR_WRITE
	IO_ERR_CLOSED
)

// error code fo encode/decode
const (
	ENCODE_ERR_UNSET int = 0xEE00 + iota
	ENCODE_ERR_EMPTY_WRITEIO
	ENCODE_ERR_BUFLEN
	ENCODE_ERR_ZCOPY
	ENCODE_ERR_NIL
	ENCODE_ERR_LAST
	//
	DECODE_ERR_UNSET
	DECODE_ERR_EMPTY_READIO
	DECODE_ERR_BUFLEN
	DECODE_ERR_ZCOPY
	DECODE_ERR_NIL
	DECODE_ERR_LAST
)

// Checksum used for frame header checksum
type Checksum interface {
	// New return a new interface
	New(seed uint32) Checksum

	// Checksum32 returns the sum of data in uint32
	Checksum32(data []byte) uint32
}

// Codec specifies a interface use for encode/decode []byte block
type Codec interface {

	// New return a new interface
	New() Codec

	// return max length of decoded []byte when decode src
	MaxDecodedLen(src []byte) (int, error)

	// decode src and save into dst
	// if len(dst) < MaxDecodedLen(src), return error with code DECODE_ERR_BUFLEN
	// returned p []byte + nil, p []byte is the sub-set of dst []byte,
	// modify p will effect underlay dst
	Decode(dst, src []byte) (p []byte, err error)

	// return max length of encoded []byte when encode src
	MaxEncodedLen(srcLen int) int

	// encode src and save into dst/
	// if len(dst) < MaxEncodedLen(src), return error with code ENCODE_ERR_BUFLEN
	// returned p []byte + nil, p []byte is the sub-set of dst []byte,
	// modify p will effect underlay dst
	Encode(dst, src []byte) (p []byte, err error)
}

// Filter implemented io.ReadWriteCloser
type Filter interface {

	// New create a new Filter, using rw io.ReadWriteCloser as underlay io
	New(ssid uint64, rw io.ReadWriteCloser) Filter

	// Reader encode raw stream from underlay io.Reader to header+body
	// encoder should marshal underlay io.Reader information into header, eg,.filename, dest host:port
	Read(p []byte) (n int, err error)

	// Writer decode header+body from raw stream
	// decoder should unmarshal information from header, eg,.filename, dest host:port
	// and create coresponing underlay io.Writer
	Write(p []byte) (n int, err error)

	// Close discard all internal resource
	// will close underlay io.ReadWriteCloser
	Close()
}

////////

// TX frame 48 bytes header
// = binary.Size(uint64(0))(frametype)
// + binary.Size(uint64(0))(ssid)
// + binary.Size(uint64(0))(index)
// + binary.Size(uint64(0))(payloadlen)
// + binary.Size(uint64(0))(bodylen)
// + binary.Size(uint32(0))(murmur3.sum32)
// + payload []byte
//
// mixerFrame carry real buffer used in io
//
// SECURITY: one-by-one byte attack
// TODO: move one-by-one byte to direct-proxy link
//
type mixerFrame struct {
	frametype  uint64 // type of this frame, check FRAME_*
	ssid       uint64 // session id of frame
	index      uint64 // frame index of frame
	payloadlen uint64 // payload length of frame
	bodylen    uint64 // body lenght of frame
	hdrsum     uint32 // checksum32 of header
	sumlen     int    // header len for marshal without checksum slot
	hdrlen     int    // header len for marshal
	framebuf   []byte // buffer use fo marshal and codec
	frameptr   int    // current opration position of buffer
	iobuf      []byte // io buffer use fo WriteTo
	ioptr      int    // current opration position for WriteTo
	iolen      int    // pre-io length
}

func newMixerFrame() *mixerFrame {
	mf := &mixerFrame{
		frametype: FRAME_MAGIC_UNSET,
		ssid:      INIT_SSID,
	}
	mf.hdrlen = binary.Size(mf.frametype) +
		binary.Size(mf.ssid) +
		binary.Size(mf.index) +
		binary.Size(mf.payloadlen) +
		binary.Size(mf.bodylen) +
		binary.Size(mf.hdrsum)

	mf.sumlen = mf.hdrlen - binary.Size(mf.hdrsum)
	mf.framebuf = make([]byte, mf.hdrlen*10)
	mf.iobuf = make([]byte, mf.hdrlen+CMTP_BUF_INIT)
	return mf
}

// headerInit fill header values
func (mf *mixerFrame) headerInit(frametype, ssid, index uint64) {
	mf.frametype = frametype
	mf.ssid = ssid
	mf.index = index
	mf.bodylen = uint64(mf.ioptr)
}

// MarshalHeader
func (mf *mixerFrame) MarshalHeader(checksum Checksum, count int) {
	//
	// marshal header
	//
	// TX frame 48 bytes header
	// = binary.Size(uint64(0))(magic code)
	// + binary.Size(uint64(0))(ssid)
	// + binary.Size(uint64(0))(index)
	// + binary.Size(uint64(0))(payloadlen)
	// + binary.Size(uint64(0))(bodylen)
	// + binary.Size(uint32(0))(frametype)
	// + binary.Size(uint32(0))(murmur3.sum32)
	// + payload []byte
	//
	//binary.Size(mf.magic) +
	//binary.Size(mf.ssid) +
	//binary.Size(mf.index) +
	//binary.Size(mf.payloadlen) +
	//binary.Size(mf.bodylen) +
	//binary.Size(mf.frametype) +
	//binary.Size(mf.hdrsum)
	//
	// marshal header
	//
	offset := 0
	err := binary.Write(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.frametype)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, marshal frametype failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	offset += binary.Size(mf.frametype)
	//
	err = binary.Write(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.ssid)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, marshal ssid failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	offset += binary.Size(mf.ssid)
	//
	err = binary.Write(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.index)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, marshal index failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	offset += binary.Size(mf.index)
	//
	err = binary.Write(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.payloadlen)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, marshal payloadlen failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	offset += binary.Size(mf.payloadlen)
	//
	err = binary.Write(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.bodylen)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, marshal bodylen failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	offset += binary.Size(mf.bodylen)
	//
	mf.hdrsum = checksum.Checksum32(mf.framebuf[:mf.sumlen])
	//
	err = binary.Write(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.hdrsum)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, marshal hdrsum failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
}

// UnMarshalHeader
func (mf *mixerFrame) UnMarshalHeader(checksum Checksum, count int) error {
	//
	// unmarshal header
	//
	// TX frame 48 bytes header
	// = binary.Size(uint64(0))(frametype)
	// + binary.Size(uint64(0))(ssid)
	// + binary.Size(uint64(0))(index)
	// + binary.Size(uint64(0))(payloadlen)
	// + binary.Size(uint64(0))(bodylen)
	// + binary.Size(uint32(0))(murmur3.sum32)
	// + payload []byte
	//
	//binary.Size(mf.frametype) +
	//binary.Size(mf.ssid) +
	//binary.Size(mf.index) +
	//binary.Size(mf.payloadlen) +
	//binary.Size(mf.bodylen) +
	//binary.Size(mf.hdrsum)
	//
	// unmarshal header
	//
	mf.index = 0
	mf.frametype = FRAME_MAGIC_UNSET
	//
	if mf.frameptr < mf.hdrlen {
		return &CMsg{
			Code: uint64(UNMARSHAL_ERR_HDRLEN),
		}
	}
	offset := 0
	err := binary.Read(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.frametype)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, unmarshal frametype failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	//
	if mf.frametype <= FRAME_MAGIC_UNSET || mf.frametype >= FRAME_MAGIC_LAST {
		return &CMsg{
			Code: uint64(UNMARSHAL_ERR_INVALID),
			Err:  fmt.Errorf("decodeLoop#%d, %d, index %d, unmarshal frametype failed: invalid frametype, should be %d > %d > %d", count, mf.ssid, mf.index, FRAME_MAGIC_LAST, mf.frametype, FRAME_MAGIC_UNSET),
		}
	}
	offset += binary.Size(mf.frametype)
	//
	err = binary.Read(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.ssid)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, unmarshal ssid failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	offset += binary.Size(mf.ssid)
	//
	err = binary.Read(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.index)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, unmarshal index failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	offset += binary.Size(mf.index)
	//
	err = binary.Read(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.payloadlen)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, unmarshal payloadlen failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	offset += binary.Size(mf.payloadlen)
	//
	err = binary.Read(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.bodylen)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, unmarshal bodylen failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	offset += binary.Size(mf.bodylen)
	//
	mf.hdrsum = checksum.Checksum32(mf.framebuf[:mf.sumlen])
	//
	err = binary.Read(misc.NewBRWC(mf.framebuf[offset:]), CMTP_ENDIAN, &mf.hdrsum)
	if err != nil {
		panic(fmt.Sprintf("decodeLoop#%d, %d, index %d, unmarshal hdrsum failed: %s", count, mf.ssid, mf.index, err.Error()))
	}
	return nil
}

func (mf *mixerFrame) Reset() {
	mf.frametype = FRAME_MAGIC_UNSET
	mf.ioptr = 0
	mf.iolen = 0
	mf.frameptr = 0
}

// frameStream use for half of a connection
type frameStream struct {
	index   uint64           // current frame index
	codec   Codec            // codec use for this session
	interCh chan *mixerFrame // internal data
}

// newFrameStream return new *frameStream
func newFrameStream(codec Codec) *frameStream {
	// use CHANNEL_BUFFER_SIZE channel buffer size for speed
	return &frameStream{
		index:   INIT_INDEX,
		codec:   codec,
		interCh: make(chan *mixerFrame, CHANNEL_BUFFER_SIZE),
	}
}

// mixerSession use for a connection
type mixerSession struct {
	ssid   uint64       // id of this session
	hash   Checksum     // hash for header
	filter Filter       // filter work with this session
	in     *frameStream // assemble side of a connection
	out    *frameStream // disassemble side of a connection
}

// newMixerSession return new *newMixerSession
func newMixerSession(ssid uint64, codec Codec, hash Checksum, filter Filter) *mixerSession {
	return &mixerSession{
		ssid:   ssid,
		hash:   hash,
		in:     newFrameStream(codec.New()),
		out:    newFrameStream(codec.New()),
		filter: filter,
	}
}

//
// CodecMixer implemented TXMixer + RXMixer
//
// CodecMixer implemented Multiplexing Transport with codec
//
type CodecMixer struct {
	ssid           uint64                      // ssid counter
	mutex          sync.Mutex                  // common lock
	token          uint64                      // token id
	aes            *keyaes.AES                 // crypter for handshake
	key            []byte                      //
	ioFreeCh       chan *mixerFrame            // idle frame
	encodeCh       chan *mixerFrame            // encode frame
	decodeCh       chan *mixerFrame            // decode frame
	assembleFastCh chan *mixerFrame            // assemble write frame
	assembleCh     chan *mixerFrame            // assemble write frame
	codec          Codec                       // frame codec
	checksum       Checksum                    // frame header hash
	filter         Filter                      // frame filter
	state          MIXER_STATE                 // mixer state
	closing        chan struct{}               // close notify for all ReadFrom/WriteTo
	closed         chan string                 // closed goroutine
	cpus           int                         // number of using cpus
	exitMsg        chan CMsg                   // goroutine exit msg
	mixerCh        chan io.ReadWriteCloser     // underlay io.ReadWriteCloser
	ssmutex        map[uint64]*sync.Mutex      // lock for session index by ssid
	ssindex        map[uint64]uint64           // index for session index by ssid
	sscodecList    map[uint64]Codec            // codec map to ssid
	ssWriteCh      map[uint64]chan *mixerFrame // disassemble write frame
}

// newCodecMixer return *CodecMixer
func newCodecMixer(token uint64, codec Codec, checksum Checksum, filter Filter) *CodecMixer {
	cpus := int(float32(runtime.GOMAXPROCS(-1)) * 2.618)
	chansize := cpus*5 + CHANNEL_BUFFER_SIZE
	//
	// expend token to be aes key
	//
	key := keyaes.FnvUintExpend(token, keyaes.AES_KEYLEN)

	tf := &CodecMixer{
		ssid:           INIT_SSID,
		token:          token,
		aes:            keyaes.NewAES(key, nil),
		key:            key,
		ioFreeCh:       make(chan *mixerFrame, chansize*2),
		encodeCh:       make(chan *mixerFrame, chansize),
		decodeCh:       make(chan *mixerFrame, chansize),
		assembleFastCh: make(chan *mixerFrame, chansize),
		assembleCh:     make(chan *mixerFrame, chansize),
		ssWriteCh:      make(map[uint64]chan *mixerFrame),
		codec:          codec,
		checksum:       checksum,
		filter:         filter,
		state:          MIXER_STATE_RESET,
		closing:        make(chan struct{}, cpus*10),
		closed:         make(chan string, cpus*10),
		cpus:           cpus,
		exitMsg:        make(chan CMsg, cpus*10),
		mixerCh:        make(chan io.ReadWriteCloser, cpus*5),
		sscodecList:    make(map[uint64]Codec),
		ssmutex:        make(map[uint64]*sync.Mutex),
		ssindex:        make(map[uint64]uint64),
	}
	tf.sscodecList[INIT_SSID] = tf.codec.New()
	tf.ssmutex[INIT_SSID] = &sync.Mutex{}
	tf.ssindex[INIT_SSID] = INIT_INDEX
	tf.ssWriteCh[INIT_SSID] = make(chan *mixerFrame, CHANNEL_BUFFER_SIZE)
	for i := 0; i < cpus; i++ {
		tf.ioFreeCh <- newMixerFrame()
	}
	// launch codec goroutine
	for i := 0; i < tf.cpus; i++ {
		go tf.encodeLoop(i)
		go tf.decodeLoop(i)
	}
	// launch ReadWriter goroutine for back2back connection
	go tf.runReadWriter()
	return tf
}

// initssid
func (tf *CodecMixer) initssid(ssid uint64) uint64 {
	tf.mutex.Lock()
	defer tf.mutex.Unlock()
	if ssid == 0 {
		tf.ssid++
		ssid = tf.ssid
	}
	if _, ok := tf.sscodecList[ssid]; ok {
		return ssid
	}
	tf.sscodecList[ssid] = tf.codec.New()
	tf.ssmutex[ssid] = &sync.Mutex{}
	tf.ssindex[ssid] = INIT_INDEX
	tf.ssWriteCh[ssid] = make(chan *mixerFrame, tf.cpus*5+1024)
	return ssid
}

// clearssid
func (tf *CodecMixer) clearssid(ssid uint64) {
	tf.mutex.Lock()
	defer tf.mutex.Unlock()
	if ssid == 0 {
		return
	}
	if _, ok := tf.sscodecList[ssid]; ok == false {
		return
	}
	delete(tf.sscodecList, ssid)
	delete(tf.ssmutex, ssid)
	delete(tf.ssindex, ssid)
	close(tf.ssWriteCh[ssid])
	delete(tf.ssWriteCh, ssid)
}

// growindex
func (tf *CodecMixer) growindex(ssid uint64) uint64 {
	tf.ssmutex[ssid].Lock()
	tf.ssindex[ssid]++
	defer tf.ssmutex[ssid].Unlock()
	return tf.ssindex[ssid]
}

// acceptClient (assamble side), running in goroutine, call newSession for new client
func (tf *CodecMixer) acceptClient(nl *net.TCPListener) {
}

// newSession, create ssid for new client session
// call ReadFrom to read plain data from new client
// call WriteTo to write session data from peer to new client
// ReadFrom/WriteTo run in goroutine
func (tf *CodecMixer) newSession(rw *net.TCPConn) error {
	return nil
}

// acceptPeer, running in goroutine, accept new peer connection
// call newPeer for new peer connection
func (tf *CodecMixer) acceptPeer(nl *net.TCPListener) {
}

// newPeer, do handshake to match token,
// call ReadFrom to read frame from peer
// call WriteTo to write frame to peer
// ReadFrom/WriteTo run in goroutine
func (tf *CodecMixer) newPeer(rw *net.TCPConn, isActive bool) error {
	if isActive {
		if err := tf.peerServerHandshake(rw); err != nil {
			return err
		}
	} else {
		if err := tf.peerClientHandshake(rw); err != nil {
			return err
		}
	}
	return nil
}

// peerServerHandshake waitting for token msg and write ack back to remote peer
func (tf *CodecMixer) peerServerHandshake(rw *net.TCPConn) error {
	return nil
}

// peerClientHandshake write token msg to remote peer and wait for ack msg
func (tf *CodecMixer) peerClientHandshake(rw *net.TCPConn) error {
	return nil
}

//
func (tf *CodecMixer) addMixerIO(rw io.ReadWriteCloser) {
	println("addMixerIO", misc.GetXID(rw), "...")
	tf.mixerCh <- rw
}

//
func (tf *CodecMixer) sendExitCMsg(code uint64, err error) {
	go func() {
		tf.exitMsg <- CMsg{
			Code: code,
			Err:  err,
		}
	}()
}

//
func (tf *CodecMixer) runReadWriter() {
	// one goroutine/mixer io
	var rw io.ReadWriteCloser
	var count int
	for {
		select {
		case rw = <-tf.mixerCh:
			println("runReadWriter", misc.GetXID(rw), "running ...")
		default:
			println("block, runReadWriter", "waiting for new mixer channel ...")
			rw = <-tf.mixerCh
			println("unblock, runReadWriter", misc.GetXID(rw), "running ...")
		}
		count++
		go tf.writeLoop(rw, count)
		go tf.readLoop(rw, count)
	}

}

// writeLoop fetch *mixerFrame from assembleFastCh+assembleCh and write to rw io.ReadWriteCloser
// no ordering contorl
// if write to rw io.ReadWriteCloser failed, put *mixerFrame back to assembleFastCh and exit
// put *mixerFrame back to assembleCh after write done
func (tf *CodecMixer) writeLoop(rw io.ReadWriteCloser, count int) {
	defer rw.Close()
	fmt.Printf("writeLoop#%d, %s, running ...\n", count, misc.GetXID(rw))

	var mf *mixerFrame
	var ok bool
	var err error
	var ioptr, iobytes, emptyio int
	for {
		// send fast channel first
		select {
		case mf, ok = <-tf.assembleFastCh:
		default:
			mf, ok = <-tf.assembleCh
		}
		if ok == false {
			err = fmt.Errorf("writeLoop#%d, %s, exit for input channel closed", count, misc.GetXID(rw))
			fmt.Printf("%s\n", err.Error())
			tf.sendExitCMsg(0, err)
			return
		}
		iobytes = 0
		ioptr = 0
		emptyio = 0
		// writing
		for {
			iobytes, err = rw.Write(mf.iobuf[ioptr:mf.ioptr])
			if err != nil {
				tf.assembleFastCh <- mf
				err = fmt.Errorf("writeLoop#%d, %s, index %d, write failed: %s", count, misc.GetXID(rw), mf.index, err.Error())
				fmt.Printf("%s\n", err.Error())
				tf.sendExitCMsg(0, err)
				return
			}
			if iobytes < 1 {
				emptyio++
				if emptyio > CMTP_MAX_EMPTY_IO {
					tf.assembleFastCh <- mf
					err = fmt.Errorf("writeLoop#%d, %s, index %d, write failed: too many empty io(%d > %d)", count, misc.GetXID(rw), mf.index, emptyio, CMTP_MAX_EMPTY_IO)
					fmt.Printf("%s\n", err.Error())
					tf.sendExitCMsg(0, err)
					return
				}
				emptyio++
				// delay 10ms
				time.Sleep(1e7)
			}
			ioptr += iobytes
			if ioptr < mf.ioptr {
				// part write, retry
				fmt.Printf("writeLoop#%d, %s, part writing index %d, total %d - out %d = pending %d\n", count, misc.GetXID(rw), mf.index, mf.ioptr, ioptr, mf.ioptr-ioptr)
				continue
			}
			// write done
			break
		}
		fmt.Printf("writeLoop#%d, %s, done writing index %d, total %d - out %d = pending %d\n", count, misc.GetXID(rw), mf.index, mf.ioptr, ioptr, mf.ioptr-ioptr)
		tf.ioFreeCh <- mf
	}
}

// encodeLoop marshal and encode mixerFrame from encodeCh
// send encoded mixerFrame to assembleCh
//
func (tf *CodecMixer) encodeLoop(count int) {
	fmt.Printf("encodeLoop#%d, running ...\n", count)

	var mf *mixerFrame
	var ok bool
	var err error
	var maxcodeclen int
	var payloadbuf []byte
	hash := tf.checksum.New(0)
	for {
		mf, ok = <-tf.encodeCh
		if ok == false {
			err = fmt.Errorf("encodeLoop#%d, exit for input channel closed", count)
			fmt.Printf("%s\n", err.Error())
			tf.sendExitCMsg(0, err)
			return
		}
		// tf.sscodecList[mf.ssid] is created by initssid()
		codec := tf.sscodecList[mf.ssid]
		// encode body
		maxcodeclen = codec.MaxEncodedLen(int(mf.bodylen))
		// expend if need
		if maxcodeclen+mf.hdrlen > len(mf.framebuf) {
			mf.framebuf = make([]byte, maxcodeclen+mf.hdrlen)
		}
		// already make sure dst buffer have enough space
		payloadbuf, err = codec.Encode(mf.framebuf[mf.hdrlen:], mf.iobuf[:mf.ioptr])
		// can not recover from encoder failed
		if err != nil {
			panic(fmt.Sprintf("encodeLoop#%d, %s, index %d, encode failed: %s", count, misc.GetXID(codec), mf.index, err.Error()))
		}
		mf.payloadlen = uint64(len(payloadbuf))
		mf.frameptr = int(mf.payloadlen) + mf.hdrlen
		fmt.Printf("encodeLoop#%d, %s, done encode index %d, frame %d - payload %d = header %d\n", count, misc.GetXID(codec), mf.index, mf.frameptr, mf.payloadlen, mf.frameptr-int(mf.payloadlen))
		//
		mf.MarshalHeader(hash, count)
		//
		if mf.payloadlen < CMTP_SMALL_PKG_SIZE {
			tf.assembleFastCh <- mf
		} else {
			tf.assembleCh <- mf
		}

	}
}

// ReadFrom associate a frontend io.ReadWriteCloser with one ssid
// marshal frame read from io.ReadWriteCloser and write to underlay io.Writer
// return to caller until read EOF or error
func (tf *CodecMixer) ReadFrom(rw Filter) (written int, err error) {
	defer rw.Close()

	// 0 for new ssid
	ssid := tf.initssid(0)
	defer tf.clearssid(ssid)
	//
	emptyio := 0
	// start to read body/frame
	var er error
	var nr int
	for {
		mf, ok := <-tf.ioFreeCh
		if ok == false {
			return 0, errors.New("CodecMixer(ioFreeCh) already closed")
		}
		// expand buff for fast link
		if mf.iolen < CMTP_BUF_MAX && mf.iolen == len(mf.iobuf[mf.hdrlen:]) {
			mf.iobuf = make([]byte, mf.iolen+mf.iolen)
			println("expand mf.iobuf to", mf.iolen+mf.iolen)
		}
		nr, er = rw.Read(mf.iobuf[mf.hdrlen:])
		if nr <= 0 {
			if er == nil {
				if emptyio < CMTP_MAX_EMPTY_IO {
					emptyio++
					// delay 10ms
					time.Sleep(1e7)
					continue
				} else {
					er = fmt.Errorf("ReadFrom#%d, %s, index %d, write failed: too many empty io(%d > %d)", ssid, misc.GetXID(rw), mf.index, emptyio, CMTP_MAX_EMPTY_IO)
					fmt.Printf("%s\n", er.Error())
				}
			}
			break
		}
		written += nr
		mf.iolen = nr
		mf.ioptr = nr + mf.hdrlen
		//
		mf.headerInit(FRAME_PAYLOADENCODED, ssid, tf.growindex(ssid))
		//
		tf.encodeCh <- mf
		if er != nil {
			break
		}
	}
	return written, er
}

// readLoop fetch *mixerFrame from ioFreeCh and read from rw io.ReadWriteCloser
// no ordering contorl
// if read to rw io.ReadWriteCloser failed, put *mixerFrame back to ioFreeCh and exit
// put *mixerFrame back to encodeCh after read done
func (tf *CodecMixer) readLoop(rw io.ReadWriteCloser, count int) {
	defer rw.Close()
	fmt.Printf("readLoop#%d, %s, running ...\n", count, misc.GetXID(rw))

	var mf *mixerFrame
	var ok bool
	var err error
	var iobytes, emptyio, errorCount int
	hash := tf.checksum.New(0)
	for {
		mf, ok = <-tf.ioFreeCh
		if ok == false {
			err = fmt.Errorf("readLoop#%d, %s, exit for input channel closed", count, misc.GetXID(rw))
			fmt.Printf("%s\n", err.Error())
			tf.sendExitCMsg(0, err)
			return
		}
		mf.frametype = FRAME_MAGIC_UNSET
		iobytes = 0
		mf.frameptr = 0
		emptyio = 0
		// reading
		for {
			if mf.frametype == FRAME_MAGIC_UNSET {
				// read header
				iobytes, err = rw.Read(mf.framebuf[mf.frameptr:mf.hdrlen])
			} else {
				// read payload
				iobytes, err = rw.Read(mf.framebuf[mf.frameptr:mf.payloadlen])
			}
			if err != nil {
				tf.ioFreeCh <- mf
				err = fmt.Errorf("readLoop#%d, %s, index %d, read failed: %s", count, misc.GetXID(rw), mf.index, err.Error())
				fmt.Printf("%s\n", err.Error())
				tf.sendExitCMsg(1, err)
				return
			}
			if iobytes < 1 {
				emptyio++
				time.Sleep(1e7)
				if emptyio > CMTP_MAX_EMPTY_IO {
					tf.ioFreeCh <- mf
					err = fmt.Errorf("readLoop#%d, %s, index %d, read failed: too many empty io(%d > %d)", count, misc.GetXID(rw), mf.index, emptyio, CMTP_MAX_EMPTY_IO)
					fmt.Printf("%s\n", err.Error())
					tf.sendExitCMsg(1, err)
					return
				}
			}
			mf.frameptr += iobytes
			//
			if mf.frametype == FRAME_MAGIC_UNSET {
				// need for header
				unerr := mf.UnMarshalHeader(hash, count)
				if unerr != nil {
					if unerr.(CMsg).Code == uint64(UNMARSHAL_ERR_HDRLEN) {
						// read more
						continue
					} else {
						fmt.Printf("readLoop#%d, %s, error UnMarshalHeader index %d, total %d: %v\n", count, misc.GetXID(rw), mf.index, mf.frameptr, mf.framebuf[:mf.hdrlen])
						mf.frametype = FRAME_MAGIC_UNSET
						errorCount++
						if errorCount > CMTP_MAX_UNMARSHAL_ERR {
							tf.ioFreeCh <- mf
							err = fmt.Errorf("readLoop#%d, %s, index %d, read failed: too many unmarshal header failed(%d > %d)", count, misc.GetXID(rw), mf.index, errorCount, CMTP_MAX_UNMARSHAL_ERR)
							fmt.Printf("%s\n", err.Error())
							tf.sendExitCMsg(1, err)
							return
						}
						break
					}
				}
				// header is ok
				tf.initssid(mf.ssid)
				continue
			}
			// reading payload
			if uint64(mf.frameptr) >= mf.payloadlen {
				// read done
				break
			}
			// read more payload
		}
		if mf.frametype == FRAME_MAGIC_UNSET {
			fmt.Printf("readLoop#%d, %s, error reading index %d, total %d\n", count, misc.GetXID(rw), mf.index, mf.frameptr)
			tf.ioFreeCh <- mf
		} else {
			fmt.Printf("readLoop#%d, %s, done reading index %d, total %d\n", count, misc.GetXID(rw), mf.index, mf.frameptr)
			tf.encodeCh <- mf
		}
	}
}

// decodeLoop marshal and decode mixerFrame from encodeCh
// send decoded mixerFrame to ioReadToCh
func (tf *CodecMixer) decodeLoop(count int) {
	fmt.Printf("decodeLoop#%d, running ...\n", count)

	var mf *mixerFrame
	var ok bool
	var err error
	var maxcodeclen int
	var payloadbuf []byte
	for {
		mf, ok = <-tf.decodeCh
		if ok == false {
			err = fmt.Errorf("decodeLoop#%d, exit for input channel closed", count)
			fmt.Printf("%s\n", err.Error())
			tf.sendExitCMsg(0, err)
			return
		}
		// tf.sscodecList[mf.ssid] is created by initssid()
		codec := tf.sscodecList[mf.ssid]
		// decode body
		maxcodeclen, err = codec.MaxDecodedLen(mf.framebuf[mf.hdrlen:mf.frameptr])
		if err != nil {
			fmt.Printf("decodeLoop#%d, %s, ssid %d index %d, decode failed: %s", count, misc.GetXID(codec), mf.ssid, mf.index, err.Error())
			tf.ioFreeCh <- mf
			continue
		}
		// expend if need
		if maxcodeclen > len(mf.iobuf) {
			mf.iobuf = make([]byte, maxcodeclen)
		}
		// already make sure dst buffer have enough space
		payloadbuf, err = codec.Decode(mf.iobuf, mf.framebuf[mf.hdrlen:mf.frameptr])
		// can not recover from decoder failed
		if err != nil {
			fmt.Printf("decodeLoop#%d, %s, ssid %d index %d, decode failed: %s", count, misc.GetXID(codec), mf.ssid, mf.index, err.Error())
			tf.ioFreeCh <- mf
			continue
		}
		mf.ioptr = len(payloadbuf)
		if uint64(mf.ioptr) != mf.bodylen {
			fmt.Printf("decodeLoop#%d, %d, decoded body lenght mismatch, index %d, header %d, decoded %d\n", count, mf.ssid, mf.index, mf.bodylen, mf.ioptr)
			tf.ioFreeCh <- mf
			continue
		}
		fmt.Printf("decodeLoop#%d, %s, done decode index %d, frame %d - payload %d = header %d\n", count, misc.GetXID(codec), mf.index, mf.frameptr, mf.payloadlen, mf.frameptr-int(mf.payloadlen))
		//
		// tf.ssWriteCh[mf.ssid] is created by initssid()
		tf.ssWriteCh[mf.ssid] <- mf
	}
}

// WriteTo associate a frontend io.ReadWriteCloser with one ssid
// return to caller until Write error
func (tf *CodecMixer) WriteTo(rw Filter, ssid uint64) (written int, err error) {
	defer rw.Close()

	//
	defer tf.clearssid(ssid)
	//
	emptyio := 0
	// start to read body/frame
	// tf.ssWriteCh[ssid] is created by initssid()
	framesource := tf.ssWriteCh[ssid]
	var nr int
	for {
		//
		mf, ok := <-framesource
		if ok == false {
			err = fmt.Errorf("WriteTo ssid %d, exit for input channel closed", ssid)
			fmt.Printf("%s\n", err.Error())
			tf.sendExitCMsg(0, err)
			return
		}
		//
		// expand buff for fast link
		if mf.iolen < CMTP_BUF_MAX && mf.iolen == len(mf.iobuf[mf.hdrlen:]) {
			mf.iobuf = make([]byte, mf.iolen+mf.iolen)
			println("expand mf.iobuf to", mf.iolen+mf.iolen)
		}
		nr, err = rw.Read(mf.iobuf[mf.hdrlen:])
		if nr <= 0 {
			if err == nil {
				if emptyio < CMTP_MAX_EMPTY_IO {
					emptyio++
					// delay 10ms
					time.Sleep(1e7)
					continue
				} else {
					err = fmt.Errorf("ReadFrom#%d, %s, index %d, write failed: too many empty io(%d > %d)", ssid, misc.GetXID(rw), mf.index, emptyio, CMTP_MAX_EMPTY_IO)
					fmt.Printf("%s\n", err.Error())
				}
			}
			break
		}
		written += nr
		mf.iolen = nr
		mf.ioptr = nr + mf.hdrlen
		//
		mf.headerInit(FRAME_PAYLOADENCODED, ssid, tf.growindex(ssid))
		//
		tf.encodeCh <- mf
		if err != nil {
			break
		}
	}
	return written, err
}

// Close discard all internal resource
// will close underlay io.ReadWriteCloser
func (tf *CodecMixer) Close() {
	tf.state = MIXER_STATE_CLOSED
}

//
//
//
//
//
//
//
//
//
//
//

//

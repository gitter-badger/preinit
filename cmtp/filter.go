//
// TCPFilter for Common Multiplexing Transport Proxy (CMTP)
//
//

//
package cmtp

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/wheelcomplex/preinit/keyaes"
	"github.com/wheelcomplex/preinit/misc"
)

// state machine for Filter
type FILTER_STATE int

const (
	FILTER_STATE_UNSET FILTER_STATE = iota
	FILTER_STATE_RESET
	FILTER_STATE_SENDHEADER
	FILTER_STATE_SENDWAITIO
	FILTER_STATE_SENDBODY
	FILTER_STATE_FILLHEADER
	FILTER_STATE_FILLINFO
	FILTER_STATE_FMTINFO
	FILTER_STATE_FILLTAIL
	FILTER_STATE_FILLBODY
	FILTER_STATE_CLOSED
	FILTER_STATE_LAST
)

// tcpStream
type tcpStream struct {
	state   FILTER_STATE       // filter state
	iobuf   []byte             // buffer for io
	ioptr   int                // current opration position of buffer
	rw      io.ReadWriteCloser //
	dstinfo string             //
	ioready chan struct{}      //
}

//
func newTcpStream() *tcpStream {
	ts := &tcpStream{
		state:   FILTER_STATE_RESET,
		ioready: make(chan struct{}, 1024),
	}
	return ts
}

//
func (ts *tcpStream) close() {
	if ts.rw != nil {
		ts.rw.Close()
		ts.rw = nil
	}
	ts.iobuf = nil
	ts.ioptr = 0
	ts.state = FILTER_STATE_UNSET
}

// max tcpStream header size
const MAX_HEADERLEN int = 1024

// timeout for DialTCP
const MAX_DIALTIME time.Duration = 15e9

// TCPFilter implemented Filter
// at assemble side, send initial header(include dstinfo) and forward raw stream from underlay io.Reader
// at disassemble side, read initial header(include dstinfo), create underlay io.Writer(by dstinfo)
// and forward raw stream to underlay io.Writer
// initial header will encrypt by aes
type TCPFilter struct {
	ssid       uint64        //
	aes        *keyaes.AES   // aes use for dst info crypt
	dstinfo    string        //
	hdrlen     int           // header len for marshal header
	encryptlen uint32        // stream initial info length after encrypt(without prefix header)
	closed     chan struct{} //
	in         *tcpStream    // assemble side
	out        *tcpStream    // disassemble side
}

// NewTCPFilter return *TCPFilter
// length of dstinfo must less then MAX_HEADERLEN
func NewTCPFilter(key []byte, dstinfo string) *TCPFilter {
	// cut
	if len(dstinfo) > MAX_HEADERLEN {
		dstinfo = dstinfo[:MAX_HEADERLEN]
	}
	tf := &TCPFilter{
		aes:     keyaes.NewAES(key, nil),
		in:      newTcpStream(),
		out:     newTcpStream(),
		dstinfo: dstinfo,
		closed:  make(chan struct{}, 1),
	}
	tf.hdrlen = binary.Size(tf.encryptlen)
	return tf
}

// New return new *TCPFilter work with ssid/rw
// if use for disassemble side, pass rw == nil
func (tf *TCPFilter) New(ssid uint64, rw io.ReadWriteCloser) Filter {
	ntf := &TCPFilter{
		ssid:    ssid,
		in:      newTcpStream(),
		out:     newTcpStream(),
		closed:  make(chan struct{}, 1),
		aes:     tf.aes,
		hdrlen:  tf.hdrlen,
		dstinfo: tf.dstinfo,
	}
	if rw != nil {
		// assemble side
		tf.in.rw = rw
		ntf.marshalHeader()
	} else {
		// disassemble side
	}
	return ntf
}

// marshalHeader
func (tf *TCPFilter) marshalHeader() {
	//
	// marshal format: uint32(tf.encryptlen)+[]byte(tf.dstinfo)
	//
	dstbuf := []byte(tf.dstinfo)
	tf.encryptlen = uint32(tf.aes.EncryptSize(len(dstbuf)))
	tf.in.iobuf = make([]byte, tf.hdrlen+int(tf.encryptlen))
	// Encrypt info, already make sure have enough buffer space for Encrypt
	tf.aes.Encrypt(tf.in.iobuf[tf.hdrlen:], dstbuf)
	// should fill encryptlen after encrypt
	binary.Write(misc.NewBRWC(tf.in.iobuf[:tf.hdrlen]), CMTP_ENDIAN, &tf.encryptlen)
	tf.in.state = FILTER_STATE_SENDHEADER
	fmt.Printf("TCPFilter, marshalInHeader: %d => %d, %x => %x\n", len(dstbuf), len(tf.in.iobuf), dstbuf, tf.in.iobuf)
}

// Close discard all internal resource
// will close underlay io.Reader
func (tf *TCPFilter) Close() {
	select {
	case <-tf.closed:
		return
	default:
		close(tf.closed)
	}
	tf.aes.Close()
	tf.in.close()
	tf.out.close()
}

// Read fill p []byte with marshalled header + stream from underlay io.Reader
// have to trace the io state machine from marshalled header to underlay io.Reader and EOF
func (tf *TCPFilter) Read(p []byte) (n int, err error) {
	for {
		switch tf.in.state {
		case FILTER_STATE_RESET:
			if tf.in.rw == nil {
				// disassemble side reading
				tf.in.state = FILTER_STATE_SENDWAITIO
				continue
			} else {
				// assemble side reading
				tf.in.state = FILTER_STATE_SENDHEADER
			}
			fallthrough
		case FILTER_STATE_SENDHEADER:
			// copy marshelled header out
			outbyte, _ := misc.NewBRWC(p).Write(tf.in.iobuf[tf.in.ioptr:])
			tf.in.ioptr += outbyte
			if tf.in.ioptr == len(tf.in.iobuf) {
				// all header is out, next read will send underlay io.Reader
				if tf.in.rw == nil {
					tf.in.state = FILTER_STATE_SENDWAITIO
				} else {
					tf.in.state = FILTER_STATE_SENDBODY
				}
			}
			return outbyte, nil
		case FILTER_STATE_SENDWAITIO:
			//
			// waiting for tf.in.rw ready
			//
			closed := false
			select {
			case <-tf.closed:
				closed = true
			default:
				select {
				case <-tf.in.ioready:
				case <-tf.closed:
					closed = true

				}
			}
			if closed == true {
				return 0, &CMsg{
					Code: IO_ERR_CLOSED,
					Err:  fmt.Errorf("TCPFilter, read failed: closed"),
				}
			}
			if tf.in.rw == nil {
				return 0, &CMsg{
					Code: IO_ERR_READ,
					Err:  fmt.Errorf("TCPFilter, read failed: io.Reader no ready when try to read"),
				}
			}
			tf.in.state = FILTER_STATE_SENDBODY
			fallthrough
		case FILTER_STATE_SENDBODY:
			// copy underlay io.Reader until EOF
			return tf.in.rw.Read(p)
		default:
			panic(fmt.Sprintf("invalid TCPFilter state %d in Read", tf.in.state))
		}
	}
	return
}

// Write unmarshal p []byte (marshalled header + stream) and write to underlay io.Writer
// have to trace the io state machine from marshalled header to underlay io.Writer and EOF
func (tf *TCPFilter) Write(p []byte) (n int, err error) {
	var pren int
	for {
		switch tf.out.state {
		case FILTER_STATE_RESET:
			tf.out.state = FILTER_STATE_FILLHEADER
			tf.out.iobuf = make([]byte, 0, tf.hdrlen)
			tf.out.ioptr = 0
			fallthrough
		case FILTER_STATE_FILLHEADER:
			// try to read encryptlen
			n = len(p)
			tf.out.iobuf = append(tf.out.iobuf, p...)
			if len(tf.out.iobuf) < tf.hdrlen {
				// need more for header
				return n, nil
			}
			// unmarshal encryptlen header
			binary.Read(misc.NewBRWC(tf.out.iobuf[:tf.hdrlen]), CMTP_ENDIAN, &tf.encryptlen)
			println("TCPFilter unmarshalled encryptlen", tf.encryptlen)
			if int(tf.encryptlen) > MAX_HEADERLEN*2 {
				return n, &CMsg{
					Code: UNMARSHAL_ERR_HDRLEN,
					Err:  fmt.Errorf("TCPFilter, unmarshal header failed: info length too large(max %d, got %d)", MAX_HEADERLEN*2, tf.encryptlen),
				}
			}
			tf.out.ioptr = int(tf.encryptlen) + tf.hdrlen
			if len(tf.out.iobuf) < tf.out.ioptr {
				// read more
				tf.out.state = FILTER_STATE_FILLINFO
				return n, nil
			}
			pren = n
			tf.out.state = FILTER_STATE_FMTINFO
			continue
		case FILTER_STATE_FILLINFO:
			// read more initial info
			n = len(p)
			tf.out.iobuf = append(tf.out.iobuf, p...)
			if len(tf.out.iobuf) < tf.out.ioptr {
				// read more
				return n, nil
			}
			// read done
			tf.out.state = FILTER_STATE_FMTINFO
			pren = n
			continue
		case FILTER_STATE_FMTINFO:
			// func (ae *AES) Decrypt(decryptText []byte, src []byte) ([]byte, error)
			dstbuf, err := tf.aes.Decrypt(nil, tf.out.iobuf[tf.hdrlen:tf.out.ioptr])
			if err != nil {
				return pren, &CMsg{
					Code: UNMARSHAL_ERR_OPTION,
					Err:  fmt.Errorf("TCPFilter, decrypt info failed: %s", err.Error()),
				}
			}
			fmt.Printf("TCPFilter, unmarshalInHeader: %d <= %d, %x <= %x\n", len(dstbuf), len(tf.out.iobuf), dstbuf, tf.out.iobuf)
			tf.out.dstinfo = string(dstbuf)
			//
			rw, derr := net.DialTimeout("tcp", tf.out.dstinfo, MAX_DIALTIME)
			if derr != nil {
				return pren, &CMsg{
					Code: IO_ERR_DIAL,
					Err:  fmt.Errorf("TCPFilter, dial dst failed(%s): %s", tf.out.dstinfo, derr.Error()),
				}
			}
			//
			tf.out.rw = rw.(*net.TCPConn)
			//
			// prepare read side
			//
			tf.in.rw = rw.(*net.TCPConn)
			tf.in.state = FILTER_STATE_SENDWAITIO
			// active read goroutine
			tf.in.ioready <- struct{}{}
			//
			if len(tf.out.iobuf[tf.out.ioptr:]) == 0 {
				tf.out.state = FILTER_STATE_FILLBODY
				return pren, nil
			}
			tf.out.state = FILTER_STATE_FILLTAIL
			fallthrough
		case FILTER_STATE_FILLTAIL:
			// write tail body to io.Writer
			nw, ew := tf.out.rw.Write(tf.out.iobuf[tf.out.ioptr:])
			tf.out.ioptr += nw
			if len(tf.out.iobuf[tf.out.ioptr:]) == 0 {
				tf.out.state = FILTER_STATE_FILLBODY
				return pren, ew
			}
			if ew != nil {
				if nw == 0 {
					// unrecoveable error
					return pren, ew
				}
				// shortWrite ?
			}
			// pending bytes to write
			continue
		case FILTER_STATE_FILLBODY:
			// write pain body to io.Writer
			return tf.out.rw.Write(p)
		default:
			panic(fmt.Sprintf("invalid TCPFilter state %d in Write", tf.out.state))
		}
	}
	return
}

//

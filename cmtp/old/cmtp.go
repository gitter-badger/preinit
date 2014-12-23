


////////

// ReadFrom Decode stream from rw io.Reader and write to underdelay io.Writer
// ReadFrom will not return befor err occured
func (tf *TCPRXFilter) ReadFrom(rw io.Reader) (written int64, err error) {
	if len(tf.iobuf) < CMTP_BUF_INIT {
		tf.iobuf = make([]byte, CMTP_BUF_INIT)
	}
	var maxBuf bool
	tf.ioptr = len(tf.iobuf)
	// code copy from io.Copy()
	for {
		nr, er := rw.Read(tf.iobuf)
		if nr > 0 {
			nw, ew := tf.Write(tf.iobuf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF && nr > 0 {
			break
		}
		if er != nil {
			err = er
			break
		}
		// more buffer for high speed link
		if maxBuf == false && nr == tf.ioptr {
			tf.iobuf = make([]byte, tf.ioptr*2)
			tf.ioptr = len(tf.iobuf)
			if tf.ioptr > CMTP_BUF_INIT*128 {
				maxBuf = true
				println("TCPRXFilter: ReadFrom reach max write buffer size:", tf.ioptr)
			} else {
				println("TCPRXFilter: ReadFrom grow write buffer size:", tf.ioptr)
			}
		}
	}
	return written, err
}

// Read fill p []byte with marshalled header + stream from underdelay io.Reader
// have to trace the io state machine from marshalled header to underdelay io.Reader and EOF
func (tf *CodecMixer) Read(p []byte) (n int, err error) {
	switch tf.state {
	case FILTER_STATE_RESET:
		tf.state = FILTER_STATE_SENDHEADER
		//
		tf.ssid = uint64(<-uuidCh)
		// marshal header
		err = binary.Write(NewBRWC(tf.hdrbuf[0:binary.Size(tf.ssid)]), CMTP_ENDIAN, &tf.ssid)
		if err != nil {
			tf.Close()
			return 0, &CommError{
				Code: MARSHAL_ERR_HDR,
				Err:  fmt.Errorf("CodecMixer Marshal ssid failed: %s", err.Error()),
			}
		}
		println("CodecMixer marshalled ssid", tf.ssid)
		err = binary.Write(NewBRWC(tf.hdrbuf[binary.Size(tf.ssid):tf.hdrlen]), CMTP_ENDIAN, &tf.token)
		if err != nil {
			tf.Close()
			return 0, &CommError{
				Code: MARSHAL_ERR_HDR,
				Err:  fmt.Errorf("CodecMixer Marshal token failed: %s", err.Error()),
			}
		}
		println("CodecMixer marshalled token", tf.token)
		fallthrough
	case FILTER_STATE_SENDHEADER:
		// copy marshelled header out
		cplen := len(p)
		blen := tf.hdrlen - tf.hdrptr
		if cplen > blen {
			cplen = blen
		}
		copy(p, tf.hdrbuf[:cplen])
		tf.hdrptr += cplen
		if tf.hdrptr == tf.hdrlen {
			// all header is out, next read will send underdelay io.Reader
			tf.state = FILTER_STATE_SENDBODY
		}
		return cplen, nil
	case FILTER_STATE_SENDBODY:
		// copy underdelay io.Reader until EOF
		return tf.rw.Read(p)
	default:
		panic(fmt.Sprintf("invalid CodecMixer state %d in Read", tf.state))
	}
	return
}

// WriteTo Encode selft to internal buffer and write to w io.Writer
// WriteTo will not return befor err occured
func (tf *CodecMixer) WriteTo(w io.Writer) (written int64, err error) {
	if len(tf.iobuf) < CMTP_BUF_INIT {
		tf.iobuf = make([]byte, CMTP_BUF_INIT)
	}
	var maxBuf bool
	tf.ioptr = len(tf.iobuf)
	// code copy from io.Copy()
	for {
		// Read from tf
		nr, er := tf.Read(tf.iobuf)
		if nr > 0 {
			nw, ew := w.Write(tf.iobuf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF && nr > 0 {
			break
		}
		if er != nil {
			err = er
			break
		}
		// more buffer for high speed link
		if maxBuf == false && nr == tf.ioptr {
			tf.iobuf = make([]byte, tf.ioptr*2)
			tf.ioptr = len(tf.iobuf)
			if tf.ioptr > CMTP_BUF_INIT*128 {
				maxBuf = true
				println("CodecMixer: WriteTo reach max write buffer size:", tf.ioptr)
			} else {
				println("CodecMixer: WriteTo grow write buffer size:", tf.ioptr)
			}
		}
	}
	return written, err
}

// AddWriteCloser for //  assemble // disassemble
func (tf *CodecMixer) AddWriter(w io.WriteCloser) (writebytes int64, err error) {
	return
}

// AddReadCloser for //  assemble // disassemble
func (tf *CodecMixer) AddReadFrom(w io.ReadCloser) (readbytes int64, err error) {
	return
}

// Close discard all internal resource
// will close underdelay io.Reader
func (tf *CodecMixer) Close() {
	if tf.rw != nil {
		tf.rw.Close()
	}
	tf.rw = nil
	tf.iobuf = nil
	tf.hdrbuf = nil
	tf.state = FILTER_STATE_CLOSED
}

//

// NewCodecDecoder return decoder for RXMixer
func NewCodecDecoder(rw io.ReadWriteCloser, token uint64) *CodecDecoder {
	return NewCodecEncoder(rw, 0, token)
}

// Write unmarshal p []byte (marshalled header + stream) and write to underdelay io.Writer
// have to trace the io state machine from marshalled header to underdelay io.Writer and EOF
func (tf *CodecMixer) Write(p []byte) (n int, err error) {
	switch tf.state {
	case FILTER_STATE_RESET:
		tf.state = FILTER_STATE_FILLHEADER
		fallthrough
	case FILTER_STATE_FILLHEADER:
		n = len(p)
		lessn := len(tf.hdrbuf) - tf.hdrlen
		tf.hdrbuf = append(tf.hdrbuf, p...)
		if len(tf.hdrbuf) < tf.hdrlen {
			// need more for header
			return n, nil
		}
		// unmarshal header
		err := binary.Read(NewBRWC(tf.hdrbuf[:tf.hdrlen]), CMTP_ENDIAN, &tf.ssid)
		if err != nil {
			tf.Close()
			return 0, &CommError{
				Code: MARSHAL_ERR_HDR,
				Err:  fmt.Errorf("CodecMixer UnMarshal ssid failed: %s", err.Error()),
			}
		}
		println("CodecMixer unmarshalled ssid", tf.ssid)
		var token uint64
		err = binary.Read(NewBRWC(tf.hdrbuf[binary.Size(tf.ssid):tf.hdrlen]), CMTP_ENDIAN, &token)
		if err != nil {
			tf.Close()
			return 0, &CommError{
				Code: MARSHAL_ERR_HDR,
				Err:  fmt.Errorf("CodecMixer UnMarshal token failed: %s", err.Error()),
			}
		}
		if token != tf.token {
			err = fmt.Errorf("CodecMixer UnMarshal token failed: token mismatch, local %d != remote %d", tf.token, token)
			println(err.Error())
			tf.Close()
			return 0, &CommError{
				Code: MARSHAL_ERR_HDR,
				Err:  err,
			}
		}
		println("CodecMixer unmarshalled token", tf.token)
		tf.state = FILTER_STATE_FILLBODY
		tf.hdrptr = tf.hdrlen
		if len(tf.hdrbuf) > tf.hdrlen {
			fmt.Printf("CodecMixer, Write head body: %v\n", tf.hdrbuf[tf.hdrlen:])
			headn, headerr := tf.Write(tf.hdrbuf[tf.hdrlen:])
			// reset buffer
			tf.hdrbuf = make([]byte, tf.hdrlen)
			if headerr != nil {
				return lessn + headn, headerr
			}
		}
		return n, nil
	case FILTER_STATE_FILLBODY:
		// write pain body to io.Writer
		return tf.rw.Write(p)
	default:
		panic(fmt.Sprintf("invalid CodecMixer state %d in Write", tf.state))
	}
	return
}

// ReadFrom Decode stream from rw io.Reader and write to underdelay io.Writer
// ReadFrom will not return befor err occured
func (tf *CodecMixer) ReadFrom(rw io.Reader) (written int64, err error) {
	if len(tf.iobuf) < CMTP_BUF_INIT {
		tf.iobuf = make([]byte, CMTP_BUF_INIT)
	}
	var maxBuf bool
	tf.ioptr = len(tf.iobuf)
	// code copy from io.Copy()
	for {
		nr, er := rw.Read(tf.iobuf)
		if nr > 0 {
			nw, ew := tf.Write(tf.iobuf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF && nr > 0 {
			break
		}
		if er != nil {
			err = er
			break
		}
		// more buffer for high speed link
		if maxBuf == false && nr == tf.ioptr {
			tf.iobuf = make([]byte, tf.ioptr*2)
			tf.ioptr = len(tf.iobuf)
			if tf.ioptr > CMTP_BUF_INIT*128 {
				maxBuf = true
				println("CodecMixer: ReadFrom reach max write buffer size:", tf.ioptr)
			} else {
				println("CodecMixer: ReadFrom grow write buffer size:", tf.ioptr)
			}
		}
	}
	return written, err
}

//
//
////
//
//
//
//
////
//
//
//
//
////
//
//
//
//
////
//
//
//
//
//

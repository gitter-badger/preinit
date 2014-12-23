//
// Cmsg for Common Multiplexing Transport Proxy (CMTP)
//

package cmtp

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/wheelcomplex/preinit/misc"
)

// CMsg
type CMsg struct {
	Code uint64
	Id   uint64
	Msg  []byte
	Err  error
}

// String return formated string of CMsg
func (mc *CMsg) String() string {
	return fmt.Sprintf("code %d, id %d, msg: %s, error: %s", mc.Code, mc.Id, mc.Msg, mc.Err.Error())
}

// Error return formated string of CMsg
func (mc CMsg) Error() string {
	return mc.String()
}

// MarshalSize return size of CMsg Marshal buffer
func (mc *CMsg) MarshalSize() int {
	// format: code+msglen+msg
	blen := binary.Size(mc.Code)
	hdrlen := binary.Size(uint32(0)) + blen*2
	elen := 0
	if len(mc.Msg) == 0 && mc.Err != nil {
		mc.Msg = []byte(mc.Err.Error())
	}
	elen = len(mc.Msg)
	if elen > 1024 {
		elen = 1024
	}
	return hdrlen + elen
}

// Marshal return byte stream of CMsg
func (mc *CMsg) Marshal() ([]byte, error) {
	// format: code+msglen+msg
	blen := binary.Size(mc.Code)
	hdrlen := binary.Size(uint32(0)) + blen*2
	if len(mc.Msg) == 0 && mc.Err != nil {
		mc.Msg = []byte(mc.Err.Error())
	}
	elen := uint32(len(mc.Msg))
	if elen > 1024 {
		elen = 1024
	}
	buf := make([]byte, hdrlen+int(elen))
	if err := binary.Write(misc.NewBRWC(buf), CMTP_ENDIAN, &mc.Code); err != nil {
		return nil, fmt.Errorf("CMsg Marshal code failed: %s", err.Error())
	}
	if err := binary.Write(misc.NewBRWC(buf[blen:]), CMTP_ENDIAN, &mc.Id); err != nil {
		return nil, fmt.Errorf("CMsg Marshal id failed: %s", err.Error())
	}
	if err := binary.Write(misc.NewBRWC(buf[blen+blen:]), CMTP_ENDIAN, &elen); err != nil {
		return nil, fmt.Errorf("CMsg Marshal code failed: %s", err.Error())
	}
	cpbuf := buf[hdrlen:]
	// do not use biuldin copy, because mc.Msg length maybe too long
	for i := uint32(0); i < elen; i++ {
		cpbuf[i] = mc.Msg[i]
	}
	return buf, nil
}

// Read fill p with Marshalled stream
func (mc *CMsg) Read(p []byte) (int, error) {
	mp, err := mc.Marshal()
	if err != nil {
		return 0, err
	}
	return misc.NewBRWC(mp).Read(p)
}

// UnMarshalSize return size of CMsg UnMarshal buffer
func (mc *CMsg) UnMarshalSize(buf []byte) (int, error) {
	// format: code+msglen+msg
	blen := binary.Size(mc.Code)
	hdrlen := binary.Size(uint32(0)) + blen*2
	if len(buf) < hdrlen {
		return 0, fmt.Errorf("CMsg UnMarshal failed: %s", "input bytes too few")
	}
	//if err := binary.Read(misc.NewBRWC(buf), CMTP_ENDIAN, &mc.Code); err != nil {
	//	return 0, fmt.Errorf("CMsg UnMarshal code failed: %s", err.Error())
	//}
	//if err := binary.Read(misc.NewBRWC(buf[blen:]), CMTP_ENDIAN, &mc.Id); err != nil {
	//	return 0, fmt.Errorf("CMsg UnMarshal id failed: %s", err.Error())
	//}
	elen := uint32(0)
	if err := binary.Read(misc.NewBRWC(buf[blen+blen:]), CMTP_ENDIAN, &elen); err != nil {
		return 0, fmt.Errorf("CMsg UnMarshal msg length failed: %s", err.Error())
	}
	return hdrlen + int(elen), nil
}

// UnMarshal return CMsg from byte stream
func (mc *CMsg) UnMarshal(buf []byte) (int, error) {
	// format: code+msglen+msg
	blen := binary.Size(mc.Code)
	hdrlen := binary.Size(uint32(0)) + blen*2
	if len(buf) < hdrlen {
		return 0, fmt.Errorf("CMsg UnMarshal failed: %s", "input bytes too few")
	}
	if err := binary.Read(misc.NewBRWC(buf), CMTP_ENDIAN, &mc.Code); err != nil {
		return 0, fmt.Errorf("CMsg UnMarshal code failed: %s", err.Error())
	}
	if err := binary.Read(misc.NewBRWC(buf[blen:]), CMTP_ENDIAN, &mc.Id); err != nil {
		return 0, fmt.Errorf("CMsg UnMarshal id failed: %s", err.Error())
	}
	elen := uint32(0)
	if err := binary.Read(misc.NewBRWC(buf[blen+blen:]), CMTP_ENDIAN, &elen); err != nil {
		return 0, fmt.Errorf("CMsg UnMarshal msg length failed: %s", err.Error())
	}
	plen := uint32(len(buf[hdrlen:]))
	// ignored short read error
	if elen > plen {
		elen = plen
	} else if elen > 1024 {
		elen = 1024
	}
	outlen := hdrlen + int(elen)
	if elen > 0 {
		if len(mc.Msg) < int(elen) {
			// expend if need
			mc.Msg = make([]byte, elen)
		} else {
			mc.Msg = mc.Msg[:elen]
		}
		copy(mc.Msg, buf[hdrlen:outlen])
		mc.Err = fmt.Errorf("%s", mc.Msg)
	} else {
		mc.Err = nil
		mc.Msg = mc.Msg[:0]
	}
	return outlen, nil
}

// Write UnMarshalled p
func (mc *CMsg) Write(p []byte) (int, error) {
	return mc.UnMarshal(p)
}

// Equal return true if mc.Code == a.Code && mc.Msg == a.Msg
func (mc *CMsg) Equal(a *CMsg) bool {
	if a == nil || mc == nil {
		return false
	}
	return mc.Code == a.Code && bytes.Equal(mc.Msg, a.Msg) && mc.Err.Error() == a.Err.Error()
}

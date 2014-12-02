/*
	package byterwcloser implementations a very simple reader/writer/closer interface for []byte
*/

package byterwcloser

//
type ByteRWCloser []byte

func NewByteRWCloser(size int) ByteRWCloser {
	return ByteRWCloser(make([]byte, size))
}

// Read Implementations Reader interface
func (brw ByteRWCloser) Read(p []byte) (n int, err error) {
	l1 := len(brw)
	l2 := len(p)
	if l1 > l2 {
		n = l2
	} else {
		n = l1
	}
	copy(p, brw)
	return
}

// Write Implementations Writer interface
func (brw ByteRWCloser) Write(p []byte) (n int, err error) {
	l1 := len(brw)
	l2 := len(p)
	if l1 > l2 {
		n = l2
	} else {
		n = l1
	}
	copy(brw, p)
	return
}

// Close Implementations Closer interface
func (brw ByteRWCloser) Close() (err error) {
	return
}

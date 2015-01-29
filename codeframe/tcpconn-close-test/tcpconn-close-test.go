/*
	TCP Connection test
*/

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

func TimePrintf(format string, v ...interface{}) {
	ts := fmt.Sprintf("[%s] ", time.Now().String())
	msg := fmt.Sprintf(format, v...)
	fmt.Printf("%s%s", ts, msg)
}

func main() {
	rbufsize := flag.Int("R", 1, "read buffer size")
	rtime := flag.Int("T", 65535, "read ttimeout")
	flag.Parse()
	laddr, lerr := net.ResolveTCPAddr("tcp", "127.0.0.1:8800")
	if lerr != nil {
		println(lerr.Error())
		os.Exit(9)
	}
	l, lerr := net.ListenTCP("tcp", laddr)
	if lerr != nil {
		println(lerr.Error())
		os.Exit(9)
	}
	synch := make(chan struct{}, 1)
	go func() {
		synch <- struct{}{}
		conn, _ := l.AcceptTCP() // *net.TCPConn
		fmt.Printf("new conn %s\n", conn.RemoteAddr().String())
		buf := []byte("01234567890123456789")
		for i := 0; i < 10; i++ {
			_, err := conn.Write(buf[i : i+1])
			if err != nil {
				fmt.Printf("out#%d: %s\n", i, err.Error())
				break
			} else {
				fmt.Printf("out#%d: %s\n", i, buf[i:i+1])
			}
		}
		conn.Close()
		println("server closed")
	}()
	defer l.Close()
	// wait for listener
	<-synch
	hostport := "127.0.0.1:8800"
	cl, ce := net.Dial("tcp4", hostport)
	if ce != nil {
		TimePrintf("ERROR: %s\n", ce.Error())
		os.Exit(1)
	}
	TimePrintf("conected: %s\n", hostport)

	rtimeout := 1e8 * time.Duration(*rtime)
	println("reading with buffer size:", *rbufsize, "timeout", *rtime)
	buflen := *rbufsize
	rbuf := make([]byte, buflen)
	tbuf := make([]byte, 0, 100*buflen)
	// read until read timeout
	// wait one seconds for server closed
	time.Sleep(1e9)
	for {
		cl.SetReadDeadline(time.Now().Add(rtimeout))
		nr, re := cl.Read(rbuf)
		if nr > 0 {
			tbuf = append(tbuf, rbuf[:nr]...)
		}
		if re != nil {
			TimePrintf("ERROR: read %s failed: %s\n", hostport, re.Error())
			break
		}
	}
	if len(tbuf) > 0 {
		TimePrintf("Got final response %s (%d)\n ------ \n%s\n", hostport, len(tbuf), tbuf)
		TimePrintf(" ------\n")
	}
}

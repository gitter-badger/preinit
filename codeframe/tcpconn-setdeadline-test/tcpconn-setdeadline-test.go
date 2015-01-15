package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
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
	go func() {
		conn, _ := l.AcceptTCP() // *net.TCPConn
		defer conn.Close()
		fmt.Printf("new conn %s\n", conn.RemoteAddr().String())
		buf := []byte("0123456789")
		tk := time.NewTicker(2e9)
		for i := 0; i < 10; i++ {
			conn.Write(buf[i : i+1])
			fmt.Printf("out#%d: %s\n", i, buf[i:i+1])
			<-tk.C
		}
	}()
	defer l.Close()
	time.Sleep(2e9)
	nfd, err := net.Dial("tcp", "127.0.0.1:8800")
	if err != nil {
		println(err.Error())
		os.Exit(9)
	}
	defer nfd.Close()
	limit := time.Now().Add(5e9)
	nfd.SetDeadline(limit)
	rbuf := make([]byte, 10)
	for i := 0; i < 5; i++ {
		nr, ne := nfd.Read(rbuf)
		fmt.Printf("read %d byte, err %v: %v\n", nr, ne, rbuf)
	}
}

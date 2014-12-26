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
	doread := flag.Bool("r", false, "read response")
	hostport := flag.String("H", "111.13.100.92:80", "connect to host")
	flag.Parse()
	cl, ce := net.Dial("tcp4", *hostport)
	if ce != nil {
		TimePrintf("ERROR: %s\n", ce.Error())
		os.Exit(1)
	}
	TimePrintf("conected: %s\n", *hostport)

	readit := make(chan struct{}, 1)
	readok := make(chan struct{}, 1)
	if *doread {
		go func() {
			var rtimeout time.Duration = 5e8
			buflen := 81920
			rbuf := make([]byte, buflen)
			for {
				<-readit
				// read until read timeout
				total := 0
				for {
					cl.SetReadDeadline(time.Now().Add(rtimeout))
					nr, re := cl.Read(rbuf[total:])
					if nr > 0 {
						total += nr
					}
					if re != nil {
						TimePrintf("ERROR: read %s failed: %s\n", *hostport, re.Error())
						break
					}
					if total >= buflen {
						TimePrintf("Got part response %s (%d)\n ------ \n%s\n", *hostport, total, rbuf[:total])
						TimePrintf(" ------\n")
						total = 0
					}
				}
				if total > 0 {
					TimePrintf("Got final response %s (%d)\n ------ \n%s\n", *hostport, total, rbuf[:total])
					TimePrintf(" ------\n")
				}
				readok <- struct{}{}
			}
		}()
	}
	rbuf := make([]byte, 128)
	loop := 0
	TimePrintf("disconnect your router befor press <ENTER>\n")
	for {
		loop++
		TimePrintf("#%d, enter data to send:", loop)
		nr, re := os.Stdin.Read(rbuf)
		if re != nil {
			TimePrintf("ERROR: read stdin failed: %s\n", re.Error())
			os.Exit(1)
		}
		TimePrintf("writing to %s(%d): %s\n", *hostport, nr, rbuf[:nr])
		// Write(b []byte) (n int, err error)
		nw, ew := cl.Write(rbuf[:nr])
		if ew != nil {
			TimePrintf("ERROR: %s\n", ew.Error())
			os.Exit(1)
		}
		TimePrintf("write done %s(%d/%d): %s\n", *hostport, nw, nr, rbuf[:nr])
		if *doread {
			readit <- struct{}{}
			<-readok
		}
	}
}

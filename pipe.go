package main

import (
	"net"
	"time"
	"./verbose"
)

const (
	NO_TIMEOUT = iota
	SET_TIMEOUT
	readTimeout = 100000
)

func SetReadTimeout(c net.Conn) {
	if readTimeout != 0 {
		c.SetReadDeadline(time.Now().Add(readTimeout))
	}
}

const pipeBufSize = 4096
const nBuf = 2048

var pipeBuf = NewLeakyBuf(nBuf, pipeBufSize)

// PipeThenClose copies data from src to dst, closes dst when done.
func PipeThenClose(src, dst net.Conn, timeoutOpt int) {
	defer dst.Close()
	buf := pipeBuf.Get()
	defer pipeBuf.Put(buf)
	for {
		if timeoutOpt == SET_TIMEOUT {
			SetReadTimeout(src)
		}
		n, err := src.Read(buf)
		// read may return EOF with n > 0
		// should always process n > 0 bytes before handling error
		if n > 0 {
			if _, err = dst.Write(buf[0:n]); err != nil {
				verbose.TSPrintf("write:", err)
				break
			}
		}
		if err != nil {
			break
		}
	}
}

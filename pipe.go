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


/*

	sendCount := 0
	for {
		select {
		case b := <-read:
			// fill buf here
			log.Println(len(b))
			//verbose.TSPrintf("client: <-read of '%s'; hex:'%x' of length %d added to buffer\n", string(b), b, len(b))
			//buf.Write(b)
			//verbose.TSPrintf("client: after write to buf of len(b)=%d, buf is now length %d\n", len(b), buf.Len())
		case <-tick.C:
			sendCount++
			verbose.Log("\n ====================\n client sendCount = %d\n ====================\n", sendCount)
			verbose.Log("client: sendCount %d, got tick.C. key as always(?) = '%x'. buf is now size %d\n", sendCount, key, buf.Len())
			// write buf to new http request, starting with key
			req := bytes.NewBuffer(key)
			buf.WriteTo(req)
			resp, err := http.Post(
				"http://"+f.revProxyAddr+"/ping",
				"application/octet-stream",
				req)
			if err != nil && err != io.EOF {
				log.Println(err.Error())
				continue
			}

			// Write http response response to conn
			// we take apart the io.Copy to print out the response for debugging.
			//_, err = io.Copy(conn, resp.Body)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			verbose.Log("client: resp.Body = '%s'\n", string(body))
			_, err = conn.Write(body)
			if err != nil {
				panic(err)
			}
			resp.Body.Close()
		}
	}

*/

package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"time"
)

const (
	NO_TIMEOUT = iota
	SET_TIMEOUT
	readTimeout = 100000
)

const pipeBufSize = 1024
const nBuf = 2048

//var pipeBuf = NewLeakyBuf(nBuf, pipeBufSize)


// Take a reader, and turn it into a channel of bufSize chunks of []byte
func makeReadChan(r io.Reader, bufSize int) chan []byte {
	read := make(chan []byte)
	go func() {
		for {
			b := make([]byte, bufSize)
			n, err := r.Read(b)
			if err != nil {
				return
			}
			read <- b[0:n]
		}
	}()
	return read
}


// PipeThenClose copies data from src to dst, closes dst when done.
func TunnelAsHTTP(src, dst net.Conn, timeoutOpt int) {
	//defer dst.Close()
	//buf := pipeBuf.Get()
	//defer pipeBuf.Put(buf)
	buf := new(bytes.Buffer)
	read := makeReadChan(src, pipeBufSize)
	for {
		if timeoutOpt == SET_TIMEOUT {
			src.SetReadDeadline(time.Now().Add(readTimeout))
		}
		select {
		case b := <-read:
			buf.Write(b)
			buf.Reset()
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

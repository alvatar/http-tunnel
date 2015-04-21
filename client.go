package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
	"./socks"
)


const (
	bufSize = 8192
	readTimeout = 100000

	NO_TIMEOUT = iota
	SET_TIMEOUT
)

var (
	listenAddr   = flag.String("listen", ":2222", "local listen address")
	tunnelRemote = flag.String("tunnel", fmt.Sprintf("%s:%d", "127.0.0.1", 8888), "remote tunnel server")
	tickInterval = flag.Int("tick", 250, "update interval (msec)")
)

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

func handleConnection(tunnelLocalConn net.Conn, tunnelRemoteAddress string) {
	defer tunnelLocalConn.Close()
	if err := socks.HandShake(tunnelLocalConn); err != nil {
		log.Println("socks handshake:", err)
		return
	}
	_, targetAddr, err := socks.GetRequest(tunnelLocalConn)
	if err != nil {
		log.Println("error getting request:", err)
		return
	}

	// Connection established message
	_, err = tunnelLocalConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	if err != nil {
		log.Println("send connection confirmation:", err)
		return
	}

	log.Println("tunneling request to", targetAddr, "through", tunnelRemoteAddress)

	// Create the HTTP session for this connection at the other end of the tunnel -> get a Key (UUID)
	createBuf := bytes.NewBuffer([]byte(targetAddr + "\n"))
	resp, err := http.Post(
		"http://"+tunnelRemoteAddress+"/connect",
		"text/plain",
		createBuf)
	if err != nil {
		log.Println("error sending CONNECT command to remote tunnel endpoint: ", err.Error())
		return
	}
	defer resp.Body.Close()
	key, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading response: ", err.Error())
		return
	}
	/*
	_, err = tunnelLocalConn.Write([]byte(`HTTP/1.1 200 OK
Date: Mon, 23 May 2005 22:38:34 GMT
Server: Apache/1.3.3.7 (Unix) (Red-Hat/Linux)
Last-Modified: Wed, 08 Jan 2003 23:11:55 GMT
ETag: \"3f80f-1b6-3e1cb03b\"
Content-Type: text/html; charset=UTF-8
Content-Length: 130
Accept-Ranges: bytes
Connection: close

<html>
<head>
  <title>An Example Page</title>
</head>
<body>
  Hello World, this is a very simple HTML document.
</body>
</html>
`))
        */
	buf := new(bytes.Buffer)
	read := makeReadChan(tunnelLocalConn, 1024)
	tick := time.NewTicker(time.Duration(int64(*tickInterval)) * time.Millisecond)
	for {
		select {
		case b := <-read:
			buf.Write(b)
		case <-tick.C:
			if buf.Len() == 0 {
				continue
			}
			req := bytes.NewBuffer(key)
			resp, err := http.Post(
				"http://"+tunnelRemoteAddress,
				"application/octet-stream",
				req)
			if err != nil {
				log.Println("error sending octet stream to remote tunnel endpoint: ", err.Error())
				continue
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("error reading response: ", err.Error())
				return
			}
			log.Println(string(body))
			_, err = tunnelLocalConn.Write(body)
			if err != nil {
				log.Println("error sending response to client application: ", err.Error())
				return
			}
			tunnelLocalConn.Close()
		}
	}
}

func main() {
	flag.Parse()
	log.SetPrefix("http/socks client: ")

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handleConnection(conn, *tunnelRemote)
	}
}

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
	bufSize = 4096
	readTimeout = 10000

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

// Handle the clien application connection. It will communicate using the SOCKS
// protocol (which is just a handshake with two commands: connect/bind). The
// rest of the data is passed through.
func handleConnection(clientConn net.Conn, tunnelRemoteAddress string) {
	defer clientConn.Close()
	if err := socks.HandShake(clientConn); err != nil {
		log.Println("socks handshake:", err)
		return
	}
	_, targetAddr, err := socks.GetRequest(clientConn)
	if err != nil {
		log.Println("error getting request:", err)
		return
	}

	// Connection established message
	_, err = clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	if err != nil {
		log.Println("send connection confirmation:", err)
		return
	}

	log.Println("tunneling request to", targetAddr, "through", tunnelRemoteAddress)

	// Create the HTTP session for this connection at the other end of the tunnel -> get a Key (UUID)
	createBuf := bytes.NewBuffer([]byte(targetAddr))
	resp, err := http.Post(
		"http://" + tunnelRemoteAddress + "/connect",
		"text/plain",
		createBuf)
	if err != nil {
		log.Println("error sending CONNECT command to remote tunnel endpoint: ", err.Error())
		return
	}
	defer resp.Body.Close()
	uuid, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading response: ", err.Error())
		return
	}
	/*
	_, err = clientConn.Write([]byte(`HTTP/1.1 200 OK
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
	read := makeReadChan(clientConn, bufSize)
	tick := time.NewTicker(time.Duration(int64(*tickInterval)) * time.Millisecond)
	for {
		select {
			// Read from the client application connection
		case b := <-read:
			buf.Write(b)
			// Periodically contact the server sending the payload on behalf of the client application
		case <-tick.C:
			// Send a request for more data (may or may not send data as well)
			req := bytes.NewBuffer(uuid)
			buf.WriteTo(req)
			resp, err := http.Post(
				"http://" + tunnelRemoteAddress,
				"application/octet-stream",
				req)
			if err != nil {
				log.Println("error sending octet stream to remote tunnel endpoint: ", err.Error())
				continue
			}
			defer resp.Body.Close()

			// Read response from the tunnel server
			body, err := ioutil.ReadAll(resp.Body)
			// No more data to receive
			if err == io.EOF {
				clientConn.Close()
				return
			}
			if err != nil && err != io.EOF {
				log.Println("error reading response: ", err.Error())
				return
			}
			log.Println(string(body))
			// Send the received data to the client application
			_, err = clientConn.Write(body)
			if err != nil {
				log.Println("error sending response to client application: ", err.Error())
				return
			}
		}
	}
}

func main() {
	flag.Parse()
	log.SetPrefix("http/socks client: ")

	// Listen for client application requests
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	for {
		// Handle each application requests concurrently
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handleConnection(conn, *tunnelRemote)
	}
}

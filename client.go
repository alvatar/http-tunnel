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
)


const (
	bufSize = 8192
	readTimeout = 100000

	NO_TIMEOUT = iota
	SET_TIMEOUT
)

var (
	listenAddr   = flag.String("listen", ":2222", "local listen address")
	tunnelRemote = flag.String("tunnel", fmt.Sprintf("%s:%d", "127.0.0.0", 8888), "remote tunnel server")
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
	/*
		if err := socks.HandShake(conn); err != nil {
		log.Println("socks handshake:", err)
		return
	}
	_, targetAddr, err := socks.GetRequest(conn)
	if err != nil {
		log.Println("error getting request:", err)
		return
	}

	// Connection established message
	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43})
	if err != nil {
		log.Println("send connection confirmation:", err)
		return
	}

	log.Println("tunneling request to", targetAddr, "through", proxyAddress)
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
			req := new(bytes.Buffer)
			buf.WriteTo(req)
			log.Println("POINT 1")
			resp, err := http.Post(
				"http://"+tunnelRemoteAddress,
				"application/octet-stream",
				req)
			if err != nil && err != io.EOF {
				log.Println(err.Error())
				continue
			}
			defer resp.Body.Close()
			log.Println("POINT 2")
			body, err := ioutil.ReadAll(resp.Body)
			log.Println("POINT 3")
			if err != nil {
				panic(err)
			}
			log.Println(body)
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
		go handleConnection(conn, "127.0.0.1:8888")
	}
}

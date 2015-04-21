package main

import (
	"bytes"
	"io/ioutil"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"./socks"
)

const (
	bufSize = 8192
)

var (
	listenAddr   = flag.String("listen", ":2222", "local listen address")
	httpAddr     = flag.String("http", fmt.Sprintf("%s:%d", ReverseProxyIp, ReverseProxyPort), "remote tunnel server")
	tickInterval = flag.Int("tick", 250, "update interval (msec)") // orig: 250
)

type ForwardProxy struct {
	listenAddr       string
	revProxyAddr     string
	tickIntervalMsec int
}

func NewForwardProxy(listenAddr string, revProxAddr string, tickIntervalMsec int) *ForwardProxy {
	return &ForwardProxy{
		listenAddr:       listenAddr,
		revProxyAddr:     revProxAddr, // http server's address
		tickIntervalMsec: tickIntervalMsec,
	}
}

func handleSocksConnection(conn net.Conn, proxyAddress string) {
	if err := socks.HandShake(conn); err != nil {
		log.Println("socks handshake:", err)
		return
	}
	_, addr, err := socks.GetRequest(conn)
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

	log.Println("tunneling request to", addr, "through", proxyAddress)

	remote, err := net.Dial("tcp", proxyAddress)
	if err != nil {
		return
	}

	TunnelAsHTTP(conn, remote, NO_TIMEOUT)

	log.Println("closed connection to", addr)
}

func (f *ForwardProxy) ListenAndServe() {
	listener, err := net.Listen("tcp", f.listenAddr)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	log.Printf("listen on '%v', with revProxAddr '%v'", f.listenAddr, f.revProxyAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		log.Println("accept with", conn.LocalAddr(), "from ", conn.RemoteAddr())
		go handleSocksConnection(conn, f.revProxyAddr)
	}

	// --------------- NOT REACHED

	buf := new(bytes.Buffer)

	// initiate new session and read key
	log.Println("Attempting connect HttpTun Server.", f.revProxyAddr)
	resp, err := http.Post(
		"http://" + "127.0.0.1:22" + "/create",
		"text/plain",
		buf)
	if err != nil {
		panic(err)
	}
	key, err := ioutil.ReadAll(resp.Body) // key, err
	log.Println(key)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()

	// log.Printf("client main(): after Post('/create') we got ResponseWriter with key = '%x'", key)

	// ticker to set a streaming rate
	tick := time.NewTicker(time.Duration(int64(f.tickIntervalMsec)) * time.Millisecond)
	log.Println(tick)

	//read := makeReadChan(conn, bufSize)
	//buf.Reset()
}

func main() {
	flag.Parse()
	log.SetPrefix("tun.client: ")

	f := NewForwardProxy(*listenAddr, *httpAddr, *tickInterval)
	f.ListenAndServe()
}

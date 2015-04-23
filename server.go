package main

import (
	"flag"
//	"io"
	"io/ioutil"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/landjur/go-uuid"
)

const (
	readTimeoutMsec = 1000
)

var (
	listenAddr = flag.String("http", fmt.Sprintf("%s:%d", "127.0.0.1", 8888), "tunnel server listen address")
	connectQueue = make(chan *proxy)
	proxyMap = make(map[string]*proxy)
)

type proxy struct {
	pktChan   chan proxyPacket
	uuid      uuid.UUID         // UUID to identify the transaction (the application request)
	conn      net.Conn          // Connection to the target address
	recvCount int
}

// A new proxy is created per request at client's application side. The tunnel
// client side takes care of packing the client application requests into HTTP
// requests sent to this server (the tunnel server side).
func NewProxy(targetAddr string) (prx *proxy, err error) {
	uuid, err := uuid.NewTimeBased()
	prx = &proxy{pktChan: make(chan proxyPacket), uuid: uuid }
	prx.conn, err = net.Dial("tcp",targetAddr)
	if err != nil {
		return
	}
	err = prx.conn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	return
}

type proxyPacket struct {
	resp    http.ResponseWriter
	request *http.Request
	body    []byte
	done    chan bool
}

// Handle the /connect path, which will create and register a new proxy for the
// client application request
func connectHandler(c http.ResponseWriter, r *http.Request) {
	targetAddr, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		http.Error(c, "could not read connect request -", http.StatusInternalServerError)
		return
	}

	prx, err := NewProxy(string(targetAddr))
	if err != nil {
		log.Println("could not connect -", err)
		return
	}

	proxyMap[prx.uuid.String()] = prx
	c.Write([]byte(prx.uuid.String()))
}

// Handle the /tunnel path, which tunnels a client application request maskered
// as HTTP packets to the final destination
func tunnelHandler(c http.ResponseWriter, r *http.Request) {
}

func main() {
	flag.Parse()
	log.SetPrefix("http/socks server: ")

	http.HandleFunc("/", tunnelHandler)
	http.HandleFunc("/connect", connectHandler)
	err := http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

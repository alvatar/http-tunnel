package main

import (
	"flag"
	"io"
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
	uuidLen = 36 // 36 is the length of a formatted ASCII-based RFC4122 UUID
)

var (
	listenAddr = flag.String("http", fmt.Sprintf("%s:%d", "127.0.0.1", 8888), "tunnel server listen address")
	connectQueue = make(chan *proxy)
	packetQueue = make(chan proxyPacket)
	proxyMap = make(map[string]*proxy)
)

// A struct representing the packets tunneled through a proxy connection
type proxyPacket struct {
	resp    http.ResponseWriter
	request *http.Request
	body    []byte
	done    chan bool
}

// A struct representing a proxy connection
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
	if err != nil {
		panic(err)
	}
	prx = &proxy{pktChan: make(chan proxyPacket), uuid: uuid }
	prx.conn, err = net.Dial("tcp",targetAddr)
	if err != nil {
		return
	}
	err = prx.conn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	return
}

// Each proxy will take care of the packets sent through with its UUID
func (prx *proxy) handlePacket(packet proxyPacket) {
	// Read tunneled packet and send it to real target
	payload := packet.body[uuidLen:]
	n, err := prx.conn.Write(payload)
	if n != len(payload) {
		log.Printf("incomplete writing of payload")
	}
	packet.request.Body.Close()
	if err == io.EOF {
		prx.conn = nil
		log.Printf("EOF in proxy with UUID:", prx.uuid.String())
		return
	}

	// Read target response
	err = prx.conn.SetReadDeadline(time.Now().Add(time.Millisecond * readTimeoutMsec))
	if err != nil {
		log.Printf("error setting timeout for connection with UUID: %s %s" , prx.uuid.String(), err)
		return
	}
	targetResp := make([]byte, 1<<16) // The max theoretical size of a TCP packet (lower in practice)
	n, err = prx.conn.Read(targetResp)
	if err != nil {
		log.Println("error reading response from target", err)
		return
	}

	// Build response to tunnel client
	packet.resp.Header().Set("Content-type", "application/octet-stream")
	_, err = packet.resp.Write(targetResp[:n])
	if err != nil {
		log.Printf("error replying to tunnel client", err)
	}

	packet.done <- true // Signal the tunnelHandler so it can return
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

	// Do the actions that touch globals through a channel to handle
	// concurrent requests
	connectQueue <- prx

	c.Write([]byte(prx.uuid.String()))
}

// Handle the /tunnel path, which tunnels a client application request maskered
// as HTTP packets to the final destination
func tunnelHandler(c http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("error reading packet from tunnel client")
		return
	}

	packet := proxyPacket{resp: c, request: r, body: body, done: make(chan bool)}
	// Enqueue a packet
	packetQueue <- packet
	// Wait for the 'done' signal. This handler will send anything written to c to the tunnel client
	<- packet.done
}

func muxer() {
	for {
		select {
		case prx := <- connectQueue:
			proxyMap[prx.uuid.String()] = prx
		case packet := <- packetQueue:
			if len(packet.body) < uuidLen {
				continue
			}
			uuid := make([]byte, uuidLen)
			copy(uuid, packet.body)

			prx, ok := proxyMap[string(uuid)]
			if !ok {
				log.Printf("couldn't find proxy for key = ", uuid)
				continue
			}
			prx.handlePacket(packet)
		}
	}
}

func main() {
	flag.Parse()
	log.SetPrefix("http/socks server: ")

	go muxer()

	http.HandleFunc("/", tunnelHandler)
	http.HandleFunc("/connect", connectHandler)
	err := http.ListenAndServe(*listenAddr, nil)
	if err != nil {
		panic(err)
	}
}

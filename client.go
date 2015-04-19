package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"./verbose"
)

const bufSize = 1024
const KeyLen = 64

var ReverseProxyIp string = ""
var ReverseProxyPort int = 8888

var (
	listenAddr   = flag.String("listen", ":2222", "local listen address")
	httpAddr     = flag.String("http", fmt.Sprintf("%s:%d", ReverseProxyIp, ReverseProxyPort), "remote tunnel server")
	tickInterval = flag.Int("tick", 250, "update interval (msec)") // orig: 250
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

func (f *ForwardProxy) ListenAndServe() error {
	listener, err := net.Listen("tcp", f.listenAddr)
	if err != nil {
		panic(err)
	}
	log.Printf("listen on '%v', with revProxAddr '%v'", f.listenAddr, f.revProxyAddr)

	conn, err := listener.Accept()
	if err != nil {
		panic(err)
	}
	log.Println("accept conn", "localAddr.", conn.LocalAddr(), "remoteAddr.", conn.RemoteAddr())

	buf := new(bytes.Buffer)

	// initiate new session and read key
	log.Println("Attempting connect HttpTun Server.", f.revProxyAddr)
	resp, err := http.Post(
		"http://"+f.revProxyAddr+"/create",
		"text/plain",
		buf)
	if err != nil {
		panic(err)
	}
	key, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	resp.Body.Close()

	// log.Printf("client main(): after Post('/create') we got ResponseWriter with key = '%x'", key)

	// ticker to set a rate at which to hit the server
	tick := time.NewTicker(time.Duration(int64(f.tickIntervalMsec)) * time.Millisecond)
	read := makeReadChan(conn, bufSize)
	buf.Reset()
	sendCount := 0
	for {
		select {
		case b := <-read:
			// fill buf here
			verbose.Log("client: <-read of '%s'; hex:'%x' of length %d added to buffer\n", string(b), b, len(b))
			buf.Write(b)
			verbose.Log("client: after write to buf of len(b)=%d, buf is now length %d\n", len(b), buf.Len())

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
}

func main() {
	flag.Parse()
	log.SetPrefix("tun.client: ")

	f := NewForwardProxy(*listenAddr, *httpAddr, *tickInterval)
	f.ListenAndServe()
}

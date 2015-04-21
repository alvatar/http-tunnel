# HTTP Tunnel

## A proof of concept to tunnel TCP over HTTP, using a SOCKS frontend

Based on: gohttptunnel by Andrew Gerrand <adg@golang.org> and Jason E. Aten <j.e.aten@gmail.com>, and Shadowsocks.

### Example usage:

Run 'server' at your endpoint, by default it listens on port 8888.

    ./server

Run 'client' on your local machine, by default it listens locally on 2222.

    ./client -tunnel=serverAddress:8888

With both of them running (you must start server first), you can then
connect via ssh to localhost:2222 on the local machine:

    ssh -p 2222 username@127.0.0.1

### Flags
  * -listen=ip:port local tunnel endpoint (server address)
    (default :2222)
  * -tunnel=ip:port remote tunnel endpoint (server address)
    (default 127.0.0.1:8888 for local testing)
  * -tick=250 HTTP stream interval
    (default 250)

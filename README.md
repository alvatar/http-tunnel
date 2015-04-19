# http-tunnel

## A tool to tunnel TCP over HTTP with SOCKS dynamic port forwarding

Based on gohttptunnel by Andrew Gerrand <adg@golang.org> and Jason E. Aten <j.e.aten@gmail.com>

## Example usage:

Run 'server' at your endpoint, by default it listens on port 8888.

    ./server

Run 'client' on your local machine, by default it listens locally on 2222.

Flags:
  * -http=serverAddress:8888 to point to your server.
  * -dest=destAddr:destPort to point to your tunnel endpoint (your final target) _DEPRECATED_
    (default is -dest=127.0.0.1:22 to connect to local sshd on the server).
  * -tick [default 250] HTTP stream interval

    ./client -http=serverAddress:8888

With both of them running (you must start server first), you can then
connect via ssh to localhost:2222 on the local machine:

    ssh -p 2222 username@127.0.0.1

You should then be tunnelling over HTTP.

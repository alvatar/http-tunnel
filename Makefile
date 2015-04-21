.phony: all debug run

all:
	go build client.go config.go buffer.go
	go build server.go config.go buffer.go

debug:
	go build -gcflags "-N -l" config.go client.go # N: disable optimizations, l: disable inlining
	go build -gcflags "-N -l" config.go server.go # N: disable optimizations, l: disable inlining

run: all
	pkill -9 client-tcp-over-http || echo
	pkill -9 server-tcp-over-http || echo
	./server &
	./client &

clean:
	pkill -9 client || true
	pkill -9 server || true
	rm -f client server config

default: all

client:
	go get github.com/cbeuw/gotfo
	go build -o ./build/gq-client ./cmd/gq-client 

server:
	go get github.com/cbeuw/gotfo
	go build -o ./build/gq-server ./cmd/gq-server

all: client server

clean:
	rm -rf ./build/gq-*

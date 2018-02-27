default: all

client:
	go build -o ./build/gq-client ./cmd/gq-client 

server:
	go build -o ./build/gq-server ./cmd/gq-server

all: client server

clean:
	rm -rf ./build/gq-*

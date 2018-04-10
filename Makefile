default: all

update-version:
	./update-version.sh

client: update-version
	go get github.com/cbeuw/gotfo
	go build -o ./build/gq-client ./cmd/gq-client 

server: update-version
	go get github.com/cbeuw/gotfo
	go build -o ./build/gq-server ./cmd/gq-server

all: client server

clean:
	rm -rf ./build/gq-*

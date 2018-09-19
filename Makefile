default: all

version=$(shell ver=$$(git log -n 1 --pretty=oneline --format=%D | awk -F, '{print $$1}' | awk '{print $$3}'); \
	if [ "$$ver" = "master" ] ; then \
	ver="master($$(git log -n 1 --pretty=oneline --format=%h))" ; \
	fi ; \
	echo $$ver)

client: 
	go build -ldflags "-X main.version=${version}" -o ./build/gq-client ./cmd/gq-client 

server: 
	go build -ldflags "-X main.version=${version}" -o ./build/gq-server ./cmd/gq-server

install:
	mv build/gq-* /usr/local/bin

all: client server

clean:
	rm -rf ./build/gq-*

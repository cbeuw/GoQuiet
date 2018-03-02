package main

import (
	"github.com/cbeuw/GoQuiet/gqclient"
	"github.com/cbeuw/GoQuiet/gqclient/TLS"
	"io"
	"log"
	"net"
	"os"
	"time"
)

// ss refers to the ss-client, remote refers to the proxy server

type pipe interface {
	remoteToSS()
	ssToRemote()
	closePipe()
}

type pair struct {
	ss     net.Conn
	remote net.Conn
}

func (p *pair) closePipe() {
	go p.ss.Close()
	go p.remote.Close()
}

func (p *pair) remoteToSS() {
	for {
		data, err := TLS.ReadTillDrain(p.remote)
		if err != nil {
			p.closePipe()
			return
		}
		data = TLS.PeelRecordLayer(data)
		_, err = p.ss.Write(data)
		if err != nil {
			p.closePipe()
			return
		}
	}
}

func (p *pair) ssToRemote() {
	for {
		buf := make([]byte, 10240)
		i, err := io.ReadAtLeast(p.ss, buf, 1)
		if err != nil {
			p.closePipe()
			return
		}
		data := buf[:i]
		data = TLS.AddRecordLayer(data, []byte{0x17}, []byte{0x03, 0x03})
		_, err = p.remote.Write(data)
		if err != nil {
			p.closePipe()
			return
		}
	}
}

func initSequence(ssConn net.Conn, sta *gqclient.State) {
	// SS likes to make TCP connections and then immediately close it
	// without sending anything. This is apperently a feature.
	// But we don't want this because it may be significant to the GFW
	// and we don't want to make meaningless handshakes.
	// So we filter these empty connections
	var err error
	data := make([]byte, 1024)
	i, err := io.ReadAtLeast(ssConn, data, 1)
	if err != nil {
		go ssConn.Close()
	}
	data = data[:i]

	var remoteConn net.Conn
	for trial := 0; err == nil && trial < 3; trial++ {
		remoteConn, err = net.Dial("tcp", sta.SS_REMOTE_HOST+":"+sta.SS_REMOTE_PORT)
	}
	if remoteConn == nil {
		log.Println("Failed to connect to the proxy server")
		return
	}
	clientHello := TLS.ComposeInitHandshake(sta)
	_, err = remoteConn.Write(clientHello)
	if err != nil {
		log.Printf("Sending ClientHello to remote: %v\n", err)
		return
	}
	// Three discarded messages: ServerHello, ChangeCipherSpec and Finished
	for c := 0; c < 3; c++ {
		_, err = TLS.ReadTillDrain(remoteConn)
		if err != nil {
			log.Printf("Reading discarded message %v: %v\n", c, err)
			return
		}
	}
	reply := TLS.ComposeReply()
	_, err = remoteConn.Write(reply)
	if err != nil {
		log.Printf("Sending reply to remote: %v\n", err)
		return
	}
	p := pair{
		ssConn,
		remoteConn,
	}

	// Send the data we got from SS in the beginning
	data = TLS.AddRecordLayer(data, []byte{0x17}, []byte{0x03, 0x03})
	_, err = p.remote.Write(data)
	if err != nil {
		log.Printf("Sending first SS data to remote: %v\n", err)
		p.closePipe()
		return
	}
	go p.remoteToSS()
	go p.ssToRemote()

}

func main() {
	opaque := gqclient.BtoInt(gqclient.CryptoRandBytes(32))
	sta := &gqclient.State{
		SS_LOCAL_HOST: os.Getenv("SS_LOCAL_HOST"),
		// IP address of this plugin listening. Should be 127.0.0.1
		SS_LOCAL_PORT: os.Getenv("SS_LOCAL_PORT"),
		// The remote port set in SS, default to 8388
		SS_REMOTE_HOST: os.Getenv("SS_REMOTE_HOST"),
		// IP address of the proxy server with the server side of this plugin running
		SS_REMOTE_PORT: os.Getenv("SS_REMOTE_PORT"),
		// Port number of the proxy server with the server side of this plugin running
		// should be 443
		Now:    time.Now,
		Opaque: opaque,
	}
	configPath := os.Getenv("SS_PLUGIN_OPTIONS")
	err := sta.ParseConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}
	sta.SetAESKey()
	listener, err := net.Listen("tcp", sta.SS_LOCAL_HOST+":"+sta.SS_LOCAL_PORT)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go initSequence(conn, sta)
	}

}

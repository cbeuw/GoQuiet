package main

import (
	"errors"
	"flag"
	"github.com/cbeuw/GoQuiet/gqserver"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type pipe interface {
	remoteToServer()
	serverToRemote()
	closePipe()
}

type ssPair struct {
	ss     net.Conn
	remote net.Conn
}

type webPair struct {
	webServer net.Conn
	remote    net.Conn
}

func (pair *webPair) closePipe() {
	go pair.webServer.Close()
	go pair.remote.Close()
}

func (pair *ssPair) closePipe() {
	go pair.ss.Close()
	go pair.remote.Close()
}

func (pair *webPair) serverToRemote() {
	for {
		length, err := io.Copy(pair.remote, pair.webServer)
		if err != nil || length == 0 {
			pair.closePipe()
			return
		}
	}
}

func (pair *webPair) remoteToServer() {
	for {
		length, err := io.Copy(pair.webServer, pair.remote)
		if err != nil || length == 0 {
			pair.closePipe()
			return
		}
	}
}

func (pair *ssPair) remoteToServer() {
	for {
		data, err := gqserver.ReadTillDrain(pair.remote)
		if err != nil {
			pair.closePipe()
			return
		}
		data = gqserver.PeelRecordLayer(data)
		_, err = pair.ss.Write(data)
		if err != nil {
			pair.closePipe()
			return
		}
	}
}

func (pair *ssPair) serverToRemote() {
	for {
		buf := make([]byte, 10240)
		i, err := io.ReadAtLeast(pair.ss, buf, 1)
		if err != nil {
			pair.closePipe()
			return
		}
		data := buf[:i]
		data = gqserver.AddRecordLayer(data, []byte{0x17}, []byte{0x03, 0x03})
		_, err = pair.remote.Write(data)
		if err != nil {
			pair.closePipe()
			return
		}
	}
}

func dispatchConnection(conn net.Conn, sta *gqserver.State) {
	goWeb := func(data []byte) {
		pair, err := makeWebPipe(conn, sta)
		if err != nil {
			log.Printf("Making connection to redirection server: %v\n", err)
			go conn.Close()
			return
		}
		pair.webServer.Write(data)
		go pair.remoteToServer()
		go pair.serverToRemote()
	}
	goSS := func() {
		pair, err := makeSSPipe(conn, sta)
		if err != nil {
			log.Fatalf("Making connection to ss-server: %v\n", err)
		}
		go pair.remoteToServer()
		go pair.serverToRemote()
	}
	buf := make([]byte, 1500)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	i, err := io.ReadAtLeast(conn, buf, 1)
	if err != nil {
		go conn.Close()
		return
	}
	conn.SetReadDeadline(time.Time{})
	data := buf[:i]
	ch, err := gqserver.ParseClientHello(data)
	if err != nil {
		goWeb(data)
		return
	}
	isSS := gqserver.IsSS(ch, sta)
	if !isSS {
		log.Printf("+1 non SS traffic from %v\n", conn.RemoteAddr())
		goWeb(data)
		return
	}
	reply := gqserver.ComposeReply(ch)
	_, err = conn.Write(reply)
	if err != nil {
		log.Printf("Sending reply to remote: %v\n", err)
		go conn.Close()
		return
	}
	// Two discarded messages: ChangeCipherSpec and Finished
	for c := 0; c < 2; c++ {
		_, err = gqserver.ReadTillDrain(conn)
		if err != nil {
			log.Printf("Reading discarded message %v: %v\n", c, err)
			go conn.Close()
			return
		}
	}
	goSS()
}

func makeWebPipe(remote net.Conn, sta *gqserver.State) (*webPair, error) {
	conn, err := net.Dial("tcp", sta.WebServerAddr)
	if err != nil {
		return &webPair{}, errors.New("Connection to web server failed")
	}
	pair := &webPair{
		conn,
		remote,
	}
	return pair, nil
}

func makeSSPipe(remote net.Conn, sta *gqserver.State) (*ssPair, error) {
	conn, err := net.Dial("tcp", sta.SS_LOCAL_HOST+":"+sta.SS_LOCAL_PORT)
	if err != nil {
		return &ssPair{}, errors.New("Connection to SS server failed")
	}
	pair := &ssPair{
		conn,
		remote,
	}
	return pair, nil
}

func usedRandomCleaner(sta *gqserver.State) {
	for {
		time.Sleep(12 * time.Hour)
		now := int(sta.Now().Unix())
		for key, t := range sta.UsedRandom {
			if now-t > 12*3600 {
				sta.DelUsedRandom(key)
			}
		}
	}
}

func main() {
	// Should be 127.0.0.1 to listen to ss-server on this machine
	var localHost string
	// server_port in ss config, same as remotePort in plugin mode
	var localPort string
	// server in ss config, the outbound listening ip
	var remoteHost string
	// Outbound listening ip, should be 443
	var remotePort string
	var configPath string
	if os.Getenv("SS_LOCAL_HOST") != "" {
		localHost = os.Getenv("SS_LOCAL_HOST")
		localPort = os.Getenv("SS_LOCAL_PORT")
		remoteHost = os.Getenv("SS_REMOTE_HOST")
		remotePort = os.Getenv("SS_REMOTE_PORT")
		configPath = os.Getenv("SS_PLUGIN_OPTIONS")
	} else {
		localAddr := flag.String("r", "", "localAddr: 127.0.0.1:server_port as set in SS config")
		flag.StringVar(&remoteHost, "s", "0.0.0.0", "remoteHost: outbound listing ip, set to 0.0.0.0 to listen to everything")
		flag.StringVar(&remotePort, "p", "443", "remotePort: outbound listing port, should be 443")
		flag.StringVar(&configPath, "c", "gqserver.json", "configPath: path to gqserver.json")
		flag.Parse()
		if *localAddr == "" {
			log.Fatal("Must specify localAddr")
		}
		localHost = strings.Split(*localAddr, ":")[0]
		localPort = strings.Split(*localAddr, ":")[1]
		log.Printf("Starting standalone mode, listening on %v:%v to ss at %v:%v\n", remoteHost, remotePort, localHost, localPort)
	}
	sta := &gqserver.State{
		SS_LOCAL_HOST:  localHost,
		SS_LOCAL_PORT:  localPort,
		SS_REMOTE_HOST: remoteHost,
		SS_REMOTE_PORT: remotePort,
		Now:            time.Now,
		UsedRandom:     map[[32]byte]int{},
	}
	err := sta.ParseConfig(configPath)
	if err != nil {
		log.Fatalf("Configuration file error: %v", err)
	}
	sta.SetAESKey()
	go usedRandomCleaner(sta)
	listener, err := net.Listen("tcp", sta.SS_REMOTE_HOST+":"+sta.SS_REMOTE_PORT)
	log.Println("Listening on " + sta.SS_REMOTE_HOST + ":" + sta.SS_REMOTE_PORT)
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("%v", err)
			continue
		}
		go dispatchConnection(conn, sta)
	}

}

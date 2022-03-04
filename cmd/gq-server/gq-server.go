package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/cbeuw/GoQuiet/gqserver"
)

var version string

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
	buf := make([]byte, 20480)
	for {
		i, err := gqserver.ReadTillDrain(pair.remote, buf)
		if err != nil {
			pair.closePipe()
			return
		}
		data := gqserver.PeelRecordLayer(buf[:i])
		_, err = pair.ss.Write(data)
		if err != nil {
			pair.closePipe()
			return
		}
	}
}

func (pair *ssPair) serverToRemote() {
	buf := make([]byte, 10240)
	for {
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
			log.Printf("Making connection to ss-server: %v\n", err)
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
		log.Printf("+1 non SS non (or malformed) TLS traffic from %v\n", conn.RemoteAddr())
		goWeb(data)
		return
	}

	isSS := gqserver.IsSS(ch, sta)
	if !isSS {
		log.Printf("+1 non SS TLS traffic from %v\n", conn.RemoteAddr())
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
	discardBuf := make([]byte, 1024)
	for c := 0; c < 2; c++ {
		_, err = gqserver.ReadTillDrain(conn, discardBuf)
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
		return &webPair{}, err
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
		return &ssPair{}, err
	}
	pair := &ssPair{
		conn,
		remote,
	}
	return pair, nil
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
	var pluginOpts string

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if os.Getenv("SS_LOCAL_HOST") != "" {
		localHost = os.Getenv("SS_LOCAL_HOST")
		localPort = os.Getenv("SS_LOCAL_PORT")
		remoteHost = os.Getenv("SS_REMOTE_HOST")
		remotePort = os.Getenv("SS_REMOTE_PORT")
		pluginOpts = os.Getenv("SS_PLUGIN_OPTIONS")
	} else {
		localAddr := flag.String("r", "", "localAddr: 127.0.0.1:server_port as set in SS config")
		flag.StringVar(&remoteHost, "s", "0.0.0.0", "remoteHost: outbound listing ip, set to 0.0.0.0 to listen to everything")
		flag.StringVar(&remotePort, "p", "443", "remotePort: outbound listing port, should be 443")
		flag.StringVar(&pluginOpts, "c", "gqserver.json", "pluginOpts: path to gqserver.json or options seperated by semicolons")
		askVersion := flag.Bool("v", false, "Print the version number")
		printUsage := flag.Bool("h", false, "Print this message")
		flag.Parse()

		if *askVersion {
			fmt.Printf("gq-server %s\n", version)
			return
		}

		if *printUsage {
			flag.Usage()
			return
		}

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
	err := sta.ParseConfig(pluginOpts)
	if err != nil {
		log.Fatalf("Configuration file error: %v", err)
	}
	if strings.IndexByte(sta.SS_LOCAL_HOST, ':') != -1 {
	    sta.SS_LOCAL_HOST = "[" + sta.SS_LOCAL_HOST + "]"
	}
	if strings.IndexByte(sta.SS_REMOTE_HOST, ':') != -1 {
	    sta.SS_REMOTE_HOST = "[" + sta.SS_REMOTE_HOST + "]"
	}

	if sta.Key == "" {
		log.Fatal("Key cannot be empty")
	}

	sta.SetAESKey()
	go sta.UsedRandomCleaner()

	listen := func(addr, port string) {
		listener, err := net.Listen("tcp", addr+":"+port)
		log.Println("Listening on " + addr + ":" + port)
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

	// When listening on an IPv6 and IPv4, SS gives REMOTE_HOST as e.g. ::|0.0.0.0
	listeningIP := strings.Split(sta.SS_REMOTE_HOST, "|")
	for i, ip := range listeningIP {
		// if net.ParseIP(ip).To4() == nil {
			// IPv6 needs square brackets
			// ip = "[" + ip + "]"
		// }

		// The last listener must block main() because the program exits on main return.
		if i == len(listeningIP)-1 {
			listen(ip, sta.SS_REMOTE_PORT)
		} else {
			go listen(ip, sta.SS_REMOTE_PORT)
		}
	}

}

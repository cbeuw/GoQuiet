package main

import (
	"encoding/binary"
	"errors"
	"github.com/imsupernewstar/GoQuiet/goquiet/gqserver"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type Pipe interface {
	RemoteToServer()
	ServerToRemote()
	ClosePipe()
}

type SSPair struct {
	ss     net.Conn
	remote net.Conn
}

type WebPair struct {
	webServer net.Conn
	remote    net.Conn
}

func readTillDrain(conn net.Conn) []byte {
	t := time.Now()
	t = t.Add(3 * time.Second)
	conn.SetReadDeadline(t) // 3 seconds
	var buf []byte
	conn.Read(buf)
	msglen := gqserver.BtoInt(buf[3:5])
	for len(buf) < msglen {
		var tempbuf []byte
		conn.Read(tempbuf)
		buf = append(buf, tempbuf...)
	}
	conn.SetReadDeadline(time.Time{})
	return buf
}

func (pair *WebPair) ClosePipe() {
	pair.webServer.Close()
	pair.remote.Close()
}

func (pair *SSPair) ClosePipe() {
	pair.ss.Close()
	pair.remote.Close()
}

func (pair *WebPair) ServerToRemote() {
	_, err := io.Copy(pair.remote, pair.webServer)
	if err != nil {
		pair.ClosePipe()
	}
}

func (pair *WebPair) RemoteToServer() {
	for {
		_, err := io.Copy(pair.webServer, pair.remote)
		if err != nil {
			pair.ClosePipe()
			return
		}
	}
}

func (pair *SSPair) RemoteToServer() {
	for {
		data := readTillDrain(pair.remote)
		data = data[5:]
		_, err := pair.ss.Write(data)
		if err != nil {
			pair.ClosePipe()
			return
		}
	}
}

func (pair *SSPair) ServerToRemote() {
	for {
		data := []byte{}
		pair.ss.Read(data)
		msglen := make([]byte, 2)
		binary.BigEndian.PutUint16(msglen, uint16(len(data)))
		record := append([]byte{0x17, 0x03, 0x03}, msglen...)
		data = append(record, data...)
		_, err := pair.remote.Write(data)
		if err != nil {
			pair.ClosePipe()
			return
		}
	}
}

func dispatchConnection(conn net.Conn) {
	goWeb := func(data []byte) {
		pair, err := MakeWebPipe(conn)
		if err != nil {
			log.Fatal(err)
		}
		pair.webServer.Write(data)
		go pair.RemoteToServer()
		go pair.ServerToRemote()
	}
	data := []byte{}
	conn.Read(data)
	client_hello, err := gqserver.ParseClientHello(data)
	if err != nil {
		goWeb(data)
		return
	}
	is_SS := gqserver.IsSS(&client_hello)
	if !is_SS {
		goWeb(data)
		return
	}

}

var SS_LOCAL_HOST string
var SS_LOCAL_PORT string
var SS_REMOTE_HOST string
var SS_REMOTE_PORT string

func MakeWebPipe(remote net.Conn) (WebPair, error) {
	conn, err := net.Dial("tcp", gqserver.Config.Web_server_addr)
	if err != nil {
		return WebPair{}, errors.New("Connection to web server failed")
	}
	pair := WebPair{
		conn,
		remote,
	}
	return pair, nil
}

func MakeSSPipe(remote net.Conn) (SSPair, error) {
	conn, err := net.Dial("tcp", SS_LOCAL_HOST+":"+SS_LOCAL_PORT)
	if err != nil {
		return SSPair{}, errors.New("Connection to SS server failed")
	}
	pair := SSPair{
		conn,
		remote,
	}
	return pair, nil
}

func main() {
	SS_LOCAL_HOST = os.Getenv("SS_LOCAL_HOST")
	// Should be 127.0.0.1 unless the plugin is deployed on another machine, which is not supported yet
	SS_LOCAL_PORT = os.Getenv("SS_LOCAL_PORT")
	// SS loopback port, default set by SS to 8388
	SS_REMOTE_HOST = os.Getenv("SS_REMOTE_HOST")
	// Outbound listening address, should be 0.0.0.0
	SS_REMOTE_PORT = os.Getenv("SS_REMOTE_PORT")
	// Since this is a TLS obfuscator, this should be 443
	err := gqserver.ParseConfig(os.Args[1])
	if err != nil {
		panic(err)
	}

	listener, _ := net.Listen("tcp", SS_REMOTE_HOST+":"+SS_REMOTE_PORT)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("%v", err)
			continue
		}
		go dispatchConnection(conn)
	}

}

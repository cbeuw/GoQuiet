package TLS

import (
	"encoding/binary"
	"github.com/cbeuw/GoQuiet/gqclient"
	"time"
)

// AddRecordLayer adds record layer to data
func AddRecordLayer(input []byte, typ []byte, ver []byte) []byte {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(input)))
	ret := append(typ, ver...)
	ret = append(ret, length...)
	ret = append(ret, input...)
	return ret
}

// PeelRecordLayer peels off the record layer
func PeelRecordLayer(data []byte) []byte {
	ret := data[5:]
	return ret
}

type browser interface {
	composeExtensions()
	composeClientHello()
}

func makeServerName(sta *gqclient.State) []byte {
	serverName := sta.ServerName
	serverNameLength := make([]byte, 2)
	binary.BigEndian.PutUint16(serverNameLength, uint16(len(serverName)))
	serverNameType := []byte{0x00} // host_name
	var ret []byte
	ret = append(serverNameType, serverNameLength...)
	ret = append(ret, serverName...)
	serverNameListLength := make([]byte, 2)
	binary.BigEndian.PutUint16(serverNameListLength, uint16(len(ret)))
	return append(serverNameListLength, ret...)
}

func makeSessionTicket(sta *gqclient.State) []byte {
	seed := int64(sta.Opaque + gqclient.BtoInt(sta.AESKey) + int(sta.Now().Unix())/sta.TicketTimeHint)
	return gqclient.PsudoRandBytes(192, seed)
}

func makeNullBytes(length int) []byte {
	var ret []byte
	for i := 0; i < length; i++ {
		ret = append(ret, 0x00)
	}
	return ret
}

// addExtensionRecord, add type, length to extension data
func addExtRec(typ []byte, data []byte) []byte {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(data)))
	var ret []byte
	ret = append(typ, length...)
	return append(ret, data...)
}

// ComposeInitHandshake composes ClientHello with record layer
func ComposeInitHandshake(sta *gqclient.State) []byte {
	var ch []byte
	switch sta.Browser {
	case "chrome":
		ch = (&chrome{}).composeClientHello(sta)
	case "firefox":
		ch = (&firefox{}).composeClientHello(sta)
	default:
		panic("Unsupported browser:" + sta.Browser)
	}
	return AddRecordLayer(ch, []byte{0x16}, []byte{0x03, 0x01})
}

// ComposeReply composes RL+ChangeCipherSpec+RL+Finished
func ComposeReply() []byte {
	TLS12 := []byte{0x03, 0x03}
	ccsBytes := AddRecordLayer([]byte{0x01}, []byte{0x14}, TLS12)
	finished := gqclient.PsudoRandBytes(40, time.Now().UnixNano())
	fBytes := AddRecordLayer(finished, []byte{0x16}, TLS12)
	return append(ccsBytes, fBytes...)
}

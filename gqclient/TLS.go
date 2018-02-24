package gqclient

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math/rand"
	"net"
	"time"
)

// ReadTillDrain reads TLS data according to its record layer
func ReadTillDrain(conn net.Conn) (ret []byte, err error) {
	// TCP is a stream. Multiple TLS messages can arrive at the same time,
	// a single message can also be segmented due to MTU of the IP layer.
	// This function guareentees a single TLS message to be read and everything
	// else is left in the buffer.
	record := make([]byte, 5)
	i, err := io.ReadFull(conn, record)
	if err != nil {
		return
	}
	ret = record
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	left := BtoInt(record[3:5])
	for left != 0 {
		buffered := bufio.NewReader(conn).Buffered()
		// min(left,buffered)
		// Read as much as the buffer has if we know the entire buffer
		// belongs to this message, or only the part left if the buffer contains
		// other stuff.
		var toRead int
		if left > buffered {
			toRead = buffered
		} else {
			toRead = left
		}
		buf := make([]byte, toRead)
		i, err = io.ReadFull(conn, buf)
		if err != nil {
			return
		}
		left -= i
		ret = append(ret, buf[:i]...)
	}
	conn.SetReadDeadline(time.Time{})
	return
}

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

// see https://tools.ietf.org/html/draft-davidben-tls-grease-01
// This is a characteristic of chrome.
func makeGREASE() []byte {
	rand.Seed(time.Now().UnixNano())
	sixteenth := rand.Intn(16)
	monoGREASE := byte(sixteenth*16 + 0xA)
	doubleGREASE := []byte{monoGREASE, monoGREASE}
	return doubleGREASE
}

func makeServerName(sta *State) []byte {
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

func makeSessionTicket(sta *State) []byte {
	seed := int64(sta.Opaque + BtoInt(sta.AESKey) + int(sta.Now().Unix())/sta.TicketTimeHint)
	return PsudoRandBytes(192, seed)
}

func makeSupportedGroups() []byte {
	suppGroupListLen := []byte{0x00, 0x08}
	suppGroup := append(makeGREASE(), []byte{0x00, 0x1d, 0x00, 0x17, 0x00, 0x18}...)
	return append(suppGroupListLen, suppGroup...)
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

func composeExtensions(sta *State) []byte {
	var extensions [14][]byte
	extensions[0] = addExtRec(makeGREASE(), nil)                          // First GREASE
	extensions[1] = addExtRec([]byte{0xff, 0x01}, []byte{0x00})           // renegotiation_info
	extensions[2] = addExtRec([]byte{0x00, 0x00}, makeServerName(sta))    // server name indication
	extensions[3] = addExtRec([]byte{0x00, 0x17}, nil)                    // extended_master_secret
	extensions[4] = addExtRec([]byte{0x00, 0x23}, makeSessionTicket(sta)) // Session tickets
	sigAlgo, _ := hex.DecodeString("0012040308040401050308050501080606010201")
	extensions[5] = addExtRec([]byte{0x00, 0x0d}, sigAlgo)                              // Signature Algorithms
	extensions[6] = addExtRec([]byte{0x00, 0x05}, []byte{0x01, 0x00, 0x00, 0x00, 0x00}) // status request
	extensions[7] = addExtRec([]byte{0x00, 0x12}, nil)                                  // signed cert timestamp
	APLN, _ := hex.DecodeString("000c02683208687474702f312e31")
	extensions[8] = addExtRec([]byte{0x00, 0x10}, APLN)                                   // app layer proto negotiation
	extensions[9] = addExtRec([]byte{0x75, 0x50}, nil)                                    // channel id
	extensions[10] = addExtRec([]byte{0x00, 0x0b}, []byte{0x01, 0x00})                    // ec point formats
	extensions[11] = addExtRec([]byte{0x00, 0x0a}, makeSupportedGroups())                 // supported groups
	extensions[12] = addExtRec(makeGREASE(), []byte{0x00})                                // Last GREASE
	extensions[13] = addExtRec([]byte{0x00, 0x15}, makeNullBytes(110-len(extensions[2]))) // padding
	var ret []byte
	for i := 0; i < 14; i++ {
		ret = append(ret, extensions[i]...)
	}
	return ret
}

func composeClientHello(sta *State) []byte {
	cipherSuites, _ := hex.DecodeString("2a2ac02bc02fc02cc030cca9cca8c013c014009c009d002f0035000a")
	var clientHello [12][]byte
	clientHello[0] = []byte{0x01}                             // handshake type
	clientHello[1] = []byte{0x00, 0x01, 0xfc}                 // length 508
	clientHello[2] = []byte{0x03, 0x03}                       // client version
	clientHello[3] = makeRandomField(sta)                     // random
	clientHello[4] = []byte{0x20}                             // session id length 32
	clientHello[5] = PsudoRandBytes(32, sta.Now().UnixNano()) // session id
	clientHello[6] = []byte{0x00, 0x1c}                       // cipher suites length 28
	clientHello[7] = cipherSuites                             // cipher suites
	clientHello[8] = []byte{0x01}                             // compression methods length 1
	clientHello[9] = []byte{0x00}                             // compression methods
	clientHello[10] = []byte{0x01, 0x97}                      // extensions length 407
	clientHello[11] = composeExtensions(sta)                  // extensions
	var ret []byte
	for i := 0; i < 12; i++ {
		ret = append(ret, clientHello[i]...)
	}
	return ret
}

func ComposeInitHandshake(sta *State) []byte {
	ch := composeClientHello(sta)
	return AddRecordLayer(ch, []byte{0x16}, []byte{0x03, 0x01})
}

func ComposeReply() []byte {
	TLS12 := []byte{0x03, 0x03}
	ccsBytes := AddRecordLayer([]byte{0x01}, []byte{0x14}, TLS12)
	finished := PsudoRandBytes(40, time.Now().UnixNano())
	fBytes := AddRecordLayer(finished, []byte{0x16}, TLS12)
	return append(ccsBytes, fBytes...)
}

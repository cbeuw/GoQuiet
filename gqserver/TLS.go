package gqserver

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
)

type ClientHello struct {
	handshake_type          byte
	length                  int
	client_version          []byte
	random                  []byte
	session_id_len          int
	session_id              []byte
	cipher_suites_len       int
	cipher_suites           []byte
	compression_methods_len int
	compression_methods     []byte
	extensions_len          int
	extensions              map[[2]byte][]byte
}

func parseExtensions(input []byte) (ret map[[2]byte][]byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
			err = errors.New("Malformed Extensions")
		}
	}()
	pointer := 0
	total_len := len(input)
	ret = make(map[[2]byte][]byte)
	for pointer < total_len {
		var typ [2]byte
		copy(typ[:], input[pointer:pointer+2])
		pointer += 2
		length := BtoInt(input[pointer : pointer+2])
		pointer += 2
		data := input[pointer : pointer+length]
		pointer += length
		ret[typ] = data
	}
	return ret, err
}

func peelRecordLayer(data []byte) (ret []byte, err error) {
	ret = data[5:]
	return
}
func ParseClientHello(data []byte) (ret *ClientHello, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("Malformed ClientHello")
		}
	}()
	data, err = peelRecordLayer(data)
	pointer := 0
	// Handshake Type
	handshake_type := data[pointer]
	if handshake_type != 0x01 {
		return ret, errors.New("Not a ClientHello")
	}
	pointer += 1
	// Length
	length := BtoInt(data[pointer : pointer+3])
	pointer += 3
	if length != len(data[pointer:]) {
		return ret, errors.New("Hello length doesn't match")
	}
	// Client Version
	client_version := data[pointer : pointer+2]
	pointer += 2
	// Random
	random := data[pointer : pointer+32]
	pointer += 32
	// Session ID
	session_id_len := int(data[pointer])
	pointer += 1
	session_id := data[pointer : pointer+session_id_len]
	pointer += session_id_len
	// Cipher Suites
	cipher_suites_len := BtoInt(data[pointer : pointer+2])
	pointer += 2
	cipher_suites := data[pointer : pointer+cipher_suites_len]
	pointer += cipher_suites_len
	// Compression Methods
	compression_methods_len := int(data[pointer])
	pointer += 1
	compression_methods := data[pointer : pointer+compression_methods_len]
	pointer += compression_methods_len
	// Extensions
	extensions_len := BtoInt(data[pointer : pointer+2])
	pointer += 2
	extensions, err := parseExtensions(data[pointer:])
	ret = &ClientHello{
		handshake_type,
		length,
		client_version,
		random,
		session_id_len,
		session_id,
		cipher_suites_len,
		cipher_suites,
		compression_methods_len,
		compression_methods,
		extensions_len,
		extensions,
	}
	return
}

func addRecordLayer(input []byte, typ []byte) []byte {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(input)))
	ret := append(typ, []byte{0x03, 0x03}...)
	ret = append(ret, length...)
	ret = append(ret, input...)
	return ret
}

func composeServerHello(client_hello *ClientHello) []byte {
	var server_hello [10][]byte
	server_hello[0] = []byte{0x02}             // handshake type
	server_hello[1] = []byte{0x00, 0x00, 0x4d} // length 77
	server_hello[2] = []byte{0x03, 0x03}       // server version
	random := make([]byte, 32)
	binary.BigEndian.PutUint32(random, rand.Uint32())
	server_hello[3] = random                               // random
	server_hello[4] = []byte{0x20}                         // session id length 32
	server_hello[5] = client_hello.session_id              // session id
	server_hello[6] = []byte{0xc0, 0x30}                   // cipher suite TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
	server_hello[7] = []byte{0x00}                         // compression method null
	server_hello[8] = []byte{0x00, 0x05}                   // extensions length 5
	server_hello[9] = []byte{0xff, 0x01, 0x00, 0x01, 0x00} // extensions renegotiation_info
	ret := []byte{}
	for i := 0; i < 10; i++ {
		ret = append(ret, server_hello[i]...)
	}
	return ret
}

func ComposeReply(client_hello *ClientHello) []byte {
	sh_bytes := addRecordLayer(composeServerHello(client_hello), []byte{0x16})
	ccs_bytes := addRecordLayer([]byte{0x01}, []byte{0x14})
	finished := make([]byte, 64)
	r := rand.Uint64()
	binary.BigEndian.PutUint64(finished, r)
	finished = finished[0:40]
	f_bytes := addRecordLayer(finished, []byte{0x16})
	ret := append(sh_bytes, ccs_bytes...)
	ret = append(ret, f_bytes...)
	return ret
}

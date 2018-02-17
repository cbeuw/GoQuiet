package gqserver

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"time"
)

func decrypt(iv []byte, key []byte, ciphertext []byte) []byte {
	block, _ := aes.NewCipher(key)
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	// "ciphertext" is now plaintext
	return ciphertext
}

func IsSS(input *ClientHello) bool {
	ticket := input.extensions[[2]byte{0x00, 0x23}]
	if len(ticket) != 192 {
		return false
	}

	h := sha256.New()
	t := int(time.Now().Unix()) / Config.ticket_time_hint
	h.Write([]byte(fmt.Sprintf("%v", t) + Config.key))
	goal := h.Sum(nil)

	plaintext := decrypt(input.random[0:16], Config.aes_key, input.random[16:])
	if !bytes.Equal(plaintext, goal[0:16]) {
		return false
	}

	plaintext = decrypt(ticket[0:16], Config.aes_key, ticket[16:])
	return bytes.Equal(plaintext[0:32], goal)
}

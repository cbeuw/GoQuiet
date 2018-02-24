package gqserver

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"log"
)

func decrypt(iv []byte, key []byte, ciphertext []byte) []byte {
	ret := make([]byte, len(ciphertext))
	copy(ret, ciphertext) // Because XORKeyStream is inplace, but we don't want the input to be changed
	block, _ := aes.NewCipher(key)
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ret, ret)
	// ret is now plaintext
	return ret
}

// IsSS checks if a ClientHello belongs to shadowsocks
func IsSS(input *ClientHello, sta *State) bool {
	var random [32]byte
	copy(random[:], input.random)

	sta.M.Lock()
	used := sta.UsedRandom[random]
	sta.M.Unlock()

	if used != 0 {
		log.Println("Replay! Duplicate random")
		return false
	}

	sta.PutUsedRandom(random)

	h := sha256.New()
	t := int(sta.Now().Unix()) / (12 * 60 * 60)
	h.Write([]byte(fmt.Sprintf("%v", t) + sta.Key))
	goal := h.Sum(nil)[0:16]
	plaintext := decrypt(input.random[0:16], sta.AESKey, input.random[16:])
	return bytes.Equal(plaintext, goal)
}

package gqclient

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
)

func encrypt(iv []byte, key []byte, plaintext []byte) []byte {
	block, _ := aes.NewCipher(key)
	ciphertext := make([]byte, len(plaintext))
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext, plaintext)
	return ciphertext
}

// MakeRandomField makes the random value that can pass the check at server side
func MakeRandomField(sta *State) []byte {
	h := sha256.New()
	t := int(sta.Now().Unix()) / (12 * 60 * 60)
	h.Write([]byte(fmt.Sprintf("%v", t) + sta.Key))
	goal := h.Sum(nil)[0:16]
	iv := CryptoRandBytes(16)
	rest := encrypt(iv, sta.AESKey, goal)
	return append(iv, rest...)
}

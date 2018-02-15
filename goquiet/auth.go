package goquiet

import (
	"crypto/aes"
	"crypto/cipher"
)

func Decrypt(iv []byte, key []byte, ciphertext []byte) []byte {
	block, _ := aes.NewCipher(key)
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	// "ciphertext" is now plaintext
	return ciphertext
}

func IsSS(input ClientHello) bool {
	return false
}

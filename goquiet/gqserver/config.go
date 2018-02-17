package gqserver

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/ioutil"
)

type Conf struct {
	Web_server_addr  string
	key              string
	ticket_time_hint int
	aes_key          []byte
}

var Config Conf

func ParseConfig(filepath string) (err error) {
	content, err := ioutil.ReadFile(filepath)
	if err != nil {
		return errors.New("Failed to read config file")
	}
	json.Unmarshal(content, &Config)
	h := sha256.New()
	h.Write([]byte(Config.key))
	Config.aes_key = h.Sum(nil)
	return
}

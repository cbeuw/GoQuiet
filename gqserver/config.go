package gqserver

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"
)

type TimeFn func() time.Time

type State struct {
	Web_server_addr  string
	Key              string
	Ticket_time_hint int
	AES_key          []byte
	Now              TimeFn
	SS_LOCAL_HOST    string
	SS_LOCAL_PORT    string
	SS_REMOTE_HOST   string
	SS_REMOTE_PORT   string
	Used_random      map[[32]byte]int
}

func ParseConfig(configPath string, sta *State) error {
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return errors.New("Failed to read config file. File may not exist")
	}
	err = json.Unmarshal(content, &sta)
	if err != nil {
		return errors.New("Bad config json format")
	}
	MakeAESKey(sta)
	return nil
}

func MakeAESKey(sta *State) {
	h := sha256.New()
	h.Write([]byte(sta.Key))
	sta.AES_key = h.Sum(nil)
}

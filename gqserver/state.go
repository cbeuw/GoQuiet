package gqserver

import (
	"crypto/sha256"
	"encoding/json"
	"io/ioutil"
	"strings"
	"sync"
	"time"
)

type stateManager interface {
	ParseConfig(string) error
	SetAESKey(string)
	PutUsedRandom([32]byte)
}

// State type stores the global state of the program
type State struct {
	WebServerAddr  string
	Key            string
	AESKey         []byte
	Now            func() time.Time
	SS_LOCAL_HOST  string
	SS_LOCAL_PORT  string
	SS_REMOTE_HOST string
	SS_REMOTE_PORT string
	M              sync.RWMutex
	UsedRandom     map[[32]byte]int
}

// semi-colon separated value.
func ssvToJson(ssv string) (ret []byte) {
	unescape := func(s string) string {
		r := strings.Replace(s, "\\\\", "\\", -1)
		r = strings.Replace(r, "\\=", "=", -1)
		r = strings.Replace(r, "\\;", ";", -1)
		return r
	}
	lines := strings.Split(unescape(ssv), ";")
	ret = []byte("{")
	for _, ln := range lines {
		if ln == "" {
			break
		}
		sp := strings.SplitN(ln, "=", 2)
		key := sp[0]
		value := sp[1]
		ret = append(ret, []byte("\""+key+"\":\""+value+"\",")...)

	}
	ret = ret[:len(ret)-1] // remove the last comma
	ret = append(ret, '}')
	return ret
}

// ParseConfig parses the config (either a path to json or in-line ssv config) into a State variable
func (sta *State) ParseConfig(config string) (err error) {
	var content []byte
	if strings.Contains(config, ";") && strings.Contains(config, "=") {
		content = ssvToJson(config)
	} else {
		content, err = ioutil.ReadFile(config)
		if err != nil {
			return err
		}
	}
	err = json.Unmarshal(content, &sta)
	if err != nil {
		return err
	}
	return nil
}

// SetAESKey calculates the SHA256 of the string key
func (sta *State) SetAESKey() {
	h := sha256.New()
	h.Write([]byte(sta.Key))
	sta.AESKey = h.Sum(nil)
}

// UsedRandomCleaner clears the cache of used random fields every 12 hours
func (sta *State) UsedRandomCleaner() {
	for {
		time.Sleep(12 * time.Hour)
		now := int(sta.Now().Unix())
		sta.M.Lock()
		for key, t := range sta.UsedRandom {
			if now-t > 12*3600 {
				delete(sta.UsedRandom, key)
			}
		}
		sta.M.Unlock()
	}
}

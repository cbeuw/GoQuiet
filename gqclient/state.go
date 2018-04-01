package gqclient

import (
	"crypto/sha256"
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"
)

type stateManager interface {
	ParseConfig(string) error
	SetAESKey(string)
}

// State stores global variables
type State struct {
	SS_LOCAL_HOST  string
	SS_LOCAL_PORT  string
	SS_REMOTE_HOST string
	SS_REMOTE_PORT string
	Now            func() time.Time
	Opaque         int
	Key            string
	TicketTimeHint int
	AESKey         []byte
	ServerName     string
	Browser        string
	FastOpen       bool
}

// semi-colon separated value. This is for Android plugin options
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
		// JSON doesn't like quotation marks around int and boolean
		// Yes this is extremely ugly but it's still better than writing a tokeniser
		if key == "TicketTimeHint" || key == "FastOpen" {
			ret = append(ret, []byte("\""+key+"\":"+value+",")...)
		} else {
			ret = append(ret, []byte("\""+key+"\":\""+value+"\",")...)
		}
	}
	ret = ret[:len(ret)-1] // remove the last comma
	ret = append(ret, '}')
	return ret
}

// ParseConfig parses the config (either a path to json or Android config) into a State variable
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

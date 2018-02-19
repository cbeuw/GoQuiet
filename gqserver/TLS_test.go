package gqserver

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func TestParseClientHello(t *testing.T) {
	dir := "tests/TLS/"
	files, _ := ioutil.ReadDir(dir)
	for _, c := range files {
		if c.Name()[0] == '.' {
			continue
		}
		content, _ := ioutil.ReadFile(dir + c.Name())
		result, err := ParseClientHello(content)
		indicat := strings.Split(c.Name(), "_")
		if indicat[0] == "OK" {
			if err != nil {
				t.Error(
					"For", c.Name(),
					"expected", "OK",
					"got", err,
				)
			} else {
				var key [2]byte
				temp, _ := hex.DecodeString(indicat[1])
				copy(key[:], temp)
				exp, _ := hex.DecodeString(indicat[2])
				if !bytes.Equal(result.extensions[key], exp) {
					t.Error(
						"For", c.Name(),
						"expected", indicat[2],
						"got", fmt.Sprintf("%x", result.extensions[key]),
					)
				}
			}
		} else if indicat[0] == "ERR" {
			if err == nil {
				t.Error(
					"For", c.Name(),
					"expected", indicat[1],
					"got", "no err",
				)
			}
		}
	}
}

func TestComposeReply(t *testing.T) {
	dir := "tests/TLS/"
	files, _ := ioutil.ReadDir(dir)
	for _, c := range files {
		indicat := strings.Split(c.Name(), "_")
		if indicat[0] != "OK" {
			continue
		}
		content, _ := ioutil.ReadFile(dir + c.Name())
		ch, _ := ParseClientHello(content)
		result := ComposeReply(ch)
		if !bytes.Equal(result[44:76], ch.sessionId) {
			t.Error(
				"For", c.Name(),
				"expected", fmt.Sprintf("%x", ch.sessionId),
				"got", fmt.Sprintf("%x", result[44:76]),
			)
		}
	}
}

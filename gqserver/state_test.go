package gqserver

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	path := "../config/gqserver.json"
	sta := &State{}
	err := sta.ParseConfig(path)
	if err != nil {
		t.Error(err)
		fmt.Println("Key: " + sta.Key)
		fmt.Println("WebServerAddr: " + sta.WebServerAddr)
	}
}

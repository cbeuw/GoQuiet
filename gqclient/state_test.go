package gqclient

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	path := "../config/gqclient.json"
	sta := &State{}
	err := sta.ParseConfig(path)
	if err != nil {
		t.Error(err)
		fmt.Println("Key: " + sta.Key)
		fmt.Println("ServerName: " + sta.ServerName)
		fmt.Printf("TicketTimeHint: %v\n", sta.TicketTimeHint)
	}
}

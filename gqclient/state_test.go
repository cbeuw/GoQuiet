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
		fmt.Printf("TicketTimeHint: %+v", *sta)
	}
}

func TestSsvToJson(t *testing.T) {
	ssv := "Browser=chrome;Key=example;TicketTimeHint=1234;"
	sta := &State{}
	err := sta.ParseConfig(ssv)
	if err != nil {
		t.Error(err)
		fmt.Printf("TicketTimeHint: %+v", *sta)
	}
}

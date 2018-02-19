package gqserver

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestIsSS(t *testing.T) {
	dir := "tests/auth/"
	files, _ := ioutil.ReadDir(dir)
	for _, c := range files {
		if c.Name()[0] == '.' {
			continue
		}
		content, _ := ioutil.ReadFile(dir + c.Name())
		indicat := strings.Split(c.Name(), "_")
		timeHint, _ := strconv.Atoi(indicat[2])
		mockTime, _ := strconv.Atoi(indicat[3])
		mockTimeFn := func() time.Time {
			return time.Unix(int64(mockTime), 0)
		}
		mockSta := &State{
			Key:              indicat[1],
			Ticket_time_hint: timeHint,
			Now:              mockTimeFn,
			Used_random:      map[[32]byte]int{},
		}
		MakeAESKey(mockSta)
		ch, err := ParseClientHello(content)
		if err != nil {
			fmt.Println(err)
		}
		isss := IsSS(ch, mockSta)
		if indicat[0] == "TRUE" && !isss {
			t.Error(
				"For", c.Name(),
				"expecting", "true",
				"got", isss,
			)
		}
	}
}

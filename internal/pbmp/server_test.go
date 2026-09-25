package pbmp

import (
	"encoding/json"
	"testing"
)

func TestHandle(t *testing.T) {
	s := NewState("engo", "irc.example", "#engo")
	s.SetConnected(true)
	b, e := Handle([]byte("{\"pbmp\":1,\"type\":\"request\",\"id\":\"42\",\"method\":\"bot.info\",\"params\":{}}"), s)
	if e != nil {
		t.Fatal(e)
	}
	var v map[string]any
	if e = json.Unmarshal(b, &v); e != nil {
		t.Fatal(e)
	}
	if v["ok"] != true || v["id"] != "42" {
		t.Fatalf("bad response: %s", b)
	}
}
func TestUnknown(t *testing.T) {
	s := NewState("engo", "irc.example", "#engo")
	b, e := Handle([]byte("{\"pbmp\":1,\"type\":\"request\",\"id\":\"1\",\"method\":\"nope\",\"params\":{}}"), s)
	if e != nil {
		t.Fatal(e)
	}
	var v map[string]any
	json.Unmarshal(b, &v)
	if v["ok"] != false {
		t.Fatalf("expected error: %s", b)
	}
}

func TestChannelsList(t *testing.T) {
	s := NewState("engo", "irc.example", "#engo", "#ops")
	s.SetConnected(true)
	b, e := Handle([]byte("{\"pbmp\":1,\"type\":\"request\",\"id\":\"c\",\"method\":\"channels.list\",\"params\":{}}"), s)
	if e != nil {
		t.Fatal(e)
	}
	var v map[string]any
	if e = json.Unmarshal(b, &v); e != nil {
		t.Fatal(e)
	}
	result := v["result"].(map[string]any)
	channels := result["channels"].([]any)
	if len(channels) != 2 {
		t.Fatalf("channels=%v", channels)
	}
}

package pbmp

import (
	"encoding/json"
	"testing"
	"strings"
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

func TestDynamicChannelLifecycle(t *testing.T){s:=NewState("engo","irc.example","#configured");s.SetConnected(true);s.SetActions(func(string)error{return nil},func(string,string)error{return nil});b,e:=Handle([]byte("{\"pbmp\":1,\"type\":\"request\",\"id\":\"j\",\"method\":\"channels.join\",\"params\":{\"network\":\"irc.example\",\"name\":\"#Dyn[Ops]\"}}"),s);if e!=nil||!strings.Contains(string(b),"\"joining\""){t.Fatalf("join=%s err=%v",b,e)};s.Observe("JOIN","ENGO",[]string{"#dyn{ops}"},"");if s.ChannelState("#DYN[OPS]")!="joined"{t.Fatal("dynamic channel not joined")};b,_=Handle([]byte("{\"pbmp\":1,\"type\":\"request\",\"id\":\"l\",\"method\":\"channels.list\",\"params\":{}}"),s);if !strings.Contains(string(b),"#Dyn[Ops]"){t.Fatalf("dynamic missing: %s",b)};Handle([]byte("{\"pbmp\":1,\"type\":\"request\",\"id\":\"p\",\"method\":\"channels.part\",\"params\":{\"network\":\"irc.example\",\"name\":\"#dyn{ops}\"}}"),s);if s.ChannelState("#Dyn[Ops]")!="parting"{t.Fatalf("state=%s",s.ChannelState("#Dyn[Ops]"))};s.Observe("PART","engo",[]string{"#dyn{ops}"},"");if s.ChannelState("#Dyn[Ops]")!="joining"{t.Fatalf("post-part state=%s",s.ChannelState("#Dyn[Ops]"))}}

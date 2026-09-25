package config

import (
	"testing"
	"time"
)

func TestValidateSASLCredentialsAsPair(t *testing.T) {
	cfg := Config{
		Server:       "irc.example:6697",
		Nick:         "engo",
		User:         "engo",
		RealName:     "Engo",
		SASLUsername: "engo",
		ReconnectMin: time.Second,
		ReconnectMax: time.Minute,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for incomplete SASL credentials")
	}
}

func TestValidateReconnectRange(t *testing.T) {
	cfg := Config{
		Server:       "irc.example:6697",
		Nick:         "engo",
		User:         "engo",
		RealName:     "Engo",
		ReconnectMin: 10 * time.Second,
		ReconnectMax: time.Second,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for reconnect range")
	}
}

func TestValidateScriptCapabilities(t *testing.T){
	valid:=Config{ScriptMaxAllocs:1,HTTPTimeout:time.Second,HTTPMaxBody:1,ScriptCapabilitiesRaw:"weather.tengo:http"}
	if err:=valid.Validate();err!=nil{t.Fatalf("valid capabilities rejected: %v",err)}
	for _,raw:=range []string{"weather.tengo:shell","weather:http","weather.tengo:","weather.tengo"}{
		cfg:=valid;cfg.ScriptCapabilitiesRaw=raw
		if err:=cfg.Validate();err==nil{t.Fatalf("expected capability validation error for %q",raw)}
	}
}

func TestCapabilityEnv(t *testing.T){
	t.Setenv("ENGO_SCRIPT_CAPABILITIES","weather.tengo:http, feed.tengo:http")
	got:=capabilityEnv("ENGO_SCRIPT_CAPABILITIES")
	if len(got["weather.tengo"])!=1||got["weather.tengo"][0]!="http"{t.Fatalf("unexpected weather capabilities: %#v",got)}
	if len(got["feed.tengo"])!=1||got["feed.tengo"][0]!="http"{t.Fatalf("unexpected feed capabilities: %#v",got)}
}

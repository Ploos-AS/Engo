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

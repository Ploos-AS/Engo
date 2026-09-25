package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server string
	Nick string
	User string
	RealName string
	TLS bool
	Script string
	ScriptsDir string
	ScriptMaxAllocs int64\n\tStateDir string
	SASLUsername string
	SASLPassword string
	ReconnectMin time.Duration
	ReconnectMax time.Duration
}

func FromEnv() Config {
	return Config{
		Server: os.Getenv("ENGO_SERVER"),
		Nick: getenv("ENGO_NICK","engo"),
		User: getenv("ENGO_USER","engo"),
		RealName: getenv("ENGO_REALNAME","Engo IRC bot"),
		TLS: getenv("ENGO_TLS","1")!="0",
		Script: getenv("ENGO_SCRIPT","scripts/examples/hello.tengo"),
		ScriptsDir: os.Getenv("ENGO_SCRIPTS_DIR"),
		ScriptMaxAllocs: int64Env("ENGO_SCRIPT_MAX_ALLOCS",100000),\n\t\tStateDir: getenv("ENGO_STATE_DIR","data/state"),
		SASLUsername: os.Getenv("ENGO_SASL_USERNAME"),
		SASLPassword: os.Getenv("ENGO_SASL_PASSWORD"),
		ReconnectMin: durationEnv("ENGO_RECONNECT_MIN",2*time.Second),
		ReconnectMax: durationEnv("ENGO_RECONNECT_MAX",2*time.Minute),
	}
}

func (c Config) Validate() error {
	if c.ScriptMaxAllocs <= 0 { return fmt.Errorf("ENGO_SCRIPT_MAX_ALLOCS must be positive") }
	if c.Server=="" { return nil }
	if c.Nick=="" || c.User=="" || c.RealName=="" { return fmt.Errorf("nick, user and real name must not be empty") }
	if (c.SASLUsername=="")!=(c.SASLPassword=="") { return fmt.Errorf("ENGO_SASL_USERNAME and ENGO_SASL_PASSWORD must be set together") }
	if c.ReconnectMin<=0 || c.ReconnectMax<c.ReconnectMin { return fmt.Errorf("invalid reconnect interval") }
	return nil
}

func getenv(key,fallback string) string { if v:=os.Getenv(key); v!="" { return v }; return fallback }
func durationEnv(key string,fallback time.Duration) time.Duration {
	v:=os.Getenv(key); if v=="" { return fallback }
	if d,err:=time.ParseDuration(v); err==nil { return d }
	if seconds,err:=strconv.Atoi(v); err==nil { return time.Duration(seconds)*time.Second }
	return fallback
}
func int64Env(key string,fallback int64) int64 {
	v:=os.Getenv(key); if v=="" { return fallback }
	n,err:=strconv.ParseInt(v,10,64); if err!=nil { return fallback }
	return n
}

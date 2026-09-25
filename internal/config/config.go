package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	ScriptMaxAllocs int64
	StateDir string
	HTTPAllow []string
	HTTPTimeout time.Duration
	HTTPMaxBody int64
	ScriptCapabilities map[string][]string
	ScriptCapabilitiesRaw string
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
		ScriptMaxAllocs: int64Env("ENGO_SCRIPT_MAX_ALLOCS",100000),
		StateDir: getenv("ENGO_STATE_DIR","data/state"),
		HTTPAllow: csvEnv("ENGO_HTTP_ALLOW"),
		HTTPTimeout: durationEnv("ENGO_HTTP_TIMEOUT",10*time.Second),
		HTTPMaxBody: int64Env("ENGO_HTTP_MAX_BODY",262144),
		ScriptCapabilities: capabilityEnv("ENGO_SCRIPT_CAPABILITIES"),
		ScriptCapabilitiesRaw: os.Getenv("ENGO_SCRIPT_CAPABILITIES"),
		SASLUsername: os.Getenv("ENGO_SASL_USERNAME"),
		SASLPassword: os.Getenv("ENGO_SASL_PASSWORD"),
		ReconnectMin: durationEnv("ENGO_RECONNECT_MIN",2*time.Second),
		ReconnectMax: durationEnv("ENGO_RECONNECT_MAX",2*time.Minute),
	}
}

func (c Config) Validate() error {
	if err:=validateCapabilities(c.ScriptCapabilitiesRaw);err!=nil{return err}
	if c.ScriptMaxAllocs <= 0 { return fmt.Errorf("ENGO_SCRIPT_MAX_ALLOCS must be positive") }
	if c.HTTPTimeout<=0 || c.HTTPMaxBody<=0 { return fmt.Errorf("invalid HTTP capability limits") }
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

func csvEnv(key string)[]string{v:=strings.TrimSpace(os.Getenv(key));if v==""{return nil};parts:=strings.Split(v,",");out:=parts[:0];for _,p:=range parts{if p=strings.TrimSpace(p);p!=""{out=append(out,p)}};return out}

func capabilityEnv(key string)map[string][]string{
	out:=make(map[string][]string)
	for _,entry:=range csvEnv(key){
		parts:=strings.SplitN(entry,":",2)
		if len(parts)!=2{continue}
		name:=strings.TrimSpace(parts[0])
		if name==""{continue}
		for _,capability:=range strings.Split(parts[1],"+"){
			if capability=strings.ToLower(strings.TrimSpace(capability));capability!=""{out[name]=append(out[name],capability)}
		}
	}
	return out
}

func validateCapabilities(raw string)error{
	raw=strings.TrimSpace(raw);if raw==""{return nil}
	for _,entry:=range strings.Split(raw,","){
		entry=strings.TrimSpace(entry);parts:=strings.SplitN(entry,":",2)
		if len(parts)!=2||strings.TrimSpace(parts[0])==""||strings.TrimSpace(parts[1])==""{return fmt.Errorf("invalid ENGO_SCRIPT_CAPABILITIES entry %q",entry)}
		if !strings.HasSuffix(strings.TrimSpace(parts[0]),".tengo"){return fmt.Errorf("invalid script name in ENGO_SCRIPT_CAPABILITIES: %q",parts[0])}
		for _,capability:=range strings.Split(parts[1],"+"){
			capability=strings.ToLower(strings.TrimSpace(capability))
			if capability==""{return fmt.Errorf("empty capability in ENGO_SCRIPT_CAPABILITIES entry %q",entry)}
			if capability!="http"{return fmt.Errorf("unknown script capability %q",capability)}
		}
	}
	return nil
}

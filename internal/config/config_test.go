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

func TestValidateSASLRequiresTLS(t *testing.T){
	base:=Config{Server:"irc.example:6667",Nick:"engo",User:"engo",RealName:"Engo",ScriptMaxAllocs:1,HTTPTimeout:time.Second,HTTPMaxBody:1,SASLUsername:"engo",SASLPassword:"secret",ReconnectMin:time.Second,ReconnectMax:time.Minute}
	if err:=base.Validate();err==nil{t.Fatal("expected insecure SASL rejection")}
	base.AllowInsecureSASL=true
	if err:=base.Validate();err!=nil{t.Fatalf("explicit insecure SASL override rejected: %v",err)}
	base.AllowInsecureSASL=false;base.TLS=true
	if err:=base.Validate();err!=nil{t.Fatalf("TLS SASL rejected: %v",err)}
}

func TestValidateIRCCapabilities(t *testing.T){
	for _,capability:=range []string{"account-notify","extended-join","server-time"}{
		if err:=validateIRCCapability(capability);err!=nil{t.Fatalf("%s rejected: %v",capability,err)}
	}
	for _,capability:=range []string{"sasl","echo-message","unknown"}{
		if err:=validateIRCCapability(capability);err==nil{t.Fatalf("expected %s to be rejected",capability)}
	}
}

func TestAccountPermissionsEnvAndValidation(t *testing.T){
	t.Setenv("ENGO_ACCOUNT_PERMISSIONS","alice:admin+operator,bob:operator")
	cfg:=FromEnv()
	if len(cfg.AccountPermissions["alice"])!=2||cfg.AccountPermissions["alice"][0]!="admin"||cfg.AccountPermissions["alice"][1]!="operator"{t.Fatalf("unexpected alice permissions: %#v",cfg.AccountPermissions)}
	if err:=validatePermissions(cfg.AccountPermissionsRaw);err!=nil{t.Fatalf("valid permissions rejected: %v",err)}
	for _,raw:=range []string{"alice","alice:"," :admin","alice:admin+"}{if err:=validatePermissions(raw);err==nil{t.Fatalf("expected invalid permissions for %q",raw)}}
}

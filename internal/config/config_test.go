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
	for _,capability:=range []string{"account-notify","account-tag","extended-join","server-time"}{
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

func TestCommandPermissionsEnvAndValidation(t *testing.T){
	t.Setenv("ENGO_COMMAND_PERMISSIONS","reload:admin,kick:operator")
	cfg:=FromEnv()
	if cfg.CommandPermissions["reload"]!="admin"||cfg.CommandPermissions["kick"]!="operator"{t.Fatalf("unexpected command permissions: %#v",cfg.CommandPermissions)}
	if err:=validateCommandPermissions(cfg.CommandPermissionsRaw);err!=nil{t.Fatalf("valid command permissions rejected: %v",err)}
	for _,raw:=range []string{"reload","reload:"," :admin","reload:admin+operator"}{if err:=validateCommandPermissions(raw);err==nil{t.Fatalf("expected invalid command permission for %q",raw)}}
}

func TestPermissionNameValidation(t *testing.T){
	for _,name:=range []string{"admin","irc.operator","script.reload","moderation-kick","ops_1"}{if err:=validatePermissionName(name);err!=nil{t.Fatalf("valid permission %q rejected: %v",name,err)}}
	for _,name:=range []string{"","Admin",".admin","admin.","admin..reload","admin reload","admin:reload","ådmín"}{if err:=validatePermissionName(name);err==nil{t.Fatalf("invalid permission %q accepted",name)}}
}

func TestAuthorizationIdentifierValidation(t *testing.T){
	for _,name:=range []string{"alice","Alice.Account","services-account","user_1"}{if err:=validateAccountName(name);err!=nil{t.Fatalf("valid account %q rejected: %v",name,err)}}
	for _,name:=range []string{"","alice account","alice:admin","alice/account"}{if err:=validateAccountName(name);err==nil{t.Fatalf("invalid account %q accepted",name)}}
	for _,name:=range []string{"reload","op-kick","script_1"}{if err:=validateCommandName(name);err!=nil{t.Fatalf("valid command %q rejected: %v",name,err)}}
	for _,name:=range []string{"","Reload","op.kick","op kick","op:kick"}{if err:=validateCommandName(name);err==nil{t.Fatalf("invalid command %q accepted",name)}}
}


func TestScriptCapabilityNamesRejectTraversal(t *testing.T){
	bad:=[]string{"../weather.tengo:http","dir/weather.tengo:http","dir\\\\weather.tengo:http","..tengo:http","weather..prod.tengo:http","weather$.tengo:http"}
	for _,raw:=range bad{if err:=validateScriptCapabilityNames(raw);err==nil{t.Fatalf("expected rejection for %q",raw)}}
	good:=[]string{"weather.tengo:http","weather-prod_1.tengo:http","weather.prod.tengo:http"}
	for _,raw:=range good{if err:=validateScriptCapabilityNames(raw);err!=nil{t.Fatalf("valid name %q rejected: %v",raw,err)}}
}


func TestPermissionPolicyRejectsDuplicateAccounts(t *testing.T){
	for _,raw:=range []string{"alice:admin,alice:operator","Alice:admin,alice:operator"}{if err:=validatePermissions(raw);err==nil{t.Fatalf("expected duplicate account rejection for %q",raw)}}
}

func TestCommandPolicyRejectsDuplicateCommands(t *testing.T){
	for _,raw:=range []string{"reload:admin,reload:operator","kick:operator,kick:admin"}{if err:=validateCommandPermissions(raw);err==nil{t.Fatalf("expected duplicate command rejection for %q",raw)}}
}


func TestCommandPermissionReferencesMustBeDeclared(t *testing.T){
	accounts:=map[string][]string{"alice":{"admin","operator"}}
	if err:=validatePermissionReferences(accounts,map[string]string{"reload":"admin","kick":"operator"});err!=nil{t.Fatalf("declared permission rejected: %v",err)}
	if err:=validatePermissionReferences(accounts,map[string]string{"reload":"admn"});err==nil{t.Fatal("expected undeclared permission rejection")}
	if err:=validatePermissionReferences(nil,map[string]string{"reload":"admin"});err==nil{t.Fatal("expected permission rejection without account grants")}
}


func TestScriptCapabilitiesRejectDuplicates(t *testing.T){
	bad:=[]string{"weather.tengo:http,weather.tengo:http","weather.tengo:http+http"}
	for _,raw:=range bad{if err:=validateCapabilities(raw);err==nil{t.Fatalf("expected duplicate capability policy rejection for %q",raw)}}
	if err:=validateCapabilities("weather.tengo:http,alerts.tengo:http");err!=nil{t.Fatalf("distinct script grants rejected: %v",err)}
}


func TestIRCCapabilitiesRejectDuplicates(t *testing.T){
	c:=Config{Server:"irc.example:6697",Nick:"engo",User:"engo",RealName:"Engo",TLS:true,ScriptMaxAllocs:1,HTTPTimeout:time.Second,HTTPMaxBody:1,ReconnectMin:time.Second,ReconnectMax:2*time.Second,IRCCapabilities:[]string{"account-tag","ACCOUNT-TAG"}}
	if err:=c.Validate();err==nil{t.Fatal("expected duplicate IRC capability rejection")}
}


func TestIRCCapabilitiesEnvNormalizesCase(t *testing.T){
	t.Setenv("ENGO_IRC_CAPABILITIES"," ACCOUNT-TAG,Server-Time ")
	cfg:=FromEnv()
	if len(cfg.IRCCapabilities)!=2||cfg.IRCCapabilities[0]!="account-tag"||cfg.IRCCapabilities[1]!="server-time"{t.Fatalf("IRC capabilities not normalized: %#v",cfg.IRCCapabilities)}
}


func TestPermissionPolicyRejectsDuplicatePermissionsPerAccount(t *testing.T) {
	for _, raw := range []string{"alice:admin+admin", "alice:Admin+admin"} {
		if err := validatePermissions(raw); err == nil {
			t.Fatalf("expected duplicate permission rejection for %q", raw)
		}
	}
}


func TestPermissionEnvNormalizesCase(t *testing.T) {
	t.Setenv("ENGO_ACCOUNT_PERMISSIONS", " Alice:Admin+Operator ")
	cfg := FromEnv()
	got := cfg.AccountPermissions["alice"]
	if len(got) != 2 || got[0] != "admin" || got[1] != "operator" {
		t.Fatalf("account permissions not normalized: %#v", cfg.AccountPermissions)
	}
}

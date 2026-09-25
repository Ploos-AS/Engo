package irc

import (
	"strings"
	"testing"
)

func TestCapabilityListed(t *testing.T) {
	line := ":irc.example CAP engo LS :multi-prefix sasl=PLAIN account-notify"
	if !capabilityListed(line, "sasl") {
		t.Fatal("expected sasl capability")
	}
	if capabilityListed(line, "echo-message") {
		t.Fatal("unexpected echo-message capability")
	}
}

func TestNumericCommand(t *testing.T) {
	got := numericCommand([]string{":irc.example", "903", "engo", ":SASL authentication successful"})
	if got != "903" {
		t.Fatalf("got %q, want 903", got)
	}
}

func TestMultilineCapabilityCollection(t *testing.T){
	first:=[]string{":irc.example","CAP","engo","LS","*",":multi-prefix","account-notify"}
	if !capLSContinues(first){t.Fatal("expected CAP LS continuation")}
	caps:=append(capabilityNames(":irc.example CAP engo LS * :multi-prefix account-notify"),capabilityNames(":irc.example CAP engo LS :sasl=PLAIN echo-message")...)
	if !containsCapability(caps,"sasl"){t.Fatal("expected SASL from final CAP LS line")}
	if !containsCapability(caps,"multi-prefix"){t.Fatal("expected capability from continuation line")}
}

func TestAuthenticateChunks(t *testing.T){
	cases:=[]struct{name string;n int;want int;terminal bool}{
		{"short",10,1,false},
		{"exact",400,2,true},
		{"long",401,2,false},
		{"double-exact",800,3,true},
	}
	for _,tc:=range cases{t.Run(tc.name,func(t *testing.T){
		chunks:=authenticateChunks(strings.Repeat("x",tc.n))
		if len(chunks)!=tc.want{t.Fatalf("got %d chunks, want %d",len(chunks),tc.want)}
		for i,ch:=range chunks{if ch!="+"&&len(ch)>400{t.Fatalf("chunk %d too long: %d",i,len(ch))}}
		if (chunks[len(chunks)-1]=="+")!=tc.terminal{t.Fatalf("terminal marker mismatch: %#v",chunks)}
	})}
}

func TestCapabilityNamesFromNAK(t *testing.T){
	caps:=capabilityNames(":irc.example CAP engo NAK :sasl echo-message")
	if !containsCapability(caps,"sasl")||!containsCapability(caps,"echo-message"){t.Fatalf("unexpected NAK capabilities: %#v",caps)}
}

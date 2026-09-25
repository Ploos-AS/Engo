package irc

import "testing"

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

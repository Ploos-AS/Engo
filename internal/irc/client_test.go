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

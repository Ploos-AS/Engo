package script

import "testing"

func TestCapabilitiesRequireHTTP(t *testing.T) {
	if err := (Capabilities{}).Require("http"); err == nil {
		t.Fatal("expected denied HTTP capability")
	}
	if err := (Capabilities{HTTP: true}).Require("http"); err != nil {
		t.Fatalf("expected HTTP capability to be granted: %v", err)
	}
}

func TestCapabilitiesRequireRejectsUnknown(t *testing.T) {
	if err := (Capabilities{HTTP: true}).Require("shell"); err == nil {
		t.Fatal("expected unknown capability error")
	}
}

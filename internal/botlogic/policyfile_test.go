package botlogic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePolicy(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "policy.pl")
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestLoadPolicyFileAcceptsMayExecute(t *testing.T) {
	p := writePolicy(t, "may_execute(A,C) :- authenticated(_,A), allowed(C).\n")
	got, err := LoadPolicyFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "may_execute(") {
		t.Fatalf("source=%q", got)
	}
}
func TestLoadPolicyFileRejectsMissingContract(t *testing.T) {
	p := writePolicy(t, "allowed(reload).\n")
	if _, err := LoadPolicyFile(p); err == nil {
		t.Fatal("expected missing may_execute/2 error")
	}
}
func TestLoadPolicyFileRejectsEmptyAndOversize(t *testing.T) {
	if _, err := LoadPolicyFile(writePolicy(t, "")); err == nil {
		t.Fatal("expected empty error")
	}
	if _, err := LoadPolicyFile(writePolicy(t, strings.Repeat("x", (64<<10)+1))); err == nil {
		t.Fatal("expected size error")
	}
}

package script

import (
	"strings"
	"testing"
)

func TestStorePersists(t *testing.T) {
	root := t.TempDir()
	if err := NewStore(root, "one").Set("key", "value"); err != nil { t.Fatal(err) }
	got, ok, err := NewStore(root, "one").Get("key")
	if err != nil || !ok || got != "value" { t.Fatalf("unexpected persisted value") }
}

func TestStoreNamespaces(t *testing.T) {
	root := t.TempDir()
	if err := NewStore(root, "one").Set("key", "value"); err != nil { t.Fatal(err) }
	_, ok, err := NewStore(root, "two").Get("key")
	if err != nil || ok { t.Fatalf("namespace isolation failed") }
}

func TestStoreValueLimit(t *testing.T) {
	if err := NewStore(t.TempDir(), "one").Set("key", strings.Repeat("x", 65537)); err == nil { t.Fatal("expected size error") }
}

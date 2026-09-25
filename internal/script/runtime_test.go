package script

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tengo")
	if err := os.WriteFile(path, []byte(`fmt := import("fmt"); fmt.sprintf("%s", "ok")`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RunFile(path); err != nil {
		t.Fatalf("RunFile() error = %v", err)
	}
}

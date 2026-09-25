package script

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ploos-AS/Engo/internal/bot"
	"github.com/Ploos-AS/Engo/internal/irc"
)

type captureSender struct{ target, text string }
func (s *captureSender) Say(target, text string) error { s.target, s.text = target, text; return nil }
func (s *captureSender) Notice(string, string) error { return nil }
func (s *captureSender) Action(string, string) error { return nil }

func TestRunFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tengo")
	if err := os.WriteFile(path, []byte(`x := 1 + 1`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RunFile(path); err != nil {
		t.Fatalf("RunFile() error = %v", err)
	}
}

func TestTengoCommandCallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "command.tengo")
	src := `bot("command", "hello", "hello")
if bot("active", "hello") {
	bot("say", event["target"], "Hello " + event["nick"])
}`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	sender := &captureSender{}
	b := bot.New(sender)
	rt := New(path, b)
	if err := rt.Load(); err != nil {
		t.Fatal(err)
	}
	if err := b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!hello")); err != nil {
		t.Fatal(err)
	}
	if sender.target != "#engo" || sender.text != "Hello alice" {
		t.Fatalf("unexpected reply: %#v", sender)
	}
}

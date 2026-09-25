package bot

import (
	"testing"

	"github.com/Ploos-AS/Engo/internal/irc"
)

type testSender struct{ target, text string }
func (s *testSender) Say(target, text string) error { s.target, s.text = target, text; return nil }
func (s *testSender) Notice(string, string) error { return nil }
func (s *testSender) Action(string, string) error { return nil }

func TestCommandDispatch(t *testing.T) {
	s := &testSender{}
	b := New(s)
	called := false
	b.Command("hello", func(ev Event) error {
		called = true
		if ev.Nick != "alice" || len(ev.Args) != 1 || ev.Args[0] != "world" {
			t.Fatalf("unexpected event: %#v", ev)
		}
		return b.Say(ev.Target, "hi "+ev.Nick)
	})
	if err := b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!hello world")); err != nil {
		t.Fatal(err)
	}
	if !called || s.target != "#engo" || s.text != "hi alice" {
		t.Fatalf("command not dispatched correctly: %#v", s)
	}
}

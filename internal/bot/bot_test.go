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


func TestAccountTagOverridesCachedIdentity(t *testing.T){
	b:=New(&testSender{})
	var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT stale-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage("@account=fresh-account :alice!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!="fresh-account"{t.Fatalf("account-tag did not override cache: %q",got)}
	got=""
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!="fresh-account"{t.Fatalf("authoritative account-tag was not cached: %q",got)}
}

func TestExtendedJoinLogoutClearsCachedIdentity(t *testing.T){
	b:=New(&testSender{})
	var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT stale-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example JOIN #engo * :Alice Example"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!=""{t.Fatalf("JOIN * left stale account identity: %q",got)}
}

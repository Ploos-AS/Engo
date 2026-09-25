package irc

import "testing"

func TestParsePrivmsg(t *testing.T) {
	m := ParseMessage(":alice!u@example PRIVMSG #engo :!hello world")
	if m.Nick != "alice" || m.Command != "PRIVMSG" || m.Target() != "#engo" || m.Trailing != "!hello world" {
		t.Fatalf("unexpected message: %#v", m)
	}
}

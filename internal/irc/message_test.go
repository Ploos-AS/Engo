package irc

import "testing"

func TestParsePrivmsg(t *testing.T) {
	m := ParseMessage(":alice!u@example PRIVMSG #engo :!hello world")
	if m.Nick != "alice" || m.Command != "PRIVMSG" || m.Target() != "#engo" || m.Trailing != "!hello world" {
		t.Fatalf("unexpected message: %#v", m)
	}
}

func TestParseMessageTags(t *testing.T) {
	m := ParseMessage("@time=2026-09-25T09:30:00.000Z;account=alice\\suser :alice!u@example PRIVMSG #engo :hello")
	if m.Tags["time"] != "2026-09-25T09:30:00.000Z" {
		t.Fatalf("unexpected time tag: %q", m.Tags["time"])
	}
	if m.Tags["account"] != "alice user" {
		t.Fatalf("unexpected account tag: %q", m.Tags["account"])
	}
	if m.Nick != "alice" || m.Command != "PRIVMSG" || m.Trailing != "hello" {
		t.Fatalf("unexpected message: %#v", m)
	}
}

func TestUnescapeTag(t *testing.T) {
	got := unescapeTag("one\\stwo\\:three\\\\four\\r\\nfive")
	want := "one two;three\\four\r\nfive"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestParseAccountNotify(t *testing.T) {
	m := ParseMessage(":alice!u@example ACCOUNT services-account")
	if m.Nick != "alice" || m.Command != "ACCOUNT" || len(m.Params) != 1 || m.Params[0] != "services-account" {
		t.Fatalf("unexpected ACCOUNT message: %#v", m)
	}
}

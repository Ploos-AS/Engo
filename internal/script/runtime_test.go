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

func writeScript(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil { t.Fatal(err) }
}

func TestRunFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.tengo")
	writeScript(t, path, `x := 1 + 1`)
	if err := RunFile(path); err != nil { t.Fatalf("RunFile() error = %v", err) }
}

func TestTengoCommandCallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "command.tengo")
	writeScript(t, path, `bot("command","hello","hello")
if bot("active","hello") { bot("say",event["target"],"Hello "+event["nick"]) }`)
	sender:=&captureSender{}; b:=bot.New(sender); rt:=New(path,b)
	if err:=rt.Load(); err!=nil { t.Fatal(err) }
	_ = b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!hello"))
	if sender.target!="#engo" || sender.text!="Hello alice" { t.Fatalf("unexpected reply: %#v",sender) }
}

func TestReloadKeepsPreviousScriptOnFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reload.tengo")
	good := `bot("command","hello","hello")
if bot("active","hello") { bot("say",event["target"],"v1") }`
	writeScript(t,path,good)
	sender:=&captureSender{}; b:=bot.New(sender); rt:=New(path,b)
	if err:=rt.Load(); err!=nil { t.Fatal(err) }

	writeScript(t,path,`this is not valid tengo {{{`)
	if err:=rt.Reload(); err==nil { t.Fatal("expected reload failure") }
	sender.text=""
	_ = b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!hello"))
	if sender.text!="v1" { t.Fatalf("previous script was not preserved: %#v",sender) }
}

func TestReloadReplacesHandlers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reload.tengo")
	v1 := `bot("command","hello","hello")
if bot("active","hello") { bot("say",event["target"],"v1") }`
	v2 := `bot("command","hello","hello")
if bot("active","hello") { bot("say",event["target"],"v2") }`
	writeScript(t,path,v1)
	sender:=&captureSender{}; b:=bot.New(sender); rt:=New(path,b)
	if err:=rt.Load(); err!=nil { t.Fatal(err) }
	writeScript(t,path,v2)
	if err:=rt.Reload(); err!=nil { t.Fatal(err) }
	_ = b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!hello"))
	if sender.text!="v2" { t.Fatalf("new script not active: %#v",sender) }
}

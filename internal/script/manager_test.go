package script

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ploos-AS/Engo/internal/bot"
	"github.com/Ploos-AS/Engo/internal/irc"
)

func TestManagerLoadsMultipleScripts(t *testing.T) {
	dir:=t.TempDir()
	writeScript(t,filepath.Join(dir,"one.tengo"),`bot("command","one","one")
if bot("active","one"){bot("say",event["target"],"one")}`)
	writeScript(t,filepath.Join(dir,"two.tengo"),`bot("command","two","two")
if bot("active","two"){bot("say",event["target"],"two")}`)
	s:=&captureSender{}; b:=bot.New(s); m:=NewManager(dir,b)
	if err:=m.ReloadAll(); err!=nil { t.Fatal(err) }
	if got:=m.Scripts(); len(got)!=2 { t.Fatalf("scripts=%v",got) }
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!two"))
	if s.text!="two" { t.Fatalf("unexpected reply %q",s.text) }
}

func TestManagerRollbackWhenOneScriptFails(t *testing.T) {
	dir:=t.TempDir()
	path:=filepath.Join(dir,"good.tengo")
	writeScript(t,path,`bot("command","ok","ok")
if bot("active","ok"){bot("say",event["target"],"old")}`)
	s:=&captureSender{}; b:=bot.New(s); m:=NewManager(dir,b)
	if err:=m.ReloadAll(); err!=nil { t.Fatal(err) }
	if err:=os.WriteFile(filepath.Join(dir,"broken.tengo"),[]byte("{{ invalid"),0o600); err!=nil { t.Fatal(err) }
	if err:=m.ReloadAll(); err==nil { t.Fatal("expected reload failure") }
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!ok"))
	if s.text!="old" { t.Fatalf("old registry not preserved: %q",s.text) }
}


func TestManagerDisableAndEnableScript(t *testing.T) {
	dir:=t.TempDir()
	writeScript(t,filepath.Join(dir,"one.tengo"),`bot("command","one","one")
if bot("active","one"){bot("say",event["target"],"one")}`)
	writeScript(t,filepath.Join(dir,"two.tengo"),`bot("command","two","two")
if bot("active","two"){bot("say",event["target"],"two")}`)
	s:=&captureSender{};b:=bot.New(s);m:=NewManager(dir,b)
	if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
	if err:=m.Disable("two.tengo");err!=nil{t.Fatal(err)}
	s.text=""
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!two"))
	if s.text!=""{t.Fatalf("disabled script still handled command: %q",s.text)}
	if got:=m.Disabled();len(got)!=1||got[0]!="two.tengo"{t.Fatalf("disabled=%v",got)}
	if err:=m.Enable("two.tengo");err!=nil{t.Fatal(err)}
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!two"))
	if s.text!="two"{t.Fatalf("enabled script did not return: %q",s.text)}
}

func TestManagerRejectsPathTraversal(t *testing.T) {
	m:=NewManager(t.TempDir(),bot.New(&captureSender{}))
	if err:=m.Disable("../outside.tengo");err==nil{t.Fatal("expected invalid script name")}
}

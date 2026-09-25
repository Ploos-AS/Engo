package script

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
	if _,text:=s.snapshot();text!="two" { t.Fatalf("unexpected reply %q",text) }
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
	if _,text:=s.snapshot();text!="old" { t.Fatalf("old registry not preserved: %q",text) }
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
	s.reset()
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!two"))
	if _,text:=s.snapshot();text!=""{t.Fatalf("disabled script still handled command: %q",text)}
	if got:=m.Disabled();len(got)!=1||got[0]!="two.tengo"{t.Fatalf("disabled=%v",got)}
	if err:=m.Enable("two.tengo");err!=nil{t.Fatal(err)}
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!two"))
	if _,text:=s.snapshot();text!="two"{t.Fatalf("enabled script did not return: %q",text)}
}

func TestManagerRejectsPathTraversal(t *testing.T) {
	m:=NewManager(t.TempDir(),bot.New(&captureSender{}))
	if err:=m.Disable("../outside.tengo");err==nil{t.Fatal("expected invalid script name")}
}

func TestManagerFailedDisableRollsBackState(t *testing.T){
	dir:=t.TempDir()
	one:=filepath.Join(dir,"one.tengo")
	two:=filepath.Join(dir,"two.tengo")
	writeScript(t,one,`bot("command","one","one")
if bot("active","one"){bot("say",event["target"],"old")}`)
	writeScript(t,two,`bot("command","two","two")`)
	s:=&captureSender{};b:=bot.New(s);m:=NewManager(dir,b)
	if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(one,[]byte("{{ invalid"),0o600);err!=nil{t.Fatal(err)}
	if err:=m.Disable("two.tengo");err==nil{t.Fatal("expected disable reload failure")}
	if got:=m.Disabled();len(got)!=0{t.Fatalf("disabled state not rolled back: %v",got)}
	s.reset()
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!one"))
	if _,text:=s.snapshot();text!="old"{t.Fatalf("old registry not preserved after failed disable: %q",text)}
}

func TestManagerFailedEnableRollsBackState(t *testing.T){
	dir:=t.TempDir()
	one:=filepath.Join(dir,"one.tengo")
	two:=filepath.Join(dir,"two.tengo")
	writeScript(t,one,`bot("command","one","one")
if bot("active","one"){bot("say",event["target"],"old")}`)
	writeScript(t,two,`bot("command","two","two")`)
	s:=&captureSender{};b:=bot.New(s);m:=NewManager(dir,b)
	if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
	if err:=m.Disable("two.tengo");err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(one,[]byte("{{ invalid"),0o600);err!=nil{t.Fatal(err)}
	if err:=m.Enable("two.tengo");err==nil{t.Fatal("expected enable reload failure")}
	if got:=m.Disabled();len(got)!=1||got[0]!="two.tengo"{t.Fatalf("disabled state not rolled back: %v",got)}
}

func TestManagerFailedReloadKeepsOldTimers(t *testing.T){
	dir:=t.TempDir()
	path:=filepath.Join(dir,"timer.tengo")
	writeScript(t,path,`bot("command","start","start")
if bot("active","start"){bot("timer_after","1s","fired")}
if bot("active","fired"){bot("say",event["target"],"timer-fired")}`)
	s:=&captureSender{};b:=bot.New(s);m:=NewManager(dir,b)
	if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!start"))
	if err:=os.WriteFile(path,[]byte("{{ invalid"),0o600);err!=nil{t.Fatal(err)}
	if err:=m.ReloadAll();err==nil{t.Fatal("expected reload failure")}
	deadline:=time.Now().Add(2*time.Second)
	for time.Now().Before(deadline){
		if _,text:=s.snapshot();text=="timer-fired"{return}
		time.Sleep(20*time.Millisecond)
	}
	t.Fatal("old generation timer was lost after failed reload")
}

func TestManagerSuccessfulReloadCancelsOldTimers(t *testing.T){
	dir:=t.TempDir()
	path:=filepath.Join(dir,"timer.tengo")
	writeScript(t,path,`bot("command","start","start")
if bot("active","start"){bot("timer_after","1s","old-fired")}
if bot("active","old-fired"){bot("say",event["target"],"old-timer-fired")}`)
	s:=&captureSender{};b:=bot.New(s);m:=NewManager(dir,b)
	if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!start"))
	writeScript(t,path,`bot("command","new","new")
if bot("active","new"){bot("say",event["target"],"new-generation")}`)
	if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
	s.reset()
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!new"))
	if _,text:=s.snapshot();text!="new-generation"{t.Fatalf("new registry not active: %q",text)}
	s.reset()
	time.Sleep(1200*time.Millisecond)
	if _,text:=s.snapshot();text!=""{t.Fatalf("old generation timer fired after successful reload: %q",text)}
}

func TestManagerStorePersistsAcrossSuccessfulReload(t *testing.T){
	dir:=t.TempDir()
	stateDir:=t.TempDir()
	path:=filepath.Join(dir,"state.tengo")
	writeScript(t,path,`bot("command","set","set")
if bot("active","set"){kv_set("value","persisted")}`)
	s:=&captureSender{};b:=bot.New(s);m:=NewManagerWithState(dir,b,100000,stateDir)
	if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!set"))
	writeScript(t,path,`bot("command","get","get")
if bot("active","get"){
	v := kv_get("value")
	bot("say",event["target"],v)
}`)
	if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
	s.reset()
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!get"))
	if _,text:=s.snapshot();text!="persisted"{t.Fatalf("store state did not survive reload: %q",text)}
}

func TestManagerStorePersistsAcrossDisableEnable(t *testing.T){
	dir:=t.TempDir()
	stateDir:=t.TempDir()
	path:=filepath.Join(dir,"state.tengo")
	writeScript(t,path,`bot("command","set","set")
bot("command","get","get")
if bot("active","set"){kv_set("value","persisted")}
if bot("active","get"){
	v := kv_get("value")
	bot("say",event["target"],v)
}`)
	s:=&captureSender{};b:=bot.New(s);m:=NewManagerWithState(dir,b,100000,stateDir)
	if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!set"))
	if err:=m.Disable("state.tengo");err!=nil{t.Fatal(err)}
	s.reset()
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!get"))
	if s.text!=""{t.Fatalf("disabled script still handled command: %q",s.text)}
	if err:=m.Enable("state.tengo");err!=nil{t.Fatal(err)}
	_ = b.Handle(irc.ParseMessage(":a!u@h PRIVMSG #x :!get"))
	if _,text:=s.snapshot();text!="persisted"{t.Fatalf("store state did not survive disable/enable: %q",text)}
}


func TestManagerConcurrentPolicyUpdateAndReload(t *testing.T){
 dir:=t.TempDir()
 writeScript(t,filepath.Join(dir,"policy.tengo"),`bot("command","reload","reload")
if bot("active","reload"){bot("say",event["target"],"ran")}`)
 b:=bot.New(&captureSender{});m:=NewManager(dir,b)
 m.SetPermissions(map[string][]string{"alice":{"admin"}})
 m.SetCommandPermissions(map[string]string{"reload":"admin"})
 if err:=m.ReloadAll();err!=nil{t.Fatal(err)}
 done:=make(chan struct{})
 go func(){defer close(done);for i:=0;i<100;i++{m.SetPermissions(map[string][]string{"alice":{"admin"}});m.SetCommandPermissions(map[string]string{"reload":"admin"})}}()
 for i:=0;i<25;i++{if err:=m.ReloadAll();err!=nil{t.Fatal(err)}}
 <-done
}


func TestManagerCopiesPermissionPolicyOnAssignment(t *testing.T){
 b:=bot.New(&captureSender{});m:=NewManager(t.TempDir(),b)
 permissions:=map[string][]string{"alice":{"admin"}}
 commands:=map[string]string{"reload":"admin"}
 m.SetPermissions(permissions);m.SetCommandPermissions(commands)
 permissions["alice"][0]="mutated";permissions["bob"]=[]string{"admin"};commands["reload"]="mutated"
 m.mu.RLock();defer m.mu.RUnlock()
 if got:=m.permissions["alice"][0];got!="admin"{t.Fatalf("permission policy aliased caller data: %q",got)}
 if _,ok:=m.permissions["bob"];ok{t.Fatal("permission map aliased caller data")}
 if got:=m.commandPermissions["reload"];got!="admin"{t.Fatalf("command policy aliased caller data: %q",got)}
}


func TestManagerDisableRejectsMissingScript(t *testing.T) {
	m := NewManager(t.TempDir(), bot.New(&captureSender{}))
	if err := m.Disable("missing.tengo"); err == nil {
		t.Fatal("expected missing script error")
	}
	if disabled := m.Disabled(); len(disabled) != 0 {
		t.Fatalf("missing script changed disabled state: %v", disabled)
	}
}

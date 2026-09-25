package script

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestHTTPRequiresCapability(t *testing.T){
	path:=filepath.Join(t.TempDir(),"http.tengo")
	writeScript(t,path,`bot("command","fetch","fetch")
if bot("active","fetch") { bot("http_get","https://example.com/") }`)
	b:=bot.New(&captureSender{});rt:=New(path,b)
	rt.SetHTTP(NewHTTPClient([]string{"example.com"},time.Second,1024))
	if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!fetch"));err==nil{t.Fatal("expected HTTP capability denial")}
}

func TestHTTPCapabilityGrantReachesHTTPPolicy(t *testing.T){
	path:=filepath.Join(t.TempDir(),"http-grant.tengo")
	writeScript(t,path,`bot("command","fetch","fetch")
if bot("active","fetch") { bot("http_get","http://example.com/") }`)
	b:=bot.New(&captureSender{});rt:=New(path,b)
	rt.SetHTTP(NewHTTPClient([]string{"example.com"},time.Second,1024))
	rt.SetCapabilities(Capabilities{HTTP:true})
	if err:=rt.Load();err!=nil{t.Fatal(err)}
	err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!fetch"))
	if err==nil{t.Fatal("expected HTTPS policy rejection")}
	if strings.Contains(err.Error(),"not granted"){t.Fatalf("capability grant was not applied: %v",err)}
	if !strings.Contains(err.Error(),"requires https"){t.Fatalf("expected HTTP policy rejection after capability grant, got: %v",err)}
}

func TestEventObjectExposesIRCv3Tags(t *testing.T){
	ev:=bot.Event{Name:"message",Nick:"alice",Message:irc.ParseMessage("@time=2026-09-25T09:30:00.000Z;account=alice :alice!u@example PRIVMSG #engo :hello")}
	obj:=eventObject(ev)
	if obj["time"]!="2026-09-25T09:30:00.000Z"{t.Fatalf("unexpected time: %#v",obj["time"])}
	if obj["account"]!="alice"{t.Fatalf("unexpected account: %#v",obj["account"])}
	tags,ok:=obj["tags"].(map[string]interface{});if !ok{t.Fatalf("unexpected tags type: %T",obj["tags"])}
	if tags["time"]!="2026-09-25T09:30:00.000Z"||tags["account"]!="alice"{t.Fatalf("unexpected tags: %#v",tags)}
}

func TestExtendedJoinIdentityInEvent(t *testing.T){
	m:=irc.ParseMessage(":alice!u@example JOIN #engo alice :Alice Example")
	ev:=bot.Event{Name:"join",Nick:m.Nick,Target:m.Target(),Account:m.Params[1],RealName:m.Trailing,Message:m}
	obj:=eventObject(ev)
	if obj["account"]!="alice"{t.Fatalf("unexpected account: %#v",obj["account"])}
	if obj["realname"]!="Alice Example"{t.Fatalf("unexpected realname: %#v",obj["realname"])}
}

func TestAccountNotifyEventObject(t *testing.T){
	for _,tc:=range []struct{line,want string}{
		{":alice!u@example ACCOUNT services-account","services-account"},
		{":alice!u@example ACCOUNT *",""},
	}{
		m:=irc.ParseMessage(tc.line)
		account:="";if len(m.Params)>0&&m.Params[0]!="*"{account=m.Params[0]}
		obj:=eventObject(bot.Event{Name:"account",Nick:m.Nick,Account:account,Message:m})
		if obj["account"]!=tc.want{t.Fatalf("%q: account=%#v, want %q",tc.line,obj["account"],tc.want)}
	}
}

func TestAccountNotifyIdentityPersistsToMessages(t *testing.T){
	path:=filepath.Join(t.TempDir(),"account.tengo")
	writeScript(t,path,`bot("command","who","who")
if bot("active","who") { bot("say",event["target"],event["account"]) }`)
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b)
	if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if sender.text!="alice-account"{t.Fatalf("persisted account=%q",sender.text)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT *"));err!=nil{t.Fatal(err)}
	sender.text="sentinel"
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if sender.text!=""{t.Fatalf("account should be cleared after logout, got %q",sender.text)}
}

func TestAccountIdentityFollowsNickAndClearsOnQuit(t *testing.T){
	path:=filepath.Join(t.TempDir(),"identity.tengo")
	writeScript(t,path,`bot("command","who","who")
if bot("active","who") { bot("say",event["target"],event["account"]) }`)
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b);if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example JOIN #engo alice-account :Alice Example"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example NICK :alice2"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice2!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if sender.text!="alice-account"{t.Fatalf("account after nick=%q",sender.text)}
	if err:=b.Handle(irc.ParseMessage(":alice2!u@example QUIT :bye"));err!=nil{t.Fatal(err)}
	sender.text="sentinel"
	if err:=b.Handle(irc.ParseMessage(":alice2!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if sender.text!=""{t.Fatalf("account after quit=%q",sender.text)}
}

func TestAccountPermissionsDefaultDeny(t *testing.T){
	path:=filepath.Join(t.TempDir(),"permissions.tengo")
	writeScript(t,path,`bot("command","admin","admin")
if bot("active","admin") { if bot("allowed","admin") { bot("say",event["target"],"yes") } else { bot("say",event["target"],"no") } }`)
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b);rt.SetPermissions(map[string][]string{"alice-account":{"admin"}});if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":bob!u@example PRIVMSG #engo :!admin"));err!=nil{t.Fatal(err)}
	if sender.text!="no"{t.Fatalf("unauthenticated/default permission result=%q",sender.text)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!admin"));err!=nil{t.Fatal(err)}
	if sender.text!="yes"{t.Fatalf("granted permission result=%q",sender.text)}
}

func TestProtectedCommandDispatchDenyAllow(t *testing.T){
	path:=filepath.Join(t.TempDir(),"protected.tengo")
	writeScript(t,path,`bot("command","reload","reload")
if bot("active","reload") { bot("say",event["target"],"ran") }`)
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b)
	rt.SetPermissions(map[string][]string{"alice-account":{"admin"}})
	rt.SetCommandPermissions(map[string]string{"reload":"admin"})
	if err:=rt.Load();err!=nil{t.Fatal(err)}

	sender.text="untouched"
	if err:=b.Handle(irc.ParseMessage(":bob!u@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="untouched"{t.Fatalf("unauthenticated protected command ran: %q",sender.text)}

	if err:=b.Handle(irc.ParseMessage(":bob!u@example ACCOUNT bob-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":bob!u@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="untouched"{t.Fatalf("unauthorized protected command ran: %q",sender.text)}

	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="ran"{t.Fatalf("authorized protected command did not run: %q",sender.text)}
}


func TestProtectedCommandRejectsUnverifiedAccountName(t *testing.T){
	path:=filepath.Join(t.TempDir(),"unverified.tengo")
	writeScript(t,path,"bot(\"command\",\"reload\",\"reload\")\nif bot(\"active\",\"reload\") { bot(\"say\",event[\"target\"],\"ran\") }")
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b)
	rt.SetPermissions(map[string][]string{"alice-account":{"admin"}});rt.SetCommandPermissions(map[string]string{"reload":"admin"});if err:=rt.Load();err!=nil{t.Fatal(err)}
	ev:=bot.Event{Name:"message",Nick:"alice",Target:"#engo",Text:"!reload",Account:"alice-account",AccountVerified:false,Message:irc.ParseMessage(":alice PRIVMSG #engo :!reload")}
	if rt.allowed(ev,"admin"){t.Fatal("unverified account name granted permission")}
}

func TestAccountTagGrantsProtectedCommand(t *testing.T){
	path:=filepath.Join(t.TempDir(),"tagged-protected.tengo")
	writeScript(t,path,"bot(\"command\",\"reload\",\"reload\")\nif bot(\"active\",\"reload\") { bot(\"say\",event[\"target\"],\"ran\") }")
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b);rt.SetPermissions(map[string][]string{"alice-account":{"admin"}});rt.SetCommandPermissions(map[string]string{"reload":"admin"});if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage("@account=alice-account :alice!u@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="ran"{t.Fatalf("verified account-tag did not authorize protected command: %q",sender.text)}
}

func TestEventObjectExposesAccountVerification(t *testing.T){
	ev:=bot.Event{Name:"message",Account:"alice-account",AccountVerified:true};obj:=eventObject(ev)
	if obj["account_verified"]!=true{t.Fatalf("account_verified=%#v",obj["account_verified"])}
}


func TestAccountNotifyCacheGrantsProtectedCommand(t *testing.T){
	path:=filepath.Join(t.TempDir(),"notify-protected.tengo")
	writeScript(t,path,"bot(\"command\",\"reload\",\"reload\")\nif bot(\"active\",\"reload\") { bot(\"say\",event[\"target\"],\"ran\") }")
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b);rt.SetPermissions(map[string][]string{"alice-account":{"admin"}});rt.SetCommandPermissions(map[string]string{"reload":"admin"});if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="ran"{t.Fatalf("account-notify cache did not authorize protected command: %q",sender.text)}
}

func TestAccountNotifyCacheDifferentUserhostDenied(t *testing.T){
	path:=filepath.Join(t.TempDir(),"notify-stale.tengo")
	writeScript(t,path,"bot(\"command\",\"reload\",\"reload\")\nif bot(\"active\",\"reload\") { bot(\"say\",event[\"target\"],\"ran\") }")
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b);rt.SetPermissions(map[string][]string{"alice-account":{"admin"}});rt.SetCommandPermissions(map[string]string{"reload":"admin"});if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!old@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	sender.text="untouched"
	if err:=b.Handle(irc.ParseMessage(":alice!new@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="untouched"{t.Fatalf("stale account-notify identity authorized command: %q",sender.text)}
}


func TestExtendedJoinCacheGrantsProtectedCommand(t *testing.T){
	path:=filepath.Join(t.TempDir(),"join-protected.tengo")
	writeScript(t,path,"bot(\"command\",\"reload\",\"reload\")\nif bot(\"active\",\"reload\") { bot(\"say\",event[\"target\"],\"ran\") }")
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b);rt.SetPermissions(map[string][]string{"alice-account":{"admin"}});rt.SetCommandPermissions(map[string]string{"reload":"admin"});if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example JOIN #engo alice-account :Alice Example"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="ran"{t.Fatalf("extended-join identity did not authorize protected command: %q",sender.text)}
}

func TestAccountLogoutRevokesProtectedCommand(t *testing.T){
	path:=filepath.Join(t.TempDir(),"logout-protected.tengo")
	writeScript(t,path,"bot(\"command\",\"reload\",\"reload\")\nif bot(\"active\",\"reload\") { bot(\"say\",event[\"target\"],\"ran\") }")
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b);rt.SetPermissions(map[string][]string{"alice-account":{"admin"}});rt.SetCommandPermissions(map[string]string{"reload":"admin"});if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT *"));err!=nil{t.Fatal(err)}
	sender.text="untouched";if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="untouched"{t.Fatalf("logout did not revoke protected command: %q",sender.text)}
}

func TestQuitRevokesProtectedCommand(t *testing.T){
	path:=filepath.Join(t.TempDir(),"quit-protected.tengo")
	writeScript(t,path,"bot(\"command\",\"reload\",\"reload\")\nif bot(\"active\",\"reload\") { bot(\"say\",event[\"target\"],\"ran\") }")
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b);rt.SetPermissions(map[string][]string{"alice-account":{"admin"}});rt.SetCommandPermissions(map[string]string{"reload":"admin"});if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)};if err:=b.Handle(irc.ParseMessage(":alice!u@example QUIT :bye"));err!=nil{t.Fatal(err)}
	sender.text="untouched";if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="untouched"{t.Fatalf("QUIT did not revoke protected command: %q",sender.text)}
}

func TestNickChangeKeepsProtectedCommandForSameUserhost(t *testing.T){
	path:=filepath.Join(t.TempDir(),"nick-protected.tengo")
	writeScript(t,path,"bot(\"command\",\"reload\",\"reload\")\nif bot(\"active\",\"reload\") { bot(\"say\",event[\"target\"],\"ran\") }")
	sender:=&captureSender{};b:=bot.New(sender);rt:=New(path,b);rt.SetPermissions(map[string][]string{"alice-account":{"admin"}});rt.SetCommandPermissions(map[string]string{"reload":"admin"});if err:=rt.Load();err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)};if err:=b.Handle(irc.ParseMessage(":alice!u@example NICK :alice2"));err!=nil{t.Fatal(err)};if err:=b.Handle(irc.ParseMessage(":alice2!u@example PRIVMSG #engo :!reload"));err!=nil{t.Fatal(err)}
	if sender.text!="ran"{t.Fatalf("same-userhost nick change lost authorization: %q",sender.text)}
}

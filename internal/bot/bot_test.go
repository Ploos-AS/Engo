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


func TestIRCNickIdentityUsesRFC1459CaseMapping(t *testing.T){
	b:=New(&testSender{})
	var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":Alice[!u@example ACCOUNT account-one"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":aLICE{!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!="account-one"{t.Fatalf("RFC1459-equivalent nick lost account identity: %q",got)}
}

func TestQuitClearsRFC1459EquivalentNickIdentity(t *testing.T){
	b:=New(&testSender{})
	var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":Alice[!u@example ACCOUNT account-one"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":ALICE{!u@example QUIT :bye"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice[!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!=""{t.Fatalf("QUIT left stale RFC1459-equivalent identity: %q",got)}
}


func TestISupportCaseMapping(t *testing.T){
	tests:=[]struct{name,mapping,source,lookup string;want bool}{
		{"ascii","ascii","Alice[","alice{",false},
		{"strict-rfc1459","strict-rfc1459","Alice[","alice{",true},
		{"strict caret","strict-rfc1459","Alice^","alice~",false},
		{"rfc1459 caret","rfc1459","Alice^","alice~",true},
	}
	for _,tc:=range tests{t.Run(tc.name,func(t *testing.T){
		b:=New(&testSender{});var got string;b.Command("who",func(ev Event)error{got=ev.Account;return nil})
		if err:=b.Handle(irc.ParseMessage(":server 005 engo CASEMAPPING="+tc.mapping+" :supported"));err!=nil{t.Fatal(err)}
		if err:=b.Handle(irc.ParseMessage(":"+tc.source+"!u@example ACCOUNT account-one"));err!=nil{t.Fatal(err)}
		if err:=b.Handle(irc.ParseMessage(":"+tc.lookup+"!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
		if (got=="account-one")!=tc.want{t.Fatalf("mapping %s lookup account=%q want match=%v",tc.mapping,got,tc.want)}
	})}
}


func TestCaseMappingChangeClearsIdentityCache(t *testing.T){
	b:=New(&testSender{});var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":server 005 engo CASEMAPPING=ascii :supported"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!=""{t.Fatalf("CASEMAPPING change retained stale identity: %q",got)}
}

func TestRepeatedCaseMappingDoesNotClearIdentityCache(t *testing.T){
	b:=New(&testSender{});var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":server 005 engo CASEMAPPING=rfc1459 :supported"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":server 005 engo CASEMAPPING=rfc1459 :supported"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!="alice-account"{t.Fatalf("unchanged CASEMAPPING cleared valid identity: %q",got)}
}


func TestNickReuseDifferentUserhostDoesNotInheritAccount(t *testing.T){
	b:=New(&testSender{});var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":alice!old@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!new@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!=""{t.Fatalf("reused nick inherited stale account: %q",got)}
}

func TestNickChangePreservesAccountForSameUserhost(t *testing.T){
	b:=New(&testSender{});var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice!u@example NICK :alice2"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice2!u@example PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!="alice-account"{t.Fatalf("nick change lost authenticated account: %q",got)}
}


func TestCachedAccountWithoutCurrentUserhostFailsClosed(t *testing.T){
	b:=New(&testSender{});var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":alice!u@example ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!=""{t.Fatalf("message without userhost inherited cached account: %q",got)}
}

func TestAccountLearnedWithoutUserhostIsNotReusable(t *testing.T){
	b:=New(&testSender{});var got string
	b.Command("who",func(ev Event)error{got=ev.Account;return nil})
	if err:=b.Handle(irc.ParseMessage(":alice ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
	if err:=b.Handle(irc.ParseMessage(":alice PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
	if got!=""{t.Fatalf("unbound account identity was reused: %q",got)}
}


func TestAccountTagWithoutUserhostIsNotVerified(t *testing.T){
 b:=New(&testSender{});var got Event
 b.Command("who",func(ev Event)error{got=ev;return nil})
 if err:=b.Handle(irc.ParseMessage("@account=alice-account :alice PRIVMSG #engo :!who"));err!=nil{t.Fatal(err)}
 if got.Account!=""{t.Fatalf("account-tag without userhost exposed account %q",got.Account)}
 if got.AccountVerified{t.Fatal("account-tag without userhost was marked verified")}
}


func TestAccountEventWithoutUserhostIsNotVerified(t *testing.T){
 b:=New(&testSender{});var got Event
 b.On("account",func(ev Event)error{got=ev;return nil})
 if err:=b.Handle(irc.ParseMessage(":alice ACCOUNT alice-account"));err!=nil{t.Fatal(err)}
 if got.Account!=""{t.Fatalf("ACCOUNT without userhost exposed account %q",got.Account)}
 if got.AccountVerified{t.Fatal("ACCOUNT without userhost was marked verified")}
}

func TestExtendedJoinWithoutUserhostIsNotVerified(t *testing.T){
 b:=New(&testSender{});var got Event
 b.On("join",func(ev Event)error{got=ev;return nil})
 if err:=b.Handle(irc.ParseMessage(":alice JOIN #engo alice-account :Alice Example"));err!=nil{t.Fatal(err)}
 if got.Account!=""{t.Fatalf("extended JOIN without userhost exposed account %q",got.Account)}
 if got.AccountVerified{t.Fatal("extended JOIN without userhost was marked verified")}
}

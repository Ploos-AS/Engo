package irc

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestCapabilityListed(t *testing.T) {
	line := ":irc.example CAP engo LS :multi-prefix sasl=PLAIN account-notify"
	if !capabilityListed(line, "sasl") {
		t.Fatal("expected sasl capability")
	}
	if capabilityListed(line, "echo-message") {
		t.Fatal("unexpected echo-message capability")
	}
}

func TestNumericCommand(t *testing.T) {
	got := numericCommand([]string{":irc.example", "903", "engo", ":SASL authentication successful"})
	if got != "903" {
		t.Fatalf("got %q, want 903", got)
	}
}

func TestMultilineCapabilityCollection(t *testing.T){
	first:=[]string{":irc.example","CAP","engo","LS","*",":multi-prefix","account-notify"}
	if !capLSContinues(first){t.Fatal("expected CAP LS continuation")}
	caps:=append(capabilityNames(":irc.example CAP engo LS * :multi-prefix account-notify"),capabilityNames(":irc.example CAP engo LS :sasl=PLAIN echo-message")...)
	if !containsCapability(caps,"sasl"){t.Fatal("expected SASL from final CAP LS line")}
	if !containsCapability(caps,"multi-prefix"){t.Fatal("expected capability from continuation line")}
}

func TestAuthenticateChunks(t *testing.T){
	cases:=[]struct{name string;n int;want int;terminal bool}{
		{"short",10,1,false},
		{"exact",400,2,true},
		{"long",401,2,false},
		{"double-exact",800,3,true},
	}
	for _,tc:=range cases{t.Run(tc.name,func(t *testing.T){
		chunks:=authenticateChunks(strings.Repeat("x",tc.n))
		if len(chunks)!=tc.want{t.Fatalf("got %d chunks, want %d",len(chunks),tc.want)}
		for i,ch:=range chunks{if ch!="+"&&len(ch)>400{t.Fatalf("chunk %d too long: %d",i,len(ch))}}
		if (chunks[len(chunks)-1]=="+")!=tc.terminal{t.Fatalf("terminal marker mismatch: %#v",chunks)}
	})}
}

func TestCapabilityNamesFromNAK(t *testing.T){
	caps:=capabilityNames(":irc.example CAP engo NAK :sasl echo-message")
	if !containsCapability(caps,"sasl")||!containsCapability(caps,"echo-message"){t.Fatalf("unexpected NAK capabilities: %#v",caps)}
}

func TestRegistrationTimeout(t *testing.T){
	clientConn,serverConn:=net.Pipe();defer serverConn.Close()
	c:=&Client{conn:clientConn,cfg:Config{},registrationTimeout:20*time.Millisecond}
	defer clientConn.Close()
	err:=c.Run()
	if err==nil||!strings.Contains(err.Error(),"registration timed out"){t.Fatalf("expected registration timeout, got %v",err)}
}

func TestRequestedCapabilitiesIncludeSASLOnce(t *testing.T){
	caps:=[]string{"multi-prefix","sasl"}
	if !containsCapability(caps,"sasl"){t.Fatal("expected SASL capability")}
	count:=0
	for _,capability:=range caps{if capability=="sasl"{count++}}
	if count!=1{t.Fatalf("SASL capability count = %d, want 1",count)}
}


func TestNonSASLCapabilityACKEndsNegotiation(t *testing.T){
	clientConn,serverConn:=net.Pipe()
	defer clientConn.Close();defer serverConn.Close()
	c:=&Client{conn:clientConn,cfg:Config{Capabilities:[]string{"account-tag"}},registrationTimeout:time.Second}
	done:=make(chan error,1);go func(){done<-c.Run()}()
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo LS :account-tag\r\n"));err!=nil{t.Fatal(err)}
	buf:=make([]byte,128);n,err:=serverConn.Read(buf);if err!=nil{t.Fatal(err)}
	if !strings.Contains(string(buf[:n]),"CAP REQ :account-tag"){t.Fatalf("missing CAP REQ: %q",buf[:n])}
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo ACK :account-tag\r\n"));err!=nil{t.Fatal(err)}
	n,err=serverConn.Read(buf);if err!=nil{t.Fatal(err)}
	if !strings.Contains(string(buf[:n]),"CAP END"){t.Fatalf("non-SASL ACK did not end negotiation: %q",buf[:n])}
	_ = clientConn.Close();<-done
}


func TestPartialCapabilityACKWaitsForAllRequested(t *testing.T){
	clientConn,serverConn:=net.Pipe();defer clientConn.Close();defer serverConn.Close()
	c:=&Client{conn:clientConn,cfg:Config{Capabilities:[]string{"account-tag","server-time"}},registrationTimeout:time.Second}
	done:=make(chan error,1);go func(){done<-c.Run()}()
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo LS :account-tag server-time\r\n"));err!=nil{t.Fatal(err)}
	buf:=make([]byte,256);n,err:=serverConn.Read(buf);if err!=nil{t.Fatal(err)}
	if !strings.Contains(string(buf[:n]),"CAP REQ :account-tag server-time"){t.Fatalf("missing CAP REQ: %q",buf[:n])}
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo ACK :account-tag\r\n"));err!=nil{t.Fatal(err)}
	_ = serverConn.SetReadDeadline(time.Now().Add(30*time.Millisecond))
	if n,err=serverConn.Read(buf);err==nil&&strings.Contains(string(buf[:n]),"CAP END"){t.Fatalf("CAP END sent before all capabilities ACKed")}
	_ = serverConn.SetReadDeadline(time.Time{})
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo ACK :server-time\r\n"));err!=nil{t.Fatal(err)}
	n,err=serverConn.Read(buf);if err!=nil{t.Fatal(err)}
	if !strings.Contains(string(buf[:n]),"CAP END"){t.Fatalf("CAP END missing after all ACKs: %q",buf[:n])}
	_ = clientConn.Close();<-done
}


func TestCapabilityNameModifiers(t *testing.T){
	cases:=map[string]string{
		"account-tag":"account-tag",
		"-account-tag":"account-tag",
		"~server-time":"server-time",
		"=sasl":"sasl",
		"sasl=PLAIN,EXTERNAL":"sasl",
		"~sasl=PLAIN":"sasl",
		"":"",
		"-":"",
	}
	for in,want:=range cases{if got:=capabilityName(in);got!=want{t.Errorf("capabilityName(%q)=%q want %q",in,got,want)}}
}

func TestCapabilityNamesNormalizeModifiers(t *testing.T){
	caps:=capabilityNames(":irc.example CAP engo ACK :-account-tag ~server-time =sasl")
	for _,want:=range []string{"account-tag","server-time","sasl"}{
		if !containsCapability(caps,want){t.Fatalf("missing normalized capability %q in %#v",want,caps)}
	}
}


func TestNegativeCapabilityACKFailsFast(t *testing.T){
	clientConn,serverConn:=net.Pipe();defer clientConn.Close();defer serverConn.Close()
	c:=&Client{conn:clientConn,cfg:Config{Capabilities:[]string{"account-tag"}},registrationTimeout:time.Second}
	done:=make(chan error,1);go func(){done<-c.Run()}()
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo LS :account-tag\r\n"));err!=nil{t.Fatal(err)}
	buf:=make([]byte,256);if _,err:=serverConn.Read(buf);err!=nil{t.Fatal(err)}
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo ACK :-account-tag\r\n"));err!=nil{t.Fatal(err)}
	select{
	case err:=<-done:
		if err==nil||!strings.Contains(err.Error(),"disabled requested IRC capability"){t.Fatalf("unexpected error: %v",err)}
	case <-time.After(100*time.Millisecond):
		t.Fatal("negative ACK did not fail fast")
	}
}


func TestCapabilityNAKReportsOnlyRequestedCapabilities(t *testing.T){
	clientConn,serverConn:=net.Pipe();defer clientConn.Close();defer serverConn.Close()
	c:=&Client{conn:clientConn,cfg:Config{Capabilities:[]string{"account-tag","server-time"}},registrationTimeout:time.Second}
	done:=make(chan error,1);go func(){done<-c.Run()}()
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo LS :account-tag server-time\r\n"));err!=nil{t.Fatal(err)}
	buf:=make([]byte,256);if _,err:=serverConn.Read(buf);err!=nil{t.Fatal(err)}
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo NAK :server-time unrelated-cap\r\n"));err!=nil{t.Fatal(err)}
	select{
	case err:=<-done:
		if err==nil{t.Fatal("expected requested capability rejection")}
		if !strings.Contains(err.Error(),"server-time"){t.Fatalf("missing requested capability in error: %v",err)}
		if strings.Contains(err.Error(),"unrelated-cap"){t.Fatalf("unrequested capability leaked into error: %v",err)}
	case <-time.After(100*time.Millisecond):t.Fatal("CAP NAK did not fail fast")
	}
}

func TestUnrequestedCapabilityNAKIsIgnored(t *testing.T){
	clientConn,serverConn:=net.Pipe();defer clientConn.Close();defer serverConn.Close()
	c:=&Client{conn:clientConn,cfg:Config{Capabilities:[]string{"account-tag"}},registrationTimeout:time.Second}
	done:=make(chan error,1);go func(){done<-c.Run()}()
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo LS :account-tag\r\n"));err!=nil{t.Fatal(err)}
	buf:=make([]byte,256);if _,err:=serverConn.Read(buf);err!=nil{t.Fatal(err)}
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo NAK :unrelated-cap\r\n"));err!=nil{t.Fatal(err)}
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo ACK :account-tag\r\n"));err!=nil{t.Fatal(err)}
	n,err:=serverConn.Read(buf);if err!=nil{t.Fatal(err)}
	if !strings.Contains(string(buf[:n]),"CAP END"){t.Fatalf("negotiation did not continue after unrelated NAK: %q",buf[:n])}
	_ = clientConn.Close();<-done
}

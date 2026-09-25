package irc

import (
	"bufio"
	"fmt"
	"io"
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


func TestCapabilityDELOfRequestedCapabilityFails(t *testing.T){
	clientConn,serverConn:=net.Pipe();defer clientConn.Close();defer serverConn.Close()
	c:=&Client{conn:clientConn,cfg:Config{Capabilities:[]string{"account-tag"}},registrationTimeout:time.Second}
	done:=make(chan error,1);go func(){done<-c.Run()}()
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo LS :account-tag\r\n"));err!=nil{t.Fatal(err)}
	buf:=make([]byte,256);if _,err:=serverConn.Read(buf);err!=nil{t.Fatal(err)}
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo ACK :account-tag\r\n"));err!=nil{t.Fatal(err)}
	if _,err:=serverConn.Read(buf);err!=nil{t.Fatal(err)}
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo DEL :account-tag\r\n"));err!=nil{t.Fatal(err)}
	select{
	case err:=<-done:
		if err==nil||!strings.Contains(err.Error(),"removed requested IRC capability"){t.Fatalf("unexpected DEL result: %v",err)}
	case <-time.After(100*time.Millisecond):t.Fatal("CAP DEL did not fail fast")
	}
}

func TestCapabilityNEWDoesNotEnableCapability(t *testing.T){
	clientConn,serverConn:=net.Pipe();defer clientConn.Close();defer serverConn.Close()
	c:=&Client{conn:clientConn,cfg:Config{},registrationTimeout:time.Second}
	done:=make(chan error,1);go func(){done<-c.Run()}()
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo LS :\r\n"));err!=nil{t.Fatal(err)}
	buf:=make([]byte,256);n,err:=serverConn.Read(buf);if err!=nil{t.Fatal(err)}
	if !strings.Contains(string(buf[:n]),"CAP END"){t.Fatalf("expected initial CAP END: %q",buf[:n])}
	if _,err:=serverConn.Write([]byte(":irc.example CAP engo NEW :account-tag\r\n"));err!=nil{t.Fatal(err)}
	_ = serverConn.SetReadDeadline(time.Now().Add(30*time.Millisecond))
	n,err=serverConn.Read(buf)
	if err==nil&&strings.Contains(string(buf[:n]),"CAP REQ"){t.Fatalf("CAP NEW silently enabled capability")}
	_ = clientConn.Close();<-done
}


func TestSanitizeTextRemovesIRCLineBreaks(t *testing.T){
	got:=sanitizeText("hello\r\nOPER root\nworld")
	if strings.ContainsAny(got,"\r\n"){t.Fatalf("sanitizeText retained line break: %q",got)}
	if got!="hello  OPER root world"{t.Fatalf("unexpected sanitized text: %q",got)}
}

func TestActionUsesCTCPFramingAndSanitizesText(t *testing.T){
	clientConn,serverConn:=net.Pipe();defer clientConn.Close();defer serverConn.Close()
	c:=&Client{conn:clientConn}
	done:=make(chan error,1)
	go func(){done<-c.Action("#engo","waves\r\nPRIVMSG #other :oops")}()
	buf:=make([]byte,256);n,err:=serverConn.Read(buf);if err!=nil{t.Fatal(err)}
	if err:=<-done;err!=nil{t.Fatal(err)}
	got:=string(buf[:n])
	want:="PRIVMSG #engo :\x01ACTION waves  PRIVMSG #other :oops\x01\r\n"
	if got!=want{t.Fatalf("Action frame=%q want %q",got,want)}
	if strings.Contains(got,"\r\nPRIVMSG #other"){t.Fatal("Action allowed IRC line injection")}
}


func TestCAPDELSASLAfterAuthenticationDoesNotDisconnect(t *testing.T){
	clientConn,serverConn:=net.Pipe();defer clientConn.Close();defer serverConn.Close()
	c:=&Client{conn:clientConn,cfg:Config{Nick:"engo",User:"engo",RealName:"Engo",SASLUsername:"engo",SASLPassword:"secret"},registrationTimeout:time.Second}
	done:=make(chan error,1);go func(){done<-c.Run()}()
	write:=func(line string){t.Helper();if _,err:=fmt.Fprintf(serverConn,"%s\r\n",line);err!=nil{t.Fatal(err)}}
	read:=bufio.NewReader(serverConn)
	write(":srv CAP engo LS :sasl");if line,err:=read.ReadString('\n');err!=nil||!strings.Contains(line,"CAP REQ :sasl"){t.Fatalf("CAP REQ=%q err=%v",line,err)}
	write(":srv CAP engo ACK :sasl");if line,err:=read.ReadString('\n');err!=nil||!strings.Contains(line,"AUTHENTICATE PLAIN"){t.Fatalf("AUTHENTICATE=%q err=%v",line,err)}
	write("AUTHENTICATE +");if _,err:=read.ReadString('\n');err!=nil{t.Fatal(err)}
	write(":srv 903 engo :SASL authentication successful");if line,err:=read.ReadString('\n');err!=nil||!strings.Contains(line,"CAP END"){t.Fatalf("CAP END=%q err=%v",line,err)}
	write(":srv 001 engo :welcome")
	write(":srv CAP engo DEL :sasl")
	_ = serverConn.Close()
	if err:=<-done;err!=io.EOF{t.Fatalf("CAP DEL sasl after auth returned %v, want EOF",err)}
}


func TestCAPDELRequiredRuntimeCapabilityDisconnects(t *testing.T){
	for _,capability:=range []string{"account-tag","account-notify","extended-join","server-time"}{
		t.Run(capability,func(t *testing.T){
			clientConn,serverConn:=net.Pipe();defer clientConn.Close();defer serverConn.Close()
			c:=&Client{conn:clientConn,cfg:Config{Nick:"engo",User:"engo",RealName:"Engo",Capabilities:[]string{capability}},registrationTimeout:time.Second}
			done:=make(chan error,1);go func(){done<-c.Run()}()
			write:=func(line string){t.Helper();if _,err:=fmt.Fprintf(serverConn,"%s\r\n",line);err!=nil{t.Fatal(err)}}
			read:=bufio.NewReader(serverConn)
			write(":srv CAP engo LS :"+capability);if line,err:=read.ReadString('\n');err!=nil||!strings.Contains(line,"CAP REQ :"+capability){t.Fatalf("CAP REQ=%q err=%v",line,err)}
			write(":srv CAP engo ACK :"+capability);if line,err:=read.ReadString('\n');err!=nil||!strings.Contains(line,"CAP END"){t.Fatalf("CAP END=%q err=%v",line,err)}
			write(":srv 001 engo :welcome")
			write(":srv CAP engo DEL :"+capability)
			err:=<-done
			if err==nil||!strings.Contains(err.Error(),"server removed requested IRC capability"){t.Fatalf("CAP DEL %s returned %v",capability,err)}
		})
	}
}


func TestNormalizeCapabilities(t *testing.T){
 got:=normalizeCapabilities([]string{" ACCOUNT-TAG ","server-time","account-tag","","Server-Time"})
 want:=[]string{"account-tag","server-time"}
 if len(got)!=len(want){t.Fatalf("normalizeCapabilities()=%v, want %v",got,want)}
 for i:=range want{if got[i]!=want[i]{t.Fatalf("normalizeCapabilities()=%v, want %v",got,want)}}
}

func TestRunNormalizesConfiguredCapabilities(t *testing.T){
 clientConn,serverConn:=net.Pipe();defer serverConn.Close()
 c:=&Client{conn:clientConn,cfg:Config{Capabilities:[]string{" ACCOUNT-TAG ","account-tag"}},registrationTimeout:time.Second}
 errCh:=make(chan error,1);go func(){errCh<-c.Run()}()
 r:=bufio.NewReader(serverConn)
 if _,err:=fmt.Fprintln(serverConn,":server CAP * LS :account-tag");err!=nil{t.Fatal(err)}
 line,err:=r.ReadString('\n');if err!=nil{t.Fatal(err)}
 if strings.TrimSpace(line)!="CAP REQ :account-tag"{t.Fatalf("unexpected CAP request %q",line)}
 if _,err:=fmt.Fprintln(serverConn,":server CAP * ACK :account-tag");err!=nil{t.Fatal(err)}
 line,err=r.ReadString('\n');if err!=nil{t.Fatal(err)}
 if strings.TrimSpace(line)!="CAP END"{t.Fatalf("unexpected CAP completion %q",line)}
 serverConn.Close()
 if err:=<-errCh;err!=io.EOF{t.Fatalf("Run() error=%v, want EOF",err)}
}

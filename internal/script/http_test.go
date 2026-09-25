package script

import (
	"net"
	"net/url"
	"testing"
	"time"
)

func TestHTTPDisabledByDefault(t *testing.T){
	h:=NewHTTPClient(nil,time.Second,1024)
	if h.Enabled(){t.Fatal("HTTP must be disabled without allowlist")}
}
func TestHTTPRequiresHTTPS(t *testing.T){
	h:=NewHTTPClient([]string{"example.com"},time.Second,1024)
	u,_:=url.Parse("http://example.com/")
	if err:=h.validateURL(u);err==nil{t.Fatal("expected plain HTTP rejection")}
}
func TestHTTPRejectsUnlistedHost(t *testing.T){
	h:=NewHTTPClient([]string{"example.com"},time.Second,1024)
	u,_:=url.Parse("https://example.org/")
	if err:=h.validateURL(u);err==nil{t.Fatal("expected host allowlist rejection")}
}
func TestBlockedIP(t *testing.T){
	for _,raw:=range []string{"127.0.0.1","10.0.0.1","192.168.1.1","169.254.1.1","::1"}{
		if !blockedIP(net.ParseIP(raw)){t.Fatalf("%s should be blocked",raw)}
	}
	if blockedIP(net.ParseIP("8.8.8.8")){t.Fatal("public address unexpectedly blocked")}
}

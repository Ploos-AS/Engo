package script

import (
	"net"
	"net/http"
	"io"
	"strings"
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

func TestHTTPRejectsUserinfo(t *testing.T){
	h:=NewHTTPClient([]string{"example.com"},time.Second,1024)
	u,_:=url.Parse("https://user:pass@example.com/")
	if err:=h.validateURL(u);err==nil{t.Fatal("expected URL userinfo rejection")}
}
func TestHTTPRejectsNon443Port(t *testing.T){
	h:=NewHTTPClient([]string{"example.com"},time.Second,1024)
	u,_:=url.Parse("https://example.com:8443/")
	if err:=h.validateURL(u);err==nil{t.Fatal("expected non-443 port rejection")}
}
func TestHTTPTransportIgnoresEnvironmentProxy(t *testing.T){
	h:=NewHTTPClient([]string{"example.com"},time.Second,1024)
	tr,ok:=h.client.Transport.(*http.Transport)
	if !ok{t.Fatal("expected HTTP transport")}
	if tr.Proxy!=nil{t.Fatal("HTTP transport must not use environment proxy")}
}

func TestHTTPRejectsIPv4MappedPrivateAddress(t *testing.T){
	ip:=net.ParseIP("::ffff:127.0.0.1")
	if ip==nil{t.Fatal("failed to parse mapped IPv4 address")}
	if !blockedIP(ip){t.Fatal("IPv4-mapped loopback address should be blocked")}
}

func TestHTTPRejectsAllowedHostResolvingPrivate(t *testing.T){
	h:=NewHTTPClient([]string{"allowed.example"},time.Second,1024)
	h.lookupIP=func(string)([]net.IP,error){return []net.IP{net.ParseIP("127.0.0.1")},nil}
	u,_:=url.Parse("https://allowed.example/")
	if err:=h.validateURL(u);err==nil{t.Fatal("expected private resolved address rejection")}
}
func TestHTTPAcceptsAllowedHostResolvingPublic(t *testing.T){
	h:=NewHTTPClient([]string{"allowed.example"},time.Second,1024)
	h.lookupIP=func(string)([]net.IP,error){return []net.IP{net.ParseIP("8.8.8.8")},nil}
	u,_:=url.Parse("https://allowed.example/")
	if err:=h.validateURL(u);err!=nil{t.Fatalf("unexpected validation error: %v",err)}
}

func TestHTTPRedirectRejectsUnlistedHost(t *testing.T){
	h:=NewHTTPClient([]string{"allowed.example"},time.Second,1024)
	req:=&http.Request{URL:&url.URL{Scheme:"https",Host:"other.example"}}
	if err:=h.client.CheckRedirect(req,nil);err==nil{t.Fatal("expected redirect host rejection")}
}
func TestHTTPRedirectLimit(t *testing.T){
	h:=NewHTTPClient([]string{"allowed.example"},time.Second,1024)
	h.lookupIP=func(string)([]net.IP,error){return []net.IP{net.ParseIP("8.8.8.8")},nil}
	req:=&http.Request{URL:&url.URL{Scheme:"https",Host:"allowed.example"}}
	via:=make([]*http.Request,5)
	if err:=h.client.CheckRedirect(req,via);err==nil{t.Fatal("expected redirect limit rejection")}
}

type roundTripFunc func(*http.Request)(*http.Response,error)
func (f roundTripFunc) RoundTrip(r *http.Request)(*http.Response,error){return f(r)}

func TestHTTPResponseBodyLimit(t *testing.T){
	h:=NewHTTPClient([]string{"allowed.example"},time.Second,4)
	h.lookupIP=func(string)([]net.IP,error){return []net.IP{net.ParseIP("8.8.8.8")},nil}
	h.client.Transport=roundTripFunc(func(*http.Request)(*http.Response,error){
		return &http.Response{StatusCode:200,Header:make(http.Header),Body:io.NopCloser(strings.NewReader("12345"))},nil
	})
	if _,err:=h.Get("https://allowed.example/");err==nil{t.Fatal("expected response body limit error")}
}
func TestHTTPResponseWithinBodyLimit(t *testing.T){
	h:=NewHTTPClient([]string{"allowed.example"},time.Second,5)
	h.lookupIP=func(string)([]net.IP,error){return []net.IP{net.ParseIP("8.8.8.8")},nil}
	h.client.Transport=roundTripFunc(func(*http.Request)(*http.Response,error){
		header:=make(http.Header);header.Set("Content-Type","text/plain")
		return &http.Response{StatusCode:200,Header:header,Body:io.NopCloser(strings.NewReader("12345"))},nil
	})
	res,err:=h.Get("https://allowed.example/")
	if err!=nil{t.Fatalf("unexpected response error: %v",err)}
	if res["body"]!="12345"{t.Fatalf("unexpected body: %v",res["body"])}
	if res["status"]!=int64(200){t.Fatalf("unexpected status: %v",res["status"])}
}

func TestHTTPRejectsHostWithNoAddresses(t *testing.T){
	h:=NewHTTPClient([]string{"allowed.example"},time.Second,1024)
	h.lookupIP=func(string)([]net.IP,error){return []net.IP{},nil}
	u,_:=url.Parse("https://allowed.example/")
	if err:=h.validateURL(u);err==nil{t.Fatal("expected empty DNS result rejection")}
}


func TestHTTPClientAppliesSafeDefaults(t *testing.T) {
	h := NewHTTPClient([]string{"example.com"}, 0, 0)
	if h.client.Timeout != 10*time.Second {
		t.Fatalf("default timeout = %v, want 10s", h.client.Timeout)
	}
	if h.maxBody != 256*1024 {
		t.Fatalf("default max body = %d, want %d", h.maxBody, 256*1024)
	}
}

package botai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/version" {
			w.Write([]byte(`{"api_version":"1.0.0"}`))
			return
		}
		if r.URL.Path == "/v1/chat" {
			w.Write([]byte(`{"expert":"irc","text":"answer","provider":"test"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer s.Close()
	c, err := New(s.URL, time.Second)
	if err != nil { t.Fatal(err) }
	if err := c.Compatible(context.Background()); err != nil { t.Fatal(err) }
	got, err := c.Chat(context.Background(), "irc", "hello")
	if err != nil { t.Fatal(err) }
	if got != "answer" { t.Fatalf("got=%q", got) }
}

func TestClientRejectsIncompatibleAPI(t *testing.T) {
	s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.Write([]byte(`{"api_version":"2.0.0"}`))
	}))
	defer s.Close()
	c,_:=New(s.URL,time.Second)
	if err:=c.Compatible(context.Background());err==nil{t.Fatal("expected compatibility error")}
}

func TestClientUnavailable(t *testing.T) {
	c,_:=New("http://127.0.0.1:1",100*time.Millisecond)
	if err:=c.Compatible(context.Background());err==nil{t.Fatal("expected unavailable error")}
}

func TestClientHTTPErrorDoesNotLeakBody(t *testing.T) {
	const secret="provider-secret-body"
	s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		http.Error(w,secret,http.StatusBadGateway)
	}))
	defer s.Close()
	c,_:=New(s.URL,time.Second)
	_,err:=c.Chat(context.Background(),"irc","hello")
	if err==nil{t.Fatal("expected HTTP error")}
	if strings.Contains(err.Error(),secret){t.Fatalf("upstream body leaked: %v",err)}
	if !strings.Contains(err.Error(),"502"){t.Fatalf("status missing: %v",err)}
}

func TestClientRejectsLongHistory(t *testing.T) {
	c,_:=New("http://127.0.0.1",time.Second)
	h:=make([]Message,21)
	if _,err:=c.ChatWithHistory(context.Background(),"irc",h,"hello");err==nil{t.Fatal("expected history error")}
}

func TestClientOversizedResponseFails(t *testing.T) {
	s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("Content-Type","application/json")
		w.Write([]byte(`{"text":"`))
		w.Write([]byte(strings.Repeat("x",(1<<20)+1024)))
		w.Write([]byte(`"}`))
	}))
	defer s.Close()
	c,_:=New(s.URL,time.Second)
	if _,err:=c.Chat(context.Background(),"irc","hello");err==nil{t.Fatal("expected bounded decode error")}
}

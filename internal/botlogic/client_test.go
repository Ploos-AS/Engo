package botlogic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCompatibleAndQuery(t *testing.T){
	s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("Content-Type","application/json")
		switch r.URL.Path{
		case "/v1/version": w.Write([]byte(`{"service":"botlogic","version":"1.0.0","api":"v1"}`))
		case "/v1/query": w.Write([]byte(`{"ruleset":"irc-policy","revision":3,"solutions":[{"Allowed":"yes"}]}`))
		default:http.NotFound(w,r)
		}
	}));defer s.Close()
	c,err:=New(s.URL,time.Second);if err!=nil{t.Fatal(err)};if err:=c.Compatible(context.Background());err!=nil{t.Fatal(err)}
	got,err:=c.Query(context.Background(),"irc-policy","may_speak(alice, Allowed).");if err!=nil{t.Fatal(err)}
	if got.Revision!=3||len(got.Solutions)!=1||got.Solutions[0]["Allowed"]!="yes"{t.Fatalf("result=%+v",got)}
}
func TestIncompatibleAPIFailsClosedForIntegration(t *testing.T){
	s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Write([]byte(`{"service":"botlogic","version":"2.0.0","api":"v2"}`))}));defer s.Close()
	c,_:=New(s.URL,time.Second);if err:=c.Compatible(context.Background());err==nil{t.Fatal("expected incompatible API")}
}

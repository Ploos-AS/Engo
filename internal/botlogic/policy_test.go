package botlogic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMayExecuteUsesDataAsQuotedAtoms(t *testing.T){
	var body string
	s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){b:=make([]byte,r.ContentLength);r.Body.Read(b);body=string(b);w.Header().Set("Content-Type","application/json");w.Write([]byte(`{"ruleset":"irc-policy","revision":9,"solutions":[{}]}`))}));defer s.Close()
	c,_:=New(s.URL,time.Second);allowed,rev,err:=c.MayExecute(context.Background(),"irc-policy","Alice","reload");if err!=nil||!allowed||rev!=9{t.Fatalf("allowed=%v rev=%d err=%v",allowed,rev,err)}
	if !strings.Contains(body,`may_execute('alice','reload').`){t.Fatalf("body=%s",body)}
}
func TestQuoteAtomEscapesQuote(t *testing.T){
	if got:=QuoteAtom("o'brien");got!="'o''brien'"{t.Fatalf("got=%q",got)}
}

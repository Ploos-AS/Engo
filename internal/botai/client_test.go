package botai

import(
 "context"
 "net/http"
 "net/http/httptest"
 "testing"
 "time"
)
func TestClient(t *testing.T){
 s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("Content-Type","application/json")
  if r.URL.Path=="/v1/version"{w.Write([]byte(`{"api_version":"1.0.0"}`));return}
  if r.URL.Path=="/v1/chat"{w.Write([]byte(`{"expert":"irc","text":"answer","provider":"test"}`));return}
  http.NotFound(w,r)
 }));defer s.Close()
 c,err:=New(s.URL,time.Second);if err!=nil{t.Fatal(err)}
 if err:=c.Compatible(context.Background());err!=nil{t.Fatal(err)}
 got,err:=c.Chat(context.Background(),"irc","hello");if err!=nil{t.Fatal(err)}
 if got!="answer"{t.Fatalf("got=%q",got)}
}

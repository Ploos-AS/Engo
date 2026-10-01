package botlogic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestValidateAndConsultOrdersRequests(t *testing.T){
	var paths []string
	s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){paths=append(paths,r.URL.Path);w.Header().Set("Content-Type","application/json");if r.URL.Path=="/v1/validate"{w.Write([]byte(`{"ok":true,"valid":true}`));return};w.Write([]byte(`{"ok":true}`))}));defer s.Close()
	c,_:=New(s.URL,time.Second);if err:=c.ValidateAndConsult(context.Background(),"irc-policy","may_execute(a,b).");err!=nil{t.Fatal(err)}
	want:=[]string{"/v1/validate","/v1/consult"};if !reflect.DeepEqual(paths,want){t.Fatalf("paths=%v want=%v",paths,want)}
}
func TestValidateAndConsultNeverConsultsAfterValidationFailure(t *testing.T){
	var paths []string
	s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){paths=append(paths,r.URL.Path);http.Error(w,"invalid",http.StatusUnprocessableEntity)}));defer s.Close()
	c,_:=New(s.URL,time.Second);if err:=c.ValidateAndConsult(context.Background(),"irc-policy","broken(");err==nil{t.Fatal("expected error")}
	if want:=[]string{"/v1/validate"};!reflect.DeepEqual(paths,want){t.Fatalf("paths=%v want=%v",paths,want)}
}

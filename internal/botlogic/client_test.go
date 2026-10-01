package botlogic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompatibleAndQuery(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/version":
			w.Write([]byte(`{"service":"botlogic","version":"1.0.0","api":"v1"}`))
		case "/v1/query":
			w.Write([]byte(`{"ruleset":"irc-policy","revision":3,"solutions":[{"Allowed":"yes"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()
	c, err := New(s.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Compatible(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := c.Query(context.Background(), "irc-policy", "may_speak(alice, Allowed).")
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 3 || len(got.Solutions) != 1 || got.Solutions[0]["Allowed"] != "yes" {
		t.Fatalf("result=%+v", got)
	}
}
func TestIncompatibleAPIFailsClosedForIntegration(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"service":"botlogic","version":"2.0.0","api":"v2"}`))
	}))
	defer s.Close()
	c, _ := New(s.URL, time.Second)
	if err := c.Compatible(context.Background()); err == nil {
		t.Fatal("expected incompatible API")
	}
}

func TestConsultUsesStableV1Endpoint(t *testing.T) {
	var gotPath string
	var got map[string]string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true}`)
	}))
	defer s.Close()
	c, err := New(s.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Consult(context.Background(), "irc-policy", "may_execute(a,b)."); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/consult" || got["ruleset"] != "irc-policy" || got["source"] != "may_execute(a,b)." {
		t.Fatalf("path=%q body=%v", gotPath, got)
	}
}
func TestConsultRejectsOversizeSource(t *testing.T) {
	c, _ := New("http://127.0.0.1", time.Second)
	if err := c.Consult(context.Background(), "x", strings.Repeat("x", (64<<10)+1)); err == nil {
		t.Fatal("expected size error")
	}
}

func TestValidateUsesStableV1Endpoint(t *testing.T) {
	var path string
	var source string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		var in struct {
			Source string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatal(err)
		}
		source = in.Source
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true,"valid":true}`)
	}))
	defer s.Close()
	c, _ := New(s.URL, time.Second)
	if err := c.Validate(context.Background(), "allowed(a)."); err != nil {
		t.Fatal(err)
	}
	if path != "/v1/validate" || source != "allowed(a)." {
		t.Fatalf("path=%q source=%q", path, source)
	}
}
func TestValidateRejectsUnconfirmedResponse(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true,"valid":false}`)
	}))
	defer s.Close()
	c, _ := New(s.URL, time.Second)
	if err := c.Validate(context.Background(), "allowed(a)."); err == nil {
		t.Fatal("expected validation failure")
	}
}

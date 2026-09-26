package pbmp

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

type State struct {
	Nick, Network string
	Channels      []string
	connected     atomic.Bool
	mu            sync.RWMutex
	joined        map[string]bool
	join          func(string) error
	part          func(string, string) error
	modules       func() []map[string]any
	moduleAction  func(string,string) error
}

func NewState(nick, network string, channels ...string) *State {
	return &State{Nick: nick, Network: network, Channels: append([]string(nil), channels...), joined: make(map[string]bool)}
}
func (s *State) SetModuleAction(fn func(string,string) error){s.mu.Lock();s.moduleAction=fn;s.mu.Unlock()}
func (s *State) SetModules(fn func() []map[string]any) { s.mu.Lock(); s.modules = fn; s.mu.Unlock() }
func (s *State) SetActions(join func(string) error, part func(string, string) error) {
	s.mu.Lock()
	s.join = join
	s.part = part
	s.mu.Unlock()
}
func (s *State) SetConnected(v bool) {
	s.connected.Store(v)
	if !v {
		s.mu.Lock()
		clear(s.joined)
		s.mu.Unlock()
	}
}
func (s *State) Observe(command, nick string, params []string, trailing string) {
	if !strings.EqualFold(nick, s.Nick) && command != "KICK" {
		return
	}
	var ch string
	switch command {
	case "JOIN":
		if len(params) > 0 {
			ch = params[0]
		} else {
			ch = trailing
		}
		if strings.EqualFold(nick, s.Nick) && ch != "" {
			s.mu.Lock()
			s.joined[strings.ToLower(ch)] = true
			s.mu.Unlock()
		}
	case "PART":
		if len(params) > 0 {
			ch = params[0]
		}
		if strings.EqualFold(nick, s.Nick) && ch != "" {
			s.mu.Lock()
			delete(s.joined, strings.ToLower(ch))
			s.mu.Unlock()
		}
	case "KICK":
		if len(params) >= 2 && strings.EqualFold(params[1], s.Nick) {
			s.mu.Lock()
			delete(s.joined, strings.ToLower(params[0]))
			s.mu.Unlock()
		}
	}
}
func (s *State) ChannelState(name string) string {
	if !s.Connected() {
		return "disconnected"
	}
	s.mu.RLock()
	joined := s.joined[strings.ToLower(name)]
	s.mu.RUnlock()
	if joined {
		return "joined"
	}
	return "joining"
}
func (s *State) Connected() bool { return s.connected.Load() }

type request struct {
	PBMP   int            `json:"pbmp"`
	Type   string         `json:"type"`
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}
type response struct {
	PBMP   int    `json:"pbmp"`
	Type   string `json:"type"`
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  any    `json:"error,omitempty"`
}

func Handle(in []byte, s *State) ([]byte, error) {
	var q request
	if err := json.Unmarshal(in, &q); err != nil {
		return nil, err
	}
	if q.PBMP != 1 || q.Type != "request" || q.ID == "" || q.Method == "" {
		return nil, errors.New("invalid PBMP/1 request")
	}
	r := response{PBMP: 1, Type: "response", ID: q.ID, OK: true}
	switch q.Method {
	case "pbmp.info":
		r.Result = map[string]any{"protocol": "PBMP/1", "implementation": "engo", "version": "0.1.0"}
	case "capabilities.list":
		r.Result = map[string]any{"capabilities": []string{"pbmp.info", "capabilities.list", "bot.info", "networks.list", "channels.list", "channels.join", "channels.part", "modules.list", "modules.reload", "modules.enable", "modules.disable"}}
	case "bot.info":
		state := "offline"
		if s.Connected() {
			state = "online"
		}
		r.Result = map[string]any{"implementation": "engo", "version": "0.1.0", "nick": s.Nick, "state": state}
	case "modules.reload", "modules.enable", "modules.disable":
		id:=param(q.Params,"id"); if id=="" { r.OK=false; r.Error=map[string]any{"code":"invalid_params","message":"module id required"}; break }; s.mu.RLock(); action:=s.moduleAction; s.mu.RUnlock(); if action==nil { r.OK=false;r.Error=map[string]any{"code":"not_supported","message":"module lifecycle unavailable"};break }; op:=strings.TrimPrefix(q.Method,"modules."); if err:=action(op,id);err!=nil{r.OK=false;r.Error=map[string]any{"code":"operation_failed","message":err.Error()};break}; state:="active";if op=="disable"{state="disabled"};r.Result=map[string]any{"id":id,"state":state}
	case "modules.list":
		s.mu.RLock()
		fn := s.modules
		s.mu.RUnlock()
		modules := []map[string]any{}
		if fn != nil {
			modules = fn()
		}
		r.Result = map[string]any{"modules": modules}
	case "channels.join", "channels.part":
		network, name, _ := paramsString(q.Params, "network", "name", "reason")
		if network != s.Network || !validPBMPChannel(name) {
			r.OK = false
			r.Error = map[string]any{"code": "invalid_params", "message": "invalid network or channel"}
			break
		}
		s.mu.RLock()
		join, part := s.join, s.part
		s.mu.RUnlock()
		if !s.Connected() || join == nil || part == nil {
			r.OK = false
			r.Error = map[string]any{"code": "unavailable", "message": "IRC connection unavailable"}
			break
		}
		var err error
		state := "joining"
		if q.Method == "channels.join" {
			err = join(name)
		} else {
			err = part(name, param(q.Params, "reason"))
			state = "parting"
		}
		if err != nil {
			r.OK = false
			r.Error = map[string]any{"code": "operation_failed", "message": err.Error()}
			break
		}
		r.Result = map[string]any{"network": network, "name": name, "state": state}
	case "channels.list":
		channels := make([]any, 0, len(s.Channels))
		for _, name := range s.Channels {
			channels = append(channels, map[string]any{"network": s.Network, "name": name, "state": s.ChannelState(name)})
		}
		r.Result = map[string]any{"channels": channels}
	case "networks.list":
		state := "disconnected"
		if s.Connected() {
			state = "connected"
		}
		r.Result = map[string]any{"networks": []any{map[string]any{"name": s.Network, "state": state}}}
	default:
		r.OK = false
		r.Error = map[string]any{"code": "not_supported", "message": "method not supported"}
	}
	return appendJSONLine(r)
}
func param(m map[string]any, k string) string { v, _ := m[k].(string); return v }
func paramsString(m map[string]any, keys ...string) (string, string, string) {
	var v [3]string
	for i, k := range keys {
		if i < 3 {
			v[i] = param(m, k)
		}
	}
	return v[0], v[1], v[2]
}
func validPBMPChannel(s string) bool {
	return len(s) > 1 && len(s) <= 200 && strings.ContainsRune("#&+!", rune(s[0])) && !strings.ContainsAny(s, " ,\\x00\\r\\n")
}
func appendJSONLine(v any) ([]byte, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	return append(b, '\n'), nil
}
func Serve(path string, s *State) error {
	if path == "" {
		return nil
	}
	_ = os.Remove(path)
	l, e := net.Listen("unix", path)
	if e != nil {
		return e
	}
	defer func() { l.Close(); os.Remove(path) }()
	if e = os.Chmod(path, 0600); e != nil {
		return e
	}
	for {
		c, e := l.Accept()
		if e != nil {
			return e
		}
		go serveConn(c, s)
	}
}
func serveConn(c net.Conn, s *State) {
	defer c.Close()
	line, e := bufio.NewReader(c).ReadBytes('\n')
	if e != nil {
		return
	}
	out, e := Handle(line, s)
	if e == nil {
		_, _ = c.Write(out)
	}
}

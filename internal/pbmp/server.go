package pbmp

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sync/atomic"
)

type State struct {
	Nick, Network string
	Channels      []string
	connected     atomic.Bool
}

func NewState(nick, network string, channels ...string) *State {
	return &State{Nick: nick, Network: network, Channels: append([]string(nil), channels...)}
}
func (s *State) SetConnected(v bool) { s.connected.Store(v) }
func (s *State) Connected() bool     { return s.connected.Load() }

type request struct {
	PBMP   int            `json:"pbmp"`
	Type   string         `json:"type"`
	ID     string         `json:"id"`
	Method string         `json:"method"`
	Params map[string]any `json:"params"`
}
type response struct {
	PBMP     int    `json:"pbmp"`
	Type string `json:"type"`
	ID string `json:"id"`
	OK       bool   `json:"ok"`
	Result   any    `json:"result,omitempty"`
	Error    any    `json:"error,omitempty"`
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
		r.Result = map[string]any{"capabilities": []string{"pbmp.info", "capabilities.list", "bot.info", "networks.list", "channels.list"}}
	case "bot.info":
		state := "offline"
		if s.Connected() {
			state = "online"
		}
		r.Result = map[string]any{"implementation": "engo", "version": "0.1.0", "nick": s.Nick, "state": state}
	case "channels.list":
		state := "disconnected"
		if s.Connected() {
			state = "configured"
		}
		channels := make([]any, 0, len(s.Channels))
		for _, name := range s.Channels {
			channels = append(channels, map[string]any{"network": s.Network, "name": name, "state": state})
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

package botai

import "sync/atomic"

type Stats struct {
	enabled atomic.Bool
	compatible atomic.Bool
	requests atomic.Uint64
	errors atomic.Uint64
	limited atomic.Uint64
	inFlight atomic.Int64
}

func NewStats(enabled bool) *Stats {
	s:=&Stats{}
	s.enabled.Store(enabled)
	return s
}
func(s *Stats)SetCompatible(v bool){s.compatible.Store(v)}
func(s *Stats)Request(){s.requests.Add(1);s.inFlight.Add(1)}
func(s *Stats)Complete(err error){s.inFlight.Add(-1);if err!=nil{s.errors.Add(1)}}
func(s *Stats)Limited(){s.limited.Add(1)}
func(s *Stats)Snapshot() map[string]any{return map[string]any{
	"enabled":s.enabled.Load(),"compatible":s.compatible.Load(),
	"requests":s.requests.Load(),"errors":s.errors.Load(),"limited":s.limited.Load(),
	"in_flight":s.inFlight.Load(),
}}

package script

import (
	"fmt"
	"sync"
	"time"
)

type Scheduler struct {
	mu sync.Mutex
	timers map[string]*time.Timer
}

func NewScheduler()*Scheduler{return &Scheduler{timers:make(map[string]*time.Timer)}}

func (s *Scheduler) After(id string,d time.Duration,fn func()){
	if d<0{d=0}
	s.mu.Lock()
	if old:=s.timers[id];old!=nil{old.Stop()}
	var t *time.Timer
	t=time.AfterFunc(d,func(){
		s.mu.Lock()
		if s.timers[id]==t{delete(s.timers,id)}
		s.mu.Unlock()
		fn()
	})
	s.timers[id]=t
	s.mu.Unlock()
}

func (s *Scheduler) Cancel(id string)bool{
	s.mu.Lock();defer s.mu.Unlock()
	t:=s.timers[id];if t==nil{return false}
	delete(s.timers,id);return t.Stop()
}

func (s *Scheduler) CancelAll(){
	s.mu.Lock();defer s.mu.Unlock()
	for id,t:=range s.timers{t.Stop();delete(s.timers,id)}
}

func parseTimerDuration(v string)(time.Duration,error){
	d,err:=time.ParseDuration(v)
	if err!=nil||d<time.Millisecond{return 0,fmt.Errorf("invalid timer duration %q",v)}
	return d,nil
}

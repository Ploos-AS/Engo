package script

import (
	"fmt"
	"sync"
	"time"
)

type timerEntry struct{ stop func() bool; token *struct{} }

type Scheduler struct {
	mu sync.Mutex
	timers map[string]timerEntry
}

const maxActiveTimers = 32

func NewScheduler()*Scheduler{return &Scheduler{timers:make(map[string]timerEntry)}}

func (s *Scheduler) After(id string,d time.Duration,fn func())error{
	s.cancelLocked(id)
	s.mu.Lock();if len(s.timers)>=maxActiveTimers{s.mu.Unlock();return fmt.Errorf("timer limit reached")}
	token:=&struct{}{}
	var t *time.Timer
	t=time.AfterFunc(d,func(){
		s.mu.Lock()
		e,ok:=s.timers[id]
		current:=ok&&e.token==token
		if current{delete(s.timers,id)}
		s.mu.Unlock()
		if current{fn()}
	})
	s.timers[id]=timerEntry{stop:t.Stop,token:token};s.mu.Unlock();return nil
}

func (s *Scheduler) Every(id string,d time.Duration,fn func())error{
	s.cancelLocked(id)
	s.mu.Lock();full:=len(s.timers)>=maxActiveTimers;s.mu.Unlock();if full{return fmt.Errorf("timer limit reached")}
	ticker:=time.NewTicker(d)
	done:=make(chan struct{})
	var once sync.Once
	stop:=func()bool{stopped:=false;once.Do(func(){ticker.Stop();close(done);stopped=true});return stopped}
	s.timers[id]=timerEntry{stop:stop};s.mu.Unlock()
	go func(){
		for{select{
		case<-ticker.C:fn()
		case<-done:return
		}}
	}()
	return nil
}

func (s *Scheduler) cancelLocked(id string)bool{
	s.mu.Lock();defer s.mu.Unlock()
	e,ok:=s.timers[id];if !ok{return false}
	delete(s.timers,id);return e.stop()
}
func (s *Scheduler) Cancel(id string)bool{return s.cancelLocked(id)}
func (s *Scheduler) CancelAll(){
	s.mu.Lock();entries:=s.timers;s.timers=make(map[string]timerEntry);s.mu.Unlock()
	for _,e:=range entries{e.stop()}
}
func parseTimerDuration(v string)(time.Duration,error){
	d,err:=time.ParseDuration(v)
	if err!=nil||d<time.Second||d>30*24*time.Hour{return 0,fmt.Errorf("timer duration must be between 1s and 720h")}
	return d,nil
}

package script

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerAfter(t *testing.T){
	s:=NewScheduler();done:=make(chan struct{},1)
	s.After("x",10*time.Millisecond,func(){done<-struct{}{}})
	select{case<-done:case<-time.After(time.Second):t.Fatal("timer did not fire")}
}
func TestSchedulerCancel(t *testing.T){
	s:=NewScheduler();var fired atomic.Bool
	s.After("x",30*time.Millisecond,func(){fired.Store(true)})
	if !s.Cancel("x"){t.Fatal("expected timer cancellation")}
	time.Sleep(60*time.Millisecond)
	if fired.Load(){t.Fatal("cancelled timer fired")}
}
func TestParseTimerDurationRejectsTooSmall(t *testing.T){
	if _,err:=parseTimerDuration("0s");err==nil{t.Fatal("expected duration error")}
}

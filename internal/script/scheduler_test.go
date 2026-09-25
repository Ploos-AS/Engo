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


func TestSchedulerEveryAndCancel(t *testing.T){
	s:=NewScheduler();var count atomic.Int32
	s.Every("repeat",10*time.Millisecond,func(){count.Add(1)})
	deadline:=time.After(time.Second)
	for count.Load()<2{select{case<-deadline:t.Fatal("repeating timer did not fire twice");default:time.Sleep(5*time.Millisecond)}}
	s.Cancel("repeat");atCancel:=count.Load();time.Sleep(40*time.Millisecond)
	if count.Load()!=atCancel{t.Fatalf("timer continued after cancel: %d -> %d",atCancel,count.Load())}
}

func TestSchedulerReplacingTimerStopsOld(t *testing.T){
	s:=NewScheduler();var old atomic.Bool;done:=make(chan struct{},1)
	s.After("same",40*time.Millisecond,func(){old.Store(true)})
	s.After("same",10*time.Millisecond,func(){done<-struct{}{}})
	select{case<-done:case<-time.After(time.Second):t.Fatal("replacement timer did not fire")}
	time.Sleep(50*time.Millisecond)
	if old.Load(){t.Fatal("replaced timer fired")}
}

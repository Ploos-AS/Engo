package script

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerAfter(t *testing.T){
	s:=NewScheduler();done:=make(chan struct{},1)
	s.After("x",time.Second,func(){done<-struct{}{}})
	select{case<-done:case<-time.After(2*time.Second):t.Fatal("timer did not fire")}
}
func TestSchedulerCancel(t *testing.T){
	s:=NewScheduler();var fired atomic.Bool
	s.After("x",time.Second,func(){fired.Store(true)})
	if !s.Cancel("x"){t.Fatal("expected timer cancellation")}
	time.Sleep(1100*time.Millisecond)
	if fired.Load(){t.Fatal("cancelled timer fired")}
}
func TestParseTimerDurationRejectsTooSmall(t *testing.T){
	if _,err:=parseTimerDuration("0s");err==nil{t.Fatal("expected duration error")}
}


func TestSchedulerEveryAndCancel(t *testing.T){
	s:=NewScheduler();var count atomic.Int32
	s.Every("repeat",time.Second,func(){count.Add(1)})
	deadline:=time.After(3*time.Second)
	for count.Load()<2{select{case<-deadline:t.Fatal("repeating timer did not fire twice");default:time.Sleep(20*time.Millisecond)}}
	s.Cancel("repeat");atCancel:=count.Load();time.Sleep(1100*time.Millisecond)
	if count.Load()!=atCancel{t.Fatalf("timer continued after cancel: %d -> %d",atCancel,count.Load())}
}

func TestSchedulerReplacingTimerStopsOld(t *testing.T){
	s:=NewScheduler();var old atomic.Bool;done:=make(chan struct{},1)
	s.After("same",time.Second,func(){old.Store(true)})
	s.After("same",time.Second,func(){done<-struct{}{}})
	select{case<-done:case<-time.After(2*time.Second):t.Fatal("replacement timer did not fire")}
	time.Sleep(1100*time.Millisecond)
	if old.Load(){t.Fatal("replaced timer fired")}
}

func TestSchedulerTimerLimit(t *testing.T){
	s:=NewScheduler()
	for i:=0;i<maxActiveTimers;i++{
		if err:=s.After(fmt.Sprintf("timer-%d",i),time.Hour,func(){});err!=nil{t.Fatalf("timer %d: %v",i,err)}
	}
	if err:=s.After("overflow",time.Hour,func(){});err==nil{t.Fatal("expected timer limit error")}
	s.CancelAll()
}

func TestSchedulerReplacingEveryStopsOld(t *testing.T){
	s:=NewScheduler()
	var old atomic.Int32
	var replacement atomic.Int32
	if err:=s.Every("same",20*time.Millisecond,func(){old.Add(1)});err!=nil{t.Fatal(err)}
	time.Sleep(30*time.Millisecond)
	if err:=s.Every("same",20*time.Millisecond,func(){replacement.Add(1)});err!=nil{t.Fatal(err)}
	oldAtReplace:=old.Load()
	deadline:=time.After(500*time.Millisecond)
	for replacement.Load()==0{
		select{
		case<-deadline:t.Fatal("replacement repeating timer did not fire")
		default:time.Sleep(5*time.Millisecond)
		}
	}
	time.Sleep(50*time.Millisecond)
	if old.Load()!=oldAtReplace{t.Fatalf("replaced repeating timer continued: %d -> %d",oldAtReplace,old.Load())}
	s.CancelAll()
}

func TestSchedulerCancelEveryPreventsFurtherCallbacks(t *testing.T){
	s:=NewScheduler()
	var count atomic.Int32
	if err:=s.Every("repeat",20*time.Millisecond,func(){count.Add(1)});err!=nil{t.Fatal(err)}
	deadline:=time.After(500*time.Millisecond)
	for count.Load()==0{
		select{
		case<-deadline:t.Fatal("repeating timer did not fire")
		default:time.Sleep(5*time.Millisecond)
		}
	}
	if !s.Cancel("repeat"){t.Fatal("expected repeating timer cancellation")}
	atCancel:=count.Load()
	time.Sleep(60*time.Millisecond)
	if count.Load()!=atCancel{t.Fatalf("callback ran after cancellation: %d -> %d",atCancel,count.Load())}
}

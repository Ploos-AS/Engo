package script

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Ploos-AS/Engo/internal/bot"
	"github.com/d5/tengo/v2"
)

type Runtime struct {
	path string
	bot *bot.Bot
	mu sync.RWMutex
	src []byte
	maxAllocs int64
	store *Store
	scheduler *Scheduler
	http *HTTPClient
}

func New(path string,b *bot.Bot)*Runtime{return NewLimited(path,b,100000)}
func NewLimited(path string,b *bot.Bot,maxAllocs int64)*Runtime{
	return &Runtime{path:path,bot:b,maxAllocs:maxAllocs,store:NewStore("",scriptNamespace(path)),scheduler:NewScheduler(),http:NewHTTPClient(nil,10*time.Second,262144)}
}
func (r *Runtime) SetStore(s *Store){r.store=s}
func (r *Runtime) SetHTTP(h *HTTPClient){r.http=h}
func (r *Runtime) Load()error{return r.Reload()}
func (r *Runtime) Reload()error{
	src,err:=os.ReadFile(r.path);if err!=nil{return fmt.Errorf("read script: %w",err)}
	reg:=bot.NewRegistry();if err:=r.prepare(src,&reg);err!=nil{return err}
	r.scheduler.CancelAll();r.mu.Lock();r.src=append([]byte(nil),src...);r.mu.Unlock();r.bot.Replace(reg);return nil
}
func RunFile(path string)error{return New(path,bot.New(discardSender{})).Load()}
func (r *Runtime) prepare(src []byte,reg *bot.Registry)error{
	s:=tengo.NewScript(src);s.SetMaxAllocs(r.maxAllocs)
	if err:=s.Add("bot",&tengo.UserFunction{Name:"bot",Value:r.registrationCall(src,reg)});err!=nil{return err}
	if err:=s.Add("event",eventObject(bot.Event{}));err!=nil{return err}
	if _,err:=s.Run();err!=nil{return fmt.Errorf("run script: %w",err)};return nil
}
func (r *Runtime) registrationCall(src []byte,reg *bot.Registry)func(...tengo.Object)(tengo.Object,error){
	return func(args ...tengo.Object)(tengo.Object,error){
		if len(args)<1{return nil,tengo.ErrWrongNumArguments};op,ok:=tengo.ToString(args[0]);if !ok{return nil,fmt.Errorf("bot operation must be a string")}
		switch op{
		case "on","command":
			if len(args)!=3{return nil,tengo.ErrWrongNumArguments};name,ok1:=tengo.ToString(args[1]);id,ok2:=tengo.ToString(args[2]);if !ok1||!ok2{return nil,fmt.Errorf("handler name and id must be strings")}
			h:=func(ev bot.Event)error{return r.runHandler(src,id,ev)};name=strings.ToLower(strings.TrimSpace(name));if name==""{return nil,fmt.Errorf("handler name must not be empty")}
			if op=="on"{reg.Events[name]=append(reg.Events[name],h)}else{reg.Commands[name]=h};return tengo.UndefinedValue,nil
		case "say","notice","action":return tengo.UndefinedValue,nil
		default:return nil,fmt.Errorf("unknown bot operation %q",op)}
	}
}
func (r *Runtime) runHandler(src []byte,id string,ev bot.Event)error{
	s:=tengo.NewScript(src);s.SetMaxAllocs(r.maxAllocs)
	if err:=s.Add("event",eventObject(ev));err!=nil{return err}
	if err:=s.Add("bot",&tengo.UserFunction{Name:"bot",Value:r.eventCall(id)});err!=nil{return err}
	if _,err:=s.Run();err!=nil{return fmt.Errorf("Tengo handler %s: %w",id,err)};return nil
}
func (r *Runtime) eventCall(active string)func(...tengo.Object)(tengo.Object,error){
	return func(args ...tengo.Object)(tengo.Object,error){
		if len(args)<1{return nil,tengo.ErrWrongNumArguments};op,ok:=tengo.ToString(args[0]);if !ok{return nil,fmt.Errorf("bot operation must be a string")}
		switch op{
		case "on","command":return tengo.UndefinedValue,nil
		case "active":if len(args)!=2{return nil,tengo.ErrWrongNumArguments};id,_:=tengo.ToString(args[1]);return tengo.FromInterface(id==active)
		case "timer_after":
			if len(args)!=3{return nil,tengo.ErrWrongNumArguments};duration,ok1:=tengo.ToString(args[1]);id,ok2:=tengo.ToString(args[2]);if !ok1||!ok2||strings.TrimSpace(id)==""{return nil,fmt.Errorf("timer_after requires duration and handler id strings")}
			d,err:=parseTimerDuration(duration);if err!=nil{return nil,err}
			timerID:=active+":"+id
			r.scheduler.After(timerID,d,func(){if err:=r.runHandler(r.currentSource(),id,bot.Event{Name:"timer"});err!=nil{fmt.Printf("Engo timer %s: %v\n",id,err)}})
			return tengo.UndefinedValue,nil
		case "timer_every":
			if len(args)!=3{return nil,tengo.ErrWrongNumArguments};duration,ok1:=tengo.ToString(args[1]);id,ok2:=tengo.ToString(args[2]);if !ok1||!ok2||strings.TrimSpace(id)==""{return nil,fmt.Errorf("timer_every requires duration and handler id strings")}
			d,err:=parseTimerDuration(duration);if err!=nil{return nil,err}
			timerID:=active+":"+id
			r.scheduler.Every(timerID,d,func(){if err:=r.runHandler(r.currentSource(),id,bot.Event{Name:"timer"});err!=nil{fmt.Printf("Engo timer %s: %v\n",id,err)}})
			return tengo.UndefinedValue,nil
		case "timer_cancel":
			if len(args)!=2{return nil,tengo.ErrWrongNumArguments};id,ok:=tengo.ToString(args[1]);if !ok{return nil,fmt.Errorf("timer id must be string")};return tengo.FromInterface(r.scheduler.Cancel(active+":"+id))
		case "http_get":
			if len(args)!=2{return nil,tengo.ErrWrongNumArguments};raw,ok:=tengo.ToString(args[1]);if !ok{return nil,fmt.Errorf("http_get URL must be string")};if !r.http.Enabled(){return nil,fmt.Errorf("HTTP capability is disabled")};res,err:=r.http.Get(raw);if err!=nil{return nil,err};return tengo.FromInterface(res)
		case "kv_get":
			if len(args)!=2{return nil,tengo.ErrWrongNumArguments};key,ok:=tengo.ToString(args[1]);if !ok{return nil,fmt.Errorf("kv key must be string")};v,found,err:=r.store.Get(key);if err!=nil{return nil,err};if !found{return tengo.UndefinedValue,nil};return tengo.FromInterface(v)
		case "kv_set":
			if len(args)!=3{return nil,tengo.ErrWrongNumArguments};key,ok1:=tengo.ToString(args[1]);value,ok2:=tengo.ToString(args[2]);if !ok1||!ok2{return nil,fmt.Errorf("kv key/value must be strings")};if err:=r.store.Set(key,value);err!=nil{return nil,err};return tengo.UndefinedValue,nil
		case "kv_delete":
			if len(args)!=2{return nil,tengo.ErrWrongNumArguments};key,ok:=tengo.ToString(args[1]);if !ok{return nil,fmt.Errorf("kv key must be string")};if err:=r.store.Delete(key);err!=nil{return nil,err};return tengo.UndefinedValue,nil
		case "say","notice","action":
			if len(args)!=3{return nil,tengo.ErrWrongNumArguments};target,ok1:=tengo.ToString(args[1]);text,ok2:=tengo.ToString(args[2]);if !ok1||!ok2||strings.TrimSpace(target)==""{return nil,fmt.Errorf("%s requires target and text strings",op)}
			var err error;switch op{case"say":err=r.bot.Say(target,text);case"notice":err=r.bot.Notice(target,text);case"action":err=r.bot.Action(target,text)};if err!=nil{return nil,err};return tengo.UndefinedValue,nil
		default:return nil,fmt.Errorf("unknown bot operation %q",op)}
	}
}
func (r *Runtime) currentSource()[]byte{r.mu.RLock();defer r.mu.RUnlock();return append([]byte(nil),r.src...)}
func (r *Runtime) Stop(){r.scheduler.CancelAll()}
func eventObject(ev bot.Event)map[string]interface{}{args:=make([]interface{},len(ev.Args));for i,a:=range ev.Args{args[i]=a};return map[string]interface{}{"name":ev.Name,"nick":ev.Nick,"target":ev.Target,"text":ev.Text,"command":ev.Command,"args":args}}
func scriptNamespace(path string)string{return strings.TrimSuffix(filepath.Base(path),filepath.Ext(path))}
type discardSender struct{}
func(discardSender)Say(string,string)error{return nil};func(discardSender)Notice(string,string)error{return nil};func(discardSender)Action(string,string)error{return nil}

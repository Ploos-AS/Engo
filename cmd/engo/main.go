package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ploos-AS/Engo/internal/bot"
	"github.com/Ploos-AS/Engo/internal/config"
	"github.com/Ploos-AS/Engo/internal/irc"
	"github.com/Ploos-AS/Engo/internal/script"
)

func main(){
	cfg:=config.FromEnv();if err:=cfg.Validate();err!=nil{fatal(err)}
	if cfg.Server==""{if err:=script.RunFile(cfg.Script);err!=nil{fatal(err)};return}
	ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
	delay:=cfg.ReconnectMin
	for{
		started:=time.Now()
		err:=runIRC(ctx,cfg);if ctx.Err()!=nil{return}
		if shouldResetBackoff(time.Since(started),cfg.ReconnectMax){delay=cfg.ReconnectMin}
		fmt.Fprintf(os.Stderr,"engo: IRC session ended: %v; reconnecting in %s\n",err,delay)
		timer:=time.NewTimer(delay);select{case<-ctx.Done():timer.Stop();return;case<-timer.C:}
		delay*=2;if delay>cfg.ReconnectMax{delay=cfg.ReconnectMax}
	}
}

func runIRC(ctx context.Context,cfg config.Config)error{
	client,err:=irc.Dial(irc.Config{Server:cfg.Server,Nick:cfg.Nick,User:cfg.User,RealName:cfg.RealName,TLS:cfg.TLS,SASLUsername:cfg.SASLUsername,SASLPassword:cfg.SASLPassword,Capabilities:cfg.IRCCapabilities});if err!=nil{return err};defer client.Close()
	b:=bot.New(client)
	var reloadScripts func()error
	if cfg.ScriptsDir!=""{
		mgr:=script.NewManagerWithCapabilities(cfg.ScriptsDir,b,cfg.ScriptMaxAllocs,cfg.StateDir,cfg.HTTPAllow,cfg.HTTPTimeout,cfg.HTTPMaxBody)
		mgr.SetPermissions(cfg.AccountPermissions)
		mgr.SetCommandPermissions(cfg.CommandPermissions)
		for name,caps:=range cfg.ScriptCapabilities{for _,capability:=range caps{if capability=="http"{if err:=mgr.SetScriptCapabilities(name,script.Capabilities{HTTP:true});err!=nil{return err}}}}
		if err:=mgr.ReloadAll();err!=nil{return err}
		reloadScripts=mgr.ReloadAll
	}else{
		rt:=script.NewLimited(cfg.Script,b,cfg.ScriptMaxAllocs)
		rt.SetStore(script.NewStore(cfg.StateDir,scriptNamespace(cfg.Script)))
		rt.SetHTTP(script.NewHTTPClient(cfg.HTTPAllow,cfg.HTTPTimeout,cfg.HTTPMaxBody))
		rt.SetPermissions(cfg.AccountPermissions)
		rt.SetCommandPermissions(cfg.CommandPermissions)
		for _,capability:=range cfg.ScriptCapabilities[filepathBase(cfg.Script)]{if capability=="http"{rt.SetCapabilities(script.Capabilities{HTTP:true})}}
		if err:=rt.Load();err!=nil{return err}
		reloadScripts=rt.Reload
	}
	client.OnMessage(b.Handle)
	reload:=make(chan os.Signal,1);signal.Notify(reload,syscall.SIGHUP);defer signal.Stop(reload)
	done:=make(chan error,1);go func(){done<-client.Run()}()
	for{select{
	case<-ctx.Done():_ = client.Close();<-done;return ctx.Err()
	case err:=<-done:return err
	case<-reload:
		if err:=reloadScripts();err!=nil{fmt.Fprintf(os.Stderr,"engo: script reload rejected; previous version remains active: %v\n",err)}else{fmt.Fprintln(os.Stderr,"engo: script reloaded")}
	}}
}

func scriptNamespace(path string)string{
	base:=path
	for i:=len(path)-1;i>=0;i--{if path[i]=='/'||path[i]=='\\'{base=path[i+1:];break}}
	for i:=len(base)-1;i>=0;i--{if base[i]=='.'{return base[:i]}}
	return base
}
func shouldResetBackoff(sessionDuration,threshold time.Duration)bool{return sessionDuration>=threshold}

func fatal(err error){fmt.Fprintln(os.Stderr,"engo:",err);os.Exit(1)}

func filepathBase(path string)string{
	base:=path
	for i:=len(path)-1;i>=0;i--{if path[i]=='/'||path[i]=='\\'{base=path[i+1:];break}}
	return base
}

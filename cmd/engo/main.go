package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Ploos-AS/Engo/internal/bot"
	botaiclient "github.com/Ploos-AS/Engo/internal/botai"
	"github.com/Ploos-AS/Engo/internal/config"
	"github.com/Ploos-AS/Engo/internal/irc"
	"github.com/Ploos-AS/Engo/internal/pbmp"
	"github.com/Ploos-AS/Engo/internal/script"
)

func main() {
	cfg := config.FromEnv()
	if err := cfg.Validate(); err != nil {
		fatal(err)
	}
	if cfg.Server == "" {
		if err := script.RunFile(cfg.Script); err != nil {
			fatal(err)
		}
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pbstate := pbmp.NewState(cfg.Nick, cfg.Server, cfg.Channels...)
	pbstate.SetConfig(map[string]any{"server": cfg.Server, "nick": cfg.Nick, "user": cfg.User, "realname": cfg.RealName, "tls": cfg.TLS, "channels": cfg.Channels, "script": cfg.Script, "scripts_dir": cfg.ScriptsDir, "reconnect_min": cfg.ReconnectMin.String(), "reconnect_max": cfg.ReconnectMax.String()})
	if cfg.PBMPSocket != "" {
		go func() {
			if err := pbmp.Serve(cfg.PBMPSocket, pbstate); err != nil {
				fmt.Fprintf(os.Stderr, "engo: PBMP server: %v\n", err)
			}
		}()
	}
	delay := cfg.ReconnectMin
	for {
		started := time.Now()
		err := runIRC(ctx, cfg, pbstate)
		pbstate.SetConnected(false)
		pbstate.Log("warn", "IRC session disconnected")
		pbstate.CountReconnect()
		if ctx.Err() != nil {
			return
		}
		if shouldResetBackoff(time.Since(started), cfg.ReconnectMax) {
			delay = cfg.ReconnectMin
		}
		fmt.Fprintf(os.Stderr, "engo: IRC session ended: %v; reconnecting in %s\n", err, delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		delay *= 2
		if delay > cfg.ReconnectMax {
			delay = cfg.ReconnectMax
		}
	}
}

func runIRC(ctx context.Context, cfg config.Config, pbstate *pbmp.State) error {
	client, err := irc.Dial(irc.Config{Server: cfg.Server, Nick: cfg.Nick, User: cfg.User, RealName: cfg.RealName, TLS: cfg.TLS, SASLUsername: cfg.SASLUsername, SASLPassword: cfg.SASLPassword, Capabilities: cfg.IRCCapabilities, Channels: cfg.Channels})
	if err != nil {
		return err
	}
	defer client.Close()
	defer pbstate.SetActions(nil, nil)
	defer pbstate.SetModules(nil)
	defer pbstate.SetModuleLifecycle(nil)
	pbstate.SetConnected(true)
	pbstate.Log("info", "IRC session connected")
	pbstate.SetActions(client.Join, client.Part)
	b := bot.New(client)
	if cfg.BotAIURL != "" {
		ai, aiErr := botaiclient.New(cfg.BotAIURL, cfg.BotAITimeout)
		if aiErr != nil {
			fmt.Fprintf(os.Stderr, "engo: BotAI disabled: %v\n", aiErr)
			pbstate.Log("warn", "BotAI disabled by invalid configuration")
		} else {
			checkCtx, cancel := context.WithTimeout(ctx, cfg.BotAITimeout)
			compatErr := ai.Compatible(checkCtx)
			cancel()
			if compatErr != nil {
				fmt.Fprintf(os.Stderr, "engo: BotAI unavailable/incompatible; continuing without AI: %v\n", compatErr)
				pbstate.Log("warn", "BotAI unavailable or incompatible; IRC operation continues")
			} else {
				conversations := botaiclient.NewConversations(int(cfg.BotAIHistoryMessages))
				b.Command("aireset", func(ev bot.Event) error {
					key := aiConversationKey(ev)
					conversations.Reset(key)
					return b.Notice(ev.Nick, "BotAI conversation context cleared")
				})
				b.Command("ai", func(ev bot.Event) error {
					if len(ev.Args) == 0 {
						return b.Notice(ev.Nick, "usage: !ai <message>")
					}
					replyTarget := ev.Target
					if replyTarget == "" || !strings.HasPrefix(replyTarget, "#") {
						replyTarget = ev.Nick
					}
					aiCtx, cancel := context.WithTimeout(ctx, cfg.BotAITimeout)
					defer cancel()
					message := strings.Join(ev.Args, " ")
					key := aiConversationKey(ev)
					reply, err := ai.ChatWithHistory(aiCtx, cfg.BotAIExpert, conversations.History(key), message)
					if err != nil {
						fmt.Fprintf(os.Stderr, "engo: BotAI request failed: %v\n", err)
						return b.Notice(ev.Nick, "BotAI is temporarily unavailable")
					}
					conversations.AddExchange(key, message, reply)
					return b.Say(replyTarget, reply)
				})
				pbstate.Log("info", "BotAI integration enabled")
			}
		}
	}
	var reloadScripts func() error
	var moduleList func() []map[string]any
	var moduleAction func(string, string) error
	if cfg.ScriptsDir != "" {
		mgr := script.NewManagerWithCapabilities(cfg.ScriptsDir, b, cfg.ScriptMaxAllocs, cfg.StateDir, cfg.HTTPAllow, cfg.HTTPTimeout, cfg.HTTPMaxBody)
		mgr.SetPermissions(cfg.AccountPermissions)
		mgr.SetCommandPermissions(cfg.CommandPermissions)
		for name, caps := range cfg.ScriptCapabilities {
			for _, capability := range caps {
				if capability == "http" {
					if err := mgr.SetScriptCapabilities(name, script.Capabilities{HTTP: true}); err != nil {
						return err
					}
				}
			}
		}
		if err := mgr.ReloadAll(); err != nil {
			return err
		}
		reloadScripts = mgr.ReloadAll
		moduleList = mgr.Modules
		moduleAction = func(op, id string) error {
			switch op {
			case "reload":
				return mgr.Reload(id)
			case "enable":
				return mgr.Enable(id)
			case "disable":
				return mgr.Disable(id)
			}
			return fmt.Errorf("unsupported module action")
		}
	} else {
		rt := script.NewLimited(cfg.Script, b, cfg.ScriptMaxAllocs)
		rt.SetStore(script.NewStore(cfg.StateDir, scriptNamespace(cfg.Script)))
		rt.SetHTTP(script.NewHTTPClient(cfg.HTTPAllow, cfg.HTTPTimeout, cfg.HTTPMaxBody))
		rt.SetPermissions(cfg.AccountPermissions)
		rt.SetCommandPermissions(cfg.CommandPermissions)
		for _, capability := range cfg.ScriptCapabilities[filepathBase(cfg.Script)] {
			if capability == "http" {
				rt.SetCapabilities(script.Capabilities{HTTP: true})
			}
		}
		if err := rt.Load(); err != nil {
			return err
		}
		reloadScripts = rt.Reload
		name := filepathBase(cfg.Script)
		caps := append([]string(nil), cfg.ScriptCapabilities[name]...)
		moduleAction = func(op, id string) error {
			if id != name {
				return fmt.Errorf("unknown module")
			}
			if op != "reload" {
				return fmt.Errorf("operation unavailable in single-script mode")
			}
			return rt.Reload()
		}
		moduleList = func() []map[string]any {
			return []map[string]any{{"id": name, "runtime": "tengo", "state": "active", "capabilities": caps}}
		}
	}
	pbstate.SetModules(moduleList)
	if cfg.ScriptsDir != "" {
		pbstate.SetModuleLifecycle(moduleAction, "reload", "enable", "disable")
	} else {
		pbstate.SetModuleLifecycle(moduleAction, "reload")
	}
	client.OnMessage(func(m irc.Message) error {
		pbstate.CountRX()
		pbstate.Observe(m.Command, m.Nick, m.Params, m.Trailing)
		return b.Handle(m)
	})
	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGHUP)
	defer signal.Stop(reload)
	done := make(chan error, 1)
	go func() { done <- client.Run() }()
	for {
		select {
		case <-ctx.Done():
			_ = client.Close()
			<-done
			return ctx.Err()
		case err := <-done:
			return err
		case <-reload:
			if err := reloadScripts(); err != nil {
				fmt.Fprintf(os.Stderr, "engo: script reload rejected; previous version remains active: %v\n", err)
				pbstate.Log("warn", "script reload rejected; previous version remains active")
			} else {
				fmt.Fprintln(os.Stderr, "engo: script reloaded")
				pbstate.Log("info", "script reloaded")
			}
		}
	}
}

func aiConversationKey(ev bot.Event) string {
	if ev.Target != "" && strings.HasPrefix(ev.Target, "#") {
		return "channel:" + strings.ToLower(ev.Target)
	}
	if ev.AccountVerified && ev.Account != "" {
		return "account:" + strings.ToLower(ev.Account)
	}
	return "nick:" + strings.ToLower(ev.Nick)
}

func scriptNamespace(path string) string {
	base := path
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			base = path[i+1:]
			break
		}
	}
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '.' {
			return base[:i]
		}
	}
	return base
}
func shouldResetBackoff(sessionDuration, threshold time.Duration) bool {
	return sessionDuration >= threshold
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "engo:", err); os.Exit(1) }

func filepathBase(path string) string {
	base := path
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			base = path[i+1:]
			break
		}
	}
	return base
}

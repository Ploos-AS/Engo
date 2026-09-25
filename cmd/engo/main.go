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

func main() {
	cfg := config.FromEnv()
	if err := cfg.Validate(); err != nil { fatal(err) }
	if cfg.Server == "" {
		if err := script.RunFile(cfg.Script); err != nil { fatal(err) }
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	delay := cfg.ReconnectMin
	for {
		err := runIRC(ctx, cfg)
		if ctx.Err() != nil { return }
		fmt.Fprintf(os.Stderr, "engo: IRC session ended: %v; reconnecting in %s\n", err, delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done(): timer.Stop(); return
		case <-timer.C:
		}
		delay *= 2
		if delay > cfg.ReconnectMax { delay = cfg.ReconnectMax }
	}
}

func runIRC(ctx context.Context, cfg config.Config) error {
	client, err := irc.Dial(irc.Config{
		Server: cfg.Server, Nick: cfg.Nick, User: cfg.User, RealName: cfg.RealName,
		TLS: cfg.TLS, SASLUsername: cfg.SASLUsername, SASLPassword: cfg.SASLPassword,
	})
	if err != nil { return err }
	defer client.Close()

	b := bot.New(client)
	rt := script.New(cfg.Script, b)
	if err := rt.Load(); err != nil { return err }
	client.OnMessage(b.Handle)

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
			} else {
				fmt.Fprintln(os.Stderr, "engo: script reloaded")
			}
		}
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "engo:", err)
	os.Exit(1)
}

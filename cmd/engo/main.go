package main

import (
	"fmt"
	"os"

	"github.com/Ploos-AS/Engo/internal/config"
	"github.com/Ploos-AS/Engo/internal/irc"
	"github.com/Ploos-AS/Engo/internal/script"
)

func main() {
	cfg := config.FromEnv()

	if err := script.RunFile(cfg.Script); err != nil {
		fatal(err)
	}

	if cfg.Server == "" {
		return
	}

	client, err := irc.Dial(irc.Config{
		Server:   cfg.Server,
		Nick:     cfg.Nick,
		User:     cfg.User,
		RealName: cfg.RealName,
		TLS:      cfg.TLS,
	})
	if err != nil {
		fatal(err)
	}
	defer client.Close()

	if err := client.Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "engo:", err)
	os.Exit(1)
}

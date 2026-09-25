package config

import "os"

type Config struct {
	Server   string
	Nick     string
	User     string
	RealName string
	TLS      bool
	Script   string
}

func FromEnv() Config {
	return Config{
		Server:   os.Getenv("ENGO_SERVER"),
		Nick:     getenv("ENGO_NICK", "engo"),
		User:     getenv("ENGO_USER", "engo"),
		RealName: getenv("ENGO_REALNAME", "Engo IRC bot"),
		TLS:      getenv("ENGO_TLS", "1") != "0",
		Script:   getenv("ENGO_SCRIPT", "scripts/examples/hello.tengo"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

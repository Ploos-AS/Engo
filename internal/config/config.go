package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server                string
	Nick                  string
	User                  string
	RealName              string
	TLS                   bool
	Script                string
	ScriptsDir            string
	ScriptMaxAllocs       int64
	StateDir              string
	HTTPAllow             []string
	HTTPTimeout           time.Duration
	HTTPMaxBody           int64
	ScriptCapabilities    map[string][]string
	ScriptCapabilitiesRaw string
	SASLUsername          string
	SASLPassword          string
	AllowInsecureSASL     bool
	IRCCapabilities       []string
	AccountPermissions    map[string][]string
	AccountPermissionsRaw string
	CommandPermissions    map[string]string
	CommandPermissionsRaw string
	ReconnectMin          time.Duration
	ReconnectMax          time.Duration
	PBMPSocket            string
	BotAIURL              string
	BotAIExpert           string
	BotAITimeout          time.Duration
	BotAIHistoryMessages  int64
	BotAIConversation     bool
	Channels              []string
}

func FromEnv() Config {
	return Config{
		Server:                os.Getenv("ENGO_SERVER"),
		Nick:                  getenv("ENGO_NICK", "engo"),
		User:                  getenv("ENGO_USER", "engo"),
		RealName:              getenv("ENGO_REALNAME", "Engo IRC bot"),
		TLS:                   getenv("ENGO_TLS", "1") != "0",
		Script:                getenv("ENGO_SCRIPT", "scripts/examples/hello.tengo"),
		ScriptsDir:            os.Getenv("ENGO_SCRIPTS_DIR"),
		ScriptMaxAllocs:       int64Env("ENGO_SCRIPT_MAX_ALLOCS", 100000),
		StateDir:              getenv("ENGO_STATE_DIR", "data/state"),
		HTTPAllow:             csvEnv("ENGO_HTTP_ALLOW"),
		HTTPTimeout:           durationEnv("ENGO_HTTP_TIMEOUT", 10*time.Second),
		HTTPMaxBody:           int64Env("ENGO_HTTP_MAX_BODY", 262144),
		ScriptCapabilities:    capabilityEnv("ENGO_SCRIPT_CAPABILITIES"),
		ScriptCapabilitiesRaw: os.Getenv("ENGO_SCRIPT_CAPABILITIES"),
		SASLUsername:          os.Getenv("ENGO_SASL_USERNAME"),
		SASLPassword:          os.Getenv("ENGO_SASL_PASSWORD"),
		AllowInsecureSASL:     getenv("ENGO_ALLOW_INSECURE_SASL", "0") == "1",
		IRCCapabilities:       lowerCSVEnv("ENGO_IRC_CAPABILITIES"),
		AccountPermissions:    permissionEnv("ENGO_ACCOUNT_PERMISSIONS"),
		AccountPermissionsRaw: os.Getenv("ENGO_ACCOUNT_PERMISSIONS"),
		CommandPermissions:    commandPermissionEnv("ENGO_COMMAND_PERMISSIONS"),
		CommandPermissionsRaw: os.Getenv("ENGO_COMMAND_PERMISSIONS"),
		ReconnectMin:          durationEnv("ENGO_RECONNECT_MIN", 2*time.Second),
		ReconnectMax:          durationEnv("ENGO_RECONNECT_MAX", 2*time.Minute),
		PBMPSocket:            os.Getenv("ENGO_PBMP_SOCKET"),
		BotAIURL:              strings.TrimSpace(os.Getenv("ENGO_BOTAI_URL")),
		BotAIExpert:           getenv("ENGO_BOTAI_EXPERT", "auto"),
		BotAITimeout:          durationEnv("ENGO_BOTAI_TIMEOUT", 30*time.Second),
		BotAIHistoryMessages:  int64Env("ENGO_BOTAI_HISTORY_MESSAGES", 10),
		BotAIConversation:     getenv("ENGO_BOTAI_CONVERSATION", "0") == "1",
		Channels:              csvEnv("ENGO_CHANNELS"),
	}
}

func (c Config) Validate() error {
	if err := validateCapabilities(c.ScriptCapabilitiesRaw); err != nil {
		return err
	}
	if err := validateScriptCapabilityNames(c.ScriptCapabilitiesRaw); err != nil {
		return err
	}
	if err := validatePermissions(c.AccountPermissionsRaw); err != nil {
		return err
	}
	if err := validateCommandPermissions(c.CommandPermissionsRaw); err != nil {
		return err
	}
	if err := validatePermissionReferences(c.AccountPermissions, c.CommandPermissions); err != nil {
		return err
	}
	if c.ScriptMaxAllocs <= 0 {
		return fmt.Errorf("ENGO_SCRIPT_MAX_ALLOCS must be positive")
	}
	if c.HTTPTimeout <= 0 || c.HTTPMaxBody <= 0 {
		return fmt.Errorf("invalid HTTP capability limits")
	}
	if c.Server == "" {
		return nil
	}
	if c.Nick == "" || c.User == "" || c.RealName == "" {
		return fmt.Errorf("nick, user and real name must not be empty")
	}
	if (c.SASLUsername == "") != (c.SASLPassword == "") {
		return fmt.Errorf("ENGO_SASL_USERNAME and ENGO_SASL_PASSWORD must be set together")
	}
	if c.SASLUsername != "" && !c.TLS && !c.AllowInsecureSASL {
		return fmt.Errorf("SASL credentials require TLS; set ENGO_ALLOW_INSECURE_SASL=1 to override")
	}
	seenIRCCaps := make(map[string]bool)
	for _, capability := range c.IRCCapabilities {
		name := strings.ToLower(strings.TrimSpace(capability))
		if err := validateIRCCapability(name); err != nil {
			return err
		}
		if seenIRCCaps[name] {
			return fmt.Errorf("duplicate ENGO_IRC_CAPABILITIES capability %q", name)
		}
		seenIRCCaps[name] = true
	}
	if c.BotAIURL != "" && c.BotAITimeout <= 0 {
		return fmt.Errorf("ENGO_BOTAI_TIMEOUT must be positive")
	}
	if c.BotAIURL != "" && (c.BotAIHistoryMessages < 0 || c.BotAIHistoryMessages > 20) {
		return fmt.Errorf("ENGO_BOTAI_HISTORY_MESSAGES must be between 0 and 20")
	}
	if c.BotAIURL != "" && strings.TrimSpace(c.BotAIExpert) == "" {
		return fmt.Errorf("ENGO_BOTAI_EXPERT must not be empty")
	}
	if c.ReconnectMin <= 0 || c.ReconnectMax < c.ReconnectMin {
		return fmt.Errorf("invalid reconnect interval")
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func durationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if seconds, err := strconv.Atoi(v); err == nil {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}
func int64Env(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func csvEnv(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func lowerCSVEnv(key string) []string {
	out := csvEnv(key)
	for i := range out {
		out[i] = strings.ToLower(out[i])
	}
	return out
}

func capabilityEnv(key string) map[string][]string {
	out := make(map[string][]string)
	for _, entry := range csvEnv(key) {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		for _, capability := range strings.Split(parts[1], "+") {
			if capability = strings.ToLower(strings.TrimSpace(capability)); capability != "" {
				out[name] = append(out[name], capability)
			}
		}
	}
	return out
}

func validateCapabilities(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	seen := make(map[string]bool)
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("invalid ENGO_SCRIPT_CAPABILITIES entry %q", entry)
		}
		name := strings.TrimSpace(parts[0])
		if !strings.HasSuffix(name, ".tengo") {
			return fmt.Errorf("invalid script name in ENGO_SCRIPT_CAPABILITIES: %q", parts[0])
		}
		if seen[name] {
			return fmt.Errorf("duplicate script in ENGO_SCRIPT_CAPABILITIES: %q", name)
		}
		seen[name] = true
		capsSeen := make(map[string]bool)
		for _, capability := range strings.Split(parts[1], "+") {
			capability = strings.ToLower(strings.TrimSpace(capability))
			if capability == "" {
				return fmt.Errorf("empty capability in ENGO_SCRIPT_CAPABILITIES entry %q", entry)
			}
			if capability != "http" {
				return fmt.Errorf("unknown script capability %q", capability)
			}
			if capsSeen[capability] {
				return fmt.Errorf("duplicate capability %q for script %q", capability, name)
			}
			capsSeen[capability] = true
		}
	}
	return nil
}

func validateIRCCapability(capability string) error {
	switch strings.ToLower(strings.TrimSpace(capability)) {
	case "account-notify", "account-tag", "extended-join", "server-time":
		return nil
	case "sasl":
		return fmt.Errorf("sasl is managed automatically from ENGO_SASL_USERNAME/ENGO_SASL_PASSWORD")
	default:
		return fmt.Errorf("unsupported ENGO_IRC_CAPABILITIES capability %q", capability)
	}
}

func permissionEnv(key string) map[string][]string {
	out := make(map[string][]string)
	for _, entry := range csvEnv(key) {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			continue
		}
		account := strings.ToLower(strings.TrimSpace(parts[0]))
		if account == "" {
			continue
		}
		for _, permission := range strings.Split(parts[1], "+") {
			if permission = strings.ToLower(strings.TrimSpace(permission)); permission != "" {
				out[account] = append(out[account], permission)
			}
		}
	}
	return out
}
func validatePermissions(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	seen := make(map[string]bool)
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(entry), ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return fmt.Errorf("invalid ENGO_ACCOUNT_PERMISSIONS entry %q", entry)
		}
		account := strings.ToLower(strings.TrimSpace(parts[0]))
		if err := validateAccountName(account); err != nil {
			return err
		}
		if seen[account] {
			return fmt.Errorf("duplicate account in ENGO_ACCOUNT_PERMISSIONS: %q", account)
		}
		seen[account] = true
		permissionsSeen := make(map[string]bool)
		for _, p := range strings.Split(parts[1], "+") {
			p = strings.ToLower(strings.TrimSpace(p))
			if p == "" {
				return fmt.Errorf("empty permission in ENGO_ACCOUNT_PERMISSIONS entry %q", entry)
			}
			if err := validatePermissionName(p); err != nil {
				return err
			}
			if permissionsSeen[p] {
				return fmt.Errorf("duplicate permission %q for account %q", p, account)
			}
			permissionsSeen[p] = true
		}
	}
	return nil
}

func commandPermissionEnv(key string) map[string]string {
	out := make(map[string]string)
	for _, entry := range csvEnv(key) {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) == 2 {
			out[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.ToLower(strings.TrimSpace(parts[1]))
		}
	}
	return out
}
func validateCommandPermissions(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	seen := make(map[string]bool)
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(entry), ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" || strings.Contains(parts[1], "+") {
			return fmt.Errorf("invalid ENGO_COMMAND_PERMISSIONS entry %q", entry)
		}
		command := strings.ToLower(strings.TrimSpace(parts[0]))
		if err := validateCommandName(command); err != nil {
			return err
		}
		if seen[command] {
			return fmt.Errorf("duplicate command in ENGO_COMMAND_PERMISSIONS: %q", command)
		}
		seen[command] = true
		if err := validatePermissionName(strings.ToLower(strings.TrimSpace(parts[1]))); err != nil {
			return err
		}
	}
	return nil
}
func validatePermissionName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 || strings.Contains(name, "..") {
		return fmt.Errorf("invalid permission name %q", name)
	}
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || (r == '.' && i > 0 && i < len(name)-1) {
			continue
		}
		return fmt.Errorf("invalid permission name %q", name)
	}
	return nil
}

func validateAccountName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return fmt.Errorf("invalid IRC account name %q", name)
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("-_.", r) {
			continue
		}
		return fmt.Errorf("invalid IRC account name %q", name)
	}
	return nil
}
func validateCommandName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return fmt.Errorf("invalid command name %q", name)
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("invalid command name %q", name)
	}
	return nil
}

func validateScriptCapabilityNames(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, entry := range strings.Split(raw, ",") {
		name := strings.TrimSpace(strings.SplitN(entry, ":", 2)[0])
		if strings.ContainsAny(name, "/\\") || name == "." || name == ".." || strings.Contains(name, "..") {
			return fmt.Errorf("invalid script name in ENGO_SCRIPT_CAPABILITIES: %q", name)
		}
		base := strings.TrimSuffix(name, ".tengo")
		if base == "" {
			return fmt.Errorf("invalid script name in ENGO_SCRIPT_CAPABILITIES: %q", name)
		}
		for _, r := range base {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
				continue
			}
			return fmt.Errorf("invalid script name in ENGO_SCRIPT_CAPABILITIES: %q", name)
		}
	}
	return nil
}

func validatePermissionReferences(accounts map[string][]string, commands map[string]string) error {
	declared := make(map[string]bool)
	for _, permissions := range accounts {
		for _, permission := range permissions {
			declared[strings.ToLower(strings.TrimSpace(permission))] = true
		}
	}
	for command, permission := range commands {
		permission = strings.ToLower(strings.TrimSpace(permission))
		if permission != "" && !declared[permission] {
			return fmt.Errorf("ENGO_COMMAND_PERMISSIONS command %q references undeclared permission %q", command, permission)
		}
	}
	return nil
}

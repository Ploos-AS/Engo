package irc

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Config struct {
	Server       string
	Nick         string
	User         string
	RealName     string
	TLS          bool
	SASLUsername string
	SASLPassword string
	Capabilities []string
	Channels     []string
}

type Client struct {
	conn                net.Conn
	cfg                 Config
	onMessage           func(Message) error
	registrationTimeout time.Duration
}

func Dial(cfg Config) (*Client, error) {
	if cfg.Server == "" {
		return nil, fmt.Errorf("server is required")
	}
	for name, value := range map[string]string{"nick": cfg.Nick, "user": cfg.User, "real name": cfg.RealName} {
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("%s contains IRC line breaks", name)
		}
	}

	d := net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	var (
		conn net.Conn
		err  error
	)
	if cfg.TLS {
		host, _, splitErr := net.SplitHostPort(cfg.Server)
		if splitErr != nil {
			return nil, fmt.Errorf("invalid server address: %w", splitErr)
		}
		conn, err = tls.DialWithDialer(&d, "tcp", cfg.Server, &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		})
	} else {
		conn, err = d.Dial("tcp", cfg.Server)
	}
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	c := &Client{conn: conn, cfg: cfg, registrationTimeout: 30 * time.Second}
	if err := c.register(); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) register() error {
	if err := c.writef("CAP LS 302"); err != nil {
		return err
	}
	if err := c.writef("NICK %s", c.cfg.Nick); err != nil {
		return err
	}
	return c.writef("USER %s 0 * :%s", c.cfg.User, c.cfg.RealName)
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) OnMessage(handler func(Message) error) { c.onMessage = handler }

func (c *Client) Say(target, text string) error {
	return c.writef("PRIVMSG %s :%s", sanitizeTarget(target), sanitizeText(text))
}
func (c *Client) Notice(target, text string) error {
	return c.writef("NOTICE %s :%s", sanitizeTarget(target), sanitizeText(text))
}
func (c *Client) Action(target, text string) error {
	return c.writef("PRIVMSG %s :\u0001ACTION %s\u0001", sanitizeTarget(target), sanitizeText(text))
}

func (c *Client) Run() error {
	r := bufio.NewReader(c.conn)
	saslWanted := c.cfg.SASLUsername != ""
	requestedCaps := normalizeCapabilities(c.cfg.Capabilities)
	if saslWanted && !containsCapability(requestedCaps, "sasl") {
		requestedCaps = append(requestedCaps, "sasl")
	}
	capRequestSent := false
	saslStarted := false
	capEnded := false
	saslComplete := false
	ackedCaps := make(map[string]bool)
	timeout := c.registrationTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	var capLS []string

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return fmt.Errorf("registration timed out: %w", err)
			}
			if err == io.EOF {
				return io.EOF
			}
			return fmt.Errorf("read IRC: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")

		if strings.HasPrefix(line, "PING ") {
			if err := c.writef("PONG %s", strings.TrimPrefix(line, "PING ")); err != nil {
				return err
			}
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		if hasToken(fields, "CAP") && strings.Contains(line, " LS ") {
			if capRequestSent || capEnded {
				return fmt.Errorf("unexpected CAP LS after capability negotiation started")
			}
			capLS = append(capLS, capabilityNames(line)...)
			if capLSContinues(fields) {
				continue
			}
			for _, capability := range requestedCaps {
				if !containsCapability(capLS, capability) {
					return fmt.Errorf("server does not advertise requested capability %q", capability)
				}
			}
			if len(requestedCaps) > 0 && !capRequestSent {
				if err := c.writef("CAP REQ :%s", strings.Join(requestedCaps, " ")); err != nil {
					return err
				}
				capRequestSent = true
			} else if len(requestedCaps) == 0 && !capEnded {
				if err := c.writef("CAP END"); err != nil {
					return err
				}
				capEnded = true
			}
			continue
		}

		if hasToken(fields, "CAP") && strings.Contains(line, " DEL ") {
			if !capRequestSent {
				return fmt.Errorf("unexpected CAP DEL before CAP REQ")
			}
			for _, name := range capabilityNames(line) {
				delete(ackedCaps, name)
				if containsCapability(requestedCaps, name) {
					if name == "sasl" && saslComplete {
						continue
					}
					return fmt.Errorf("server removed requested IRC capability %q", name)
				}
			}
			continue
		}

		if hasToken(fields, "CAP") && strings.Contains(line, " NEW ") {
			// NEW only advertises availability. Requested capabilities are
			// negotiated during registration and are not silently enabled here.
			// Before registration completes, accepting NEW would mutate the
			// capability state outside the LS/REQ/ACK transaction.
			if !capEnded {
				return fmt.Errorf("unexpected CAP NEW before CAP END")
			}
			continue
		}

		if hasToken(fields, "CAP") && strings.Contains(line, " NAK ") {
			if capEnded {
				return fmt.Errorf("unexpected CAP NAK after CAP END")
			}
			if !capRequestSent {
				return fmt.Errorf("unexpected CAP NAK before CAP REQ")
			}
			rejected := capabilityNames(line)
			var required []string
			for _, name := range rejected {
				if containsCapability(requestedCaps, name) {
					required = append(required, name)
				}
			}
			if containsCapability(required, "sasl") && saslWanted {
				return fmt.Errorf("server rejected requested SASL capability")
			}
			if len(required) > 0 {
				return fmt.Errorf("server rejected requested IRC capability: %s", strings.Join(required, ", "))
			}
			continue
		}

		if hasToken(fields, "CAP") && strings.Contains(line, " ACK ") {
			if capEnded {
				return fmt.Errorf("unexpected CAP ACK after CAP END")
			}
			if !capRequestSent {
				return fmt.Errorf("unexpected CAP ACK before CAP REQ")
			}
			for _, token := range capabilityTokens(line) {
				name := capabilityName(token)
				if name == "" {
					continue
				}
				if !containsCapability(requestedCaps, name) {
					return fmt.Errorf("server ACKed unrequested IRC capability %q", name)
				}
				if strings.HasPrefix(token, "-") {
					delete(ackedCaps, name)
					if containsCapability(requestedCaps, name) {
						return fmt.Errorf("server disabled requested IRC capability %q", name)
					}
					continue
				}
				ackedCaps[name] = true
			}
			allAcked := true
			for _, capability := range requestedCaps {
				if !ackedCaps[capability] {
					allAcked = false
					break
				}
			}
			if !allAcked {
				continue
			}
			if saslWanted && ackedCaps["sasl"] && !saslStarted {
				if err := c.writef("AUTHENTICATE PLAIN"); err != nil {
					return err
				}
				saslStarted = true
				continue
			}
			if !saslWanted && !capEnded {
				if err := c.writef("CAP END"); err != nil {
					return err
				}
				capEnded = true
			}
			continue
		}

		if len(fields) >= 2 && fields[0] == "AUTHENTICATE" && fields[1] == "+" && saslStarted {
			payload := "\x00" + c.cfg.SASLUsername + "\x00" + c.cfg.SASLPassword
			encoded := base64.StdEncoding.EncodeToString([]byte(payload))
			for _, chunk := range authenticateChunks(encoded) {
				if err := c.writef("AUTHENTICATE %s", chunk); err != nil {
					return err
				}
			}
			continue
		}

		if c.onMessage != nil {
			if err := c.onMessage(ParseMessage(line)); err != nil {
				return fmt.Errorf("message handler: %w", err)
			}
		}

		numeric := numericCommand(fields)
		switch numeric {
		case "903":
			if !saslWanted || !saslStarted {
				return fmt.Errorf("unexpected SASL success numeric 903")
			}
			saslComplete = true
			if !capEnded {
				if err := c.writef("CAP END"); err != nil {
					return err
				}
				capEnded = true
			}
		case "904", "905", "906", "907":
			if !saslWanted || !saslStarted {
				return fmt.Errorf("unexpected SASL failure numeric %s", numeric)
			}
			return fmt.Errorf("SASL authentication failed (%s)", numeric)
		case "001":
			if !capEnded {
				return fmt.Errorf("registered before capability negotiation completed")
			}
			if saslWanted && !saslComplete {
				return fmt.Errorf("registered before SASL completed")
			}
			for _, channel := range c.cfg.Channels {
				if err := c.writef("JOIN %s", sanitizeTarget(channel)); err != nil {
					return err
				}
			}
			_ = c.conn.SetReadDeadline(time.Time{})
		}
	}
}

func (c *Client) writef(format string, args ...any) error {
	if _, err := fmt.Fprintf(c.conn, format+"\r\n", args...); err != nil {
		return fmt.Errorf("write IRC: %w", err)
	}
	return nil
}

func hasToken(fields []string, token string) bool {
	for _, field := range fields {
		if field == token {
			return true
		}
	}
	return false
}

func capabilityTokens(line string) []string {
	i := strings.LastIndex(line, " :")
	if i < 0 {
		return nil
	}
	return strings.Fields(line[i+2:])
}

func capabilityNames(line string) []string {
	i := strings.LastIndex(line, " :")
	if i < 0 {
		return nil
	}
	var out []string
	for _, cap := range capabilityTokens(line) {
		if name := capabilityName(cap); name != "" {
			out = append(out, name)
		}
	}
	return out
}

func capabilityName(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	for len(token) > 0 && strings.ContainsRune("-~=", rune(token[0])) {
		token = token[1:]
	}
	if token == "" {
		return ""
	}
	return strings.ToLower(strings.SplitN(token, "=", 2)[0])
}

func normalizeCapabilities(caps []string) []string {
	out := make([]string, 0, len(caps))
	seen := make(map[string]bool)
	for _, capability := range caps {
		name := strings.ToLower(strings.TrimSpace(capability))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func containsCapability(caps []string, capability string) bool {
	for _, cap := range caps {
		if cap == capability {
			return true
		}
	}
	return false
}

func capLSContinues(fields []string) bool {
	for i, field := range fields {
		if field == "LS" && i+1 < len(fields) && fields[i+1] == "*" {
			return true
		}
	}
	return false
}

func capabilityListed(line, capability string) bool {
	i := strings.LastIndex(line, " :")
	if i < 0 {
		return false
	}
	for _, cap := range strings.Fields(line[i+2:]) {
		if capabilityName(cap) == capability {
			return true
		}
	}
	return false
}

func numericCommand(fields []string) string {
	for _, field := range fields {
		if len(field) == 3 && field[0] >= '0' && field[0] <= '9' &&
			field[1] >= '0' && field[1] <= '9' && field[2] >= '0' && field[2] <= '9' {
			return field
		}
	}
	return ""
}

func sanitizeTarget(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

func sanitizeText(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.ReplaceAll(s, "\n", " ")
}

func authenticateChunks(encoded string) []string {
	if encoded == "" {
		return []string{"+"}
	}
	chunks := make([]string, 0, (len(encoded)+399)/400+1)
	for len(encoded) > 400 {
		chunks = append(chunks, encoded[:400])
		encoded = encoded[400:]
	}
	if encoded != "" {
		chunks = append(chunks, encoded)
	}
	if len(encoded) == 400 || len(chunks) > 0 && len(chunks[len(chunks)-1]) == 400 {
		chunks = append(chunks, "+")
	}
	return chunks
}

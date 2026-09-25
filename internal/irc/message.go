package irc

import "strings"

// Message is a parsed IRC message suitable for delivery to the bot layer.
type Message struct {
	Raw     string
	Prefix  string
	Nick    string
	Command string
	Params  []string
	Trailing string
}

func ParseMessage(line string) Message {
	m := Message{Raw: line}
	rest := line
	if strings.HasPrefix(rest, ":") {
		if i := strings.IndexByte(rest, ' '); i >= 0 {
			m.Prefix = rest[1:i]
			rest = strings.TrimLeft(rest[i+1:], " ")
			m.Nick = m.Prefix
			if j := strings.IndexByte(m.Nick, '!'); j >= 0 {
				m.Nick = m.Nick[:j]
			}
		}
	}
	if i := strings.Index(rest, " :"); i >= 0 {
		m.Trailing = rest[i+2:]
		rest = rest[:i]
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return m
	}
	m.Command = strings.ToUpper(fields[0])
	if len(fields) > 1 {
		m.Params = append([]string(nil), fields[1:]...)
	}
	return m
}

func (m Message) Target() string {
	if len(m.Params) == 0 {
		return ""
	}
	return m.Params[0]
}

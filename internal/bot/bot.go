package bot

import (
	"fmt"
	"strings"

	"github.com/Ploos-AS/Engo/internal/irc"
)

type Sender interface {
	Say(target, text string) error
	Notice(target, text string) error
	Action(target, text string) error
}

type Event struct {
	Name    string
	Nick    string
	Target  string
	Text    string
	Command string
	Args    []string
	Message irc.Message
}

type Handler func(Event) error

type Bot struct {
	sender   Sender
	handlers map[string][]Handler
	commands map[string]Handler
	prefix   string
}

func New(sender Sender) *Bot {
	return &Bot{
		sender: sender,
		handlers: make(map[string][]Handler),
		commands: make(map[string]Handler,
		),
		prefix: "!",
	}
}

func (b *Bot) On(name string, handler Handler) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name != "" && handler != nil {
		b.handlers[name] = append(b.handlers[name], handler)
	}
}

func (b *Bot) Command(name string, handler Handler) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name != "" && handler != nil {
		b.commands[name] = handler
	}
}

func (b *Bot) Handle(m irc.Message) error {
	ev := eventFromMessage(m)
	if ev.Name == "" {
		return nil
	}
	for _, h := range b.handlers[ev.Name] {
		if err := h(ev); err != nil {
			return err
		}
	}
	if ev.Name == "message" && strings.HasPrefix(ev.Text, b.prefix) {
		fields := strings.Fields(strings.TrimPrefix(ev.Text, b.prefix))
		if len(fields) > 0 {
			name := strings.ToLower(fields[0])
			if h := b.commands[name]; h != nil {
				ev.Command = name
				ev.Args = fields[1:]
				if err := h(ev); err != nil {
					return fmt.Errorf("command %s: %w", name, err)
				}
			}
		}
	}
	return nil
}

func (b *Bot) Say(target, text string) error { return b.sender.Say(target, text) }
func (b *Bot) Notice(target, text string) error { return b.sender.Notice(target, text) }
func (b *Bot) Action(target, text string) error { return b.sender.Action(target, text) }

func eventFromMessage(m irc.Message) Event {
	ev := Event{Nick: m.Nick, Target: m.Target(), Text: m.Trailing, Message: m}
	switch m.Command {
	case "PRIVMSG":
		ev.Name = "message"
	case "JOIN":
		ev.Name = "join"
		if ev.Target == "" {
			ev.Target = m.Trailing
		}
	case "PART":
		ev.Name = "part"
	case "NOTICE":
		ev.Name = "notice"
	default:
		if m.Command != "" {
			ev.Name = strings.ToLower(m.Command)
		}
	}
	return ev
}

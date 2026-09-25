package script

import (
	"fmt"
	"os"
	"strings"

	"github.com/Ploos-AS/Engo/internal/bot"
	"github.com/d5/tengo/v2"
)

type Runtime struct {
	path string
	bot  *bot.Bot
	src  []byte
}

func New(path string, b *bot.Bot) *Runtime {
	return &Runtime{path: path, bot: b}
}

func (r *Runtime) Load() error {
	src, err := os.ReadFile(r.path)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}
	r.src = src

	// Registration mode evaluates the script once and records handlers.
	s := tengo.NewScript(src)
	if err := s.Add("bot", &tengo.UserFunction{Name: "bot", Value: r.registrationCall}); err != nil {
		return fmt.Errorf("add bot API: %w", err)
	}
	if err := s.Add("event", eventObject(bot.Event{})); err != nil {
		return fmt.Errorf("add event: %w", err)
	}
	if _, err := s.Run(); err != nil {
		return fmt.Errorf("run script: %w", err)
	}
	return nil
}

func RunFile(path string) error {
	return New(path, bot.New(discardSender{})).Load()
}

func (r *Runtime) registrationCall(args ...tengo.Object) (tengo.Object, error) {
	if len(args) < 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	op, ok := tengo.ToString(args[0])
	if !ok {
		return nil, fmt.Errorf("bot operation must be a string")
	}
	switch op {
	case "on", "command":
		if len(args) != 3 {
			return nil, tengo.ErrWrongNumArguments
		}
		name, ok := tengo.ToString(args[1])
		if !ok {
			return nil, fmt.Errorf("handler name must be a string")
		}
		handlerID, ok := tengo.ToString(args[2])
		if !ok {
			return nil, fmt.Errorf("handler id must be a string")
		}
		h := func(ev bot.Event) error { return r.runHandler(handlerID, ev) }
		if op == "on" {
			r.bot.On(name, h)
		} else {
			r.bot.Command(name, h)
		}
		return tengo.UndefinedValue, nil
	case "say", "notice", "action":
		// Sending during registration is intentionally ignored.
		return tengo.UndefinedValue, nil
	default:
		return nil, fmt.Errorf("unknown bot operation %q", op)
	}
}

func (r *Runtime) runHandler(handlerID string, ev bot.Event) error {
	s := tengo.NewScript(r.src)
	if err := s.Add("event", eventObject(ev)); err != nil {
		return err
	}
	if err := s.Add("bot", &tengo.UserFunction{Name: "bot", Value: r.eventCall(handlerID)}); err != nil {
		return err
	}
	if _, err := s.Run(); err != nil {
		return fmt.Errorf("Tengo handler %s: %w", handlerID, err)
	}
	return nil
}

func (r *Runtime) eventCall(activeHandler string) func(...tengo.Object) (tengo.Object, error) {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return nil, tengo.ErrWrongNumArguments
		}
		op, ok := tengo.ToString(args[0])
		if !ok {
			return nil, fmt.Errorf("bot operation must be a string")
		}
		switch op {
		case "on", "command":
			// Handler declarations are no-ops while processing an event.
			return tengo.UndefinedValue, nil
		case "active":
			if len(args) != 2 {
				return nil, tengo.ErrWrongNumArguments
			}
			id, _ := tengo.ToString(args[1])
			return tengo.FromInterface(id == activeHandler)
		case "say", "notice", "action":
			if len(args) != 3 {
				return nil, tengo.ErrWrongNumArguments
			}
			target, ok1 := tengo.ToString(args[1])
			text, ok2 := tengo.ToString(args[2])
			if !ok1 || !ok2 || strings.TrimSpace(target) == "" {
				return nil, fmt.Errorf("%s requires target and text strings", op)
			}
			var err error
			switch op {
			case "say":
				err = r.bot.Say(target, text)
			case "notice":
				err = r.bot.Notice(target, text)
			case "action":
				err = r.bot.Action(target, text)
			}
			if err != nil {
				return nil, err
			}
			return tengo.UndefinedValue, nil
		default:
			return nil, fmt.Errorf("unknown bot operation %q", op)
		}
	}
}

func eventObject(ev bot.Event) map[string]interface{} {
	args := make([]interface{}, len(ev.Args))
	for i, arg := range ev.Args {
		args[i] = arg
	}
	return map[string]interface{}{
		"name": ev.Name, "nick": ev.Nick, "target": ev.Target, "text": ev.Text,
		"command": ev.Command, "args": args,
	}
}

type discardSender struct{}
func (discardSender) Say(string, string) error { return nil }
func (discardSender) Notice(string, string) error { return nil }
func (discardSender) Action(string, string) error { return nil }

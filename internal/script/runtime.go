package script

import (
	"fmt"
	"os"
	"sync"

	"github.com/Ploos-AS/Engo/internal/bot"
	"github.com/d5/tengo/v2"
)

type Runtime struct {
	path string
	bot  *bot.Bot
	mu   sync.Mutex
}

func New(path string, b *bot.Bot) *Runtime {
	return &Runtime{path: path, bot: b}
}

func (r *Runtime) Load() error {
	src, err := os.ReadFile(r.path)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}

	s := tengo.NewScript(src)
	if err := s.Add("bot", &tengo.UserFunction{Name: "bot", Value: r.botCall}); err != nil {
		return fmt.Errorf("add bot API: %w", err)
	}
	if _, err := s.Run(); err != nil {
		return fmt.Errorf("run script: %w", err)
	}
	return nil
}

func RunFile(path string) error {
	return New(path, bot.New(discardSender{})).Load()
}

func (r *Runtime) botCall(args ...tengo.Object) (tengo.Object, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(args) < 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	op, ok := tengo.ToString(args[0])
	if !ok {
		return nil, tengo.ErrInvalidArgumentType{Name: "operation", Expected: "string", Found: args[0].TypeName()}
	}
	switch op {
	case "on", "command":
		if len(args) != 3 {
			return nil, tengo.ErrWrongNumArguments
		}
		name, ok := tengo.ToString(args[1])
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "name", Expected: "string", Found: args[1].TypeName()}
		}
		fn, ok := args[2].(*tengo.CompiledFunction)
		if !ok {
			return nil, tengo.ErrInvalidArgumentType{Name: "handler", Expected: "function", Found: args[2].TypeName()}
		}
		handler := r.handler(fn)
		if op == "on" {
			r.bot.On(name, handler)
		} else {
			r.bot.Command(name, handler)
		}
		return tengo.UndefinedValue, nil
	case "say", "notice", "action":
		if len(args) != 3 {
			return nil, tengo.ErrWrongNumArguments
		}
		target, _ := tengo.ToString(args[1])
		text, _ := tengo.ToString(args[2])
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

func (r *Runtime) handler(fn *tengo.CompiledFunction) bot.Handler {
	return func(ev bot.Event) error {
		// Tengo functions registered from the initial program are retained here;
		// direct invocation support will be expanded with isolated per-event VMs.
		_ = fn
		_ = ev
		return nil
	}
}

type discardSender struct{}
func (discardSender) Say(string, string) error { return nil }
func (discardSender) Notice(string, string) error { return nil }
func (discardSender) Action(string, string) error { return nil }

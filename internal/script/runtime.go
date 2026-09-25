package script

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Ploos-AS/Engo/internal/bot"
	"github.com/d5/tengo/v2"
)

type Runtime struct {
	path string
	bot *bot.Bot
	mu sync.RWMutex
	src []byte
}

func New(path string, b *bot.Bot) *Runtime { return &Runtime{path: path, bot: b} }

func (r *Runtime) Load() error { return r.Reload() }

// Reload validates a fresh script and atomically replaces the active handlers.
// If compilation/registration fails, the previous script remains active.
func (r *Runtime) Reload() error {
	src, err := os.ReadFile(r.path)
	if err != nil { return fmt.Errorf("read script: %w", err) }

	reg := bot.NewRegistry()
	register := r.registrationCall(src, &reg)
	s := tengo.NewScript(src)
	if err := s.Add("bot", &tengo.UserFunction{Name: "bot", Value: register}); err != nil { return err }
	if err := s.Add("event", eventObject(bot.Event{})); err != nil { return err }
	if _, err := s.Run(); err != nil { return fmt.Errorf("run script: %w", err) }

	r.mu.Lock()
	r.src = append([]byte(nil), src...)
	r.mu.Unlock()
	r.bot.Replace(reg)
	return nil
}

func RunFile(path string) error { return New(path, bot.New(discardSender{})).Load() }

func (r *Runtime) registrationCall(src []byte, reg *bot.Registry) func(...tengo.Object) (tengo.Object, error) {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 { return nil, tengo.ErrWrongNumArguments }
		op, ok := tengo.ToString(args[0])
		if !ok { return nil, fmt.Errorf("bot operation must be a string") }
		switch op {
		case "on", "command":
			if len(args) != 3 { return nil, tengo.ErrWrongNumArguments }
			name, ok1 := tengo.ToString(args[1])
			id, ok2 := tengo.ToString(args[2])
			if !ok1 || !ok2 { return nil, fmt.Errorf("handler name and id must be strings") }
			h := func(ev bot.Event) error { return r.runHandler(src, id, ev) }
			name = strings.ToLower(strings.TrimSpace(name))
			if name == "" { return nil, fmt.Errorf("handler name must not be empty") }
			if op == "on" { reg.Events[name] = append(reg.Events[name], h) } else { reg.Commands[name] = h }
			return tengo.UndefinedValue, nil
		case "say", "notice", "action":
			return tengo.UndefinedValue, nil
		default:
			return nil, fmt.Errorf("unknown bot operation %q", op)
		}
	}
}

func (r *Runtime) runHandler(src []byte, handlerID string, ev bot.Event) error {
	s := tengo.NewScript(src)
	if err := s.Add("event", eventObject(ev)); err != nil { return err }
	if err := s.Add("bot", &tengo.UserFunction{Name: "bot", Value: r.eventCall(handlerID)}); err != nil { return err }
	if _, err := s.Run(); err != nil { return fmt.Errorf("Tengo handler %s: %w", handlerID, err) }
	return nil
}

func (r *Runtime) eventCall(activeHandler string) func(...tengo.Object) (tengo.Object, error) {
	return func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 { return nil, tengo.ErrWrongNumArguments }
		op, ok := tengo.ToString(args[0])
		if !ok { return nil, fmt.Errorf("bot operation must be a string") }
		switch op {
		case "on", "command":
			return tengo.UndefinedValue, nil
		case "active":
			if len(args) != 2 { return nil, tengo.ErrWrongNumArguments }
			id, _ := tengo.ToString(args[1])
			return tengo.FromInterface(id == activeHandler)
		case "say", "notice", "action":
			if len(args) != 3 { return nil, tengo.ErrWrongNumArguments }
			target, ok1 := tengo.ToString(args[1]); text, ok2 := tengo.ToString(args[2])
			if !ok1 || !ok2 || strings.TrimSpace(target) == "" { return nil, fmt.Errorf("%s requires target and text strings", op) }
			var err error
			switch op { case "say": err=r.bot.Say(target,text); case "notice": err=r.bot.Notice(target,text); case "action": err=r.bot.Action(target,text) }
			if err != nil { return nil, err }
			return tengo.UndefinedValue, nil
		default:
			return nil, fmt.Errorf("unknown bot operation %q", op)
		}
	}
}

func eventObject(ev bot.Event) map[string]interface{} {
	args:=make([]interface{},len(ev.Args)); for i,arg:=range ev.Args { args[i]=arg }
	return map[string]interface{}{"name":ev.Name,"nick":ev.Nick,"target":ev.Target,"text":ev.Text,"command":ev.Command,"args":args}
}

type discardSender struct{}
func (discardSender) Say(string,string) error{return nil}
func (discardSender) Notice(string,string) error{return nil}
func (discardSender) Action(string,string) error{return nil}

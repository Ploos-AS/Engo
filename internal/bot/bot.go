package bot

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Ploos-AS/Engo/internal/irc"
)

type Sender interface {
	Say(target, text string) error
	Notice(target, text string) error
	Action(target, text string) error
}

type Event struct {
	Name string
	Nick string
	Target string
	Text string
	Command string
	Args []string
	Account string
	RealName string
	Message irc.Message
}

type Handler func(Event) error

type Registry struct {
	Events map[string][]Handler
	Commands map[string]Handler
}

type Bot struct {
	sender Sender
	mu sync.RWMutex
	handlers map[string][]Handler
	commands map[string]Handler
	prefix string
	accounts map[string]string
}

func New(sender Sender) *Bot {
	return &Bot{sender: sender, handlers: make(map[string][]Handler), commands: make(map[string]Handler), prefix: "!", accounts: make(map[string]string)}
}

func NewRegistry() Registry {
	return Registry{Events: make(map[string][]Handler), Commands: make(map[string]Handler)}
}

func (b *Bot) Replace(reg Registry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = reg.Events
	b.commands = reg.Commands
}

func (b *Bot) On(name string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	if name != "" && handler != nil { b.handlers[name] = append(b.handlers[name], handler) }
}

func (b *Bot) Command(name string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	if name != "" && handler != nil { b.commands[name] = handler }
}

func (b *Bot) Handle(m irc.Message) error {
	b.mu.Lock()
	account:=b.accounts[m.Nick]
	if tagged,ok:=m.Tags["account"];ok {
		account=tagged
		if account==""||account=="*"{account="";delete(b.accounts,m.Nick)}else{b.accounts[m.Nick]=account}
	}
	switch m.Command {
	case "ACCOUNT":
		account="";if len(m.Params)>0&&m.Params[0]!="*"{account=m.Params[0]};if account==""{delete(b.accounts,m.Nick)}else{b.accounts[m.Nick]=account}
	case "JOIN":
		if len(m.Params)>=2 { account=m.Params[1]; if account=="*"{account=""}; if account==""{delete(b.accounts,m.Nick)}else{b.accounts[m.Nick]=account} }
	case "NICK":
		newNick:=m.Trailing;if newNick==""&&len(m.Params)>0{newNick=m.Params[0]};if newNick!=""&&account!=""{delete(b.accounts,m.Nick);b.accounts[newNick]=account}
	case "QUIT":
		delete(b.accounts,m.Nick)
	}
	b.mu.Unlock()
	ev := eventFromMessage(m)
	if ev.Account==""{ev.Account=account}
	if ev.Name == "" { return nil }

	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[ev.Name]...)
	var command Handler
	var commandName string
	var commandArgs []string
	if ev.Name == "message" && strings.HasPrefix(ev.Text, b.prefix) {
		fields := strings.Fields(strings.TrimPrefix(ev.Text, b.prefix))
		if len(fields) > 0 {
			commandName = strings.ToLower(fields[0])
			commandArgs = append([]string(nil), fields[1:]...)
			command = b.commands[commandName]
		}
	}
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ev); err != nil {
			// A broken script handler must not tear down the IRC connection.
			fmt.Printf("engo: event handler %s failed: %v\n", ev.Name, err)
		}
	}
	if command != nil {
		ev.Command, ev.Args = commandName, commandArgs
		if err := command(ev); err != nil {
			fmt.Printf("engo: command %s failed: %v\n", commandName, err)
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
	case "PRIVMSG": ev.Name = "message"
	case "JOIN":
		ev.Name = "join"
		if ev.Target == "" { ev.Target = m.Trailing }
		if len(m.Params)>=2 { ev.Account=m.Params[1]; if ev.Account=="*"{ev.Account=""} }
		if len(m.Params)>=3 { ev.RealName=m.Params[2] } else if len(m.Params)>=2 { ev.RealName=m.Trailing }
	case "PART": ev.Name = "part"
	case "NOTICE": ev.Name = "notice"
	case "ACCOUNT":
		ev.Name = "account"
		if len(m.Params)>0 { ev.Account=m.Params[0]; if ev.Account=="*"{ev.Account=""} } else { ev.Account=m.Trailing; if ev.Account=="*"{ev.Account=""} }
	default:
		if m.Command != "" { ev.Name = strings.ToLower(m.Command) }
	}
	return ev
}

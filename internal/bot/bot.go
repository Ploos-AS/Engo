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
	AccountVerified bool
	RealName string
	Message irc.Message
}

type Handler func(Event) error

type accountIdentity struct { account, userhost string }

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
	accounts map[string]accountIdentity
	caseMapping string
}

func New(sender Sender) *Bot {
	return &Bot{sender: sender, handlers: make(map[string][]Handler), commands: make(map[string]Handler), prefix: "!", accounts: make(map[string]accountIdentity), caseMapping: "rfc1459"}
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
	if m.Command=="005"{b.applyISupport(m)}
	nickKey:=ircNickKey(m.Nick,b.caseMapping)
	identity:=b.accounts[nickKey]
	currentUserhost:=messageUserhost(m)
	if identity.account!="" {
		// Cached authorization identity is only reusable when the message carries
		// the same user@host. Missing or changed provenance fails closed.
		if identity.userhost==""||currentUserhost==""||identity.userhost!=currentUserhost {
			delete(b.accounts,nickKey)
			identity=accountIdentity{}
		}
	}
	account:=identity.account
	accountVerified:=account!=""
	if tagged,ok:=m.Tags["account"];ok {
		previous:=identity
		account=tagged
		accountVerified=account!=""&&account!="*"&&currentUserhost!=""&&(previous.account==""||previous.userhost==currentUserhost)
		if !accountVerified{account="";delete(b.accounts,nickKey)}else{b.accounts[nickKey]=accountIdentity{account:account,userhost:currentUserhost}}
	}
	switch m.Command {
	case "ACCOUNT":
		account="";accountVerified=false;if len(m.Params)>0&&m.Params[0]!="*"&&currentUserhost!=""{account=m.Params[0];accountVerified=true};if account==""{delete(b.accounts,nickKey)}else{b.accounts[nickKey]=accountIdentity{account:account,userhost:currentUserhost}}
	case "JOIN":
		if len(m.Params)>=2 { account=m.Params[1]; accountVerified=account!=""&&account!="*"&&currentUserhost!=""; if !accountVerified{account=""}; if account==""{delete(b.accounts,nickKey)}else{b.accounts[nickKey]=accountIdentity{account:account,userhost:currentUserhost}} }
	case "NICK":
		newNick:=m.Trailing;if newNick==""&&len(m.Params)>0{newNick=m.Params[0]};if newNick!=""&&account!=""&&accountVerified&&currentUserhost!=""{delete(b.accounts,nickKey);b.accounts[ircNickKey(newNick,b.caseMapping)]=accountIdentity{account:account,userhost:currentUserhost}}else{delete(b.accounts,nickKey)}
	case "QUIT":
		delete(b.accounts,nickKey)
	case "KICK":
		// A KICK describes another nick leaving a channel. Account identity is
		// network-scoped and may still be valid in other shared channels, so do
		// not revoke it here; provenance is rechecked on every later message.
	}
	b.mu.Unlock()
	ev := eventFromMessage(m)
	if !accountVerified{ev.Account=""}else{ev.Account=account;ev.AccountVerified=true}
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

func (b *Bot) applyISupport(m irc.Message){
	for _,p:=range m.Params{
		if strings.HasPrefix(strings.ToUpper(p),"CASEMAPPING="){
			v:=strings.ToLower(strings.TrimSpace(strings.SplitN(p,"=",2)[1]))
			switch v{
			case "ascii","rfc1459","strict-rfc1459":
				if v!=b.caseMapping{
					// Identity cache keys depend on CASEMAPPING. Fail closed rather
					// than risk granting permissions through a stale nick mapping.
					b.accounts=make(map[string]accountIdentity)
					b.caseMapping=v
				}
			}
		}
	}
}
func ircNickKey(nick,caseMapping string)string{
	var b strings.Builder;b.Grow(len(nick))
	for _,r:=range nick{
		if r>='A'&&r<='Z'{r+=32}
		if caseMapping!="ascii"{
			switch r{case '[':r='{';case ']':r='}';case '\\':r='|'}
			if caseMapping=="rfc1459"&&r=='^'{r='~'}
		}
		b.WriteRune(r)
	}
	return b.String()
}

func messageUserhost(m irc.Message) string {
	if i:=strings.IndexByte(m.Prefix,'!');i>=0&&i+1<len(m.Prefix){return m.Prefix[i+1:]}
	return ""
}

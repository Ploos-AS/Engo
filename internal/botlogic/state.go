package botlogic

import (
	"sort"
	"strings"
	"sync"
)

type IRCState struct {
	mu        sync.Mutex
	channels  map[string]map[string]bool
	accounts  map[string]string
	operators map[string]map[string]bool
	voiced    map[string]map[string]bool
}

func NewIRCState() *IRCState {
	return &IRCState{channels: map[string]map[string]bool{}, accounts: map[string]string{}, operators: map[string]map[string]bool{}, voiced: map[string]map[string]bool{}}
}

func (s *IRCState) Observe(command, nick, target, account string, verified bool, params []string, trailing string) []FactOperation {
	s.mu.Lock()
	defer s.mu.Unlock()
	command = strings.ToUpper(strings.TrimSpace(command))
	nick = strings.ToLower(strings.TrimSpace(nick))
	target = strings.ToLower(strings.TrimSpace(target))
	account = strings.ToLower(strings.TrimSpace(account))
	f := func(op, p string, args ...string) FactOperation {
		return FactOperation{Op: op, Fact: Fact{Predicate: p, Args: args}}
	}
	var ops []FactOperation
	switch command {
	case "JOIN":
		if nick == "" || target == "" {
			return nil
		}
		if s.channels[target] == nil {
			s.channels[target] = map[string]bool{}
		}
		if !s.channels[target][nick] {
			s.channels[target][nick] = true
			ops = append(ops, f("assert", "channel_member", target, nick))
		}
		ops = append(ops, f("assert", "online", nick))
		if verified && account != "" {
			if old := s.accounts[nick]; old != "" && old != account {
				ops = append(ops, f("retract", "authenticated", nick, old))
			}
			s.accounts[nick] = account
			ops = append(ops, f("assert", "authenticated", nick, account))
		}
	case "PART":
		if target != "" && nick != "" && s.channels[target] != nil && s.channels[target][nick] {
			delete(s.channels[target], nick)
			ops = append(ops, f("retract", "channel_member", target, nick))
			ops = append(ops, s.removeRoles(target, nick, f)...)
		}
	case "KICK":
		if target == "" || len(params) < 2 {
			return nil
		}
		victim := strings.ToLower(params[1])
		if s.channels[target] != nil && s.channels[target][victim] {
			delete(s.channels[target], victim)
			ops = append(ops, f("retract", "channel_member", target, victim))
			ops = append(ops, s.removeRoles(target, victim, f)...)
		}
	case "NICK":
		if nick == "" {
			return nil
		}
		newNick := strings.ToLower(strings.TrimSpace(trailing))
		if newNick == "" && len(params) > 0 {
			newNick = strings.ToLower(params[0])
		}
		if newNick == "" {
			return nil
		}
		for ch, members := range s.channels {
			if members[nick] {
				delete(members, nick)
				members[newNick] = true
				ops = append(ops, f("retract", "channel_member", ch, nick), f("assert", "channel_member", ch, newNick))
				ops = append(ops, s.moveRoles(ch, nick, newNick, f)...)
			}
		}
		ops = append(ops, f("retract", "online", nick), f("assert", "online", newNick))
		if a := s.accounts[nick]; a != "" {
			delete(s.accounts, nick)
			s.accounts[newNick] = a
			ops = append(ops, f("retract", "authenticated", nick, a), f("assert", "authenticated", newNick, a))
		}
	case "QUIT":
		if nick == "" {
			return nil
		}
		chs := make([]string, 0)
		for ch, members := range s.channels {
			if members[nick] {
				chs = append(chs, ch)
			}
		}
		sort.Strings(chs)
		for _, ch := range chs {
			delete(s.channels[ch], nick)
			ops = append(ops, f("retract", "channel_member", ch, nick))
			ops = append(ops, s.removeRoles(ch, nick, f)...)
		}
		ops = append(ops, f("retract", "online", nick))
		if a := s.accounts[nick]; a != "" {
			delete(s.accounts, nick)
			ops = append(ops, f("retract", "authenticated", nick, a))
		}
	case "MODE":
		if target == "" || len(params) < 2 || !strings.HasPrefix(target, "#") {
			return nil
		}
		modes := params[1]
		argi := 2
		adding := true
		for _, mode := range modes {
			switch mode {
			case '+':
				adding = true
				continue
			case '-':
				adding = false
				continue
			}
			if mode != 'o' && mode != 'v' {
				continue
			}
			if argi >= len(params) {
				break
			}
			subject := strings.ToLower(strings.TrimSpace(params[argi]))
			argi++
			if subject == "" {
				continue
			}
			var pred string
			var roles map[string]map[string]bool
			if mode == 'o' {
				pred = "channel_operator"
				roles = s.operators
			} else {
				pred = "voiced"
				roles = s.voiced
			}
			if roles[target] == nil {
				roles[target] = map[string]bool{}
			}
			if adding {
				if !roles[target][subject] {
					roles[target][subject] = true
					ops = append(ops, f("assert", pred, target, subject))
				}
			} else if roles[target][subject] {
				delete(roles[target], subject)
				ops = append(ops, f("retract", pred, target, subject))
			}
		}
	case "ACCOUNT":
		if nick == "" {
			return nil
		}
		if old := s.accounts[nick]; old != "" && (!verified || account == "" || old != account) {
			delete(s.accounts, nick)
			ops = append(ops, f("retract", "authenticated", nick, old))
		}
		if verified && account != "" {
			s.accounts[nick] = account
			ops = append(ops, f("assert", "authenticated", nick, account))
		}
	}
	return ops
}

type factMaker func(string, string, ...string) FactOperation

func (s *IRCState) removeRoles(ch, nick string, f factMaker) []FactOperation {
	var ops []FactOperation
	for pred, roles := range map[string]map[string]map[string]bool{"channel_operator": s.operators, "voiced": s.voiced} {
		if roles[ch] != nil && roles[ch][nick] {
			delete(roles[ch], nick)
			ops = append(ops, f("retract", pred, ch, nick))
		}
	}
	return ops
}
func (s *IRCState) moveRoles(ch, oldNick, newNick string, f factMaker) []FactOperation {
	var ops []FactOperation
	for pred, roles := range map[string]map[string]map[string]bool{"channel_operator": s.operators, "voiced": s.voiced} {
		if roles[ch] != nil && roles[ch][oldNick] {
			delete(roles[ch], oldNick)
			roles[ch][newNick] = true
			ops = append(ops, f("retract", pred, ch, oldNick), f("assert", pred, ch, newNick))
		}
	}
	return ops
}

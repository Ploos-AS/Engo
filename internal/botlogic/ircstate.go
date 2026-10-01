package botlogic

import "strings"

func IRCStateOperations(command,nick,target,account string,verified bool) []FactOperation {
	command=strings.ToUpper(strings.TrimSpace(command));nick=strings.ToLower(strings.TrimSpace(nick));target=strings.ToLower(strings.TrimSpace(target));account=strings.ToLower(strings.TrimSpace(account))
	if nick==""{return nil}
	f:=func(op,p string,args ...string) FactOperation{return FactOperation{Op:op,Fact:Fact{Predicate:p,Args:args}}}
	switch command {
	case "JOIN":
		if target==""{return nil};ops:=[]FactOperation{f("assert","channel_member",target,nick),f("assert","online",nick)}
		if verified&&account!=""{ops=append(ops,f("assert","authenticated",nick,account))}
		return ops
	case "PART":
		if target==""{return nil};return []FactOperation{f("retract","channel_member",target,nick)}
	case "QUIT":
		// Channel membership is intentionally not guessed here: Engo does not
		// maintain a complete nick->channels index yet. Online/auth state is safe.
		ops:=[]FactOperation{f("retract","online",nick)}
		if verified&&account!=""{ops=append(ops,f("retract","authenticated",nick,account))}
		return ops
	case "ACCOUNT":
		if verified&&account!=""{return []FactOperation{f("assert","authenticated",nick,account)}}
	}
	return nil
}

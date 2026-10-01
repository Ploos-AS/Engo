package botlogic

import "testing"

func TestJoinStateOperations(t *testing.T){
	ops:=IRCStateOperations("JOIN","Alice","#Engo","Alice.Account",true)
	if len(ops)!=3{t.Fatalf("ops=%+v",ops)}
	if ops[0].Fact.Predicate!="channel_member"||ops[0].Fact.Args[0]!="#engo"{t.Fatalf("membership=%+v",ops[0])}
	if ops[2].Fact.Predicate!="authenticated"||ops[2].Fact.Args[1]!="alice.account"{t.Fatalf("auth=%+v",ops[2])}
}
func TestUnverifiedJoinDoesNotPublishAccount(t *testing.T){
	ops:=IRCStateOperations("JOIN","Alice","#Engo","spoofed",false)
	for _,op:=range ops{if op.Fact.Predicate=="authenticated"{t.Fatalf("published unverified account: %+v",op)}}
}
func TestPartOnlyRetractsMembership(t *testing.T){
	ops:=IRCStateOperations("PART","Alice","#Engo","alice.account",true)
	if len(ops)!=1||ops[0].Op!="retract"||ops[0].Fact.Predicate!="channel_member"{t.Fatalf("ops=%+v",ops)}
}

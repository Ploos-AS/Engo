package botai

import "testing"

func TestConversationsBounded(t *testing.T){
	c:=NewConversations(4)
	c.AddExchange("#a","u1","a1")
	c.AddExchange("#a","u2","a2")
	c.AddExchange("#a","u3","a3")
	h:=c.History("#a")
	if len(h)!=4||h[0].Content!="u2"||h[3].Content!="a3"{t.Fatalf("history=%#v",h)}
	if len(c.History("#b"))!=0{t.Fatal("conversation scopes leaked")}
	c.Reset("#a")
	if len(c.History("#a"))!=0{t.Fatal("reset failed")}
}
func TestConversationsClampToBotAILimit(t *testing.T){
	c:=NewConversations(99)
	for i:=0;i<12;i++{c.AddExchange("x","u","a")}
	if got:=len(c.History("x"));got!=20{t.Fatalf("len=%d",got)}
}

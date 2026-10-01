package botlogic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

func QuoteAtom(s string) string { return "'" + strings.ReplaceAll(s,"'","''") + "'" }

func (c *Client) MayExecute(ctx context.Context,ruleset,account,command string)(bool,uint64,error){
	if strings.TrimSpace(account)==""||strings.TrimSpace(command)==""{return false,0,fmt.Errorf("account and command are required")}
	q:=fmt.Sprintf("may_execute(%s,%s).",QuoteAtom(strings.ToLower(account)),QuoteAtom(strings.ToLower(command)))
	result,err:=c.Query(ctx,ruleset,q);if err!=nil{return false,0,err};return len(result.Solutions)>0,result.Revision,nil
}

func PolicyFact(account,command string) string {
	return "may_execute("+strconv.Quote(strings.ToLower(account))+","+strconv.Quote(strings.ToLower(command))+")."
}

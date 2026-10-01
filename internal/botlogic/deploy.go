package botlogic

import (
	"context"
	"fmt"
)

// ValidateAndConsult preserves the deployment ordering invariant: the exact
// source is validated by BotLogic before it can replace the active ruleset.
func (c *Client) ValidateAndConsult(ctx context.Context, ruleset, source string) error {
	if err:=c.Validate(ctx,source);err!=nil{return fmt.Errorf("validate: %w",err)}
	if err:=c.Consult(ctx,ruleset,source);err!=nil{return fmt.Errorf("consult: %w",err)}
	return nil
}

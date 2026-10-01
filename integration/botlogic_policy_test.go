package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Ploos-AS/Engo/internal/botlogic"
)

func TestReferencePolicyAgainstBotLogic(t *testing.T) {
	url := os.Getenv("ENGO_TEST_BOTLOGIC_URL")
	if url == "" {
		t.Skip("ENGO_TEST_BOTLOGIC_URL not set")
	}
	sourceBytes, err := os.ReadFile("../rulesets/irc-policy.pl")
	if err != nil {
		t.Fatal(err)
	}
	source := string(sourceBytes)
	c, err := botlogic.New(url, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := c.Compatible(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateAndConsult(ctx, "irc-policy", source); err != nil {
		t.Fatal(err)
	}

	apply := func(ops ...botlogic.FactOperation) {
		t.Helper()
		if _, err := c.ApplyFacts(ctx, "irc-policy", ops); err != nil {
			t.Fatal(err)
		}
	}
	assert := func(pred string, args ...string) botlogic.FactOperation {
		return botlogic.FactOperation{Op: "assert", Fact: botlogic.Fact{Predicate: pred, Args: args}}
	}
	apply(assert("account_command", "alice", "reload"), assert("operator_command", "kick"), assert("voiced_command", "topic"))
	apply(assert("authenticated", "AliceNick", "alice"), assert("online", "AliceNick"))
	apply(assert("authenticated", "OpNick", "opacct"), assert("online", "OpNick"), assert("channel_operator", "#engo", "OpNick"))
	apply(assert("authenticated", "VoiceNick", "voiceacct"), assert("online", "VoiceNick"), assert("voiced", "#engo", "VoiceNick"))
	apply(assert("channel_operator", "#engo", "GhostNick"))

	cases := []struct {
		name, account, command string
		allow                  bool
	}{
		{"explicit account grant", "alice", "reload", true},
		{"deny by default", "alice", "unknown", false},
		{"authenticated operator", "opacct", "kick", true},
		{"operator cannot use unrelated command", "opacct", "topic", false},
		{"authenticated voiced user", "voiceacct", "topic", true},
		{"voice cannot use operator command", "voiceacct", "kick", false},
		{"unauthenticated operator nick grants nothing", "ghost", "kick", false},
	}
	for _, q := range []string{
		"account_command('alice','reload').",
		"authenticated('AliceNick','alice').",
		"online('AliceNick').",
		"channel_operator('#engo','OpNick').",
		"operator_command('kick').",
		"voiced('#engo','VoiceNick').",
		"voiced_command('topic').",
	} {
		r, err := c.Query(ctx, "irc-policy", q)
		t.Logf("M2.4 diagnostic query=%s solutions=%v err=%v", q, r.Solutions, err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := c.Query(ctx, "irc-policy", "may_execute("+quote(tc.account)+","+quote(tc.command)+").")
			if err != nil {
				t.Fatal(err)
			}
			if got := len(r.Solutions) > 0; got != tc.allow {
				t.Fatalf("allow=%v want=%v solutions=%v", got, tc.allow, r.Solutions)
			}
		})
	}
}

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

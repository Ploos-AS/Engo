package main

import (
	"github.com/Ploos-AS/Engo/internal/bot"
	"testing"
)

func TestAIConversationMessage(t *testing.T) {
	tests := []struct {
		name, nick string
		ev         bot.Event
		want       string
		ok         bool
	}{
		{"pm", "engo", bot.Event{Target: "engo", Text: "hello"}, "hello", true},
		{"mention colon", "engo", bot.Event{Target: "#test", Text: "Engo: explain SASL"}, "explain SASL", true},
		{"mention comma", "engo", bot.Event{Target: "#test", Text: "engo, hello"}, "hello", true},
		{"ordinary channel", "engo", bot.Event{Target: "#test", Text: "hello everyone"}, "", false},
		{"command pm", "engo", bot.Event{Target: "engo", Text: "!help"}, "", false},
		{"other mention", "engo", bot.Event{Target: "#test", Text: "engobot: hello"}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := aiConversationMessage(tt.nick, tt.ev)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("got=(%q,%v)", got, ok)
			}
		})
	}
}

package main

import (
	"testing"
	"time"
)

func TestShouldResetBackoff(t *testing.T) {
	threshold := 2 * time.Minute
	if shouldResetBackoff(30*time.Second, threshold) {
		t.Fatal("short session should not reset backoff")
	}
	if !shouldResetBackoff(threshold, threshold) {
		t.Fatal("session at threshold should reset backoff")
	}
	if !shouldResetBackoff(10*time.Minute, threshold) {
		t.Fatal("stable session should reset backoff")
	}
}

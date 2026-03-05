package scheduler

import (
	"context"
	"testing"
)

func TestTrafficHintsFromContext(t *testing.T) {
	base := context.Background()
	hints := TrafficHints{
		SessionID:      "  session-1  ",
		TrafficTag:     " Gray ",
		ExperimentID:   "Exp-A",
		IdempotencyKey: " idem-key ",
		ForceCanary:    true,
	}

	ctx := WithTrafficHints(base, hints)
	got := TrafficHintsFromContext(ctx)

	if got.SessionID != "session-1" {
		t.Fatalf("unexpected session id: %s", got.SessionID)
	}
	if got.TrafficTag != "gray" {
		t.Fatalf("unexpected traffic tag: %s", got.TrafficTag)
	}
	if got.ExperimentID != "exp-a" {
		t.Fatalf("unexpected experiment id: %s", got.ExperimentID)
	}
	if got.IdempotencyKey != "idem-key" {
		t.Fatalf("unexpected idempotency key: %s", got.IdempotencyKey)
	}
	if !got.ForceCanary {
		t.Fatalf("force canary should be true")
	}
}

func TestDeterministicBucketStable(t *testing.T) {
	seed := "session-1|chat.completions|gpt-4"
	left := DeterministicBucket(seed, 100)
	right := DeterministicBucket(seed, 100)

	if left != right {
		t.Fatalf("bucket should be stable: left=%d right=%d", left, right)
	}
	if left < 0 || left >= 100 {
		t.Fatalf("bucket out of range: %d", left)
	}
}

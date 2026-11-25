package app

import (
	"testing"
	"time"
)

func TestRendererSilence(t *testing.T) {
	var state rendererState
	asked := time.Now()

	if !state.silent(asked) {
		t.Fatal("a page that never beat must count as silent")
	}
	state.beat(asked.Add(-time.Second))
	if !state.silent(asked) {
		t.Fatal("a beat from before the probe is not an answer")
	}
	state.beat(asked.Add(time.Second))
	if state.silent(asked) {
		t.Fatal("an answer to the probe must clear the silence")
	}
	state.closing = true
	if state.silent(asked.Add(time.Hour)) {
		t.Fatal("a closing window is never revived")
	}
}

func TestRendererWaitsForTheGap(t *testing.T) {
	var state rendererState
	now := time.Now()

	if wait, open := state.when(now); wait != 0 || !open {
		t.Fatalf("the first recovery must run at once: %s %v", wait, open)
	}
	state.revived = now

	if wait, open := state.when(now.Add(time.Second)); wait != reviveGap-time.Second || !open {
		t.Fatalf("a recovery inside the gap must come back at its end: %s %v", wait, open)
	}
	if wait, open := state.when(now.Add(reviveGap)); wait != 0 || !open {
		t.Fatalf("the gap must pass: %s %v", wait, open)
	}

	state.closing = true
	if _, open := state.when(now.Add(reviveGap)); open {
		t.Fatal("a closing window must not be revived")
	}
}

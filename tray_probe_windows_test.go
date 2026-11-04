//go:build windows

package desktopkit

import (
	"context"
	"errors"
	"testing"
)

func TestLatchingTrayProbeIgnoresTransientFailureAfterReady(t *testing.T) {
	calls := 0
	probe := latchingTrayProbe(func() error {
		calls++
		if calls > 1 {
			return errors.New("transient bounds/shell observation failure")
		}
		return nil
	})
	if err := probe(context.Background()); err != nil {
		t.Fatalf("initial probe: %v", err)
	}
	if err := probe(context.Background()); err != nil {
		t.Fatalf("latched probe must stay ready: %v", err)
	}
	if calls != 1 {
		t.Fatalf("check calls = %d, want 1 after readiness latch", calls)
	}
}

func TestLatchingTrayProbeRetriesUntilReady(t *testing.T) {
	calls := 0
	probe := latchingTrayProbe(func() error {
		calls++
		if calls == 1 {
			return errors.New("shell not ready")
		}
		return nil
	})
	if err := probe(context.Background()); err == nil {
		t.Fatal("first probe unexpectedly succeeded")
	}
	if err := probe(context.Background()); err != nil {
		t.Fatalf("second probe should establish readiness: %v", err)
	}
	if err := probe(context.Background()); err != nil {
		t.Fatalf("latched probe failed: %v", err)
	}
	if calls != 2 {
		t.Fatalf("check calls = %d, want 2", calls)
	}
}

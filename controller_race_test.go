package desktopkit

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// A late native hide must not undo a manual show or recovery from tray failure.
func TestLateHidePreservesNewerVisibilityRequest(t *testing.T) {
	for _, recovery := range []string{"manual show", "tray failure"} {
		t.Run(recovery, func(t *testing.T) {
			c := newController(true, true)
			c.setContext(context.Background())
			var visible atomic.Bool
			visible.Store(true)
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			c.hide = func(context.Context) {
				close(entered)
				<-release
				visible.Store(false)
			}
			c.show = func(context.Context) { visible.Store(true) }
			go func() { c.trayAvailable(); close(done) }()
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("hide did not start")
			}
			if recovery == "manual show" {
				c.ShowWindow()
			} else {
				c.trayFailed()
			}
			close(release)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("visibility update deadlocked")
			}
			if !visible.Load() {
				t.Fatal("late hide undid the newer visibility request")
			}
		})
	}
}

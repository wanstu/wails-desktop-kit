package desktopkit

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func testController() (*Controller, *int, *int, *int) {
	c := newController(true, true)
	c.setContext(context.Background())
	shown, hidden, quit := new(int), new(int), new(int)
	c.show = func(context.Context) { *shown++ }
	c.hide = func(context.Context) { *hidden++ }
	c.quit = func(context.Context) { *quit++ }
	return c, shown, hidden, quit
}
func TestWindowWaitsForTrayAndRecovers(t *testing.T) {
	c, shown, hidden, _ := testController()
	c.HideWindow()
	if *hidden != 0 {
		t.Fatal("hid before tray became available")
	}
	c.trayAvailable()
	if *hidden != 1 {
		t.Fatal("autostart did not hide after readiness")
	}
	if !c.beforeClose(context.Background(), nil) {
		t.Fatal("close should hide a recoverable window")
	}
	c.trayFailed()
	if *shown != 1 {
		t.Fatal("tray failure did not recover window")
	}
	c.HideWindow()
	if *hidden != 2 {
		t.Fatal("hid after tray failure")
	}
	if c.beforeClose(context.Background(), nil) {
		t.Fatal("failed tray must not intercept closing")
	}
}
func TestManualShowCancelsPendingAutostartHide(t *testing.T) {
	c, _, hidden, _ := testController()
	c.ShowWindow()
	c.trayAvailable()
	if *hidden != 0 {
		t.Fatal("late readiness hid a manually opened window")
	}
}
func TestQuitBypassesHideAndCancellationResetsIntent(t *testing.T) {
	c, _, _, quit := testController()
	c.trayAvailable()
	c.Quit()
	if *quit != 1 || c.beforeClose(context.Background(), nil) {
		t.Fatal("explicit quit was converted to hide")
	}
	c.quitting = false
	c.Quit()
	if !c.beforeClose(context.Background(), func(context.Context) bool { return true }) {
		t.Fatal("quit veto ignored")
	}
	if !c.beforeClose(context.Background(), nil) {
		t.Fatal("cancelled quit intent leaked into next window close")
	}
	c.stop()
	c.trayAvailable()
	if c.ready {
		t.Fatal("late ready callback resurrected shutdown")
	}
}
func TestHideNeverBlocksManualHide(t *testing.T) {
	c, _, hidden, _ := testController()
	c.allowHide = false
	c.trayAvailable()
	c.HideWindow()
	if *hidden != 0 {
		t.Fatal("HideNever was bypassed")
	}
}

type fakeTray struct{}

func (*fakeTray) Run(context.Context, []nativeItem, func(int), func()) error {
	return errors.New("unused")
}
func (*fakeTray) Update(int, bool, bool) {}
func (*fakeTray) Close()                 {}
func TestTrayActionsAreSerializedAndDuplicatesDropped(t *testing.T) {
	c, _, _, _ := testController()
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	tm := newTrayManager(c, TrayConfig{Items: []TrayItem{Action("slow", func(*Controller) error {
		calls.Add(1)
		started <- struct{}{}
		<-release
		return nil
	})}}, nil)
	tm.actions = make(map[int]func() error)
	tm.getters = make(map[int]func() (bool, bool))
	tm.pending = make(map[int]bool)
	tm.jobs = make(chan int, 8)
	tm.config.DisableShowHide = true
	tm.config.DisableQuit = true
	tm.menu()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { tm.work(ctx, &fakeTray{}); close(done) }()
	tm.enqueue(0)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	tm.enqueue(0)
	if len(tm.jobs) != 0 {
		t.Fatal("duplicate action was queued")
	}
	tm.Shutdown(context.Background())
	tm.enqueue(0)
	close(release)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not finish")
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate action executed")
	}
}

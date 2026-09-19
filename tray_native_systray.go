//go:build windows || linux

package desktopkit

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/gogpu/systray"
)

const traySupported = true

type systrayBackend struct {
	icon   []byte
	title  string
	mu     sync.Mutex
	tray   *systray.SystemTray
	items  map[int]*systray.MenuItem
	closed bool
	wake   func()
}

func newTrayBackend(icon []byte, title string) trayBackend {
	return &systrayBackend{icon: icon, title: title, items: make(map[int]*systray.MenuItem)}
}
func (b *systrayBackend) Run(ctx context.Context, items []nativeItem, click func(int), ready func()) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if ctx.Err() != nil {
		return nil
	}
	tray := systray.New()
	menu := systray.NewMenu()
	mapped := make(map[int]*systray.MenuItem)
	for id, n := range items {
		id, n := id, n
		if n.Separator {
			menu.AddSeparator()
			continue
		}
		action := func() {
			code := id
			if n.WindowAction != 0 {
				code = n.WindowAction
			}
			click(code)
		}
		var m *systray.MenuItem
		if n.Checkbox {
			m = menu.AddCheckbox(n.Label, n.Checked, action)
		} else {
			m = menu.Add(n.Label, action)
		}
		m.SetDisabled(n.Disabled)
		mapped[id] = m
	}
	tray.SetIcon(b.icon).SetTooltip(b.title).SetMenu(menu)
	tray.OnClick(func() { click(-1) }).OnDoubleClick(func() { click(-1) })
	tray.Show()
	b.mu.Lock()
	b.tray = tray
	b.items = mapped
	b.wake = trayLoopWake()
	closed := b.closed
	b.mu.Unlock()
	if closed {
		tray.Remove()
		b.wake()
	}
	probeCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		probe, closeProbe, err := newTrayProbe(tray, b.title)
		if err == nil {
			defer closeProbe()
			err = monitorTray(probeCtx, func(ctx context.Context) error {
				b.mu.Lock()
				defer b.mu.Unlock()
				if b.closed {
					return context.Canceled
				}
				return probe(ctx)
			}, ready)
		}
		result <- err
		b.Close()
	}()
	runErr := tray.Run()
	cancel()
	probeErr := <-result
	if ctx.Err() != nil {
		return nil
	}
	if probeErr != nil {
		return probeErr
	}
	return runErr
}
func monitorTray(ctx context.Context, probe func(context.Context) error, ready func()) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.Now().Add(3 * time.Second)
	available := false
	for {
		if ctx.Err() != nil {
			return nil
		}
		err := probe(ctx)
		if err == nil {
			if !available {
				available = true
				ready()
			}
		} else if available || time.Now().After(deadline) {
			return fmt.Errorf("desktop-kit: tray unavailable: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
func (b *systrayBackend) Update(id int, checked, enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	if item := b.items[id]; item != nil {
		item.SetChecked(checked)
		item.SetDisabled(!enabled)
	}
}
func (b *systrayBackend) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	if b.tray != nil {
		b.tray.Remove()
	}
	if b.wake != nil {
		b.wake()
	}
}

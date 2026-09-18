package desktopkit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type nativeItem struct {
	Label                                  string
	WindowAction                           int
	Separator, Checkbox, Checked, Disabled bool
}
type trayBackend interface {
	Run(context.Context, []nativeItem, func(int), func()) error
	Update(int, bool, bool)
	Close()
}

type trayManager struct {
	controller       *Controller
	config           TrayConfig
	mu               sync.Mutex
	started, stopped bool
	cancel           context.CancelFunc
	backend          trayBackend
	actions          map[int]func() error
	getters          map[int]func() (bool, bool)
	pending          map[int]bool
	jobs             chan int
	report           func(error)
}

func newTrayManager(c *Controller, cfg TrayConfig, report func(error)) *trayManager {
	return &trayManager{controller: c, config: cfg, report: report}
}
func (t *trayManager) DomReady(ctx context.Context) {
	if !t.config.Enabled {
		return
	}
	t.mu.Lock()
	if t.started || t.stopped {
		t.mu.Unlock()
		return
	}
	t.started = true
	runCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	t.actions = make(map[int]func() error)
	t.getters = make(map[int]func() (bool, bool))
	t.pending = make(map[int]bool)
	// At most one pending execution per item; this bounds work and drops repeats.
	t.jobs = make(chan int, len(t.config.Items)+len(t.config.FooterItems)+8)
	t.mu.Unlock()
	go t.run(runCtx)
}
func (t *trayManager) Shutdown(context.Context) {
	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return
	}
	t.stopped = true
	cancel, backend := t.cancel, t.backend
	t.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if backend != nil {
		backend.Close()
	}
	// Do not join a worker here: an action may itself have requested Quit.
}
func (t *trayManager) enqueue(id int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped || t.pending[id] {
		return
	}
	t.pending[id] = true
	select {
	case t.jobs <- id:
	default:
		delete(t.pending, id)
	}
}
func (t *trayManager) run(ctx context.Context) {
	items := t.menu()
	for id, get := range t.getters {
		checked, enabled := get()
		items[id].Checked = checked
		items[id].Disabled = !enabled
	}
	backend := newTrayBackend(t.config.Icon, t.config.Tooltip)
	t.mu.Lock()
	t.backend = backend
	stopped := t.stopped
	t.mu.Unlock()
	if stopped {
		backend.Close()
	}
	// All business callbacks and state readers use this single worker. Native
	// menu callbacks only enqueue work; window operations remain responsive.
	go t.work(ctx, backend)
	err := backend.Run(ctx, items, func(id int) {
		if id == -1 {
			t.controller.ShowWindow()
			return
		}
		if id == -2 {
			t.controller.HideWindow()
			return
		}
		t.enqueue(id)
	}, t.controller.trayAvailable)
	backend.Close()
	t.mu.Lock()
	cancel := t.cancel
	stopped = t.stopped
	t.mu.Unlock()
	cancelled := ctx.Err() != nil
	if cancel != nil {
		cancel()
	}
	if stopped || cancelled {
		return
	}
	t.controller.trayFailed()
	if err == nil {
		err = fmt.Errorf("desktop-kit: tray event loop exited unexpectedly")
	}
	if t.report != nil {
		t.report(err)
	}
}
func (t *trayManager) work(ctx context.Context, b trayBackend) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	t.refresh(ctx, b)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.refresh(ctx, b)
		case id := <-t.jobs:
			if ctx.Err() != nil {
				return
			}
			action := t.actions[id]
			var err error
			if action != nil {
				err = action()
			}
			t.mu.Lock()
			delete(t.pending, id)
			t.mu.Unlock()
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				t.controller.ShowError("操作失败", err)
			}
			t.refresh(ctx, b)
		}
	}
}
func (t *trayManager) refresh(ctx context.Context, b trayBackend) {
	for id, get := range t.getters {
		if ctx.Err() != nil {
			return
		}
		checked, enabled := get()
		b.Update(id, checked, enabled)
	}
}
func (t *trayManager) menu() []nativeItem {
	var items []nativeItem
	add := func(item TrayItem) {
		id := len(items)
		n := nativeItem{Label: item.Label, Separator: item.Kind == TraySeparatorItem, Checkbox: item.Kind == TrayCheckboxItem}
		if n.Checkbox && (item.Checkbox == nil || item.Checkbox.Get == nil || item.Checkbox.Set == nil) {
			return
		}
		items = append(items, n)
		if n.Separator {
			return
		}
		if n.Checkbox {
			cb := item.Checkbox
			t.getters[id] = func() (bool, bool) { checked, err := cb.Get(t.controller); return checked, err == nil }
			t.actions[id] = func() error {
				checked, err := cb.Get(t.controller)
				if err != nil {
					return err
				}
				return cb.Set(t.controller, !checked)
			}
		} else {
			t.actions[id] = func() error {
				if item.Action == nil {
					return nil
				}
				if err := item.Action(t.controller); err != nil {
					title := item.ErrorTitle
					if title == "" {
						title = item.Label + "失败"
					}
					return fmt.Errorf("%s: %w", title, err)
				}
				return nil
			}
		}
	}
	if !t.config.DisableShowHide {
		add(Action(t.config.ShowLabel, func(c *Controller) error { c.ShowWindow(); return nil }))
		add(Action(t.config.HideLabel, func(c *Controller) error { c.HideWindow(); return nil }))
		items[0].WindowAction = -1
		items[1].WindowAction = -2
		add(Separator())
	}
	for _, i := range t.config.Items {
		add(i)
	}
	if a := t.config.AutoStart; a != nil {
		if len(t.config.Items) > 0 {
			add(Separator())
		}
		id := len(items)
		add(Checkbox(t.config.LaunchAtLoginLabel, TrayCheckbox{
			Get: func(*Controller) (bool, error) { return a.Enabled() },
			Set: func(_ *Controller, v bool) error { return a.SetEnabled(v) },
		}))
		t.getters[id] = func() (bool, bool) { v, err := a.Enabled(); return v, a.Supported() && err == nil }
	}
	if len(t.config.FooterItems) > 0 {
		add(Separator())
		for _, i := range t.config.FooterItems {
			add(i)
		}
	}
	if !t.config.DisableQuit {
		add(Separator())
		add(Action(t.config.QuitLabel, func(c *Controller) error { c.Quit(); return nil }))
	}
	return items
}

package desktopkit

import (
	"context"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Controller owns window visibility and distinguishes an explicit quit from
// closing a window. Applications should use Quit for explicit exit actions.
type Controller struct {
	mu                                sync.Mutex
	ctx                               context.Context
	ready, stopped, quitting, closing bool
	allowHide, pendingHide            bool
	desiredVisible                    bool
	visibilityVersion                 uint64
	show, hide, quit                  func(context.Context)
}

func newController(allowHide, startHidden bool) *Controller {
	return &Controller{
		allowHide: allowHide, pendingHide: startHidden, desiredVisible: true,
		show: func(ctx context.Context) { wailsruntime.WindowShow(ctx); wailsruntime.WindowUnminimise(ctx) },
		hide: wailsruntime.WindowHide, quit: wailsruntime.Quit,
	}
}
func (c *Controller) setContext(ctx context.Context) {
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
}
func (c *Controller) Context() context.Context {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ctx
}
func (c *Controller) ShowWindow() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.pendingHide = false
	c.desiredVisible = true
	c.visibilityVersion++
	c.mu.Unlock()
	c.applyVisibility()
}

// HideWindow only hides a recoverable window under the configured policy.
func (c *Controller) HideWindow() {
	c.mu.Lock()
	if !c.allowHide || !c.ready || c.stopped || c.quitting {
		c.mu.Unlock()
		return
	}
	c.desiredVisible = false
	c.visibilityVersion++
	c.mu.Unlock()
	c.applyVisibility()
}

// Do not hold mu across native calls: Wails may dispatch them to its main
// thread, which can concurrently invoke a lifecycle callback. If an older
// operation finishes after a newer request, apply the latest intent again.
func (c *Controller) applyVisibility() {
	for {
		c.mu.Lock()
		ctx, version, stopped := c.ctx, c.visibilityVersion, c.stopped
		apply := c.hide
		if c.desiredVisible {
			apply = c.show
		}
		c.mu.Unlock()
		if stopped || ctx == nil || apply == nil {
			return
		}
		apply(ctx)
		c.mu.Lock()
		current := version == c.visibilityVersion
		c.mu.Unlock()
		if current {
			return
		}
	}
}
func (c *Controller) Quit() {
	c.mu.Lock()
	if c.stopped || c.quitting {
		c.mu.Unlock()
		return
	}
	c.quitting = true
	ctx, quit := c.ctx, c.quit
	c.mu.Unlock()
	if ctx != nil && quit != nil {
		quit(ctx)
	}
}
func (c *Controller) beforeClose(ctx context.Context, hook func(context.Context) bool) bool {
	c.mu.Lock()
	if c.closing {
		c.mu.Unlock()
		return true
	}
	c.closing = true
	c.mu.Unlock()
	defer func() { c.mu.Lock(); c.closing = false; c.mu.Unlock() }()
	if hook != nil && hook(ctx) {
		c.mu.Lock()
		c.quitting = false
		c.mu.Unlock()
		return true
	}
	c.mu.Lock()
	hide := !c.quitting && c.allowHide && c.ready && !c.stopped
	c.mu.Unlock()
	if hide {
		c.HideWindow()
		return true
	}
	return false
}
func (c *Controller) trayAvailable() {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	c.ready = true
	hide := c.pendingHide && c.allowHide && !c.quitting
	c.pendingHide = false
	if hide {
		c.desiredVisible = false
		c.visibilityVersion++
	}
	c.mu.Unlock()
	if hide {
		c.applyVisibility()
	}
}
func (c *Controller) trayFailed() {
	c.mu.Lock()
	c.ready = false
	c.pendingHide = false
	c.desiredVisible = true
	c.visibilityVersion++
	c.mu.Unlock()
	c.applyVisibility()
}
func (c *Controller) stop() {
	c.mu.Lock()
	c.stopped = true
	c.ready = false
	c.ctx = nil
	c.mu.Unlock()
}
func (c *Controller) ShowError(title string, err error) {
	if err == nil {
		return
	}
	c.ShowWindow()
	if ctx := c.Context(); ctx != nil {
		_, _ = wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
			Type: wailsruntime.ErrorDialog, Title: title, Message: err.Error(),
		})
	}
}

package desktopkit

import (
	"context"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Controller exposes framework-owned window/runtime operations to tray actions
// without forcing application code to duplicate Wails runtime plumbing.
type Controller struct {
	mu  sync.RWMutex
	ctx context.Context
}

func (c *Controller) setContext(ctx context.Context) {
	c.mu.Lock()
	c.ctx = ctx
	c.mu.Unlock()
}

// Context returns the current Wails runtime context, or nil before startup.
func (c *Controller) Context() context.Context {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

// ShowWindow shows and restores the main window.
func (c *Controller) ShowWindow() {
	if ctx := c.Context(); ctx != nil {
		wailsruntime.WindowShow(ctx)
		wailsruntime.WindowUnminimise(ctx)
	}
}

// HideWindow hides the main window.
func (c *Controller) HideWindow() {
	if ctx := c.Context(); ctx != nil {
		wailsruntime.WindowHide(ctx)
	}
}

// Quit requests normal Wails application shutdown.
func (c *Controller) Quit() {
	if ctx := c.Context(); ctx != nil {
		wailsruntime.Quit(ctx)
	}
}

// ShowError brings the main window forward and displays a native error dialog.
func (c *Controller) ShowError(title string, err error) {
	if err == nil {
		return
	}
	c.ShowWindow()
	if ctx := c.Context(); ctx != nil {
		_, _ = wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
			Type:    wailsruntime.ErrorDialog,
			Title:   title,
			Message: err.Error(),
		})
	}
}

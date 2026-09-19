package desktopkit

import (
	"errors"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var ErrRuntimeNotReady = errors.New("desktop-kit: runtime context is not ready")

func (c *Controller) OpenURL(url string) error {
	ctx := c.Context()
	if ctx == nil {
		return ErrRuntimeNotReady
	}
	wailsruntime.BrowserOpenURL(ctx, url)
	return nil
}

func (c *Controller) Emit(event string, data ...interface{}) error {
	ctx := c.Context()
	if ctx == nil {
		return ErrRuntimeNotReady
	}
	wailsruntime.EventsEmit(ctx, event, data...)
	return nil
}

func (c *Controller) OpenFileDialog(options wailsruntime.OpenDialogOptions) (string, error) {
	ctx := c.Context()
	if ctx == nil {
		return "", ErrRuntimeNotReady
	}
	return wailsruntime.OpenFileDialog(ctx, options)
}

func (c *Controller) MessageDialog(options wailsruntime.MessageDialogOptions) (string, error) {
	ctx := c.Context()
	if ctx == nil {
		return "", ErrRuntimeNotReady
	}
	return wailsruntime.MessageDialog(ctx, options)
}

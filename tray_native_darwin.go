//go:build darwin && cgo

package desktopkit

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include <stdint.h>
int dkTrayCreate(uintptr_t token, const char *items, const char *title, const void *icon, int length);
void dkTrayUpdate(uintptr_t token, int index, int checked, int enabled);
void dkTrayClose(uintptr_t token);
void dkTrayForget(uintptr_t token);
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/cgo"
	"sync"
	"unsafe"
)

const traySupported = true

type cocoaTray struct {
	icon       []byte
	title      string
	token      cgo.Handle
	once       sync.Once
	mu         sync.Mutex
	click      func(int)
	closed     bool
	closedDone chan struct{}
}

func newTrayBackend(icon []byte, title string) trayBackend {
	b := &cocoaTray{icon: icon, title: title, closedDone: make(chan struct{})}
	b.token = cgo.NewHandle(b)
	return b
}
func (b *cocoaTray) Run(ctx context.Context, items []nativeItem, click func(int), ready func()) error {
	defer func() { b.Close(); <-b.closedDone; C.dkTrayForget(C.uintptr_t(b.token)); b.token.Delete() }()
	b.mu.Lock()
	b.click = click
	closed := b.closed
	b.mu.Unlock()
	if closed || ctx.Err() != nil {
		return nil
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	spec := C.CString(string(data))
	defer C.free(unsafe.Pointer(spec))
	title := C.CString(b.title)
	defer C.free(unsafe.Pointer(title))
	icon := C.CBytes(b.icon)
	defer C.free(icon)
	if C.dkTrayCreate(C.uintptr_t(b.token), spec, title, icon, C.int(len(b.icon))) == 0 {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("desktop-kit: Cocoa could not create the tray icon")
	}
	if ctx.Err() == nil {
		ready()
	}
	<-ctx.Done()
	return nil
}
func (b *cocoaTray) Update(id int, checked, enabled bool) {
	b.mu.Lock()
	closed := b.closed
	b.mu.Unlock()
	if closed {
		return
	}
	c, e := 0, 0
	if checked {
		c = 1
	}
	if enabled {
		e = 1
	}
	C.dkTrayUpdate(C.uintptr_t(b.token), C.int(id), C.int(c), C.int(e))
}
func (b *cocoaTray) Close() {
	b.once.Do(func() {
		b.mu.Lock()
		b.closed = true
		b.mu.Unlock()
		C.dkTrayClose(C.uintptr_t(b.token))
	})
}

//export dkTrayClicked
func dkTrayClicked(token C.uintptr_t, index C.int) {
	b := cgo.Handle(token).Value().(*cocoaTray)
	b.mu.Lock()
	click, closed := b.click, b.closed
	b.mu.Unlock()
	if !closed && click != nil {
		click(int(index))
	}
}

//export dkTrayClosed
func dkTrayClosed(token C.uintptr_t) {
	b := cgo.Handle(token).Value().(*cocoaTray)
	close(b.closedDone)
}

//go:build windows || linux || darwin

package desktopkit

import (
	"context"
	"runtime"
	"sync"

	"github.com/gogpu/systray"
)

const traySupported = true

type trayManager struct {
	controller *Controller
	config     TrayConfig

	mu       sync.RWMutex
	ctx      context.Context
	tray     *systray.SystemTray
	started  bool
	stopping bool
	syncers  []func()
}

func newTrayManager(controller *Controller, cfg TrayConfig) *trayManager {
	return &trayManager{controller: controller, config: cfg}
}

func (t *trayManager) Startup(ctx context.Context) {
	t.mu.Lock()
	t.ctx = ctx
	t.mu.Unlock()
}

func (t *trayManager) DomReady(ctx context.Context) {
	if !t.config.Enabled {
		return
	}
	t.mu.Lock()
	t.ctx = ctx
	if t.started {
		t.mu.Unlock()
		return
	}
	t.started = true
	t.stopping = false
	t.mu.Unlock()
	go t.run()
}

func (t *trayManager) Shutdown(context.Context) {
	t.mu.Lock()
	t.stopping = true
	tray := t.tray
	t.mu.Unlock()
	if tray != nil {
		tray.Remove()
	}
}

func (t *trayManager) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tray := systray.New()
	menu := systray.NewMenu()

	if !t.config.DisableShowHide {
		menu.Add(t.config.ShowLabel, t.controller.ShowWindow)
		menu.Add(t.config.HideLabel, t.controller.HideWindow)
		menu.AddSeparator()
	}

	for _, item := range t.config.Items {
		t.addItem(menu, item)
	}

	if t.config.AutoStart != nil {
		if len(t.config.Items) > 0 {
			menu.AddSeparator()
		}
		t.addAutoStartItem(menu)
	}

	if len(t.config.FooterItems) > 0 {
		menu.AddSeparator()
		for _, item := range t.config.FooterItems {
			t.addItem(menu, item)
		}
	}

	if !t.config.DisableQuit {
		menu.AddSeparator()
		menu.Add(t.config.QuitLabel, t.controller.Quit)
	}

	tray.SetIcon(t.config.Icon).SetTooltip(t.config.Tooltip).SetMenu(menu)
	tray.OnClick(t.controller.ShowWindow)
	tray.OnDoubleClick(t.controller.ShowWindow)
	tray.OnRightClick(t.syncState)
	tray.Show()

	t.mu.Lock()
	if t.stopping {
		t.started = false
		t.mu.Unlock()
		tray.Remove()
		return
	}
	t.tray = tray
	t.mu.Unlock()

	_ = tray.Run()

	t.mu.Lock()
	t.tray = nil
	t.syncers = nil
	t.started = false
	t.mu.Unlock()
}

func (t *trayManager) addItem(menu *systray.Menu, item TrayItem) {
	switch item.Kind {
	case TraySeparatorItem:
		menu.AddSeparator()
	case TrayCheckboxItem:
		t.addCheckbox(menu, item)
	case TrayActionItem:
		menu.Add(item.Label, func() {
			if item.Action == nil {
				return
			}
			if err := item.Action(t.controller); err != nil {
				title := item.ErrorTitle
				if title == "" {
					title = item.Label + "失败"
				}
				t.controller.ShowError(title, err)
			}
		})
	}
}

func (t *trayManager) addCheckbox(menu *systray.Menu, item TrayItem) {
	checkbox := item.Checkbox
	if checkbox == nil || checkbox.Get == nil || checkbox.Set == nil {
		return
	}
	checked, err := checkbox.Get(t.controller)
	if err != nil {
		checked = false
	}
	var menuItem *systray.MenuItem
	menuItem = menu.AddCheckbox(item.Label, checked, func() {
		current, err := checkbox.Get(t.controller)
		if err != nil {
			t.controller.ShowError(item.Label+"失败", err)
			return
		}
		if err := checkbox.Set(t.controller, !current); err != nil {
			t.controller.ShowError(item.Label+"失败", err)
			return
		}
		t.syncCheckbox(menuItem, checkbox)
	})
	t.registerSyncer(func() { t.syncCheckbox(menuItem, checkbox) })
}

func (t *trayManager) syncCheckbox(item *systray.MenuItem, checkbox *TrayCheckbox) {
	if item == nil || checkbox == nil || checkbox.Get == nil {
		return
	}
	checked, err := checkbox.Get(t.controller)
	if err == nil {
		item.SetChecked(checked)
	}
}

func (t *trayManager) addAutoStartItem(menu *systray.Menu) {
	manager := t.config.AutoStart
	checked, err := manager.Enabled()
	if err != nil {
		checked = false
	}
	var item *systray.MenuItem
	item = menu.AddCheckbox(t.config.LaunchAtLoginLabel, checked, func() {
		current, err := manager.Enabled()
		if err != nil {
			t.controller.ShowError("读取开机启动状态失败", err)
			return
		}
		if err := manager.SetEnabled(!current); err != nil {
			t.controller.ShowError("设置开机启动失败", err)
			return
		}
		t.syncAutoStart(item, manager)
	})
	if !manager.Supported() {
		item.SetDisabled(true)
	}
	t.registerSyncer(func() { t.syncAutoStart(item, manager) })
}

func (t *trayManager) syncAutoStart(item *systray.MenuItem, manager interface {
	Supported() bool
	Enabled() (bool, error)
}) {
	if item == nil || manager == nil {
		return
	}
	item.SetDisabled(!manager.Supported())
	checked, err := manager.Enabled()
	if err == nil {
		item.SetChecked(checked)
	}
}

func (t *trayManager) registerSyncer(syncer func()) {
	t.mu.Lock()
	t.syncers = append(t.syncers, syncer)
	t.mu.Unlock()
}

func (t *trayManager) syncState() {
	t.mu.RLock()
	syncers := append([]func(){}, t.syncers...)
	t.mu.RUnlock()
	for _, syncer := range syncers {
		syncer()
	}
}

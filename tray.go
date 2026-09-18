package desktopkit

import "github.com/wanstu/wails-desktop-kit/autostart"

// TrayItemKind identifies a declarative tray menu item.
type TrayItemKind uint8

const (
	TrayActionItem TrayItemKind = iota + 1
	TrayCheckboxItem
	TraySeparatorItem
)

// TrayActionFunc handles a tray action.
type TrayActionFunc func(*Controller) error

// TrayCheckbox binds a tray checkbox to application state.
type TrayCheckbox struct {
	Get func(*Controller) (bool, error)
	Set func(*Controller, bool) error
}

// TrayItem describes an application-specific menu item.
type TrayItem struct {
	Kind       TrayItemKind
	Label      string
	ErrorTitle string
	Action     TrayActionFunc
	Checkbox   *TrayCheckbox
}

// Action creates a normal tray menu item.
func Action(label string, action TrayActionFunc) TrayItem {
	return TrayItem{Kind: TrayActionItem, Label: label, Action: action}
}

// Checkbox creates a state-backed tray checkbox.
func Checkbox(label string, checkbox TrayCheckbox) TrayItem {
	return TrayItem{Kind: TrayCheckboxItem, Label: label, Checkbox: &checkbox}
}

// Separator creates a tray menu separator.
func Separator() TrayItem {
	return TrayItem{Kind: TraySeparatorItem}
}

// TrayConfig describes standard and application-specific tray behavior.
type TrayConfig struct {
	Enabled bool
	Icon    []byte
	Tooltip string

	ShowLabel          string
	HideLabel          string
	LaunchAtLoginLabel string
	QuitLabel          string

	DisableShowHide bool
	DisableQuit     bool

	// Items are application actions shown before the standard launch-at-login
	// control.
	Items []TrayItem
	// FooterItems are application actions shown after the standard
	// launch-at-login control and before the optional standard quit item.
	FooterItems []TrayItem

	// AutoStart enables the standard launch-at-login checkbox.
	AutoStart *autostart.Manager
}

func (c TrayConfig) normalized(title string) TrayConfig {
	if c.Tooltip == "" {
		c.Tooltip = title
	}
	if c.ShowLabel == "" {
		c.ShowLabel = "显示主窗口"
	}
	if c.HideLabel == "" {
		c.HideLabel = "隐藏主窗口"
	}
	if c.LaunchAtLoginLabel == "" {
		c.LaunchAtLoginLabel = "开机启动"
	}
	if c.QuitLabel == "" {
		c.QuitLabel = "退出"
	}
	return c
}

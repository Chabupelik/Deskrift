package ui

import (
	"github.com/energye/systray"

	"deskrift/internal/config"
	"deskrift/internal/relay"
)

type TrayApp struct {
	cfg        *config.Config
	client     *relay.Client
	win        *SettingsWindow
	mStatus    *systray.MenuItem
	mAutostart *systray.MenuItem
}

func NewTrayApp(cfg *config.Config, client *relay.Client, win *SettingsWindow) *TrayApp {
	return &TrayApp{
		cfg:    cfg,
		client: client,
		win:    win,
	}
}

func (t *TrayApp) Run(startMinimized bool) {
	systray.Run(func() {
		t.onReady(startMinimized)
	}, t.onExit)
}

func (t *TrayApp) onReady(startMinimized bool) {
	systray.SetIcon(CreateDefaultIcon())
	systray.SetTitle("Deskrift")
	systray.SetTooltip("Deskrift — " + t.client.GetStatus())

	systray.SetOnClick(func(menu systray.IMenu) {
		t.win.Show()
	})

	systray.SetOnRClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})

	t.mStatus = systray.AddMenuItem("Статус: "+t.client.GetStatus(), "Статус подключения")
	t.mStatus.Disable()

	systray.AddSeparator()

	mSettings := systray.AddMenuItem("Настройки...", "Открыть окно настроек")
	mSettings.Click(func() {
		t.win.Show()
	})

	t.mAutostart = systray.AddMenuItemCheckbox("Автозапуск", "Запускать вместе с Windows", t.cfg.Autostart)
	t.mAutostart.Click(func() {
		t.toggleAutostart()
	})

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Выход", "Завершить работу приложения")
	mQuit.Click(func() {
		systray.Quit()
	})

	t.client.SetStatusCallback(func(status string) {
		t.UpdateStatus(status)
	})

	if !startMinimized && (t.cfg.ServerURL == "" || t.cfg.PCSecret == "") {
		t.win.Show()
	}
}

func (t *TrayApp) toggleAutostart() {
	newState := !t.cfg.Autostart
	if err := config.SetAutostart(newState); err == nil {
		t.cfg.Autostart = newState
		_ = t.cfg.Save()
		if newState {
			t.mAutostart.Check()
		} else {
			t.mAutostart.Uncheck()
		}
		if t.win.chkAutostart != nil {
			t.win.chkAutostart.SetChecked(newState)
		}
	}
}

func (t *TrayApp) UpdateStatus(status string) {
	systray.SetTooltip("Deskrift — " + status)
	if t.mStatus != nil {
		t.mStatus.SetTitle("Статус: " + status)
	}
	if t.win != nil {
		t.win.UpdateStatus(status)
	}
}

func (t *TrayApp) onExit() {
	t.client.Stop()
}

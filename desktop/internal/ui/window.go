package ui

import (
	"strings"
	"sync"
	"syscall"

	"github.com/gonutz/wui/v2"

	"deskrift/internal/config"
	"deskrift/internal/relay"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procShowWindow          = user32.NewProc("ShowWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

const (
	swHide = 0
	swShow = 5
)

type SettingsWindow struct {
	mu           sync.Mutex
	cfg          *config.Config
	client       *relay.Client
	win          *wui.Window
	txtServer    *wui.EditLine
	txtSecret    *wui.EditLine
	cmbQuality   *wui.ComboBox
	chkAutostart *wui.CheckBox
	lblStatus    *wui.Label
	isShown      bool
}

func NewSettingsWindow(cfg *config.Config, client *relay.Client) *SettingsWindow {
	sw := &SettingsWindow{
		cfg:    cfg,
		client: client,
	}
	sw.initUI()
	return sw
}

func (s *SettingsWindow) initUI() {
	font, _ := wui.NewFont(wui.FontDesc{Name: "Segoe UI", Height: -13})
	fontBold, _ := wui.NewFont(wui.FontDesc{Name: "Segoe UI", Height: -13, Bold: true})

	win := wui.NewWindow()
	win.SetTitle("Deskrift — Настройки")
	win.SetInnerSize(420, 360)
	win.SetFont(font)
	win.SetResizable(false)
	win.SetHasMaxButton(false)

	lblHeader := wui.NewLabel()
	lblHeader.SetText("Параметры подключения")
	lblHeader.SetFont(fontBold)
	lblHeader.SetBounds(24, 18, 370, 22)
	win.Add(lblHeader)

	lblStatus := wui.NewLabel()
	lblStatus.SetText("Статус: " + s.client.GetStatus())
	lblStatus.SetBounds(24, 44, 370, 20)
	win.Add(lblStatus)
	s.lblStatus = lblStatus

	lblServer := wui.NewLabel()
	lblServer.SetText("Адрес сервера (Server URL):")
	lblServer.SetBounds(24, 76, 370, 18)
	win.Add(lblServer)

	txtServer := wui.NewEditLine()
	txtServer.SetBounds(24, 98, 370, 24)
	txtServer.SetText(s.cfg.ServerURL)
	win.Add(txtServer)
	s.txtServer = txtServer

	lblSecret := wui.NewLabel()
	lblSecret.SetText("Ключ доступа (PC Secret):")
	lblSecret.SetBounds(24, 134, 370, 18)
	win.Add(lblSecret)

	txtSecret := wui.NewEditLine()
	txtSecret.SetBounds(24, 156, 370, 24)
	txtSecret.SetText(s.cfg.PCSecret)
	txtSecret.SetIsPassword(true)
	win.Add(txtSecret)
	s.txtSecret = txtSecret

	lblQuality := wui.NewLabel()
	lblQuality.SetText("Качество трансляции:")
	lblQuality.SetBounds(24, 192, 370, 18)
	win.Add(lblQuality)

	cmbQuality := wui.NewComboBox()
	cmbQuality.SetBounds(24, 214, 370, 24)
	cmbQuality.SetItems([]string{
		"High (60 FPS, 6-8 Mbps)",
		"Medium (30 FPS, 3-4 Mbps)",
		"Low (24 FPS, 1.5-2 Mbps)",
	})
	switch s.cfg.Quality {
	case "medium":
		cmbQuality.SetSelectedIndex(1)
	case "low":
		cmbQuality.SetSelectedIndex(2)
	default:
		cmbQuality.SetSelectedIndex(0)
	}
	win.Add(cmbQuality)
	s.cmbQuality = cmbQuality

	chkAutostart := wui.NewCheckBox()
	chkAutostart.SetBounds(24, 252, 370, 22)
	chkAutostart.SetText("Запускать вместе с Windows (автозапуск)")
	chkAutostart.SetChecked(s.cfg.Autostart)
	win.Add(chkAutostart)
	s.chkAutostart = chkAutostart

	btnSave := wui.NewButton()
	btnSave.SetText("Сохранить")
	btnSave.SetBounds(160, 300, 110, 30)
	btnSave.SetOnClick(func() {
		s.save()
	})
	win.Add(btnSave)

	btnHide := wui.NewButton()
	btnHide.SetText("Свернуть в трей")
	btnHide.SetBounds(280, 300, 114, 30)
	btnHide.SetOnClick(func() {
		s.Hide()
	})
	win.Add(btnHide)

	win.SetOnCanClose(func() bool {
		s.Hide()
		return false
	})

	s.win = win
}

func (s *SettingsWindow) save() {
	server := strings.TrimSpace(s.txtServer.Text())
	secret := strings.TrimSpace(s.txtSecret.Text())

	if server != "" && !strings.HasPrefix(server, "ws://") && !strings.HasPrefix(server, "wss://") {
		server = "wss://" + server
		s.txtServer.SetText(server)
	}

	quality := "high"
	switch s.cmbQuality.SelectedIndex() {
	case 1:
		quality = "medium"
	case 2:
		quality = "low"
	}

	autostart := s.chkAutostart.Checked()

	s.cfg.ServerURL = server
	s.cfg.PCSecret = secret
	s.cfg.Quality = quality
	s.cfg.Autostart = autostart

	_ = config.SetAutostart(autostart)
	_ = s.cfg.Save()

	s.client.UpdateConfig(s.cfg)
	wui.MessageBox("Deskrift", "Настройки сохранены")
	s.Hide()
}

func (s *SettingsWindow) UpdateStatus(status string) {
	if s.lblStatus != nil {
		s.lblStatus.SetText("Статус: " + status)
	}
}

func (s *SettingsWindow) Show() {
	s.mu.Lock()
	if !s.isShown {
		s.isShown = true
		s.mu.Unlock()
		go func() {
			_ = s.win.Show()
			s.mu.Lock()
			s.isShown = false
			s.mu.Unlock()
		}()
		return
	}
	s.mu.Unlock()

	if s.win != nil && s.win.Handle() != 0 {
		procShowWindow.Call(s.win.Handle(), uintptr(swShow))
		procSetForegroundWindow.Call(s.win.Handle())
	}
}

func (s *SettingsWindow) Hide() {
	if s.win != nil && s.win.Handle() != 0 {
		procShowWindow.Call(s.win.Handle(), uintptr(swHide))
	}
}

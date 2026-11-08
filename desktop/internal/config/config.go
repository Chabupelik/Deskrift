package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

type Config struct {
	ServerURL string `json:"server_url"`
	PCSecret  string `json:"pc_secret"`
	Quality   string `json:"quality"`
	Autostart bool   `json:"autostart"`
}

func GetConfigPath() string {
	appData := os.Getenv("APPDATA")
	if appData != "" {
		dir := filepath.Join(appData, "Deskrift")
		_ = os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "config.json")
	}
	return "config.json"
}

func Load() *Config {
	cfg := &Config{
		Quality: "high",
	}

	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, cfg)
	}

	cfg.Autostart = IsAutostartEnabled()
	return cfg
}

func (c *Config) Save() error {
	path := GetConfigPath()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func SetAutostart(enable bool) error {
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE|registry.QUERY_VALUE,
	)
	if err != nil {
		return err
	}
	defer k.Close()

	if enable {
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		cmd := fmt.Sprintf(`"%s" --tray`, exePath)
		return k.SetStringValue("Deskrift", cmd)
	}

	_ = k.DeleteValue("Deskrift")
	return nil
}

func IsAutostartEnabled() bool {
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer k.Close()

	val, _, err := k.GetStringValue("Deskrift")
	return err == nil && val != ""
}

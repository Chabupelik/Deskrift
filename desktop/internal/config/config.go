package config

import (
	"os"
)

type Config struct {
	ServerURL string
	PCSecret  string
	// Качество: low | medium | high
	Quality string
}

func Load() *Config {
	serverURL := os.Getenv("DESKRIFT_SERVER_URL")
	if serverURL == "" {
		serverURL = "wss://chabupelik.su"
	}
	pcSecret := os.Getenv("DESKRIFT_PC_SECRET")
	quality := os.Getenv("DESKRIFT_QUALITY")
	if quality == "" {
		quality = "high"
	}

	return &Config{
		ServerURL: serverURL,
		PCSecret:  pcSecret,
		Quality:   quality,
	}
}


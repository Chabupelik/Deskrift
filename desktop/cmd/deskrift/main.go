package main

import (
	"flag"
	"os"

	"deskrift/internal/config"
	"deskrift/internal/encoder"
	"deskrift/internal/input"
	"deskrift/internal/relay"
	"deskrift/internal/ui"
)

func main() {
	trayFlag := flag.Bool("tray", false, "Start minimized to system tray")
	flag.Parse()

	cfg := config.Load()

	inputHandler := input.New()
	enc := encoder.New()
	client := relay.New(cfg, enc, inputHandler)

	win := ui.NewSettingsWindow(cfg, client)
	trayApp := ui.NewTrayApp(cfg, client, win)

	go func() {
		_ = client.Run()
	}()

	startMinimized := *trayFlag
	if !startMinimized {
		for _, arg := range os.Args[1:] {
			if arg == "--tray" || arg == "-tray" {
				startMinimized = true
				break
			}
		}
	}

	trayApp.Run(startMinimized)

	client.Stop()
	enc.Stop()
}

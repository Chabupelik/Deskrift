package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/chabupelik/deskrift/internal/config"
	"github.com/chabupelik/deskrift/internal/encoder"
	"github.com/chabupelik/deskrift/internal/input"
	"github.com/chabupelik/deskrift/internal/relay"
)

func main() {
	cfg := config.Load()

	log.Printf("Deskrift starting, server: %s", cfg.ServerURL)

	inputHandler := input.New()
	enc := encoder.New(cfg)
	client := relay.New(cfg, enc, inputHandler)

	go func() {
		if err := client.Run(); err != nil {
			log.Fatalf("relay error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	client.Stop()
	enc.Stop()
}


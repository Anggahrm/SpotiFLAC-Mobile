package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/zarz/spotiflac-bot/pkg/api"
	"github.com/zarz/spotiflac-bot/pkg/bot"
	"github.com/zarz/spotiflac-bot/pkg/config"
)

func main() {
	cfg := config.LoadConfig()

	// Check if we have bot token
	if cfg.TelegramBotToken == "" {
		fmt.Println("Warning: TELEGRAM_BOT_TOKEN not set. Telegram bot will not start.")
		fmt.Println("Only API server will be running.")
	}

	// Create channels for graceful shutdown
	done := make(chan bool)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start API server in goroutine
	go func() {
		apiServer := api.NewServer(cfg.APIPort)
		if err := apiServer.Start(); err != nil {
			fmt.Printf("API server error: %v\n", err)
		}
	}()

	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  SpotiFLAC Telegram Bot & API Server")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  API Server: http://0.0.0.0:%s\n", cfg.APIPort)
	fmt.Printf("  Download Dir: %s\n", cfg.DownloadDir)
	fmt.Println("═══════════════════════════════════════════")

	// Start Telegram bot if token is available
	if cfg.TelegramBotToken != "" {
		go func() {
			tgBot, err := bot.NewBot(cfg)
			if err != nil {
				fmt.Printf("Failed to create Telegram bot: %v\n", err)
				return
			}
			if err := tgBot.Start(); err != nil {
				fmt.Printf("Telegram bot error: %v\n", err)
			}
		}()
	}

	// Wait for shutdown signal
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		done <- true
	}()

	<-done
	fmt.Println("Goodbye!")
}

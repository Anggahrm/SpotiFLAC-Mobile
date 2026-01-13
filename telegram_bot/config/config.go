package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	// Telegram bot token from @BotFather
	TelegramBotToken string
	// API server port
	APIPort string
	// Download directory for temporary files
	DownloadDir string
	// Spotify API credentials (optional)
	SpotifyClientID     string
	SpotifyClientSecret string
	// Maximum file size for Telegram (50MB for bots)
	MaxTelegramFileSize int64
	// Debug mode
	Debug bool
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	cfg := &Config{
		TelegramBotToken:    getEnv("TELEGRAM_BOT_TOKEN", ""),
		APIPort:             getEnv("API_PORT", "8080"),
		DownloadDir:         getEnv("DOWNLOAD_DIR", "/tmp/spotiflac_downloads"),
		SpotifyClientID:     getEnv("SPOTIFY_CLIENT_ID", ""),
		SpotifyClientSecret: getEnv("SPOTIFY_CLIENT_SECRET", ""),
		MaxTelegramFileSize: 50 * 1024 * 1024, // 50MB
		Debug:               getEnv("DEBUG", "false") == "true",
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gobackend "github.com/zarz/spotiflac_android/go_backend"
	"github.com/zarz/spotiflac_telegram/config"
)

// Bot represents the Telegram bot
type Bot struct {
	api    *tgbotapi.BotAPI
	config *config.Config
}

// NewBot creates a new Telegram bot
func NewBot(cfg *config.Config) (*Bot, error) {
	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is not set")
	}

	api, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	api.Debug = cfg.Debug

	// Set Spotify credentials if provided
	if cfg.SpotifyClientID != "" && cfg.SpotifyClientSecret != "" {
		gobackend.SetSpotifyAPICredentials(cfg.SpotifyClientID, cfg.SpotifyClientSecret)
	}

	// Ensure download directory exists
	if err := os.MkdirAll(cfg.DownloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	return &Bot{
		api:    api,
		config: cfg,
	}, nil
}

// Start starts the bot and listens for updates
func (b *Bot) Start() error {
	fmt.Printf("Authorized on account %s\n", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.InlineQuery != nil {
			b.handleInlineQuery(update.InlineQuery)
			continue
		}

		if update.CallbackQuery != nil {
			b.handleCallbackQuery(update.CallbackQuery)
			continue
		}

		if update.Message == nil {
			continue
		}

		// Handle commands
		if update.Message.IsCommand() {
			b.handleCommand(update.Message)
			continue
		}

		// Handle text messages (URLs or search queries)
		b.handleMessage(update.Message)
	}

	return nil
}

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.sendStartMessage(message)
	case "help":
		b.sendHelpMessage(message)
	case "search":
		b.handleSearchCommand(message)
	case "download":
		b.handleDownloadCommand(message)
	default:
		b.sendMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
	}
}

func (b *Bot) sendStartMessage(message *tgbotapi.Message) {
	text := `🎵 *Welcome to SpotiFLAC Bot!*

I can download high-quality FLAC music from Tidal, Qobuz, and Amazon Music using Spotify or Deezer links.

*How to use:*
1. Send me a Spotify or Deezer track/album/playlist URL
2. Or use /search to find tracks

*Commands:*
/search \<query\> - Search for tracks
/help - Show help message

_No account required!_`

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) sendHelpMessage(message *tgbotapi.Message) {
	text := `🎵 *SpotiFLAC Bot Help*

*Usage:*
• Send a Spotify/Deezer URL directly
• Use /search to find tracks

*Supported URLs:*
• Spotify track: https://open.spotify.com/track/...
• Spotify album: https://open.spotify.com/album/...
• Spotify playlist: https://open.spotify.com/playlist/...
• Deezer track: https://deezer.com/track/...
• Deezer album: https://deezer.com/album/...

*Quality:*
Downloads are in FLAC format with the highest available quality (up to 24-bit/192kHz).

*Limits:*
• Telegram has a 50MB file size limit
• Large files may be uploaded as documents

*Note:* This bot is for educational purposes only.`

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

func (b *Bot) handleSearchCommand(message *tgbotapi.Message) {
	query := message.CommandArguments()
	if query == "" {
		b.sendMessage(message.Chat.ID, "Please provide a search query.\nExample: /search Never Gonna Give You Up")
		return
	}

	b.performSearch(message.Chat.ID, query)
}

func (b *Bot) handleDownloadCommand(message *tgbotapi.Message) {
	args := message.CommandArguments()
	if args == "" {
		b.sendMessage(message.Chat.ID, "Please provide a URL or track info.\nExample: /download https://open.spotify.com/track/...")
		return
	}

	b.handleMessage(&tgbotapi.Message{
		Chat: message.Chat,
		Text: args,
	})
}

func (b *Bot) handleMessage(message *tgbotapi.Message) {
	text := strings.TrimSpace(message.Text)

	// Check if it's a URL
	if strings.Contains(text, "spotify.com") || strings.Contains(text, "spotify:") ||
		strings.Contains(text, "deezer.com") {
		b.handleURL(message.Chat.ID, text)
		return
	}

	// Treat as search query
	b.performSearch(message.Chat.ID, text)
}

func (b *Bot) performSearch(chatID int64, query string) {
	// Send "searching" status
	statusMsg, _ := b.sendMessage(chatID, "🔍 Searching...")

	result, err := gobackend.SearchDeezerAll(query, 5, 0)
	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Search failed: "+err.Error())
		return
	}

	var searchResult struct {
		Tracks []struct {
			SpotifyID   string `json:"spotify_id"`
			Artists     string `json:"artists"`
			Name        string `json:"name"`
			AlbumName   string `json:"album_name"`
			DurationMS  int    `json:"duration_ms"`
			Images      string `json:"images"`
			ISRC        string `json:"isrc"`
			ReleaseDate string `json:"release_date"`
			TrackNumber int    `json:"track_number"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal([]byte(result), &searchResult); err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Failed to parse search results")
		return
	}

	if len(searchResult.Tracks) == 0 {
		b.editMessage(chatID, statusMsg.MessageID, "❌ No tracks found for: "+query)
		return
	}

	// Build response with inline keyboard
	text := "🎵 *Search Results*\n\n"
	var buttons [][]tgbotapi.InlineKeyboardButton

	for i, track := range searchResult.Tracks {
		duration := formatDuration(track.DurationMS)
		text += fmt.Sprintf("%d. *%s* - %s\n   📀 %s • %s\n\n",
			i+1, escapeMarkdown(track.Name), escapeMarkdown(track.Artists),
			escapeMarkdown(track.AlbumName), duration)

		// Create download button for each track
		callbackData := fmt.Sprintf("dl:%s", track.SpotifyID)
		if len(callbackData) > 64 {
			callbackData = callbackData[:64]
		}
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("⬇️ %d. %s", i+1, truncateString(track.Name, 30)),
				callbackData,
			),
		})
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	// Delete the "searching" message and send results
	b.api.Request(tgbotapi.NewDeleteMessage(chatID, statusMsg.MessageID))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handleURL(chatID int64, url string) {
	statusMsg, _ := b.sendMessage(chatID, "📥 Fetching metadata...")

	// Parse the URL first
	var parseResult string
	var err error

	if strings.Contains(url, "spotify.com") || strings.Contains(url, "spotify:") {
		parseResult, err = gobackend.ParseSpotifyURL(url)
	} else if strings.Contains(url, "deezer.com") {
		parseResult, err = gobackend.ParseDeezerURLExport(url)
	}

	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Invalid URL: "+err.Error())
		return
	}

	var parsed struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	json.Unmarshal([]byte(parseResult), &parsed)

	switch parsed.Type {
	case "track":
		b.handleTrackURL(chatID, statusMsg.MessageID, url, parsed.Type, parsed.ID)
	case "album":
		b.handleAlbumURL(chatID, statusMsg.MessageID, url)
	case "playlist":
		b.handlePlaylistURL(chatID, statusMsg.MessageID, url)
	default:
		b.editMessage(chatID, statusMsg.MessageID, "❌ Unsupported URL type: "+parsed.Type)
	}
}

func (b *Bot) handleTrackURL(chatID int64, statusMsgID int, url string, resourceType string, resourceID string) {
	// Fetch metadata
	var metadataResult string
	var err error

	if strings.Contains(url, "spotify.com") || strings.Contains(url, "spotify:") {
		metadataResult, err = gobackend.GetSpotifyMetadataWithDeezerFallback(url)
	} else if strings.Contains(url, "deezer.com") {
		metadataResult, err = gobackend.GetDeezerMetadata(resourceType, resourceID)
	}

	if err != nil {
		b.editMessage(chatID, statusMsgID, "❌ Failed to fetch metadata: "+err.Error())
		return
	}

	var trackData struct {
		Track struct {
			SpotifyID   string `json:"spotify_id"`
			Artists     string `json:"artists"`
			Name        string `json:"name"`
			AlbumName   string `json:"album_name"`
			AlbumArtist string `json:"album_artist"`
			DurationMS  int    `json:"duration_ms"`
			Images      string `json:"images"`
			ReleaseDate string `json:"release_date"`
			TrackNumber int    `json:"track_number"`
			TotalTracks int    `json:"total_tracks"`
			DiscNumber  int    `json:"disc_number"`
			ExternalURL string `json:"external_urls"`
			ISRC        string `json:"isrc"`
		} `json:"track"`
	}

	if err := json.Unmarshal([]byte(metadataResult), &trackData); err != nil {
		b.editMessage(chatID, statusMsgID, "❌ Failed to parse metadata")
		return
	}

	track := trackData.Track

	// Show track info with download button
	text := fmt.Sprintf("🎵 *%s*\n👤 %s\n📀 %s\n⏱ %s",
		escapeMarkdown(track.Name),
		escapeMarkdown(track.Artists),
		escapeMarkdown(track.AlbumName),
		formatDuration(track.DurationMS))

	callbackData := fmt.Sprintf("dl:%s", track.SpotifyID)
	if len(callbackData) > 64 {
		callbackData = callbackData[:64]
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬇️ Download FLAC", callbackData),
		),
	)

	// Delete status message and send track info
	b.api.Request(tgbotapi.NewDeleteMessage(chatID, statusMsgID))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handleAlbumURL(chatID int64, statusMsgID int, url string) {
	var metadataResult string
	var err error

	if strings.Contains(url, "spotify.com") || strings.Contains(url, "spotify:") {
		metadataResult, err = gobackend.GetSpotifyMetadataWithDeezerFallback(url)
	} else if strings.Contains(url, "deezer.com") {
		parseResult, _ := gobackend.ParseDeezerURLExport(url)
		var parsed struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		json.Unmarshal([]byte(parseResult), &parsed)
		metadataResult, err = gobackend.GetDeezerMetadata(parsed.Type, parsed.ID)
	}

	if err != nil {
		b.editMessage(chatID, statusMsgID, "❌ Failed to fetch album: "+err.Error())
		return
	}

	var albumData struct {
		AlbumInfo struct {
			TotalTracks int    `json:"total_tracks"`
			Name        string `json:"name"`
			ReleaseDate string `json:"release_date"`
			Artists     string `json:"artists"`
			Images      string `json:"images"`
		} `json:"album_info"`
		TrackList []struct {
			SpotifyID   string `json:"spotify_id"`
			Artists     string `json:"artists"`
			Name        string `json:"name"`
			DurationMS  int    `json:"duration_ms"`
			TrackNumber int    `json:"track_number"`
			ISRC        string `json:"isrc"`
		} `json:"track_list"`
	}

	if err := json.Unmarshal([]byte(metadataResult), &albumData); err != nil {
		b.editMessage(chatID, statusMsgID, "❌ Failed to parse album data")
		return
	}

	// Show album info
	text := fmt.Sprintf("📀 *%s*\n👤 %s\n📅 %s\n🎵 %d tracks\n\n",
		escapeMarkdown(albumData.AlbumInfo.Name),
		escapeMarkdown(albumData.AlbumInfo.Artists),
		albumData.AlbumInfo.ReleaseDate,
		albumData.AlbumInfo.TotalTracks)

	// List first 10 tracks
	maxTracks := 10
	if len(albumData.TrackList) < maxTracks {
		maxTracks = len(albumData.TrackList)
	}

	for i := 0; i < maxTracks; i++ {
		track := albumData.TrackList[i]
		text += fmt.Sprintf("%d. %s (%s)\n",
			track.TrackNumber,
			escapeMarkdown(track.Name),
			formatDuration(track.DurationMS))
	}

	if len(albumData.TrackList) > maxTracks {
		text += fmt.Sprintf("\n_...and %d more tracks_\n", len(albumData.TrackList)-maxTracks)
	}

	text += "\n⚠️ _Album downloads may take a while and large files may exceed Telegram's limit._"

	// Create download buttons for individual tracks (first 5)
	var buttons [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < 5 && i < len(albumData.TrackList); i++ {
		track := albumData.TrackList[i]
		callbackData := fmt.Sprintf("dl:%s", track.SpotifyID)
		if len(callbackData) > 64 {
			callbackData = callbackData[:64]
		}
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("⬇️ %d. %s", track.TrackNumber, truncateString(track.Name, 25)),
				callbackData,
			),
		})
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	b.api.Request(tgbotapi.NewDeleteMessage(chatID, statusMsgID))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handlePlaylistURL(chatID int64, statusMsgID int, url string) {
	// Similar to album handling
	b.editMessage(chatID, statusMsgID, "📋 Playlist support coming soon! For now, please share individual track URLs.")
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	// Answer callback to remove loading indicator
	b.api.Request(tgbotapi.NewCallback(callback.ID, ""))

	if strings.HasPrefix(data, "dl:") {
		trackID := strings.TrimPrefix(data, "dl:")
		b.downloadTrack(callback.Message.Chat.ID, trackID)
	}
}

func (b *Bot) downloadTrack(chatID int64, trackID string) {
	statusMsg, _ := b.sendMessage(chatID, "⏳ Starting download...")

	// First, get track metadata
	var metadataResult string
	var err error

	if strings.HasPrefix(trackID, "deezer:") {
		deezerID := strings.TrimPrefix(trackID, "deezer:")
		metadataResult, err = gobackend.GetDeezerMetadata("track", deezerID)
	} else {
		// Assume Spotify
		spotifyURL := "https://open.spotify.com/track/" + trackID
		metadataResult, err = gobackend.GetSpotifyMetadataWithDeezerFallback(spotifyURL)
	}

	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Failed to get track info: "+err.Error())
		return
	}

	var trackData struct {
		Track struct {
			SpotifyID   string `json:"spotify_id"`
			Artists     string `json:"artists"`
			Name        string `json:"name"`
			AlbumName   string `json:"album_name"`
			AlbumArtist string `json:"album_artist"`
			DurationMS  int    `json:"duration_ms"`
			Images      string `json:"images"`
			ReleaseDate string `json:"release_date"`
			TrackNumber int    `json:"track_number"`
			TotalTracks int    `json:"total_tracks"`
			DiscNumber  int    `json:"disc_number"`
			ISRC        string `json:"isrc"`
		} `json:"track"`
	}

	if err := json.Unmarshal([]byte(metadataResult), &trackData); err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Failed to parse track metadata")
		return
	}

	track := trackData.Track

	b.editMessage(chatID, statusMsg.MessageID, fmt.Sprintf("⬇️ Downloading: *%s* - %s",
		escapeMarkdown(track.Name), escapeMarkdown(track.Artists)))

	// Check availability
	availResult, err := gobackend.CheckAvailability(track.SpotifyID, track.ISRC)
	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Track not available on any service")
		return
	}

	var availability struct {
		TidalAvailable  bool `json:"tidal_available"`
		QobuzAvailable  bool `json:"qobuz_available"`
		AmazonAvailable bool `json:"amazon_available"`
	}
	json.Unmarshal([]byte(availResult), &availability)

	// Determine which service to use
	service := ""
	if availability.TidalAvailable {
		service = "tidal"
	} else if availability.QobuzAvailable {
		service = "qobuz"
	} else if availability.AmazonAvailable {
		service = "amazon"
	} else {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Track not available on any supported service")
		return
	}

	b.editMessage(chatID, statusMsg.MessageID, fmt.Sprintf("⬇️ Downloading from %s: *%s* - %s",
		capitalizeFirst(service), escapeMarkdown(track.Name), escapeMarkdown(track.Artists)))

	// Create unique item ID for tracking
	itemID := fmt.Sprintf("tg_%d_%d", chatID, time.Now().UnixNano())

	// Prepare download request
	downloadReq := map[string]interface{}{
		"isrc":                   track.ISRC,
		"service":                service,
		"spotify_id":             track.SpotifyID,
		"track_name":             track.Name,
		"artist_name":            track.Artists,
		"album_name":             track.AlbumName,
		"album_artist":           track.AlbumArtist,
		"cover_url":              track.Images,
		"output_dir":             b.config.DownloadDir,
		"filename_format":        "{artist} - {title}",
		"quality":                "HI_RES_LOSSLESS",
		"embed_lyrics":           true,
		"embed_max_quality_cover": true,
		"track_number":           track.TrackNumber,
		"disc_number":            track.DiscNumber,
		"total_tracks":           track.TotalTracks,
		"release_date":           track.ReleaseDate,
		"item_id":                itemID,
		"duration_ms":            track.DurationMS,
	}

	reqJSON, _ := json.Marshal(downloadReq)

	// Start download with fallback
	result, err := gobackend.DownloadWithFallback(string(reqJSON))
	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Download failed: "+err.Error())
		return
	}

	var downloadResult struct {
		Success          bool   `json:"success"`
		FilePath         string `json:"file_path"`
		Error            string `json:"error"`
		ActualBitDepth   int    `json:"actual_bit_depth"`
		ActualSampleRate int    `json:"actual_sample_rate"`
		Service          string `json:"service"`
	}

	if err := json.Unmarshal([]byte(result), &downloadResult); err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Failed to parse download result")
		return
	}

	if !downloadResult.Success {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Download failed: "+downloadResult.Error)
		return
	}

	b.editMessage(chatID, statusMsg.MessageID, "📤 Uploading to Telegram...")

	// Upload file to Telegram
	err = b.uploadFile(chatID, downloadResult.FilePath, track.Name, track.Artists,
		downloadResult.ActualBitDepth, downloadResult.ActualSampleRate)
	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "❌ Upload failed: "+err.Error())
		return
	}

	// Delete status message after successful upload
	b.api.Request(tgbotapi.NewDeleteMessage(chatID, statusMsg.MessageID))

	// Clean up downloaded file
	os.Remove(downloadResult.FilePath)
}

func (b *Bot) uploadFile(chatID int64, filePath, trackName, artists string, bitDepth, sampleRate int) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()

	// Check file size limit
	if fileSize > b.config.MaxTelegramFileSize {
		return fmt.Errorf("file too large (%.1f MB). Telegram limit is 50MB", float64(fileSize)/(1024*1024))
	}

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create file upload
	fileName := filepath.Base(filePath)
	fileBytes := tgbotapi.FileBytes{
		Name:  fileName,
		Bytes: fileContent,
	}

	// Create caption with quality info
	caption := fmt.Sprintf("🎵 *%s*\n👤 %s",
		escapeMarkdown(trackName), escapeMarkdown(artists))
	if bitDepth > 0 && sampleRate > 0 {
		caption += fmt.Sprintf("\n🎛 %d-bit / %.1f kHz", bitDepth, float64(sampleRate)/1000)
	}

	// Send as audio if it's a FLAC file
	if strings.HasSuffix(strings.ToLower(filePath), ".flac") ||
		strings.HasSuffix(strings.ToLower(filePath), ".m4a") {
		audio := tgbotapi.NewAudio(chatID, fileBytes)
		audio.Title = trackName
		audio.Performer = artists
		audio.Caption = caption
		audio.ParseMode = "Markdown"
		_, err = b.api.Send(audio)
	} else {
		// Send as document
		doc := tgbotapi.NewDocument(chatID, fileBytes)
		doc.Caption = caption
		doc.ParseMode = "Markdown"
		_, err = b.api.Send(doc)
	}

	return err
}

func (b *Bot) handleInlineQuery(query *tgbotapi.InlineQuery) {
	if query.Query == "" {
		return
	}

	result, err := gobackend.SearchDeezerAll(query.Query, 10, 0)
	if err != nil {
		return
	}

	var searchResult struct {
		Tracks []struct {
			SpotifyID  string `json:"spotify_id"`
			Artists    string `json:"artists"`
			Name       string `json:"name"`
			AlbumName  string `json:"album_name"`
			DurationMS int    `json:"duration_ms"`
			Images     string `json:"images"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal([]byte(result), &searchResult); err != nil {
		return
	}

	var results []interface{}
	for i, track := range searchResult.Tracks {
		resultID := fmt.Sprintf("%d_%s", i, track.SpotifyID)

		text := fmt.Sprintf("🎵 *%s*\n👤 %s\n📀 %s\n⏱ %s\n\n_Send this to download_",
			escapeMarkdown(track.Name),
			escapeMarkdown(track.Artists),
			escapeMarkdown(track.AlbumName),
			formatDuration(track.DurationMS))

		article := tgbotapi.NewInlineQueryResultArticle(
			resultID,
			track.Name+" - "+track.Artists,
			text,
		)
		article.Description = track.AlbumName + " • " + formatDuration(track.DurationMS)
		if track.Images != "" {
			article.ThumbURL = track.Images
		}

		results = append(results, article)
	}

	inlineConfig := tgbotapi.InlineConfig{
		InlineQueryID: query.ID,
		Results:       results,
		CacheTime:     300, // Cache for 5 minutes
	}

	b.api.Request(inlineConfig)
}

// Helper functions

func (b *Bot) sendMessage(chatID int64, text string) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	sent, err := b.api.Send(msg)
	return &sent, err
}

func (b *Bot) editMessage(chatID int64, messageID int, text string) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	b.api.Send(edit)
}

func formatDuration(ms int) string {
	seconds := ms / 1000
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func escapeMarkdown(text string) string {
	// Escape special Markdown characters
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

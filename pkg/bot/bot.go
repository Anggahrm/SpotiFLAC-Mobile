package bot

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/zarz/spotiflac-bot/pkg/backend"
	"github.com/zarz/spotiflac-bot/pkg/config"
)

// Provider options
const (
	ProviderTidal  = "tidal"
	ProviderQobuz  = "qobuz"
	ProviderAmazon = "amazon"
	ProviderAuto   = "auto"
)

// Bot represents the Telegram bot
type Bot struct {
	api    *tgbotapi.BotAPI
	config *config.Config
	// User preferences (in-memory, reset on restart)
	userProvider map[int64]string
	// Rate limiting for button clicks
	lastAction     map[int64]time.Time
	activeDownload map[int64]bool
	// Batch download storage: chatID -> batchKey -> []trackIDs
	batchTracks map[int64]map[string][]string
	mu          sync.RWMutex
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
		backend.SetSpotifyAPICredentials(cfg.SpotifyClientID, cfg.SpotifyClientSecret)
	}

	// Ensure download directory exists
	if err := os.MkdirAll(cfg.DownloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	return &Bot{
		api:            api,
		config:         cfg,
		userProvider:   make(map[int64]string),
		lastAction:     make(map[int64]time.Time),
		activeDownload: make(map[int64]bool),
		batchTracks:    make(map[int64]map[string][]string),
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
	case "provider":
		b.showProviderMenu(message.Chat.ID, 0)
	default:
		b.sendMessage(message.Chat.ID, "Unknown command. Use /help to see available commands.")
	}
}

func (b *Bot) sendStartMessage(message *tgbotapi.Message) {
	// Set default provider for new user
	userID := message.Chat.ID
	if _, ok := b.userProvider[userID]; !ok {
		b.userProvider[userID] = ProviderAuto
	}

	text := `SpotiFLAC Bot

Download high-quality FLAC music from Tidal, Qobuz, and Amazon Music using Spotify or Deezer links.

Features:
- Highest quality FLAC available on each platform
- Embedded lyrics
- Multiple provider support

Usage:
1. Send a Spotify or Deezer URL
2. Or use /search to find tracks

Commands:
/search <query> - Search for tracks
/provider - Select download provider
/help - Show help

Send a link or search query to get started.`

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.api.Send(msg)
}

func (b *Bot) sendHelpMessage(message *tgbotapi.Message) {
	text := `SpotiFLAC Bot Help

Commands:
/search <query> - Search for tracks
/provider - Select download provider

Providers:
- Tidal
- Qobuz
- Amazon Music
- Auto (tries all)

Supported URLs:
- Spotify track/album/playlist
- Deezer track/album

Note: Telegram has a 50MB file size limit.`

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	b.api.Send(msg)
}

func (b *Bot) showProviderMenu(chatID int64, messageID int) {
	currentProvider := b.getUserProvider(chatID)

	text := "Select Download Provider\n\nCurrent: " + getProviderName(currentProvider)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				getCheckmark(currentProvider == ProviderAuto)+" Auto",
				"provider:"+ProviderAuto),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				getCheckmark(currentProvider == ProviderTidal)+" Tidal",
				"provider:"+ProviderTidal),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				getCheckmark(currentProvider == ProviderQobuz)+" Qobuz",
				"provider:"+ProviderQobuz),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				getCheckmark(currentProvider == ProviderAmazon)+" Amazon Music",
				"provider:"+ProviderAmazon),
		),
	)

	if messageID > 0 {
		// Edit existing message
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		edit.ReplyMarkup = &keyboard
		b.api.Send(edit)
	} else {
		// Send new message
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = keyboard
		b.api.Send(msg)
	}
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
	statusMsg, _ := b.sendMessage(chatID, "Searching Spotify...")

	// Try Spotify Web search first
	result, err := backend.SearchSpotifyWeb(query, 10)
	if err != nil {
		// Fallback to Deezer if Spotify Web fails
		b.editMessage(chatID, statusMsg.MessageID, "Spotify failed, trying Deezer...")
		result, err = backend.SearchDeezerAll(query, 5, 0)
		if err != nil {
			b.editMessage(chatID, statusMsg.MessageID, "Search failed: "+err.Error())
			return
		}
	}

	var searchResult struct {
		Tracks []struct {
			SpotifyID   string `json:"spotify_id"`
			ID          string `json:"id"`
			Artists     string `json:"artists"`
			Name        string `json:"name"`
			AlbumName   string `json:"album_name"`
			DurationMS  int    `json:"duration_ms"`
			Images      string `json:"images"`
			ISRC        string `json:"isrc"`
			ReleaseDate string `json:"release_date"`
			TrackNumber int    `json:"track_number"`
			ItemType    string `json:"item_type"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal([]byte(result), &searchResult); err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "Failed to parse search results")
		return
	}

	if len(searchResult.Tracks) == 0 {
		b.editMessage(chatID, statusMsg.MessageID, "No tracks found for: "+query)
		return
	}

	// Build response with inline keyboard
	text := "Spotify Search Results\n\n"
	var buttons [][]tgbotapi.InlineKeyboardButton

	trackCount := 0
	for _, track := range searchResult.Tracks {
		// Only show tracks (not albums/artists/playlists)
		if track.ItemType != "" && track.ItemType != "track" {
			continue
		}
		trackCount++
		if trackCount > 8 {
			break
		}

		duration := formatDuration(track.DurationMS)
		text += fmt.Sprintf("%d. %s - %s\n   %s | %s\n\n",
			trackCount, track.Name, track.Artists,
			track.AlbumName, duration)

		// Use SpotifyID or ID
		trackID := track.SpotifyID
		if trackID == "" {
			trackID = track.ID
		}

		// Create download button for each track
		callbackData := fmt.Sprintf("dl:%s", trackID)
		if len(callbackData) > 64 {
			callbackData = callbackData[:64]
		}
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%d. %s", trackCount, truncateString(track.Name, 35)),
				callbackData,
			),
		})
	}

	if trackCount == 0 {
		b.editMessage(chatID, statusMsg.MessageID, "No tracks found for: "+query)
		return
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	// Delete the "searching" message and send results
	b.api.Request(tgbotapi.NewDeleteMessage(chatID, statusMsg.MessageID))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handleURL(chatID int64, url string) {
	statusMsg, _ := b.sendMessage(chatID, "Fetching metadata...")

	// Parse the URL first
	var parseResult string
	var err error

	if strings.Contains(url, "spotify.com") || strings.Contains(url, "spotify:") {
		parseResult, err = backend.ParseSpotifyURL(url)
	} else if strings.Contains(url, "deezer.com") {
		parseResult, err = backend.ParseDeezerURLExport(url)
	}

	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "Invalid URL: "+err.Error())
		return
	}

	var parsed struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal([]byte(parseResult), &parsed); err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "Failed to parse URL")
		return
	}

	switch parsed.Type {
	case "track":
		b.handleTrackURL(chatID, statusMsg.MessageID, url, parsed.Type, parsed.ID)
	case "album":
		b.handleAlbumURL(chatID, statusMsg.MessageID, url)
	case "playlist":
		b.handlePlaylistURL(chatID, statusMsg.MessageID, url)
	default:
		b.editMessage(chatID, statusMsg.MessageID, "Unsupported URL type: "+parsed.Type)
	}
}

func (b *Bot) handleTrackURL(chatID int64, statusMsgID int, url string, resourceType string, resourceID string) {
	// Fetch metadata
	var metadataResult string
	var err error

	if strings.Contains(url, "spotify.com") || strings.Contains(url, "spotify:") {
		metadataResult, err = backend.GetSpotifyMetadataWithDeezerFallback(url)
	} else if strings.Contains(url, "deezer.com") {
		metadataResult, err = backend.GetDeezerMetadata(resourceType, resourceID)
	}

	if err != nil {
		b.editMessage(chatID, statusMsgID, "Failed to fetch metadata: "+err.Error())
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
		b.editMessage(chatID, statusMsgID, "Failed to parse metadata")
		return
	}

	track := trackData.Track

	// Check availability - use different method based on track source
	var availResult string
	var availErr error
	if strings.HasPrefix(track.SpotifyID, "deezer:") {
		deezerID := strings.TrimPrefix(track.SpotifyID, "deezer:")
		availResult, availErr = backend.CheckAvailabilityFromDeezerID(deezerID)
	} else {
		availResult, availErr = backend.CheckAvailability(track.SpotifyID, track.ISRC)
	}
	
	var availability struct {
		Tidal  bool `json:"tidal"`
		Qobuz  bool `json:"qobuz"`
		Amazon bool `json:"amazon"`
	}
	
	// Only parse if we got a valid result
	if availErr == nil && availResult != "" {
		json.Unmarshal([]byte(availResult), &availability)
	}

	// Build availability text
	var availServices []string
	if availability.Tidal {
		availServices = append(availServices, "Tidal")
	}
	if availability.Qobuz {
		availServices = append(availServices, "Qobuz")
	}
	if availability.Amazon {
		availServices = append(availServices, "Amazon")
	}
	availText := "Not available"
	if len(availServices) > 0 {
		availText = strings.Join(availServices, ", ")
	}

	// Show track info with download button
	text := fmt.Sprintf(`%s - %s
Album: %s
Duration: %s
Available on: %s`,
		track.Name,
		track.Artists,
		track.AlbumName,
		formatDuration(track.DurationMS),
		availText)

	callbackData := fmt.Sprintf("dl:%s", track.SpotifyID)
	if len(callbackData) > 64 {
		callbackData = callbackData[:64]
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Download", callbackData),
		),
	)

	// Delete status message and send track info
	b.api.Request(tgbotapi.NewDeleteMessage(chatID, statusMsgID))

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) handleAlbumURL(chatID int64, statusMsgID int, url string) {
	var metadataResult string
	var err error

	if strings.Contains(url, "spotify.com") || strings.Contains(url, "spotify:") {
		metadataResult, err = backend.GetSpotifyMetadataWithDeezerFallback(url)
	} else if strings.Contains(url, "deezer.com") {
		parseResult, _ := backend.ParseDeezerURLExport(url)
		var parsed struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		json.Unmarshal([]byte(parseResult), &parsed)
		metadataResult, err = backend.GetDeezerMetadata(parsed.Type, parsed.ID)
	}

	if err != nil {
		b.editMessage(chatID, statusMsgID, "Failed to fetch album: "+err.Error())
		return
	}

	var albumData struct {
		AlbumInfo struct {
			TotalTracks int    `json:"total_tracks"`
			Name        string `json:"name"`
			ReleaseDate string `json:"release_date"`
			Artists     string `json:"artists"`
		} `json:"album_info"`
		TrackList []struct {
			SpotifyID string `json:"spotify_id"`
			ID        string `json:"id"`
		} `json:"track_list"`
	}

	if err := json.Unmarshal([]byte(metadataResult), &albumData); err != nil {
		b.editMessage(chatID, statusMsgID, "Failed to parse album data")
		return
	}

	// Collect track IDs
	var trackIDs []string
	for _, t := range albumData.TrackList {
		id := t.SpotifyID
		if id == "" {
			id = t.ID
		}
		if id != "" {
			trackIDs = append(trackIDs, id)
		}
	}

	if len(trackIDs) == 0 {
		b.editMessage(chatID, statusMsgID, "No tracks found in album")
		return
	}

	// Generate album ID from URL or use hash
	albumID := fmt.Sprintf("%x", md5.Sum([]byte(url)))[:16]

	// Store track IDs for batch download
	b.storeBatchTracks(chatID, "album", albumID, trackIDs)

	// Simple display
	text := fmt.Sprintf("*%s*\n", escapeMarkdown(albumData.AlbumInfo.Name))
	text += fmt.Sprintf("by %s\n", escapeMarkdown(albumData.AlbumInfo.Artists))
	if albumData.AlbumInfo.ReleaseDate != "" {
		text += fmt.Sprintf("%s\n", albumData.AlbumInfo.ReleaseDate)
	}
	text += fmt.Sprintf("\n%d tracks", len(trackIDs))

	// Single "Download All" button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("Download All (%d)", len(trackIDs)),
				fmt.Sprintf("dlall:album:%s", albumID),
			),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(chatID, statusMsgID, text)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard
	b.api.Send(editMsg)
}

func (b *Bot) handlePlaylistURL(chatID int64, statusMsgID int, url string) {
	b.editMessage(chatID, statusMsgID, "Fetching playlist...")

	// Fetch playlist using existing Spotify API with fallback
	playlistJSON, err := backend.GetSpotifyMetadataWithDeezerFallback(url)
	if err != nil {
		// Try Spotify Web API as fallback
		parseResult, parseErr := backend.ParseSpotifyURL(url)
		if parseErr == nil {
			var parsed struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			}
			if json.Unmarshal([]byte(parseResult), &parsed) == nil {
				playlistJSON, err = backend.GetSpotifyWebPlaylist(parsed.ID)
			}
		}
		if err != nil {
			b.editMessage(chatID, statusMsgID, "Failed to fetch playlist: "+err.Error())
			return
		}
	}

	// Parse the response
	var playlistName, playlistOwner, playlistID string
	var trackIDs []string

	// Try new format first (playlist_info + tracks)
	var newFormat struct {
		PlaylistInfo struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Owner       string `json:"owner"`
			TotalTracks int    `json:"total_tracks"`
		} `json:"playlist_info"`
		Tracks []struct {
			ID        string `json:"id"`
			SpotifyID string `json:"spotify_id"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal([]byte(playlistJSON), &newFormat); err == nil && newFormat.PlaylistInfo.Name != "" {
		playlistName = newFormat.PlaylistInfo.Name
		playlistOwner = newFormat.PlaylistInfo.Owner
		playlistID = newFormat.PlaylistInfo.ID
		for _, t := range newFormat.Tracks {
			id := t.SpotifyID
			if id == "" {
				id = t.ID
			}
			if id != "" {
				trackIDs = append(trackIDs, id)
			}
		}
	}

	if len(trackIDs) == 0 {
		b.editMessage(chatID, statusMsgID, "No tracks found in playlist")
		return
	}

	// Store track IDs for batch download
	b.storeBatchTracks(chatID, "playlist", playlistID, trackIDs)

	// Simple display: name, owner, track count, download all button
	text := fmt.Sprintf("*%s*\n", escapeMarkdown(playlistName))
	if playlistOwner != "" {
		text += fmt.Sprintf("by %s\n", escapeMarkdown(playlistOwner))
	}
	text += fmt.Sprintf("\n%d tracks", len(trackIDs))

	// Single "Download All" button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("Download All (%d)", len(trackIDs)),
				fmt.Sprintf("dlall:playlist:%s", playlistID),
			),
		),
	)

	editMsg := tgbotapi.NewEditMessageText(chatID, statusMsgID, text)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &keyboard
	b.api.Send(editMsg)
}

// truncateString truncates a string to maxLen and adds ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// escapeMarkdown escapes special characters for Telegram Markdown
func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"`", "\\`",
	)
	return replacer.Replace(s)
}

// canPerformAction checks if user can perform an action (rate limiting)
func (b *Bot) canPerformAction(chatID int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	lastTime, exists := b.lastAction[chatID]
	if exists && time.Since(lastTime) < 2*time.Second {
		return false
	}
	b.lastAction[chatID] = time.Now()
	return true
}

// isDownloading checks if user has active download
func (b *Bot) isDownloading(chatID int64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.activeDownload[chatID]
}

// setDownloading sets download status
func (b *Bot) setDownloading(chatID int64, status bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.activeDownload[chatID] = status
}

// storeBatchTracks stores track IDs for batch download
func (b *Bot) storeBatchTracks(chatID int64, itemType, itemID string, trackIDs []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.batchTracks[chatID] == nil {
		b.batchTracks[chatID] = make(map[string][]string)
	}
	batchKey := itemType + ":" + itemID
	b.batchTracks[chatID][batchKey] = trackIDs
}

// getBatchTracks retrieves stored track IDs
func (b *Bot) getBatchTracks(chatID int64, batchKey string) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.batchTracks[chatID] == nil {
		return nil
	}
	return b.batchTracks[chatID][batchKey]
}

// downloadBatch downloads all tracks in a batch
func (b *Bot) downloadBatch(chatID int64, messageID int, batchKey string) {
	trackIDs := b.getBatchTracks(chatID, batchKey)
	if len(trackIDs) == 0 {
		b.editMessage(chatID, messageID, "No tracks found. Please send the URL again.")
		return
	}

	b.setDownloading(chatID, true)
	defer b.setDownloading(chatID, false)

	total := len(trackIDs)
	success := 0
	failed := 0

	for i, trackID := range trackIDs {
		// Update progress
		progress := fmt.Sprintf("Downloading %d/%d...", i+1, total)
		b.editMessage(chatID, messageID, progress)

		// Download track silently (no individual messages)
		err := b.downloadTrackSilent(chatID, trackID)
		if err != nil {
			failed++
		} else {
			success++
		}
	}

	// Final summary
	summary := fmt.Sprintf("Download complete\n\nSuccess: %d\nFailed: %d\nTotal: %d", success, failed, total)
	b.editMessage(chatID, messageID, summary)
}

// downloadTrackSilent downloads a track and sends the file, returns error
func (b *Bot) downloadTrackSilent(chatID int64, trackID string) error {
	// Get track metadata
	var metadataResult string
	var err error

	if strings.HasPrefix(trackID, "deezer:") {
		deezerID := strings.TrimPrefix(trackID, "deezer:")
		metadataResult, err = backend.GetDeezerMetadata("track", deezerID)
	} else {
		spotifyURL := "https://open.spotify.com/track/" + trackID
		metadataResult, err = backend.GetSpotifyMetadataWithDeezerFallback(spotifyURL)
	}

	if err != nil {
		return err
	}

	var trackData struct {
		Track struct {
			SpotifyID   string `json:"spotify_id"`
			Name        string `json:"name"`
			Artists     string `json:"artists"`
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
		return fmt.Errorf("parse metadata: %w", err)
	}

	track := trackData.Track
	if track.ISRC == "" && track.SpotifyID == "" {
		return fmt.Errorf("no ISRC or SpotifyID")
	}

	// Check availability
	var availResult string
	if strings.HasPrefix(track.SpotifyID, "deezer:") {
		deezerID := strings.TrimPrefix(track.SpotifyID, "deezer:")
		availResult, err = backend.CheckAvailabilityFromDeezerID(deezerID)
	} else if track.SpotifyID != "" {
		availResult, err = backend.CheckAvailability(track.SpotifyID, track.ISRC)
	} else {
		availResult, err = backend.CheckAvailability("", track.ISRC)
	}

	if err != nil {
		return fmt.Errorf("availability: %w", err)
	}

	var availability struct {
		Tidal  bool `json:"tidal"`
		Qobuz  bool `json:"qobuz"`
		Amazon bool `json:"amazon"`
	}
	if err := json.Unmarshal([]byte(availResult), &availability); err != nil {
		return fmt.Errorf("parse availability: %w", err)
	}

	// Determine service
	provider := b.getUserProvider(chatID)
	service := ""
	if provider == ProviderAuto {
		if availability.Tidal {
			service = ProviderTidal
		} else if availability.Qobuz {
			service = ProviderQobuz
		} else if availability.Amazon {
			service = ProviderAmazon
		}
	} else {
		switch provider {
		case ProviderTidal:
			if availability.Tidal {
				service = ProviderTidal
			}
		case ProviderQobuz:
			if availability.Qobuz {
				service = ProviderQobuz
			}
		case ProviderAmazon:
			if availability.Amazon {
				service = ProviderAmazon
			}
		}
		// Fallback if preferred not available
		if service == "" {
			if availability.Tidal {
				service = ProviderTidal
			} else if availability.Qobuz {
				service = ProviderQobuz
			} else if availability.Amazon {
				service = ProviderAmazon
			}
		}
	}

	if service == "" {
		return fmt.Errorf("not available")
	}

	// Prepare download request
	itemID := fmt.Sprintf("tg_%d_%d", chatID, time.Now().UnixNano())
	downloadReq := map[string]interface{}{
		"isrc":                    track.ISRC,
		"service":                 service,
		"spotify_id":              track.SpotifyID,
		"track_name":              track.Name,
		"artist_name":             track.Artists,
		"album_name":              track.AlbumName,
		"album_artist":            track.AlbumArtist,
		"cover_url":               track.Images,
		"output_dir":              b.config.DownloadDir,
		"filename_format":         "{artist} - {title}",
		"quality":                 "HI_RES_LOSSLESS",
		"embed_lyrics":            true,
		"embed_max_quality_cover": true,
		"track_number":            track.TrackNumber,
		"disc_number":             track.DiscNumber,
		"total_tracks":            track.TotalTracks,
		"release_date":            track.ReleaseDate,
		"item_id":                 itemID,
		"duration_ms":             track.DurationMS,
	}

	reqJSON, _ := json.Marshal(downloadReq)

	// Download
	var result string
	result, err = backend.DownloadWithFallback(string(reqJSON))
	if err != nil {
		return fmt.Errorf("download: %w", err)
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
		return fmt.Errorf("parse result: %w", err)
	}

	if !downloadResult.Success {
		return fmt.Errorf(downloadResult.Error)
	}

	// Upload to Telegram
	err = b.uploadFile(chatID, downloadResult.FilePath, track.Name, track.Artists,
		downloadResult.ActualBitDepth, downloadResult.ActualSampleRate, downloadResult.Service)

	// Clean up
	os.Remove(downloadResult.FilePath)

	return err
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	// Rate limiting
	if !b.canPerformAction(chatID) {
		b.api.Request(tgbotapi.NewCallback(callback.ID, "Please wait..."))
		return
	}

	// Handle different callback types
	switch {
	case strings.HasPrefix(data, "dlall:"):
		// Download all tracks from playlist/album
		if b.isDownloading(chatID) {
			b.api.Request(tgbotapi.NewCallback(callback.ID, "Download in progress"))
			return
		}
		b.api.Request(tgbotapi.NewCallback(callback.ID, "Starting batch download..."))
		// Parse: dlall:type:id
		parts := strings.SplitN(strings.TrimPrefix(data, "dlall:"), ":", 2)
		if len(parts) == 2 {
			batchKey := parts[0] + ":" + parts[1]
			go b.downloadBatch(chatID, messageID, batchKey)
		}

	case strings.HasPrefix(data, "dl:"):
		// Check if already downloading
		if b.isDownloading(chatID) {
			b.api.Request(tgbotapi.NewCallback(callback.ID, "Download in progress"))
			return
		}
		b.api.Request(tgbotapi.NewCallback(callback.ID, "Starting download..."))
		trackID := strings.TrimPrefix(data, "dl:")
		go b.downloadTrack(chatID, trackID)

	case strings.HasPrefix(data, "provider:"):
		provider := strings.TrimPrefix(data, "provider:")
		b.userProvider[chatID] = provider
		b.api.Request(tgbotapi.NewCallback(callback.ID, "Provider: "+getProviderName(provider)))
		b.showProviderMenu(chatID, messageID)

	default:
		b.api.Request(tgbotapi.NewCallback(callback.ID, ""))
	}
}

func (b *Bot) downloadTrack(chatID int64, trackID string) {
	// Mark as downloading
	b.setDownloading(chatID, true)
	defer b.setDownloading(chatID, false)

	statusMsg, _ := b.sendMessage(chatID, "Starting download...")

	// First, get track metadata
	var metadataResult string
	var err error

	if strings.HasPrefix(trackID, "deezer:") {
		deezerID := strings.TrimPrefix(trackID, "deezer:")
		metadataResult, err = backend.GetDeezerMetadata("track", deezerID)
	} else {
		// Assume Spotify
		spotifyURL := "https://open.spotify.com/track/" + trackID
		metadataResult, err = backend.GetSpotifyMetadataWithDeezerFallback(spotifyURL)
	}

	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "Failed to get track info: "+err.Error())
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
		b.editMessage(chatID, statusMsg.MessageID, "Failed to parse track metadata")
		return
	}

	track := trackData.Track

	// Get user provider preference
	provider := b.getUserProvider(chatID)

	b.editMessage(chatID, statusMsg.MessageID, fmt.Sprintf("Checking availability: %s - %s", track.Name, track.Artists))

	// Check availability - use different method based on track source
	var availResult string
	var availErr error

	if strings.HasPrefix(track.SpotifyID, "deezer:") {
		deezerID := strings.TrimPrefix(track.SpotifyID, "deezer:")
		availResult, availErr = backend.CheckAvailabilityFromDeezerID(deezerID)
	} else if track.SpotifyID != "" {
		availResult, availErr = backend.CheckAvailability(track.SpotifyID, track.ISRC)
	} else if track.ISRC != "" {
		availResult, availErr = backend.CheckAvailability("", track.ISRC)
	} else {
		b.editMessage(chatID, statusMsg.MessageID, "Track has no identifier for availability check")
		return
	}

	if availErr != nil {
		b.editMessage(chatID, statusMsg.MessageID, "Track not available: "+availErr.Error())
		return
	}

	var availability struct {
		Tidal  bool `json:"tidal"`
		Qobuz  bool `json:"qobuz"`
		Amazon bool `json:"amazon"`
	}
	if err := json.Unmarshal([]byte(availResult), &availability); err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "Failed to check availability")
		return
	}

	// Determine which service to use based on user preference
	service := ""
	if provider == ProviderAuto {
		if availability.Tidal {
			service = ProviderTidal
		} else if availability.Qobuz {
			service = ProviderQobuz
		} else if availability.Amazon {
			service = ProviderAmazon
		}
	} else {
		switch provider {
		case ProviderTidal:
			if availability.Tidal {
				service = ProviderTidal
			}
		case ProviderQobuz:
			if availability.Qobuz {
				service = ProviderQobuz
			}
		case ProviderAmazon:
			if availability.Amazon {
				service = ProviderAmazon
			}
		}
	}

	if service == "" {
		// If selected provider not available, try others
		if provider != ProviderAuto {
			b.editMessage(chatID, statusMsg.MessageID, fmt.Sprintf("Not available on %s, trying other providers...", getProviderName(provider)))
			if availability.Tidal {
				service = ProviderTidal
			} else if availability.Qobuz {
				service = ProviderQobuz
			} else if availability.Amazon {
				service = ProviderAmazon
			}
		}

		if service == "" {
			b.editMessage(chatID, statusMsg.MessageID, "Track not available on any supported service")
			return
		}
	}

	b.editMessage(chatID, statusMsg.MessageID, fmt.Sprintf("Downloading from %s: %s - %s",
		capitalizeFirst(service), track.Name, track.Artists))

	// Create unique item ID for tracking
	itemID := fmt.Sprintf("tg_%d_%d", chatID, time.Now().UnixNano())

	// Prepare download request - always use highest quality
	downloadReq := map[string]interface{}{
		"isrc":                    track.ISRC,
		"service":                 service,
		"spotify_id":              track.SpotifyID,
		"track_name":              track.Name,
		"artist_name":             track.Artists,
		"album_name":              track.AlbumName,
		"album_artist":            track.AlbumArtist,
		"cover_url":               track.Images,
		"output_dir":              b.config.DownloadDir,
		"filename_format":         "{artist} - {title}",
		"quality":                 "HI_RES_LOSSLESS", // Always highest quality
		"embed_lyrics":            true,              // Always embed lyrics
		"embed_max_quality_cover": true,
		"track_number":            track.TrackNumber,
		"disc_number":             track.DiscNumber,
		"total_tracks":            track.TotalTracks,
		"release_date":            track.ReleaseDate,
		"item_id":                 itemID,
		"duration_ms":             track.DurationMS,
	}

	reqJSON, _ := json.Marshal(downloadReq)

	// Start download
	var result string
	if provider == ProviderAuto {
		result, err = backend.DownloadWithFallback(string(reqJSON))
	} else {
		result, err = backend.DownloadTrack(string(reqJSON))
	}

	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "Download failed: "+err.Error())
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
		b.editMessage(chatID, statusMsg.MessageID, "Failed to parse download result")
		return
	}

	if !downloadResult.Success {
		b.editMessage(chatID, statusMsg.MessageID, "Download failed: "+downloadResult.Error)
		return
	}

	b.editMessage(chatID, statusMsg.MessageID, "Uploading to Telegram...")

	// Upload file to Telegram
	err = b.uploadFile(chatID, downloadResult.FilePath, track.Name, track.Artists,
		downloadResult.ActualBitDepth, downloadResult.ActualSampleRate, downloadResult.Service)
	if err != nil {
		b.editMessage(chatID, statusMsg.MessageID, "Upload failed: "+err.Error())
		return
	}

	// Delete status message after successful upload
	b.api.Request(tgbotapi.NewDeleteMessage(chatID, statusMsg.MessageID))

	// Clean up downloaded file
	os.Remove(downloadResult.FilePath)
}

func (b *Bot) uploadFile(chatID int64, filePath, trackName, artists string, bitDepth, sampleRate int, service string) error {
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
	caption := fmt.Sprintf("%s - %s\nSource: %s", trackName, artists, capitalizeFirst(service))
	if bitDepth > 0 && sampleRate > 0 {
		caption += fmt.Sprintf("\nQuality: %d-bit / %.1f kHz", bitDepth, float64(sampleRate)/1000)
	}

	// Send as audio if it's a FLAC file
	if strings.HasSuffix(strings.ToLower(filePath), ".flac") ||
		strings.HasSuffix(strings.ToLower(filePath), ".m4a") {
		audio := tgbotapi.NewAudio(chatID, fileBytes)
		audio.Title = trackName
		audio.Performer = artists
		audio.Caption = caption
		_, err = b.api.Send(audio)
	} else {
		// Send as document
		doc := tgbotapi.NewDocument(chatID, fileBytes)
		doc.Caption = caption
		_, err = b.api.Send(doc)
	}

	return err
}

func (b *Bot) handleInlineQuery(query *tgbotapi.InlineQuery) {
	if query.Query == "" {
		return
	}

	// Try Spotify Web search first
	result, err := backend.SearchSpotifyWeb(query.Query, 10)
	if err != nil {
		// Fallback to Deezer
		result, err = backend.SearchDeezerAll(query.Query, 10, 0)
		if err != nil {
			return
		}
	}

	var searchResult struct {
		Tracks []struct {
			SpotifyID  string `json:"spotify_id"`
			ID         string `json:"id"`
			Artists    string `json:"artists"`
			Name       string `json:"name"`
			AlbumName  string `json:"album_name"`
			DurationMS int    `json:"duration_ms"`
			Images     string `json:"images"`
			ItemType   string `json:"item_type"`
		} `json:"tracks"`
	}

	if err := json.Unmarshal([]byte(result), &searchResult); err != nil {
		return
	}

	var results []interface{}
	for i, track := range searchResult.Tracks {
		// Only show tracks
		if track.ItemType != "" && track.ItemType != "track" {
			continue
		}

		trackID := track.SpotifyID
		if trackID == "" {
			trackID = track.ID
		}

		resultID := fmt.Sprintf("%d_%s", i, trackID)

		text := fmt.Sprintf("%s - %s\nAlbum: %s\nDuration: %s",
			track.Name,
			track.Artists,
			track.AlbumName,
			formatDuration(track.DurationMS))

		article := tgbotapi.NewInlineQueryResultArticle(
			resultID,
			track.Name+" - "+track.Artists,
			text,
		)
		article.Description = track.AlbumName + " | " + formatDuration(track.DurationMS)
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

func (b *Bot) getUserProvider(chatID int64) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if p, ok := b.userProvider[chatID]; ok {
		return p
	}
	return ProviderAuto // Default
}

func (b *Bot) sendMessage(chatID int64, text string) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	sent, err := b.api.Send(msg)
	return &sent, err
}

func (b *Bot) editMessage(chatID int64, messageID int, text string) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	b.api.Send(edit)
}

func formatDuration(ms int) string {
	seconds := ms / 1000
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func getProviderName(provider string) string {
	switch provider {
	case ProviderTidal:
		return "Tidal"
	case ProviderQobuz:
		return "Qobuz"
	case ProviderAmazon:
		return "Amazon Music"
	case ProviderAuto:
		return "Auto"
	default:
		return provider
	}
}

func getCheckmark(selected bool) string {
	if selected {
		return "[x]"
	}
	return "[ ]"
}

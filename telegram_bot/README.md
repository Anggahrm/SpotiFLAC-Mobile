# SpotiFLAC Telegram Bot & API

A Telegram bot and REST API for downloading high-quality FLAC music from Tidal, Qobuz, and Amazon Music using Spotify or Deezer links.

## Features

### Telegram Bot
- 🔍 Search for tracks by name/artist
- 📥 Download tracks from Spotify/Deezer URLs
- 🎵 Automatic FLAC quality selection (up to 24-bit/192kHz)
- 📋 Album and playlist browsing
- 🎤 Embedded lyrics
- 🖼️ Album artwork embedding
- 🔄 Inline query support for quick search

### REST API
- `/api/search` - Search for tracks on Deezer or Spotify
- `/api/metadata` - Get track/album/playlist metadata
- `/api/availability` - Check track availability on streaming services
- `/api/download` - Download a track
- `/api/progress` - Check download progress
- `/api/lyrics` - Fetch lyrics for a track

## Setup

### Environment Variables

```bash
# Required for Telegram bot
export TELEGRAM_BOT_TOKEN="your_bot_token_from_botfather"

# Optional
export API_PORT="8080"                    # API server port (default: 8080)
export DOWNLOAD_DIR="/tmp/spotiflac"      # Download directory
export SPOTIFY_CLIENT_ID="your_id"        # Spotify API credentials (optional)
export SPOTIFY_CLIENT_SECRET="your_secret"
export DEBUG="false"                      # Enable debug mode
```

### Building

```bash
cd telegram_bot
go build -o spotiflac-bot .
```

### Running

```bash
# Run both bot and API
./spotiflac-bot

# Or using go run
go run main.go
```

## Usage

### Telegram Bot Commands

- `/start` - Start the bot and see welcome message
- `/help` - Show help information
- `/search <query>` - Search for tracks
- `/download <url>` - Download from URL

### Direct Messages

Simply send a Spotify or Deezer URL to download:
- `https://open.spotify.com/track/...`
- `https://open.spotify.com/album/...`
- `https://deezer.com/track/...`

Or send a search query to find tracks.

### Inline Queries

Type `@YourBotUsername query` in any chat to search for tracks inline.

## API Documentation

### Search Tracks

```
GET /api/search?q=<query>&source=deezer&track_limit=10&artist_limit=3
```

**Parameters:**
- `q` (required): Search query
- `source`: "deezer" or "spotify" (default: deezer)
- `track_limit`: Max tracks to return (default: 10)
- `artist_limit`: Max artists to return (default: 3)

### Get Metadata

```
GET /api/metadata?url=<spotify_or_deezer_url>
```

### Check Availability

```
GET /api/availability?spotify_id=<id>&isrc=<isrc>
GET /api/availability?deezer_id=<id>
```

### Download Track

```
POST /api/download
Content-Type: application/json

{
  "isrc": "USRC12345678",
  "service": "tidal",
  "spotify_id": "4iV5W9uYEdYUVa79Axb7Rh",
  "track_name": "Never Gonna Give You Up",
  "artist_name": "Rick Astley",
  "album_name": "Whenever You Need Somebody",
  "quality": "HI_RES_LOSSLESS",
  "embed_lyrics": true
}
```

### Get Lyrics

```
GET /api/lyrics?track_name=<name>&artist_name=<artist>&spotify_id=<id>
```

## Limitations

- Telegram has a 50MB file size limit for bots
- Some tracks may not be available on all services
- Rate limits may apply on search APIs

## Disclaimer

This project is for **educational and private use only**. The developer does not condone or encourage copyright infringement.

Users are solely responsible for:
1. Ensuring their use of this software complies with local laws
2. Reading and adhering to the Terms of Service of the respective platforms
3. Any legal consequences resulting from the misuse of this tool

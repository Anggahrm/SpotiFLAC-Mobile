# SpotiFLAC Bot

A Telegram bot and REST API for downloading high-quality FLAC music from Tidal, Qobuz, and Amazon Music using Spotify or Deezer links.

## Features

### 🎛 Quality Selection
- **FLAC Lossless** (16-bit/44.1kHz)
- **Hi-Res FLAC** (24-bit/96kHz)
- **Hi-Res FLAC Max** (24-bit/192kHz)

### 📦 Provider Selection
- 🔷 **Tidal**
- 🟣 **Qobuz**
- 🟠 **Amazon Music**
- 🔄 **Auto** (tries all providers)

### 🎤 Lyrics Support
- Synced lyrics embedding
- Toggle on/off per user

### 🔍 Search & Download
- Search by track name/artist
- Direct Spotify/Deezer URL support
- Album browsing
- Inline query support

## Telegram Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Start the bot |
| `/help` | Show help |
| `/search <query>` | Search for tracks |
| `/settings` | View/change settings |
| `/quality` | Change audio quality |
| `/provider` | Select download provider |
| `/lyrics` | Toggle lyrics embedding |

## Deployment

### Environment Variables

```bash
TELEGRAM_BOT_TOKEN=your_bot_token    # Required for bot
PORT=8080                             # API port (Heroku sets this)
DOWNLOAD_DIR=/tmp/downloads           # Temporary download directory
SPOTIFY_CLIENT_ID=                    # Optional - for Spotify search
SPOTIFY_CLIENT_SECRET=                # Optional - for Spotify search
DEBUG=false                           # Enable debug mode
```

### Deploy to Heroku

1. Create a new Heroku app:
```bash
heroku create your-app-name
```

2. Set the bot token:
```bash
heroku config:set TELEGRAM_BOT_TOKEN=your_token
```

3. Deploy:
```bash
git push heroku main
```

### Run Locally

```bash
# Set environment variables
export TELEGRAM_BOT_TOKEN="your_token"

# Build and run
go build -o spotiflac-bot .
./spotiflac-bot
```

### Docker

```bash
docker build -t spotiflac-bot .
docker run -e TELEGRAM_BOT_TOKEN="your_token" -p 8080:8080 spotiflac-bot
```

## REST API Documentation

### Health Check
```
GET /health
```
Response:
```json
{"success": true, "data": {"status": "healthy"}}
```

### Search Tracks
```
GET /api/search?q=<query>&source=deezer&track_limit=10&artist_limit=3
```
Parameters:
- `q` (required): Search query
- `source`: `deezer` (default) or `spotify`
- `track_limit`: Max tracks (default: 10)
- `artist_limit`: Max artists (default: 3)

Example:
```bash
curl "http://localhost:8080/api/search?q=wildflower&track_limit=5"
```

### Get Metadata
```
GET /api/metadata?url=<spotify_or_deezer_url>
```
Example:
```bash
curl "http://localhost:8080/api/metadata?url=https://open.spotify.com/track/4iV5W9uYEdYUVa79Axb7Rh"
```

### Check Availability
```
GET /api/availability?spotify_id=<id>&isrc=<isrc>
GET /api/availability?deezer_id=<id>
```
Example:
```bash
curl "http://localhost:8080/api/availability?deezer_id=123456"
```
Response:
```json
{
  "success": true,
  "data": {
    "tidal": true,
    "qobuz": true,
    "amazon": false,
    "tidal_url": "https://tidal.com/track/...",
    "qobuz_url": "https://www.qobuz.com/..."
  }
}
```

### Download Track
```
POST /api/download
Content-Type: application/json
```
Body:
```json
{
  "isrc": "USRC12345678",
  "service": "tidal",
  "track_name": "Track Name",
  "artist_name": "Artist Name",
  "album_name": "Album Name",
  "quality": "HI_RES_LOSSLESS",
  "embed_lyrics": true,
  "output_dir": "/tmp/downloads"
}
```

Quality options:
- `LOSSLESS` - 16-bit/44.1kHz
- `HI_RES` - 24-bit/96kHz
- `HI_RES_LOSSLESS` - 24-bit/192kHz

Service options:
- `tidal`
- `qobuz`
- `amazon`

### Get Download Progress
```
GET /api/progress
GET /api/progress?item_id=<id>
```

### Fetch Lyrics
```
GET /api/lyrics?track_name=<name>&artist_name=<artist>&spotify_id=<id>
```
Example:
```bash
curl "http://localhost:8080/api/lyrics?track_name=Wildflower&artist_name=Billie%20Eilish"
```

## Project Structure

```
.
├── main.go              # Entry point
├── Procfile             # Heroku process file
├── go.mod               # Go module
├── pkg/
│   ├── api/             # REST API server
│   ├── bot/             # Telegram bot
│   ├── backend/         # Download core
│   └── config/          # Configuration
└── README.md
```

## License

MIT License - See [LICENSE](LICENSE) file.

## Disclaimer

This project is for **educational and personal use only**. Users are responsible for ensuring their use complies with applicable laws and terms of service.

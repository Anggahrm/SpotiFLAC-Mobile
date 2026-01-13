# SpotiFLAC Bot

A Telegram bot and REST API for downloading high-quality FLAC/M4A music from Tidal, Qobuz, and Amazon Music using Spotify or Deezer links.

## Features

### Audio Quality
- Always downloads the **highest quality** available from each provider
- **Tidal**: Up to 24-bit/192kHz (Hi-Res FLAC or ALAC/M4A for DASH streams)
- **Qobuz**: Up to 24-bit/192kHz FLAC
- **Amazon Music**: Up to 24-bit/192kHz FLAC

### Provider Selection
- **Tidal** - Primary source
- **Qobuz** - High-quality alternative
- **Amazon Music** - Fallback option
- **Auto** - Tries all providers in order

### Additional Features
- Lyrics always embedded when available
- Search by track name/artist
- Direct Spotify/Deezer URL support
- Album browsing
- Inline query support

### Note on File Formats
- **Qobuz & Amazon**: Files are downloaded as `.flac`
- **Tidal**: May be `.flac` (BTS format) or `.m4a` (DASH streams)
  - DASH streams contain ALAC audio in M4A container
  - Quality is equivalent - only the container differs
  - Telegram supports both formats natively

## Telegram Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Start the bot and see welcome message |
| `/help` | Show available commands |
| `/search <query>` | Search for tracks |
| `/provider` | Select download provider |

## Usage

1. Send a Spotify or Deezer URL directly to the bot
2. Or use `/search` to find tracks
3. Click on a track to download
4. File will be sent when download completes

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

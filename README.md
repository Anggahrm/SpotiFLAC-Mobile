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

## Commands

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
TELEGRAM_BOT_TOKEN=your_bot_token    # Required
PORT=8080                             # API port (Heroku sets this)
DOWNLOAD_DIR=/tmp/downloads           # Temporary download directory
SPOTIFY_CLIENT_ID=                    # Optional
SPOTIFY_CLIENT_SECRET=                # Optional
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

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/search` | GET | Search tracks |
| `/api/metadata` | GET | Get track metadata |
| `/api/availability` | GET | Check provider availability |
| `/api/download` | POST | Download a track |
| `/api/progress` | GET | Get download progress |
| `/api/lyrics` | GET | Fetch lyrics |

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

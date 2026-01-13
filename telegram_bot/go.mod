module github.com/zarz/spotiflac_telegram

go 1.24.0

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/zarz/spotiflac_android/go_backend v0.0.0
)

require (
	github.com/go-flac/flacpicture v0.3.0 // indirect
	github.com/go-flac/flacvorbis v0.2.0 // indirect
	github.com/go-flac/go-flac v1.0.0 // indirect
)

replace github.com/zarz/spotiflac_android/go_backend => ../go_backend

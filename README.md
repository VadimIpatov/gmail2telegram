# Gmail to Telegram Forwarder 🚀

[![codecov](https://codecov.io/gh/vadimipatov/gmail2telegram/graph/badge.svg?token=YOUR_TOKEN)](https://codecov.io/gh/vadimipatov/gmail2telegram)

A Go application that forwards emails from Gmail to Telegram, with automatic translation to a specified language using Google's Gemini.

> 🎨 This project was created as an expirment using the "vibe coding" approach in the Cursor IDE ✨

## Features

- Monitors Gmail inbox for new messages
- Filters messages based on sender, subject keywords, and content keywords
- Translates content to a specified language
- Forwards messages to Telegram channel or chat
- Configurable polling interval
- Docker support

## Prerequisites

- Go 1.24 or later
- Gmail API credentials
- Telegram Bot Token
- Gemini API key for translation

## Configuration

Create a `config.yaml` file with the following structure:

```yaml
gmail:
  credentials_file: "credentials.json"
  token_file: "token.json"
  poll_interval: "1m"
  forwarded_label: "Forwarded"
  filter:
    from:
      - "example@gmail.com"
    subject_keywords:
      - "important"
    content_keywords:
      - "urgent"

telegram:
  bot_token: "your_bot_token"
  channel_id: "your_channel_id"
  chat_id: "your_chat_id"

translation:
  gemini_api_key: "your_gemini_api_key"
  target_language: "ru"
  model_name: "gemini-2.0-flash"
```

## Building and Running

### Local Build

1. Clone the repository:
   ```bash
   git clone https://github.com/VadimIpatov/gmail2telegram.git
   cd gmail2telegram
   ```

2. Install dependencies:
   ```bash
   make deps
   ```

3. Build the application:
   ```bash
   make build
   ```

4. Run the application:
   ```bash
   ./gmail2telegram
   ```

### Using Make

The project includes a Makefile with common commands:

```bash
make deps     # Install dependencies
make build    # Build the application
make run      # Run the application
make test     # Run tests
make clean    # Clean build artifacts
make lint     # Run linter
make token    # Generate Gmail token
```

### Using Docker

1. Build the Docker image:
   ```bash
   docker build -t gmail2telegram .
   ```

2. Run the container:
   ```bash
   docker run -d \
     --name gmail2telegram \
     -v /path/to/credentials.json:/app/credentials.json \
     -v /path/to/token.json:/app/token.json \
     -v /path/to/config.yaml:/app/config.yaml \
     gmail2telegram
   ```

Replace `/path/to/credentials.json`, `/path/to/token.json`, and `/path/to/config.yaml` with the actual paths to your files.

## Gmail API Setup

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Gmail API
4. Create OAuth 2.0 credentials
5. Download the credentials and save as `credentials.json`
6. Run the application once to generate the token file

## Telegram Setup

1. Create a new bot using [@BotFather](https://t.me/botfather)
2. Get the bot token
3. Add the bot to your channel/chat
4. Get the channel/chat ID
5. Update the `config.yaml` with your bot token and channel/chat ID

## Translation Service

The application uses Google's Gemini API for translation. You'll need to:

1. Get a Gemini API key from the [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Add the API key to your `config.yaml`

## Project Structure

```
gmail2telegram/
├── src/                  # Source code
├── config.yaml           # Application configuration
├── credentials.json      # Gmail API credentials
├── token.json            # Gmail OAuth token
├── go.mod                # Go module definition
├── go.sum                # Go dependencies checksum
├── Makefile              # Build automation
└── README.md             # Project documentation
```

## License

MIT License 
# Gmail to Telegram Forwarder

[![codecov](https://codecov.io/gh/vadimipatov/gmail2telegram/graph/badge.svg?token=YOUR_TOKEN)](https://codecov.io/gh/vadimipatov/gmail2telegram)

Forwards emails from Gmail to a Telegram channel with automatic translation via Google Gemini.

## Features

- Polls Gmail inbox at a configurable interval
- Filters messages by sender, subject keywords, and content keywords
- Translates content to a target language using Gemini
- Forwards messages to a Telegram channel or chat
- Handles multipart MIME emails including HTML-only messages
- Configurable prompt template for translation behaviour
- Docker support

## Prerequisites

- Go 1.24+
- Gmail API OAuth2 credentials
- Telegram Bot Token
- Gemini API key

## Configuration

Copy `config.yaml.example` to `config.yaml` and fill in your values:

```yaml
gmail:
  credentials_file: "credentials.json"
  token_file: "token.json"
  poll_interval: "15m"
  forwarded_label: "fwd"
  filter:
    from:
      - "@example.com"
    # subject_keywords:
    #   - "important"
    # content_keywords:
    #   - "urgent"

telegram:
  bot_token: "your_bot_token"
  channel_id: "-100your_channel_id"
  chat_id: "-100your_channel_id"

translation:
  gemini_api_key: "your_gemini_api_key"
  target_language: "Russian"
  model_name: "gemini-2.5-flash"
  # prompt_template: "..."  # optional, see default in translation.go
```

`prompt_template` supports `{target_language}` and `{text}` variables.

## Development

```bash
make deps      # install dependencies
make build     # build binary
make run       # run locally
make test      # run tests
make lint      # gofumpt + golangci-lint
make lint-fix  # auto-fix lint issues
make token     # generate Gmail OAuth token
```

Always run `make test` and `make lint` before committing.

## Docker

```bash
docker build -t gmail2telegram .

docker run -d \
  --name gmail2telegram \
  --restart unless-stopped \
  -v /path/to/credentials.json:/app/credentials.json \
  -v /path/to/token.json:/app/token.json \
  -v /path/to/config.yaml:/app/config.yaml \
  gmail2telegram ./gmail2telegram -config /app/config.yaml
```

## Gmail API Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a project and enable the Gmail API
3. Create OAuth 2.0 credentials (Desktop application)
4. Download and save as `credentials.json`
5. Run `make token` to generate `token.json`

## Telegram Setup

1. Create a bot via [@BotFather](https://t.me/botfather) and copy the token
2. Add the bot to your channel/chat as an admin
3. Get the channel/chat ID (use `@userinfobot` or the Telegram API)
4. Fill in `bot_token` and `channel_id`/`chat_id` in config

## Project Structure

```
gmail2telegram/
├── src/
│   ├── main.go          # config, main loop, service wiring
│   ├── gmail.go         # Gmail API client, MIME parsing, filtering
│   ├── translation.go   # Gemini translation service
│   └── telegram.go      # Telegram Bot API client
├── Dockerfile
├── Makefile
├── config.yaml.example
├── CLAUDE.md            # developer context for AI-assisted development
└── REVIEW.md            # code review findings
```

## License

MIT

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

type TelegramBot struct {
	client    *http.Client
	botToken  string
	channelID string
	chatID    string
	baseURL   string
}

func NewTelegramBot(config *Config) (*TelegramBot, error) {
	if config.Telegram.BotToken == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}

	return &TelegramBot{
		client:    &http.Client{},
		botToken:  config.Telegram.BotToken,
		channelID: config.Telegram.ChannelID,
		chatID:    config.Telegram.ChatID,
		baseURL:   "https://api.telegram.org/bot" + config.Telegram.BotToken,
	}, nil
}

func (b *TelegramBot) SendMessage(
	ctx context.Context,
	subject, content, from, date string,
	originalContent string,
) error {
	message := fmt.Sprintf("*%s*\n\n", subject)
	message += fmt.Sprintf("📅 %s\n", date)
	message += fmt.Sprintf("📧 From: %s\n\n", from)

	if originalContent != "" {
		message += fmt.Sprintf("🇷🇺 Translation:\n%s\n\n", content)
		message += fmt.Sprintf("🇬🇧 Original:\n%s", originalContent)
	} else {
		message += content
	}

	// Try to send to channel first
	if b.channelID != "" {
		if err := b.sendToChat(ctx, b.channelID, message); err == nil {
			return nil
		}
	}

	// Fallback to chat if channel fails or is not configured
	if b.chatID != "" {
		return b.sendToChat(ctx, b.chatID, message)
	}

	return fmt.Errorf("neither channel_id nor chat_id is configured")
}

func (b *TelegramBot) sendToChat(ctx context.Context, chatID, message string) error {
	apiURL, err := url.Parse(b.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %v", err)
	}

	apiURL.Path = path.Join(apiURL.Path, "sendMessage")

	params := url.Values{}
	params.Add("chat_id", chatID)
	params.Add("text", message)
	params.Add("parse_mode", "Markdown")

	apiURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}

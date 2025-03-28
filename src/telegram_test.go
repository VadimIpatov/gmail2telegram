package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewTelegramBot(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Telegram: struct {
					BotToken  string `yaml:"bot_token"`
					ChannelID string `yaml:"channel_id"`
					ChatID    string `yaml:"chat_id"`
				}{
					BotToken: "test-token",
				},
			},
			wantErr: false,
		},
		{
			name: "missing bot token",
			config: &Config{
				Telegram: struct {
					BotToken  string `yaml:"bot_token"`
					ChannelID string `yaml:"channel_id"`
					ChatID    string `yaml:"chat_id"`
				}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bot, err := NewTelegramBot(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTelegramBot() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr && bot == nil {
				t.Error("NewTelegramBot() returned nil without error")
			}
		})
	}
}

func TestSendMessage(t *testing.T) {
	tests := []struct {
		name            string
		bot             *TelegramBot
		subject         string
		content         string
		from            string
		date            string
		originalContent string
		serverResponse  func(w http.ResponseWriter, r *http.Request)
		wantErr         bool
	}{
		{
			name: "successful send to channel",
			bot: &TelegramBot{
				client:    &http.Client{},
				botToken:  "test-token",
				channelID: "test-channel",
				baseURL:   "http://test-server",
			},
			subject:         "Test Subject",
			content:         "Test Content",
			from:            "test@example.com",
			date:            "2024-03-28",
			originalContent: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "channel fails, fallback to chat",
			bot: &TelegramBot{
				client:    &http.Client{},
				botToken:  "test-token",
				channelID: "test-channel",
				chatID:    "test-chat",
				baseURL:   "http://test-server",
			},
			subject:         "Test Subject",
			content:         "Test Content",
			from:            "test@example.com",
			date:            "2024-03-28",
			originalContent: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/bot/test-token/sendMessage" && r.URL.Query().Get("chat_id") == "test-channel" {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			},
			wantErr: false,
		},
		{
			name: "both channel and chat fail",
			bot: &TelegramBot{
				client:    &http.Client{},
				botToken:  "test-token",
				channelID: "test-channel",
				chatID:    "test-chat",
				baseURL:   "http://test-server",
			},
			subject:         "Test Subject",
			content:         "Test Content",
			from:            "test@example.com",
			date:            "2024-03-28",
			originalContent: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "neither channel nor chat configured",
			bot: &TelegramBot{
				client:   &http.Client{},
				botToken: "test-token",
				baseURL:  "http://test-server",
			},
			subject:         "Test Subject",
			content:         "Test Content",
			from:            "test@example.com",
			date:            "2024-03-28",
			originalContent: "",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			tt.bot.baseURL = server.URL

			err := tt.bot.SendMessage(context.Background(), tt.subject, tt.content, tt.from, tt.date, tt.originalContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

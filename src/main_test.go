package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"google.golang.org/api/gmail/v1"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
gmail:
  credentials_file: "credentials.json"
  token_file: "token.json"
  poll_interval: "1m"
  forwarded_label: "Forwarded"
  filter:
    from: ["test@example.com"]
    subject_keywords: ["test"]
    content_keywords: ["test"]
telegram:
  bot_token: "test-token"
  channel_id: "test-channel"
  chat_id: "test-chat"
translation:
  gemini_api_key: "test-key"
  target_language: "en"
  model_name: "test-model"
  prompt_template: "Translate to {target_language}: {text}"
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}

	// Test loading config
	config, err := loadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	// Verify config values
	if config.Gmail.CredentialsFile != "credentials.json" {
		t.Errorf("Expected credentials_file to be 'credentials.json', got %s", config.Gmail.CredentialsFile)
	}

	if config.Gmail.PollInterval != "1m" {
		t.Errorf("Expected poll_interval to be '1m', got %s", config.Gmail.PollInterval)
	}

	if config.Gmail.ForwardedLabel != "Forwarded" {
		t.Errorf("Expected forwarded_label to be 'Forwarded', got %s", config.Gmail.ForwardedLabel)
	}

	if len(config.Gmail.Filter.From) != 1 || config.Gmail.Filter.From[0] != "test@example.com" {
		t.Errorf("Expected from filter to contain 'test@example.com', got %v", config.Gmail.Filter.From)
	}

	if config.Telegram.BotToken != "test-token" {
		t.Errorf("Expected bot_token to be 'test-token', got %s", config.Telegram.BotToken)
	}

	if config.Translation.TargetLanguage != "en" {
		t.Errorf("Expected target_language to be 'en', got %s", config.Translation.TargetLanguage)
	}
}

func TestProcessMessage(t *testing.T) {
	// Create test message
	msg := Message{
		ID:      "test-id",
		Subject: "Test Subject",
		Content: "Test Content",
		From:    "test@example.com",
		Date:    "2024-03-28",
	}

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create mock services
	mockTranslationService := &TranslationService{
		config: &Config{
			Translation: struct {
				GeminiAPIKey   string `yaml:"gemini_api_key"`
				TargetLanguage string `yaml:"target_language"`
				ModelName      string `yaml:"model_name"`
				PromptTemplate string `yaml:"prompt_template"`
			}{
				TargetLanguage: "en",
				PromptTemplate: "Translate to {target_language}: {text}",
			},
		},
		translate: func(ctx context.Context, text string) (string, error) {
			return "Translated: " + text, nil
		},
	}

	mockTelegramBot := &TelegramBot{
		client:    server.Client(),
		botToken:  "test-token",
		channelID: "test-channel",
		chatID:    "test-chat",
		baseURL:   server.URL,
	}

	mockGmailClient := &GmailClient{
		labelID: "test-label",
		markAsForwarded: func(ctx context.Context, messageID string) error {
			return nil
		},
	}

	// Test processing message
	ctx := context.Background()

	err := processMessage(ctx, msg, mockTranslationService, mockTelegramBot, mockGmailClient)
	if err != nil {
		t.Errorf("processMessage failed: %v", err)
	}
}

func TestProcessMessages(_ *testing.T) {
	// Create test messages
	messages := []Message{
		{
			ID:      "test-id-1",
			Subject: "Test Subject 1",
			Content: "Test Content 1",
			From:    "test1@example.com",
			Date:    "2024-03-28",
		},
		{
			ID:      "test-id-2",
			Subject: "Test Subject 2",
			Content: "Test Content 2",
			From:    "test2@example.com",
			Date:    "2024-03-28",
		},
	}

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create mock services
	mockTranslationService := &TranslationService{
		config: &Config{
			Translation: struct {
				GeminiAPIKey   string `yaml:"gemini_api_key"`
				TargetLanguage string `yaml:"target_language"`
				ModelName      string `yaml:"model_name"`
				PromptTemplate string `yaml:"prompt_template"`
			}{
				TargetLanguage: "en",
				PromptTemplate: "Translate to {target_language}: {text}",
			},
		},
		translate: func(ctx context.Context, text string) (string, error) {
			return "Translated: " + text, nil
		},
	}

	mockTelegramBot := &TelegramBot{
		client:    server.Client(),
		botToken:  "test-token",
		channelID: "test-channel",
		chatID:    "test-chat",
		baseURL:   server.URL,
	}

	mockGmailClient := &GmailClient{
		labelID: "test-label",
		markAsForwarded: func(ctx context.Context, messageID string) error {
			return nil
		},
	}

	// Test processing messages
	ctx := context.Background()
	processMessages(ctx, messages, mockTranslationService, mockTelegramBot, mockGmailClient)
}

func TestStartMessageProcessing(_ *testing.T) {
	// Create test messages
	testMessages := []Message{
		{
			ID:      "test-id-1",
			Subject: "Test Subject 1",
			Content: "Test Content 1",
			From:    "test1@example.com",
			Date:    "2024-03-28",
		},
	}

	// Create test HTTP server with longer timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a small delay to test timeout handling
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Track if we've returned the message
	messageReturned := false

	// Create mock services
	mockService := NewMockGmailService()
	mockService.labels = []*gmail.Label{
		{Id: "test-label", Name: "test-label"},
	}

	mockGmailClient := &GmailClient{
		service: mockService,
		labelID: "test-label",
		config: &Config{
			Gmail: struct {
				CredentialsFile string `yaml:"credentials_file"`
				TokenFile       string `yaml:"token_file"`
				PollInterval    string `yaml:"poll_interval"`
				ForwardedLabel  string `yaml:"forwarded_label"`
				Filter          struct {
					From            []string `yaml:"from"`
					SubjectKeywords []string `yaml:"subject_keywords"`
					ContentKeywords []string `yaml:"content_keywords"`
				} `yaml:"filter"`
			}{
				ForwardedLabel: "test-label",
			},
		},
		getNewMessages: func(ctx context.Context) ([]Message, error) {
			if messageReturned {
				return []Message{}, nil
			}
			messageReturned = true
			return testMessages, nil
		},
		markAsForwarded: func(ctx context.Context, messageID string) error {
			return nil
		},
	}

	mockTranslationService := &TranslationService{
		config: &Config{
			Translation: struct {
				GeminiAPIKey   string `yaml:"gemini_api_key"`
				TargetLanguage string `yaml:"target_language"`
				ModelName      string `yaml:"model_name"`
				PromptTemplate string `yaml:"prompt_template"`
			}{
				TargetLanguage: "en",
				PromptTemplate: "Translate to {target_language}: {text}",
			},
		},
		translate: func(ctx context.Context, text string) (string, error) {
			return "Translated: " + text, nil
		},
	}

	mockTelegramBot := &TelegramBot{
		client:    server.Client(),
		botToken:  "test-token",
		channelID: "test-channel",
		chatID:    "test-chat",
		baseURL:   server.URL,
	}

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start message processing with a short poll interval
	startMessageProcessing(ctx, 50*time.Millisecond, mockGmailClient, mockTranslationService, mockTelegramBot)
}

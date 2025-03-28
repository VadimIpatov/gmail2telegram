package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Gmail struct {
		CredentialsFile string `yaml:"credentials_file"`
		TokenFile       string `yaml:"token_file"`
		PollInterval    string `yaml:"poll_interval"`
		ForwardedLabel  string `yaml:"forwarded_label"`
		Filter          struct {
			From            []string `yaml:"from"`
			SubjectKeywords []string `yaml:"subject_keywords"`
			ContentKeywords []string `yaml:"content_keywords"`
		} `yaml:"filter"`
	} `yaml:"gmail"`
	Telegram struct {
		BotToken  string `yaml:"bot_token"`
		ChannelID string `yaml:"channel_id"`
		ChatID    string `yaml:"chat_id"`
	} `yaml:"telegram"`
	Translation struct {
		GeminiAPIKey   string `yaml:"gemini_api_key"`
		TargetLanguage string `yaml:"target_language"`
		ModelName      string `yaml:"model_name"`
	} `yaml:"translation"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func processMessage(
	ctx context.Context,
	msg Message,
	translationService *TranslationService,
	telegramBot *TelegramBot,
	gmailClient *GmailClient,
) error {
	log.Printf("Processing message: %s", msg.Subject)

	// Process message content
	log.Printf("Processing message content...")

	translatedContent, err := translationService.Translate(ctx, msg.Content)
	if err != nil {
		return fmt.Errorf("error processing message content: %w", err)
	}

	// Send to Telegram
	log.Printf("Sending message to Telegram...")

	err = telegramBot.SendMessage(ctx, msg.Subject, translatedContent, msg.From, msg.Date, "")
	if err != nil {
		return fmt.Errorf("error sending message to Telegram: %w", err)
	}

	log.Printf("Message processing completed successfully")

	// Mark message as forwarded
	log.Printf("Marking message as forwarded in Gmail...")

	err = gmailClient.MarkAsForwarded(ctx, msg.ID)
	if err != nil {
		return fmt.Errorf("error marking message as forwarded: %w", err)
	}

	log.Println("Message marked as forwarded successfully")

	return nil
}

func processMessages(
	ctx context.Context,
	messages []Message,
	translationService *TranslationService,
	telegramBot *TelegramBot,
	gmailClient *GmailClient,
) {
	for i, msg := range messages {
		log.Printf("Processing message %d/%d: %s", i+1, len(messages), msg.Subject)

		err := processMessage(ctx, msg, translationService, telegramBot, gmailClient)
		if err != nil {
			log.Printf("Error processing message: %v", err)

			continue
		}

		log.Printf("Message %d/%d processed successfully", i+1, len(messages))
	}
}

func startMessageProcessing(
	ctx context.Context,
	pollInterval time.Duration,
	gmailClient *GmailClient,
	translationService *TranslationService,
	telegramBot *TelegramBot,
) {
	// Process messages immediately on startup
	log.Println("Performing initial message check...")

	messages, err := gmailClient.GetNewMessages(ctx)
	if err != nil {
		log.Printf("Error getting new messages: %v", err)
	} else if len(messages) > 0 {
		log.Printf("Found %d new messages to process", len(messages))
		processMessages(ctx, messages, translationService, telegramBot, gmailClient)
	}

	// Start regular polling with ticker
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Message processing loop stopped")

			return

		case <-ticker.C:
			log.Println("Checking for new messages...")

			messages, err = gmailClient.GetNewMessages(ctx)
			if err != nil {
				log.Printf("Error getting new messages: %v", err)

				continue
			}

			if len(messages) > 0 {
				log.Printf("Found %d new messages to process", len(messages))
				processMessages(ctx, messages, translationService, telegramBot, gmailClient)
			}
		}
	}
}

func initializeServices(config *Config) (*GmailClient, *TranslationService, *TelegramBot, error) {
	// Initialize Gmail client
	log.Println("Initializing Gmail client...")

	gmailClient, err := NewGmailClient(context.Background(), config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Gmail client: %w", err)
	}

	log.Println("Gmail client initialized successfully")

	// Initialize translation service
	log.Println("Initializing translation service...")

	translationService, err := NewTranslationService(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create translation service: %w", err)
	}

	log.Println("Translation service initialized successfully")

	// Initialize Telegram bot
	log.Println("Initializing Telegram bot...")

	telegramBot, err := NewTelegramBot(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	log.Println("Telegram bot initialized successfully")

	return gmailClient, translationService, telegramBot, nil
}

func main() {
	log.Println("Starting Gmail to Telegram forwarder...")

	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	generateToken := flag.Bool("generate-token", false, "Generate Gmail OAuth token")
	flag.Parse()

	log.Printf("Loading configuration from %s...", *configPath)

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Configuration loaded successfully")

	// Parse poll interval
	pollInterval, err := time.ParseDuration(config.Gmail.PollInterval)
	if err != nil {
		log.Fatalf("Invalid poll interval: %v", err)
	}

	log.Printf("Poll interval set to %v", pollInterval)

	ctx, cancel := context.WithCancel(context.Background())

	if *generateToken {
		log.Println("Gmail OAuth token generated successfully")
		cancel()

		return
	}

	defer cancel()

	// Initialize all services
	gmailClient, translationService, telegramBot, err := initializeServices(config)
	if err != nil {
		cancel()
		// nolint: gocritic
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Start message processing
	log.Println("Starting message processing loop...")

	messageProcessor := startMessageProcessing

	go messageProcessor(ctx, pollInterval, gmailClient, translationService, telegramBot)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()
}

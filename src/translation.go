package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	defaultModelName      = "gemini-2.0-flash"
	defaultPromptTemplate = "Clean up and translate the following email to {target_language}.\n\nKeep ALL meaningful content.\nRemove only pure technical noise: email footer links (\"Unsubscribe\", \"Update settings\", \"Read more on ...\"), navigation menus, and system-generated metadata.\nTranslate every non-{target_language} word. Return ONLY the result, without any additional text, markers, or explanations:\n\n{text}"
)

type TranslationService struct {
	client    *genai.Client
	config    *Config
	translate func(ctx context.Context, text string) (string, error)
}

func NewTranslationService(config *Config) (*TranslationService, error) {
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(config.Translation.GeminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %v", err)
	}

	service := &TranslationService{
		client: client,
		config: config,
	}
	service.translate = service.defaultTranslate

	return service, nil
}

func (s *TranslationService) Close() {
	if s.client != nil {
		s.client.Close()
	}
}

func (s *TranslationService) Translate(ctx context.Context, text string) (string, error) {
	return s.translate(ctx, text)
}

// TODO:
// 2025/04/01 06:22:56 Processing message 1/1: Aprīļa rēķins
// 2025/04/01 06:22:56 Processing message: Aprīļa rēķins
// 2025/04/01 06:22:56 Processing message content...
// 2025/04/01 06:22:56 Error processing message: error processing message content: empty text provided for translation
func (s *TranslationService) defaultTranslate(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("empty text provided for translation")
	}

	// Use the configured model name or fall back to a default
	modelName := s.config.Translation.ModelName
	if modelName == "" {
		modelName = defaultModelName
	}

	// Use the configured prompt template or fall back to default
	promptTemplate := s.config.Translation.PromptTemplate
	if promptTemplate == "" {
		promptTemplate = defaultPromptTemplate
	}

	// Replace variables in the prompt template
	prompt := strings.ReplaceAll(promptTemplate, "{target_language}", s.config.Translation.TargetLanguage)
	prompt = strings.ReplaceAll(prompt, "{text}", text)

	model := s.client.GenerativeModel(modelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return strings.TrimSpace(fmt.Sprint(resp.Candidates[0].Content.Parts[0])), nil
}

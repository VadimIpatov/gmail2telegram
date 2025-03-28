package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	defaultModelName = "gemini-2.0-flash"
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

func (s *TranslationService) defaultTranslate(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("empty text provided for translation")
	}

	// Use the configured model name or fall back to a default
	modelName := s.config.Translation.ModelName
	if modelName == "" {
		modelName = defaultModelName
	}

	model := s.client.GenerativeModel(modelName)
	prompt := fmt.Sprintf(`Translate this text to %s. Translate ALL non-%s parts of the text, 
including English, Latvian, and any other languages. Keep %s text unchanged. 
Preserve all formatting (bold, italic, etc.) and line breaks. Return ONLY the result, 
without any additional text, markers, or explanations:

%s`, s.config.Translation.TargetLanguage, s.config.Translation.TargetLanguage,
		s.config.Translation.TargetLanguage, text)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response from model")
	}

	return strings.TrimSpace(fmt.Sprint(resp.Candidates[0].Content.Parts[0])), nil
}

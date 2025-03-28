package main

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/api/gmail/v1"
)

// TestGmailService is a test helper for mocking Gmail service responses
type TestGmailService struct {
	service *gmail.Service
}

func NewTestGmailService() *TestGmailService {
	return &TestGmailService{
		service: &gmail.Service{},
	}
}

func (s *TestGmailService) Users() *gmail.UsersService {
	return s.service.Users
}

// MockGmailService implements GmailServiceInterface for testing
type MockGmailService struct {
	labels   []*gmail.Label
	messages []*gmail.Message
	err      error
}

// MockUsersService implements the necessary Users methods for testing
type MockUsersService struct {
	service *MockGmailService
}

// MockLabelsService handles label-related operations
type MockLabelsService struct {
	service *MockGmailService
}

// MockMessagesService handles message-related operations
type MockMessagesService struct {
	service *MockGmailService
}

func NewMockGmailService() *MockGmailService {
	return &MockGmailService{}
}

func (m *MockGmailService) Users() GmailUsersInterface {
	return &MockUsersService{service: m}
}

func (s *MockUsersService) Labels() GmailLabelsInterface {
	return &MockLabelsService{service: s.service}
}

func (s *MockUsersService) Messages() GmailMessagesInterface {
	return &MockMessagesService{service: s.service}
}

func (s *MockLabelsService) List(userId string) ([]*gmail.Label, error) {
	if s.service.err != nil {
		return nil, s.service.err
	}
	return s.service.labels, nil
}

func (s *MockLabelsService) Create(userId string, label *gmail.Label) (*gmail.Label, error) {
	if s.service.err != nil {
		return nil, s.service.err
	}
	newLabel := &gmail.Label{
		Id:   "new_label_id",
		Name: label.Name,
	}
	s.service.labels = append(s.service.labels, newLabel)
	return newLabel, nil
}

func (s *MockMessagesService) List(userId string, q string) ([]*gmail.Message, error) {
	if s.service.err != nil {
		return nil, s.service.err
	}
	return s.service.messages, nil
}

func (s *MockMessagesService) Get(userId string, id string) (*gmail.Message, error) {
	if s.service.err != nil {
		return nil, s.service.err
	}
	for _, msg := range s.service.messages {
		if msg.Id == id {
			return msg, nil
		}
	}
	return nil, fmt.Errorf("message not found")
}

func (s *MockMessagesService) Modify(userId string, id string, mods *gmail.ModifyMessageRequest) (*gmail.Message, error) {
	if s.service.err != nil {
		return nil, s.service.err
	}
	for _, msg := range s.service.messages {
		if msg.Id == id {
			// Apply modifications
			msg.LabelIds = append(msg.LabelIds, mods.AddLabelIds...)
			return msg, nil
		}
	}
	return nil, fmt.Errorf("message not found")
}

func TestShouldProcessMessage(t *testing.T) {
	tests := []struct {
		name           string
		msg            Message
		config         *Config
		expectedResult bool
	}{
		{
			name: "no keywords specified",
			msg: Message{
				Subject: "Test Subject",
				Content: "Test Content",
			},
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
				}{},
			},
			expectedResult: true,
		},
		{
			name: "matching subject keyword",
			msg: Message{
				Subject: "Test Subject",
				Content: "Test Content",
			},
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
					Filter: struct {
						From            []string `yaml:"from"`
						SubjectKeywords []string `yaml:"subject_keywords"`
						ContentKeywords []string `yaml:"content_keywords"`
					}{
						SubjectKeywords: []string{"test"},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "matching content keyword",
			msg: Message{
				Subject: "Test Subject",
				Content: "Test Content",
			},
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
					Filter: struct {
						From            []string `yaml:"from"`
						SubjectKeywords []string `yaml:"subject_keywords"`
						ContentKeywords []string `yaml:"content_keywords"`
					}{
						ContentKeywords: []string{"test"},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "no matching keywords",
			msg: Message{
				Subject: "Different Subject",
				Content: "Different Content",
			},
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
					Filter: struct {
						From            []string `yaml:"from"`
						SubjectKeywords []string `yaml:"subject_keywords"`
						ContentKeywords []string `yaml:"content_keywords"`
					}{
						SubjectKeywords: []string{"test"},
						ContentKeywords: []string{"test"},
					},
				},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GmailClient{
				config: tt.config,
			}

			result := client.shouldProcessMessage(tt.msg)
			if result != tt.expectedResult {
				t.Errorf("shouldProcessMessage() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      *gmail.Message
		expected Message
		wantErr  bool
	}{
		{
			name: "simple message with plain text",
			msg: &gmail.Message{
				Id: "123",
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{Name: "Subject", Value: "Test Subject"},
						{Name: "From", Value: "test@example.com"},
						{Name: "Date", Value: "2024-03-28"},
					},
					Body: &gmail.MessagePartBody{
						Data: "SGVsbG8gV29ybGQ=", // "Hello World" in base64
					},
				},
			},
			expected: Message{
				ID:      "123",
				Subject: "Test Subject",
				From:    "test@example.com",
				Date:    "2024-03-28",
				Content: "Hello World",
			},
			wantErr: false,
		},
		{
			name: "multipart message with text/plain",
			msg: &gmail.Message{
				Id: "123",
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{Name: "Subject", Value: "Test Subject"},
						{Name: "From", Value: "test@example.com"},
						{Name: "Date", Value: "2024-03-28"},
					},
					Parts: []*gmail.MessagePart{
						{
							MimeType: "text/plain",
							Body: &gmail.MessagePartBody{
								Data: "SGVsbG8gV29ybGQ=",
							},
						},
						{
							MimeType: "text/html",
							Body: &gmail.MessagePartBody{
								Data: "PGgxPkhlbGxvPC9oMT4=",
							},
						},
					},
				},
			},
			expected: Message{
				ID:      "123",
				Subject: "Test Subject",
				From:    "test@example.com",
				Date:    "2024-03-28",
				Content: "Hello World",
			},
			wantErr: false,
		},
		{
			name: "message with no content",
			msg: &gmail.Message{
				Id: "123",
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{Name: "Subject", Value: "Test Subject"},
						{Name: "From", Value: "test@example.com"},
						{Name: "Date", Value: "2024-03-28"},
					},
					Body: &gmail.MessagePartBody{},
					Parts: []*gmail.MessagePart{
						{
							MimeType: "text/plain",
							Body:     &gmail.MessagePartBody{},
						},
					},
				},
			},
			expected: Message{
				ID:      "123",
				Subject: "Test Subject",
				From:    "test@example.com",
				Date:    "2024-03-28",
				Content: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GmailClient{
				config: &Config{},
			}

			got, err := client.parseMessage(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("parseMessage() = %+v, want %+v", got, tt.expected)
			}
		})
	}
}

func TestEnsureLabelExists(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		labels        []*gmail.Label
		err           error
		expectedLabel string
		wantErr       bool
	}{
		{
			name: "label exists",
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
					ForwardedLabel: "Forwarded",
				},
			},
			labels: []*gmail.Label{
				{Id: "label1", Name: "Forwarded"},
			},
			expectedLabel: "label1",
			wantErr:       false,
		},
		{
			name: "label needs to be created",
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
					ForwardedLabel: "Forwarded",
				},
			},
			labels:        []*gmail.Label{},
			expectedLabel: "new_label_id",
			wantErr:       false,
		},
		{
			name: "list labels error",
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
					ForwardedLabel: "Forwarded",
				},
			},
			err:     fmt.Errorf("list error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GmailClient{
				config: tt.config,
			}

			mockService := NewMockGmailService()
			mockService.labels = tt.labels
			mockService.err = tt.err
			client.service = mockService

			got, err := client.ensureLabelExists(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureLabelExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expectedLabel {
				t.Errorf("ensureLabelExists() = %v, want %v", got, tt.expectedLabel)
			}
		})
	}
}

func TestGetNewMessages(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		messages      []*gmail.Message
		err           error
		expectedCount int
		wantErr       bool
	}{
		{
			name: "successful message retrieval",
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
					ForwardedLabel: "Forwarded",
				},
			},
			messages: []*gmail.Message{
				{
					Id: "msg1",
					Payload: &gmail.MessagePart{
						Headers: []*gmail.MessagePartHeader{
							{Name: "Subject", Value: "Test Subject"},
							{Name: "From", Value: "test@example.com"},
							{Name: "Date", Value: "2024-03-28"},
						},
						Body: &gmail.MessagePartBody{
							Data: "SGVsbG8gV29ybGQ=",
						},
					},
				},
			},
			expectedCount: 1,
			wantErr:       false,
		},
		{
			name: "list messages error",
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
					ForwardedLabel: "Forwarded",
				},
			},
			err:     fmt.Errorf("list error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GmailClient{
				config: tt.config,
			}

			mockService := NewMockGmailService()
			mockService.messages = tt.messages
			mockService.err = tt.err
			client.service = mockService

			got, err := client.GetNewMessages(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNewMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.expectedCount {
				t.Errorf("GetNewMessages() returned %d messages, want %d", len(got), tt.expectedCount)
			}
		})
	}
}

package main

import (
	"testing"

	"google.golang.org/api/gmail/v1"
)

func TestParseMessagePart(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected string
		wantErr  bool
	}{
		{
			name:     "empty data",
			data:     "",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "valid base64 data",
			data:     "SGVsbG8gV29ybGQ=",
			expected: "Hello World",
			wantErr:  false,
		},
		{
			name:    "invalid base64 data",
			data:    "invalid-base64!",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GmailClient{}
			part := &gmail.MessagePart{
				Body: &gmail.MessagePartBody{
					Data: tt.data,
				},
			}

			got, err := client.parseMessagePart(part)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMessagePart() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if got != tt.expected {
				t.Errorf("parseMessagePart() = %v, want %v", got, tt.expected)
			}
		})
	}
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

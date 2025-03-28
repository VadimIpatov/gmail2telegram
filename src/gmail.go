package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type Message struct {
	ID      string
	Subject string
	Content string
	From    string
	Date    string
}

// GmailServiceInterface defines the interface for Gmail service operations
type GmailServiceInterface interface {
	Users() GmailUsersInterface
}

// GmailUsersInterface defines the interface for Gmail users operations
type GmailUsersInterface interface {
	Labels() GmailLabelsInterface
	Messages() GmailMessagesInterface
}

// GmailLabelsInterface defines the interface for Gmail labels operations
type GmailLabelsInterface interface {
	List(userId string) ([]*gmail.Label, error)
	Create(userId string, label *gmail.Label) (*gmail.Label, error)
}

// GmailMessagesInterface defines the interface for Gmail messages operations
type GmailMessagesInterface interface {
	List(userId string, q string) ([]*gmail.Message, error)
	Get(userId string, id string) (*gmail.Message, error)
	Modify(userId string, id string, mods *gmail.ModifyMessageRequest) (*gmail.Message, error)
}

// GmailServiceWrapper wraps the Gmail service for easier mocking in tests
type GmailServiceWrapper struct {
	service *gmail.Service
}

// NewGmailServiceWrapper creates a new wrapper around the Gmail service
func NewGmailServiceWrapper(service *gmail.Service) *GmailServiceWrapper {
	return &GmailServiceWrapper{service: service}
}

// GmailUsersWrapper wraps the Gmail users service
type GmailUsersWrapper struct {
	service *gmail.Service
}

// GmailLabelsWrapper wraps the Gmail labels service
type GmailLabelsWrapper struct {
	service *gmail.Service
}

// GmailMessagesWrapper wraps the Gmail messages service
type GmailMessagesWrapper struct {
	service *gmail.Service
}

func (w *GmailServiceWrapper) Users() GmailUsersInterface {
	return &GmailUsersWrapper{service: w.service}
}

func (w *GmailUsersWrapper) Labels() GmailLabelsInterface {
	return &GmailLabelsWrapper{service: w.service}
}

func (w *GmailUsersWrapper) Messages() GmailMessagesInterface {
	return &GmailMessagesWrapper{service: w.service}
}

func (w *GmailLabelsWrapper) List(userId string) ([]*gmail.Label, error) {
	resp, err := w.service.Users.Labels.List(userId).Do()
	if err != nil {
		return nil, err
	}
	return resp.Labels, nil
}

func (w *GmailLabelsWrapper) Create(userId string, label *gmail.Label) (*gmail.Label, error) {
	return w.service.Users.Labels.Create(userId, label).Do()
}

func (w *GmailMessagesWrapper) List(userId string, q string) ([]*gmail.Message, error) {
	resp, err := w.service.Users.Messages.List(userId).Q(q).Do()
	if err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

func (w *GmailMessagesWrapper) Get(userId string, id string) (*gmail.Message, error) {
	return w.service.Users.Messages.Get(userId, id).Do()
}

func (w *GmailMessagesWrapper) Modify(userId string, id string, mods *gmail.ModifyMessageRequest) (*gmail.Message, error) {
	return w.service.Users.Messages.Modify(userId, id, mods).Do()
}

// GmailClient struct
type GmailClient struct {
	service         GmailServiceInterface
	config          *Config
	labelID         string
	getNewMessages  func(ctx context.Context) ([]Message, error)
	markAsForwarded func(ctx context.Context, messageID string) error
}

func NewGmailClient(ctx context.Context, config *Config) (*GmailClient, error) {
	credentials, err := os.ReadFile(config.Gmail.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	oauthConfig, err := google.ConfigFromJSON(credentials, gmail.GmailModifyScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	// Create a token file path
	tokenFile := "token.json"

	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok, err = getTokenFromWeb(oauthConfig)
		if err != nil {
			return nil, fmt.Errorf("unable to get token: %v", err)
		}

		err = saveToken(tokenFile, tok)
		if err != nil {
			return nil, fmt.Errorf("unable to save token: %v", err)
		}
	}

	client := oauthConfig.Client(ctx, tok)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %v", err)
	}

	gc := &GmailClient{
		service: NewGmailServiceWrapper(srv),
		config:  config,
	}

	// Create or get the forwarded label
	labelID, err := gc.ensureLabelExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to ensure label exists: %v", err)
	}

	gc.labelID = labelID
	gc.getNewMessages = gc.defaultGetNewMessages
	gc.markAsForwarded = gc.defaultMarkAsForwarded

	return gc, nil
}

func (c *GmailClient) ensureLabelExists(ctx context.Context) (string, error) {
	// List all labels
	labels, err := c.service.Users().Labels().List("me")
	if err != nil {
		return "", fmt.Errorf("failed to list labels: %v", err)
	}

	// Check if the label already exists
	for _, label := range labels {
		if label.Name == c.config.Gmail.ForwardedLabel {
			return label.Id, nil
		}
	}

	// Create the label if it doesn't exist
	newLabel := &gmail.Label{
		Name: c.config.Gmail.ForwardedLabel,
	}
	createdLabel, err := c.service.Users().Labels().Create("me", newLabel)
	if err != nil {
		return "", fmt.Errorf("failed to create label: %v", err)
	}

	return createdLabel.Id, nil
}

func (c *GmailClient) GetNewMessages(ctx context.Context) ([]Message, error) {
	// Get messages that don't have the forwarded label
	labelId, err := c.ensureLabelExists(ctx)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("-label:%s", c.config.Gmail.ForwardedLabel)
	messages, err := c.service.Users().Messages().List("me", query)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %v", err)
	}

	var result []Message
	for _, msg := range messages {
		// Get the full message details
		fullMsg, err := c.service.Users().Messages().Get("me", msg.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to get message %s: %v", msg.Id, err)
		}

		// Parse the message
		parsedMsg, err := c.parseMessage(fullMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse message %s: %v", msg.Id, err)
		}

		if !c.shouldProcessMessage(parsedMsg) {
			continue
		}

		result = append(result, parsedMsg)

		// Mark the message as processed by adding the label
		modReq := &gmail.ModifyMessageRequest{
			AddLabelIds: []string{labelId},
		}
		_, err = c.service.Users().Messages().Modify("me", msg.Id, modReq)
		if err != nil {
			return nil, fmt.Errorf("failed to modify message %s: %v", msg.Id, err)
		}
	}

	return result, nil
}

func (c *GmailClient) defaultGetNewMessages(ctx context.Context) ([]Message, error) {
	return c.GetNewMessages(ctx)
}

func (c *GmailClient) MarkAsForwarded(ctx context.Context, messageID string) error {
	return c.markAsForwarded(ctx, messageID)
}

func (c *GmailClient) defaultMarkAsForwarded(ctx context.Context, messageID string) error {
	modReq := &gmail.ModifyMessageRequest{
		AddLabelIds: []string{c.labelID},
	}
	_, err := c.service.Users().Messages().Modify("me", messageID, modReq)
	return err
}

func (c *GmailClient) parseMessage(msg *gmail.Message) (Message, error) {
	var result Message
	result.ID = msg.Id

	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "Subject":
			result.Subject = header.Value
		case "From":
			result.From = header.Value
		case "Date":
			result.Date = header.Value
		}
	}

	// Get message content
	content, err := c.getMessageContent(msg)
	if err != nil {
		return result, fmt.Errorf("failed to get message content: %v", err)
	}
	result.Content = content

	return result, nil
}

func (c *GmailClient) getMessageContent(msg *gmail.Message) (string, error) {
	if msg == nil || msg.Payload == nil {
		return "", fmt.Errorf("invalid message: payload is nil")
	}

	var content string

	if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		if err != nil {
			return "", err
		}
		content = string(data)
	} else if len(msg.Payload.Parts) > 0 {
		for _, part := range msg.Payload.Parts {
			if part != nil && part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err != nil {
					return "", err
				}
				content = string(data)
				break
			}
		}
	}

	return content, nil
}

func (c *GmailClient) shouldProcessMessage(msg Message) bool {
	// Check From filter
	if len(c.config.Gmail.Filter.From) > 0 {
		fromMatched := false
		for _, from := range c.config.Gmail.Filter.From {
			if strings.Contains(strings.ToLower(msg.From), strings.ToLower(from)) {
				fromMatched = true
				break
			}
		}
		if !fromMatched {
			return false
		}
	}

	// Check Subject keywords
	if len(c.config.Gmail.Filter.SubjectKeywords) > 0 {
		subjectMatched := false
		for _, keyword := range c.config.Gmail.Filter.SubjectKeywords {
			if strings.Contains(strings.ToLower(msg.Subject), strings.ToLower(keyword)) {
				subjectMatched = true
				break
			}
		}
		if !subjectMatched {
			return false
		}
	}

	// Check Content keywords
	if len(c.config.Gmail.Filter.ContentKeywords) > 0 {
		contentMatched := false
		for _, keyword := range c.config.Gmail.Filter.ContentKeywords {
			if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(keyword)) {
				contentMatched = true
				break
			}
		}
		if !contentMatched {
			return false
		}
	}

	return true
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)

	return tok, err
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	// Set up a local redirect URI
	config.RedirectURL = "http://localhost:8080/oauth2callback"

	// Create a channel to receive the authorization code
	codeChan := make(chan string)

	// Start a temporary server to handle the callback
	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth2callback" {
				http.Error(w, "Invalid path", http.StatusNotFound)

				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				http.Error(w, "Code not found", http.StatusBadRequest)

				return
			}

			// Send the code to our channel
			codeChan <- code

			// Respond to the user
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("Authorization successful! You can close this window.")); err != nil {
				log.Printf("Error writing response: %v", err)
			}
		}),
	}

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Generate the authorization URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n%v\n", authURL)

	// Wait for the authorization code
	code := <-codeChan

	// Exchange the code for a token
	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %v", err)
	}

	return tok, nil
}

func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}

	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

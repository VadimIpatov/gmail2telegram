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

type GmailClient struct {
	service         *gmail.Service
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
		service: srv,
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

func (c *GmailClient) ensureLabelExists(_ context.Context) (string, error) {
	// Try to find existing label
	labels, err := c.service.Users.Labels.List("me").Do()
	if err != nil {
		return "", fmt.Errorf("unable to list labels: %v", err)
	}

	for _, label := range labels.Labels {
		if label.Name == c.config.Gmail.ForwardedLabel {
			return label.Id, nil
		}
	}

	// Create new label if it doesn't exist
	label := &gmail.Label{
		Name: c.config.Gmail.ForwardedLabel,
	}

	created, err := c.service.Users.Labels.Create("me", label).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create label: %v", err)
	}

	return created.Id, nil
}

func (c *GmailClient) GetNewMessages(ctx context.Context) ([]Message, error) {
	return c.getNewMessages(ctx)
}

func (c *GmailClient) defaultGetNewMessages(_ context.Context) ([]Message, error) {
	query := fmt.Sprintf("is:unread -label:%s", c.config.Gmail.ForwardedLabel)

	// Build OR condition for from addresses
	if len(c.config.Gmail.Filter.From) > 0 {
		fromConditions := make([]string, len(c.config.Gmail.Filter.From))
		for i, from := range c.config.Gmail.Filter.From {
			fromConditions[i] = fmt.Sprintf("from:%s", from)
		}

		query += " (" + strings.Join(fromConditions, " OR ") + ")"
	}

	log.Printf("Using Gmail query: %s", query)

	messages, err := c.service.Users.Messages.List("me").Q(query).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve messages: %v", err)
	}

	log.Printf("Gmail API returned %d messages", len(messages.Messages))

	var result []Message

	for _, msg := range messages.Messages {
		message, err := c.service.Users.Messages.Get("me", msg.Id).Do()
		if err != nil {
			log.Printf("Error getting message details for ID %s: %v", msg.Id, err)

			continue
		}

		// Log message details for debugging
		var subject, from string

		for _, header := range message.Payload.Headers {
			switch header.Name {
			case "Subject":
				subject = header.Value
			case "From":
				from = header.Value
			}
		}

		log.Printf("Found message - ID: %s, Subject: %s, From: %s", msg.Id, subject, from)

		parsedMsg, err := c.parseMessage(message)
		if err != nil {
			log.Printf("Error parsing message %s: %v", msg.Id, err)

			continue
		}

		if c.shouldProcessMessage(parsedMsg) {
			log.Printf("Message %s matches processing criteria", msg.Id)

			result = append(result, parsedMsg)
		} else {
			log.Printf("Message %s does not match processing criteria", msg.Id)
		}
	}

	log.Printf("Returning %d messages that match all criteria", len(result))

	return result, nil
}

func (c *GmailClient) MarkAsForwarded(ctx context.Context, messageID string) error {
	return c.markAsForwarded(ctx, messageID)
}

func (c *GmailClient) defaultMarkAsForwarded(_ context.Context, messageID string) error {
	modify := &gmail.ModifyMessageRequest{
		AddLabelIds: []string{c.labelID},
	}

	_, err := c.service.Users.Messages.Modify("me", messageID, modify).Do()
	if err != nil {
		return fmt.Errorf("unable to mark message as forwarded: %v", err)
	}

	return nil
}

func (c *GmailClient) parseMessagePart(part *gmail.MessagePart) (string, error) {
	if part.Body.Data == "" {
		return "", nil
	}

	decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode message part: %w", err)
	}

	return string(decoded), nil
}

func (c *GmailClient) parseMessage(msg *gmail.Message) (Message, error) {
	var result Message
	result.ID = msg.Id

	// Parse headers
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

	// Parse content
	if msg.Payload.Parts == nil {
		if msg.Payload.Body.Data == "" {
			return Message{}, fmt.Errorf("no content found in message")
		}

		content, err := c.parseMessagePart(&gmail.MessagePart{Body: msg.Payload.Body})
		if err != nil {
			return Message{}, err
		}

		result.Content = content

		return result, nil
	}

	// Handle multipart message
	for _, part := range msg.Payload.Parts {
		if part.MimeType != "text/plain" && part.MimeType != "text/html" {
			continue
		}

		content, err := c.parseMessagePart(part)
		if err != nil {
			return Message{}, err
		}

		if content != "" {
			result.Content = content

			break
		}
	}

	if result.Content == "" {
		return Message{}, fmt.Errorf("no content found in message")
	}

	return result, nil
}

func (c *GmailClient) shouldProcessMessage(msg Message) bool {
	// If no keywords are specified, consider all messages as matches
	if len(c.config.Gmail.Filter.SubjectKeywords) == 0 && len(c.config.Gmail.Filter.ContentKeywords) == 0 {
		log.Printf("No keywords specified, considering message as match")

		return true
	}

	// Check subject keywords
	for _, keyword := range c.config.Gmail.Filter.SubjectKeywords {
		if strings.Contains(strings.ToLower(msg.Subject), strings.ToLower(keyword)) {
			log.Printf("Message matches subject keyword: %s", keyword)

			return true
		}
	}

	// Check content keywords
	for _, keyword := range c.config.Gmail.Filter.ContentKeywords {
		if strings.Contains(strings.ToLower(msg.Content), strings.ToLower(keyword)) {
			log.Printf("Message matches content keyword: %s", keyword)

			return true
		}
	}

	log.Printf("Message does not match any keywords")

	return false
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

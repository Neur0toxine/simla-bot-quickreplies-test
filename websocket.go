package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/gorilla/websocket"
	mgbot "github.com/retailcrm/bot-api-client-go"
	mgbotws "github.com/retailcrm/bot-api-client-go/ws"
)

type WebsocketListener struct {
	wsEndpoint  string
	token       string
	mg          mgbot.ClientInterface
	scope       string
	trigger     string
	suggestions []mgbot.Suggestion
}

func NewWebsocketListener(endpoint, token, trigger, scope string, textOptions []string) *WebsocketListener {
	suggestions := []mgbot.Suggestion{
		{
			Type:  mgbot.SuggestionTypePhone,
			Title: "Phone",
		},
		{
			Type:  mgbot.SuggestionTypeEmail,
			Title: "E-Mail",
		},
	}

	for _, s := range textOptions {
		suggestions = append(suggestions, mgbot.Suggestion{
			Type:  mgbot.SuggestionTypeText,
			Title: s,
		})
	}

	uri := endpoint + "/api/bot/v1/"
	mg, err := mgbot.NewClientWithResponses(uri, mgbot.WithBotToken(token))
	if err != nil {
		log.Fatal(err)
	}

	wsEndpoint := regexp.MustCompile(`^http(s)?\:`).ReplaceAllString(uri, "ws$1:") + "ws"

	return &WebsocketListener{
		wsEndpoint:  wsEndpoint,
		token:       token,
		mg:          mg,
		scope:       scope,
		trigger:     trigger,
		suggestions: suggestions,
	}
}

func (l *WebsocketListener) Listen(ctx context.Context) error {
	log.Println("Listening for the new messages:", l.wsEndpoint)

	conn, err := l.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}

		var event mgbotws.EventMessageFromEventsChannel
		if err := json.Unmarshal(message, &event.Payload); err != nil {
			log.Printf("error: cannot parse websocket event: %s\n", err)
			continue
		}

		if err := l.handleEvent(ctx, event); err != nil {
			log.Printf("error: cannot handle websocket event: %s\n", err)
		}
	}
}

func (l *WebsocketListener) connect(ctx context.Context) (*websocket.Conn, error) {
	wsURL, err := url.Parse(l.wsEndpoint)
	if err != nil {
		return nil, err
	}

	query := wsURL.Query()
	query.Set("events", string(mgbotws.EventTypeMessageNew))
	wsURL.RawQuery = query.Encode()

	headers := http.Header{
		"X-Bot-Token": []string{l.token},
	}

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, wsURL.String(), headers)
	if err == nil {
		return conn, nil
	}

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("%w: websocket handshake failed with HTTP %d; additionally failed to read response body: %v", err, resp.StatusCode, readErr)
	}

	return nil, fmt.Errorf("%w: websocket handshake failed with HTTP %d: %s", err, resp.StatusCode, string(body))
}

func (l *WebsocketListener) handleEvent(ctx context.Context, event mgbotws.EventMessageFromEventsChannel) error {
	wh, ok := event.Payload.Data.(mgbotws.MessageDataSchema)
	if !ok {
		return nil
	}

	if wh.Message.From != nil && wh.Message.From.Type != mgbotws.UserTypeCustomer {
		return nil
	}

	if l.trigger != "" && wh.Message.Content != nil && *wh.Message.Content != l.trigger {
		return nil
	}

	if wh.Message.From != nil {
		log.Printf("Received message from %s with id=%d\n", wh.Message.From.Name, wh.Message.Id)
	} else {
		log.Printf("Received message with id=%d\n", wh.Message.Id)
	}

	_, err := l.mg.SendMessage(ctx, mgbot.SendMessageJSONRequestBody{
		Type:           ptr(mgbot.MessageTypeText),
		Content:        ptr("The quick brown fox jumps over the lazy dog."),
		Scope:          mgbot.MessageScope(l.scope),
		ChatID:         wh.Message.ChatId,
		QuoteMessageID: wh.Message.Id,
		TransportAttachments: &mgbot.MessageTransportAttachments{
			Suggestions: l.suggestions,
		},
	})
	if err != nil {
		return fmt.Errorf("cannot respond to the message: %w", err)
	}

	return nil
}

func ptr[T any](v T) *T {
	return &v
}

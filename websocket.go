package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	v1 "github.com/retailcrm/mg-bot-api-client-go/v1"
)

type WebsocketListener struct {
	mg          *v1.MgClient
	ws          *websocket.Conn
	scope       string
	suggestions []v1.Suggestion
}

func NewWebsocketListener(endpoint, token, scope string, textOptions []string) *WebsocketListener {
	suggestions := []v1.Suggestion{
		{
			Type:  v1.SuggestionTypePhone,
			Title: "Phone",
		},
		{
			Type:  v1.SuggestionTypeEmail,
			Title: "E-Mail",
		},
	}

	for _, s := range textOptions {
		suggestions = append(suggestions, v1.Suggestion{
			Type:  v1.SuggestionTypeText,
			Title: s,
		})
	}

	return &WebsocketListener{
		mg:          v1.New(endpoint, token),
		scope:       scope,
		suggestions: suggestions,
	}
}

func (l *WebsocketListener) Listen() error {
	data, header, err := l.mg.WsMeta([]string{v1.WsEventMessageNew})
	if err != nil {
		return fmt.Errorf("cannot get meta for connection: %w", err)
	}

	ws, _, err := websocket.DefaultDialer.Dial(data, header)
	if err != nil {
		return fmt.Errorf("cannot estabilish WebSocket connection to %s: %w", data, err)
	}

	log.Println("Listening for the new messages...")

	for {
		var wsEvent v1.WsEvent
		if err := ws.ReadJSON(&wsEvent); err != nil {
			log.Fatal("unexpected websocket error:", err)
		}

		var event v1.WsEventMessageNewData
		err = json.Unmarshal(wsEvent.Data, &event)
		if err != nil {
			log.Printf("cannot unmarshal payload: %s\n", err)
			continue
		}

		if event.Message == nil {
			log.Print("invalid payload - nil message")
			continue
		}

		if event.Message.From != nil && event.Message.From.Type != "customer" {
			continue
		}

		log.Printf("Received message from %s with id=%d\n", event.Message.From.Name, event.Message.ID)

		_, _, err := l.mg.MessageSend(v1.MessageSendRequest{
			Type:    v1.MsgTypeText,
			Content: "The quick brown fox jumps over the lazy dog.",
			// Items:                nil,
			Scope:          l.scope,
			ChatID:         event.Message.ChatID,
			QuoteMessageId: event.Message.ID,
			TransportAttachments: &v1.TransportAttachments{
				Suggestions: l.suggestions,
			},
		})
		if err != nil {
			log.Printf("error: cannot respond to the message: %s\n", err)
		}
	}
}

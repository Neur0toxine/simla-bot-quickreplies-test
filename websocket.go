package main

import (
	"context"
	"log"
	"regexp"

	mgbot "github.com/retailcrm/bot-api-client-go"
	mgbotws "github.com/retailcrm/bot-api-client-go/ws"
)

type WebsocketListener struct {
	ctx         context.Context
	ws          *mgbotws.AppController
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

	ws, err := mgbotws.NewController(regexp.MustCompile(`^http(s)?\:`).ReplaceAllString(uri, "ws$1:")+"ws", token)
	if err != nil {
		log.Fatal(err)
	}

	return &WebsocketListener{
		ws:          ws,
		mg:          mg,
		scope:       scope,
		trigger:     trigger,
		suggestions: suggestions,
	}
}

func (l *WebsocketListener) Listen(ctx context.Context) error {
	log.Println("Listening for the new messages...")

	return l.ws.SubscribeToReceiveEventsOperation(ctx, mgbotws.EventsChannelParameters{
		Events: "message_new",
	}, func(ctx context.Context, event mgbotws.EventMessageFromEventsChannel) error {
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

		_, err := l.mg.SendMessage(context.Background(), mgbot.SendMessageJSONRequestBody{
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
			log.Printf("error: cannot respond to the message: %s\n", err)
		}

		return nil
	})
}

func ptr[T any](v T) *T {
	return &v
}

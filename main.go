package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	mgbot "github.com/retailcrm/bot-api-client-go"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	apiURL := os.Getenv("API_URL")
	apiKey := os.Getenv("API_KEY")

	if apiURL == "" {
		log.Fatal("API_URL environment variable must be set")
	}
	if apiKey == "" {
		log.Fatal("API_KEY environment variable must be set")
	}

	botCode := os.Getenv("BOT_CODE")
	botName := os.Getenv("BOT_NAME")
	trigger := os.Getenv("BOT_TRIGGER")
	msgScope := os.Getenv("MESSAGE_SCOPE")
	options := os.Getenv("TEXT_OPTIONS")

	if botCode == "" {
		botCode = "test-quick-reply-bot"
	}
	if botName == "" {
		botName = "Test Quick Reply Bot"
	}
	if msgScope == "" {
		msgScope = string(mgbot.MessageScopePrivate)
	}
	if options == "" {
		options = "Text reply"
	}

	endpoint, token, err := updateIntegrationModule(apiURL, apiKey, botCode, botName)
	if err != nil {
		log.Fatal("Error updating integration module: ", err)
	}

	log.Println("Integration module has been updated. Endpoint: ", endpoint, ", token: ", token)

	stopCh := make(chan struct{}, 1)
	stop := func() {
		stopCh <- struct{}{}
		close(stopCh)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer stop()
		if err := NewWebsocketListener(endpoint, token, trigger, msgScope, strings.Split(options, ",")).Listen(ctx); err != nil {
			log.Fatal("listen error: ", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c)
	for sig := range c {
		switch sig {
		case os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM:
			cancel()
			select {
			case <-stopCh:
				return
			case <-time.After(time.Second * 5):
				log.Fatal("did not stop gracefully after 5 seconds")
			}
		default:
		}
	}
}

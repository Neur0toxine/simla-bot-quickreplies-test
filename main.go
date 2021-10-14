package main

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	v1 "github.com/retailcrm/mg-bot-api-client-go/v1"
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
	msgScope := os.Getenv("MESSAGE_SCOPE")
	options := os.Getenv("TEXT_OPTIONS")

	if botCode == "" {
		botCode = "test-quick-reply-bot"
	}
	if botName == "" {
		botName = "Test Quick Reply Bot"
	}
	if msgScope == "" {
		msgScope = v1.MessageScopePrivate
	}
	if options == "" {
		options = "Text reply"
	}

	endpoint, token, err := updateIntegrationModule(apiURL, apiKey, botCode, botName)
	if err != nil {
		log.Fatal("Error updating integration module: ", err)
	}

	log.Println("Integration module has been updated. Endpoint: ", endpoint, ", token: ", token)

	if err := NewWebsocketListener(endpoint, token, msgScope, strings.Split(options, ",")).Listen(); err != nil {
		log.Fatal(err)
	}
}

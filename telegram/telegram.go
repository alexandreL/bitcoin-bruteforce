package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Bot struct {
	Token  string
	ChatID string
}

func NewBot(token, chatID string) *Bot {
	return &Bot{
		Token:  token,
		ChatID: chatID,
	}
}

func (b *Bot) SendMessage(message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.Token)

	payload := map[string]string{
		"chat_id": b.ChatID,
		"text":    message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling message payload: %v", err)
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to send message: %s", resp.Status)
		return fmt.Errorf("failed to send message: %s", resp.Status)
	}

	log.Printf("Message sent successfully: %s", message)
	return nil
}

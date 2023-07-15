package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type WebhookData struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func SendWebhook(url string, data WebhookData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send webhook, status code: %d", resp.StatusCode)
	}

	return nil
}

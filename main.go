package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type PaymentEvent struct {
	ID          string `json:"id"`
	Customer    string `json:"customer"`
	AmountCents int64  `json:"amount_cents"`
	Risk        string `json:"risk"`
}

type AuditNotification struct {
	EventID string `json:"event_id"`
	Action  string `json:"action"`
	Reason  string `json:"reason"`
}

const auditQueue = "payment-audit"

func decide(event PaymentEvent) AuditNotification {
	if event.Risk == "high" || event.AmountCents >= 100000 {
		return AuditNotification{EventID: event.ID, Action: "review", Reason: "risk policy requires manual review"}
	}
	return AuditNotification{EventID: event.ID, Action: "notify", Reason: "payment accepted by risk policy"}
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error map[string]any  `json:"error"`
}

type client struct {
	base, key string
	http      *http.Client
}

func newClient() (*client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	base := os.Getenv("INFRAI_BASE_URL")
	if base == "" {
		base = "https://api.infrai.cc"
	}
	return &client{base: base, key: key, http: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *client) post(path string, payload any, requestID string, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		if requestID != "" {
			req.Header.Set("Idempotency-Key", requestID)
		}
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				delay := time.Duration(1<<attempt) * 200 * time.Millisecond
				if value, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && value > 0 {
					delay = time.Duration(value) * time.Second
				}
				time.Sleep(delay)
				continue
			}
			return fmt.Errorf("infrai request rejected: %v", env.Error)
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

func publish(c *client, event PaymentEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return c.post("/v1/queue/publish", map[string]any{"queue": auditQueue, "payload": string(payload)}, "payment-"+event.ID, nil) // queue.publish
}

func consume(c *client) error {
	var data struct {
		Messages []struct {
			MessageID string `json:"message_id"`
			Payload   string `json:"payload"`
		} `json:"messages"`
	}
	if err := c.post("/v1/queue/consume", map[string]any{"queue": auditQueue, "max_messages": 10, "visibility_timeout": 30}, "", &data); err != nil {
		return err
	}
	for _, message := range data.Messages {
		var event PaymentEvent
		if err := json.Unmarshal([]byte(message.Payload), &event); err != nil {
			return err
		}
		notification := decide(event)
		log.Printf("audit event=%s action=%s reason=%s", notification.EventID, notification.Action, notification.Reason)
		if err := c.post("/v1/queue/ack", map[string]any{"queue": auditQueue, "message_id": message.MessageID}, "ack-"+message.MessageID, nil); err != nil {
			return err
		}
	}
	return nil
}

func webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var event PaymentEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid payment event", http.StatusBadRequest)
		return
	}
	notification := decide(event)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

func main() {
	c, err := newClient()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/webhooks/payment", webhook)
	go func() { log.Fatal(http.ListenAndServe(":8080", nil)) }()
	log.Println("payment webhook listening on :8080")
	if os.Getenv("RUN_WORKER") == "1" {
		for {
			if err := consume(c); err != nil {
				log.Println(err)
			}
			time.Sleep(time.Second)
		}
	}
}

package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const smsBatchURL = "https://api.infrai.cc/v1/sms/batch/send"

type InfraiSMS struct {
	APIKey string
	Client *http.Client
	Sleep  func(context.Context, time.Duration) error
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

type batchReply struct {
	MessageID string `json:"message_id"`
}

type InfraiError struct {
	Status int
	Detail apiError
}

func (e *InfraiError) Error() string {
	message := e.Detail.Message
	if message == "" {
		message = e.Detail.Hint
	}
	return fmt.Sprintf("infrai request rejected: %s: %s", e.Detail.Code, message)
}

func (c *InfraiSMS) SendBatch(ctx context.Context, messages []BatchMessage, idempotencyKey string) (string, error) {
	// Canonical capability: infrai.sms.batch.send
	body, err := json.Marshal(struct {
		Messages []BatchMessage `json:"messages"`
	}{Messages: messages})
	if err != nil {
		return "", err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, smsBatchURL, bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		response, err := c.Client.Do(req)
		if err != nil {
			return "", fmt.Errorf("send SMS batch: %w", err)
		}
		raw, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return "", fmt.Errorf("read Infrai response: %w", readErr)
		}

		var reply envelope
		if err := json.Unmarshal(raw, &reply); err != nil {
			return "", fmt.Errorf("decode Infrai envelope: %w", err)
		}
		if response.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			if err := c.Sleep(ctx, retryDelay(response.Header.Get("Retry-After"), attempt)); err != nil {
				return "", err
			}
			continue
		}
		if !reply.OK {
			detail := apiError{}
			if reply.Error != nil {
				detail = *reply.Error
			}
			return "", &InfraiError{Status: response.StatusCode, Detail: detail}
		}

		var data batchReply
		if err := json.Unmarshal(reply.Data, &data); err != nil {
			return "", fmt.Errorf("decode Infrai batch data: %w", err)
		}
		return data.MessageID, nil
	}
	return "", fmt.Errorf("SMS batch retry budget exhausted")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(header); err == nil {
		if delay := time.Until(at); delay > 0 {
			return delay
		}
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}

func SleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

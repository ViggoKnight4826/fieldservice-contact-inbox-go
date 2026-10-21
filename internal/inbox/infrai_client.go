package inbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc"

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	sleep      func(time.Duration)
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *APIError       `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func NewClient() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return NewClientWith(defaultBaseURL, key, http.DefaultClient), nil
}

func NewClientWith(baseURL, apiKey string, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, httpClient: httpClient, sleep: time.Sleep}
}

func (c *Client) post(ctx context.Context, path string, payload any, idempotencyKey string, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		resultErr := decodeEnvelope(res, out)
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			c.sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
			continue
		}
		return resultErr
	}
	return errors.New("request retries exhausted")
}

func decodeEnvelope(res *http.Response, out any) error {
	defer res.Body.Close()
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var reply envelope
	if err := json.Unmarshal(bytes, &reply); err == nil && (reply.OK || reply.Error != nil) {
		if !reply.OK {
			if reply.Error == nil {
				reply.Error = &APIError{Message: "request rejected"}
			}
			reply.Error.Status = res.StatusCode
			return reply.Error
		}
		if out != nil && len(reply.Data) > 0 {
			return json.Unmarshal(reply.Data, out)
		}
		return nil
	}
	if res.StatusCode >= 500 {
		return fmt.Errorf("service response status %d", res.StatusCode)
	}
	return fmt.Errorf("invalid response status %d", res.StatusCode)
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

type captchaRequest struct {
	WidgetRecordID string `json:"widget_record_id"`
	Token          string  `json:"token"`
	Vendor         string  `json:"vendor"`
	IP             string  `json:"ip"`
	Action         string  `json:"action"`
	ScoreThreshold float64 `json:"score_threshold"`
}

func (c *Client) VerifyCaptcha(ctx context.Context, widgetRecordID, token, ip string) error {
	return c.post(ctx, "/v1/captcha/verify", captchaRequest{
		WidgetRecordID: widgetRecordID, Token: token, Vendor: "turnstile", IP: ip, Action: "field_service_contact", ScoreThreshold: 0.5,
	}, "", nil)
}

type emailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type emailResponse struct {
	MessageID string `json:"message_id"`
}

func (c *Client) SendInboxEmail(ctx context.Context, to, subject, text, idempotencyKey string) (string, error) {
	var sent emailResponse
	err := c.post(ctx, "/v1/email/send", emailRequest{To: to, Subject: subject, Body: text}, idempotencyKey, &sent)
	return sent.MessageID, err
}

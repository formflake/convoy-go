package convoy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type WebhookInterface interface {
	GetEndpoint(projectID, endpointID string) (*Endpoint, error)
	CreateEndpoint(projectID string, params UpsertEndpointParams) (*CreateEndpointResponse, error)
	UpdateEndpoint(projectID, endpointID string, params UpsertEndpointParams) (*EndpointResponse, error)
	DeleteEndpoint(projectID, endpointID string) (*EndpointResponse, error)
	TogglePause(projectID, endpointID string) (string, error)
	CreateEvent(projectID string, webhookData *Webhook) error
}

type webhookService struct {
	url string
	key string
}

var _ WebhookInterface = &webhookService{}

func NewWebhook(url, key, defaultProject string) *webhookService {
	return &webhookService{
		url: url,
		key: key,
	}
}

type EndpointToggleStatus struct {
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
}

type EndpointResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type CreateEndpointResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Uid    string `json:"uid"`
		Status string `json:"status"`
	} `json:"data"`
}

type UpsertEndpointParams struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	AdvancedSignatures bool   `json:"advanced_signatures"`
	AppID              string `json:"appID"` // deprecated but required
	// Authentication
	Description       string `json:"description"`
	HttpTimeout       int64  `json:"http_timeout"`
	IsDisabled        bool   `json:"is_disabled"`
	OwnerID           string `json:"owner_id"`
	RateLimit         int64  `json:"rate_limit"`
	RateLimitDuration int64  `json:"rate_limit_duration"`
	Secret            string `json:"secret"`
	SlackWebhookURL   string `json:"slack_webhook_url"`
	SupportEmail      string `json:"support_email"`
}

type Endpoint struct {
	Message string       `json:"message"`
	Status  bool         `json:"status"`
	Data    EndpointData `json:"data"`
}

type EndpointData struct {
	// Authentication
	// Secrets
	SlackWebhookURL   string     `json:"slack_webhook_url"`
	Status            string     `json:"status"`
	SupportEmail      string     `json:"support_email"`
	UID               string     `json:"uid"`
	UpdatedAt         time.Time  `json:"updated_at"`
	URL               string     `json:"url"`
	CreatedAt         time.Time  `json:"created_at"`
	DeletedAt         *time.Time `json:"deleted_at"`
	Description       string     `json:"description"`
	Events            int64      `json:"events"`
	HttpTimeout       int64      `json:"http_timeout"`
	Name              string     `json:"name"`
	OwnerID           string     `json:"owner_id"`
	ProjectID         string     `json:"project_id"`
	RateLimit         int64      `json:"rate_limit"`
	RateLimitDuration int64      `json:"rate_limit_duration"`
}

type Webhook struct {
	Data    WebhookData
	Headers map[string][]string
}

type EventType string

type WebhookData struct {
	Data       interface{} `json:"data"`
	EventType  EventType   `json:"event_type"`
	EndpointID string      `json:"endpoint_id"`
}

func (we *webhookService) TogglePause(projectID, endpointID string) (string, error) {
	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprint(we.url, "/api/v1/projects/", projectID, "/endpoints/", endpointID, "/pause"),
		nil,
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprint("Bearer ", we.key))

	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("response code %d invalid", resp.StatusCode)
	}

	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("error closing response body", "err", err)
		}
	}(resp.Body)

	var endpoint EndpointToggleStatus
	if err := json.NewDecoder(resp.Body).Decode(&endpoint); err != nil {
		return "", err
	}

	return endpoint.Data.Status, nil
}

func (we *webhookService) CreateEndpoint(projectID string, params UpsertEndpointParams) (*CreateEndpointResponse, error) {
	buff := new(bytes.Buffer)
	err := json.NewEncoder(buff).Encode(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprint(we.url, "/api/v1/projects/", projectID, "/endpoints"),
		buff,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprint("Bearer ", we.key))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode > http.StatusBadRequest {
		return nil, fmt.Errorf("response code %d invalid", resp.StatusCode)
	}

	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("error closing response body", "err", err)
		}
	}(resp.Body)

	var response CreateEndpointResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (we *webhookService) UpdateEndpoint(projectID, endpointID string, params UpsertEndpointParams) (*EndpointResponse, error) {
	buff := new(bytes.Buffer)
	err := json.NewEncoder(buff).Encode(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprint(we.url, "/api/v1/projects/", projectID, "/endpoints/", endpointID),
		buff,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprint("Bearer ", we.key))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode > http.StatusBadRequest {
		return nil, fmt.Errorf("response code %d invalid", resp.StatusCode)
	}

	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("error closing response body", "err", err)
		}
	}(resp.Body)

	var response EndpointResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (we *webhookService) DeleteEndpoint(projectID, endpointID string) (*EndpointResponse, error) {
	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprint(we.url, "/api/v1/projects/", projectID, "/endpoints/", endpointID),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprint("Bearer ", we.key))

	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("response code %d invalid", resp.StatusCode)
	}

	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("error closing response body", "err", err)
		}
	}(resp.Body)

	var endpoint EndpointResponse
	if err := json.NewDecoder(resp.Body).Decode(&endpoint); err != nil {
		return nil, err
	}

	return &endpoint, nil
}

func (we *webhookService) GetEndpoint(projectID, endpointID string) (*Endpoint, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprint(we.url, "/api/v1/projects/", projectID, "/endpoints/", endpointID),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprint("Bearer ", we.key))

	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response code %d invalid", resp.StatusCode)
	}

	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("error closing response body", "err", err)
		}
	}(resp.Body)

	var endpoint Endpoint
	if err := json.NewDecoder(resp.Body).Decode(&endpoint); err != nil {
		return nil, err
	}

	return &endpoint, nil
}

func (we *webhookService) CreateEvent(projectID string, webhookData *Webhook) error {
	if webhookData == nil {
		return errors.New("webhook data undefined")
	}

	jsonBytes, err := json.Marshal(webhookData.Data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprint(we.url, "/api/v1/projects/", projectID, "/events"),
		bytes.NewBuffer(jsonBytes),
	)
	if err != nil {
		return err
	}
	if webhookData.Headers != nil {
		req.Header = map[string][]string(webhookData.Headers)
	}
	req.Header.Set("Authorization", fmt.Sprint("Bearer ", we.key))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("error closing response body", "err", err)
		}
	}(resp.Body)

	if body, err := io.ReadAll(resp.Body); err == nil {
		slog.Info(string(body)) // TODO
	}

	if resp.StatusCode >= 400 {
		return errors.New("error status code " + strconv.FormatInt(int64(resp.StatusCode), 10))
	}

	return nil
}

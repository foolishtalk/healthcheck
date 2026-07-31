package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultTimeoutSeconds = 10

type ServiceConfig struct {
	TimeoutSeconds  int       `json:"timeout_seconds"`
	WecomWebhookURL string    `json:"wecom_webhook_url,omitempty"`
	Services        []Service `json:"services"`
}

type Service struct {
	Name        string `json:"name"`
	Host        string `json:"host,omitempty"`
	DownloadURL string `json:"download_url"`
}

func loadConfig(path string) (ServiceConfig, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return ServiceConfig{}, err
	}

	var config ServiceConfig
	if err := json.Unmarshal(contents, &config); err != nil {
		return ServiceConfig{}, err
	}
	if len(config.Services) == 0 {
		return ServiceConfig{}, fmt.Errorf("services must contain at least one item")
	}
	if config.TimeoutSeconds < 0 {
		return ServiceConfig{}, fmt.Errorf("timeout_seconds cannot be negative")
	}

	for index := range config.Services {
		service := &config.Services[index]
		service.Name = strings.TrimSpace(service.Name)
		service.Host = strings.TrimSpace(service.Host)
		service.DownloadURL = strings.TrimSpace(service.DownloadURL)
		if service.Name == "" {
			return ServiceConfig{}, fmt.Errorf("services[%d].name is required", index)
		}
		if service.DownloadURL == "" {
			return ServiceConfig{}, fmt.Errorf("services[%d].download_url is required", index)
		}
		if _, _, err := service.tcpEndpoint(); err != nil {
			return ServiceConfig{}, fmt.Errorf("services[%d]: %w", index, err)
		}
	}

	return config, nil
}

func (config ServiceConfig) timeout() time.Duration {
	seconds := config.TimeoutSeconds
	if seconds == 0 {
		seconds = defaultTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (config ServiceConfig) webhookURL() string {
	if fromEnvironment := strings.TrimSpace(os.Getenv("WECOM_WEBHOOK_URL")); fromEnvironment != "" {
		return fromEnvironment
	}
	return strings.TrimSpace(config.WecomWebhookURL)
}

func (service Service) tcpEndpoint() (string, string, error) {
	parsed, err := url.ParseRequestURI(service.DownloadURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid download_url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf("download_url has unsupported scheme %q", parsed.Scheme)
	}

	host := service.Host
	if host == "" {
		host = parsed.Hostname()
	}
	if host == "" {
		return "", "", fmt.Errorf("download_url has no hostname")
	}

	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return host, port, nil
}

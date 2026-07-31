package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

type checkResult struct {
	Service string
	Check   string
	Err     error
}

func main() {
	configPath := flag.String("config", "services.json", "path to the service configuration")
	flag.Parse()

	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(2)
	}

	results := runChecks(config)
	failed := make([]checkResult, 0)
	for _, result := range results {
		if result.Err == nil {
			fmt.Printf("[OK]   %-20s %s\n", result.Service, result.Check)
			continue
		}

		fmt.Printf("[FAIL] %-20s %s: %v\n", result.Service, result.Check, result.Err)
		failed = append(failed, result)
	}

	if len(failed) == 0 {
		fmt.Printf("all %d services are healthy\n", len(config.Services))
		return
	}

	message := failureMessage(failed)
	if webhookURL := config.webhookURL(); webhookURL != "" {
		if err := wecomNotify(message, webhookURL, config.timeout()); err != nil {
			fmt.Fprintf(os.Stderr, "notification failed: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "%d health check(s) failed\n", len(failed))
	os.Exit(1)
}

func runChecks(config ServiceConfig) []checkResult {
	results := make([]checkResult, 0, len(config.Services)*2)
	for _, service := range config.Services {
		host, err := service.hostname()
		if err != nil {
			results = append(results,
				checkResult{Service: service.Name, Check: "ping", Err: err},
				checkResult{Service: service.Name, Check: "download headers", Err: err},
			)
			continue
		}

		results = append(results, checkResult{
			Service: service.Name,
			Check:   "ping " + host,
			Err:     pingHost(host, config.timeout()),
		})
		results = append(results, checkResult{
			Service: service.Name,
			Check:   "download headers " + service.DownloadURL,
			Err:     checkDownloadHeaders(service.DownloadURL, config.timeout()),
		})
	}
	return results
}

func pingHost(host string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// No shell is involved: host is passed as one argument. The context provides
	// a portable timeout while -c 1 keeps the check to a single ICMP packet.
	output, err := exec.CommandContext(ctx, "ping", "-c", "1", host).CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timed out after %s", timeout)
	}
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("unreachable: %s", detail)
	}
	return nil
}

func checkDownloadHeaders(rawURL string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	return checkDownloadHeadersWithClient(rawURL, timeout, client)
}

func checkDownloadHeadersWithClient(rawURL string, timeout time.Duration, client *http.Client) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "github-actions-healthcheck/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}
	return nil
}

func failureMessage(results []checkResult) string {
	var builder strings.Builder
	builder.WriteString("服务健康检查失败：")
	for _, result := range results {
		fmt.Fprintf(&builder, "\n- %s / %s：%v", result.Service, result.Check, result.Err)
	}
	return builder.String()
}

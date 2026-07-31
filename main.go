package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
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
	testNotification := flag.Bool("test-notification", false, "send a test WeCom notification without checking targets")
	flag.Parse()

	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(2)
	}
	if *testNotification {
		if err := sendTestNotification(config); err != nil {
			fmt.Fprintf(os.Stderr, "test notification failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("test notification sent successfully; no targets were checked")
		return
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

func sendTestNotification(config ServiceConfig) error {
	webhookURL := config.webhookURL()
	if webhookURL == "" {
		return fmt.Errorf("WECOM_WEBHOOK_URL is not configured")
	}

	repository := strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY"))
	if repository == "" {
		repository = "local run"
	}
	message := fmt.Sprintf(
		"✅ Healthcheck 测试通知\n企业微信 Webhook 配置正常。\n仓库：%s\n时间：%s UTC\n本次测试未检查任何监测目标。",
		repository,
		time.Now().UTC().Format(time.RFC3339),
	)
	return wecomNotify(message, webhookURL, config.timeout())
}

func runChecks(config ServiceConfig) []checkResult {
	results := make([]checkResult, 0, len(config.Services)*2)
	checkedEndpoints := make(map[string]struct{})
	for _, service := range config.Services {
		host, port, err := service.tcpEndpoint()
		if err != nil {
			results = append(results,
				checkResult{Service: service.Name, Check: "tcp connectivity", Err: err},
				checkResult{Service: service.Name, Check: "download headers", Err: err},
			)
			continue
		}

		endpoint := net.JoinHostPort(host, port)
		if _, alreadyChecked := checkedEndpoints[endpoint]; !alreadyChecked {
			checkedEndpoints[endpoint] = struct{}{}
			results = append(results, checkResult{
				Service: service.Name,
				Check:   "tcp " + endpoint,
				Err:     checkTCPConnectivity(host, port, config.timeout()),
			})
		}
		results = append(results, checkResult{
			Service: service.Name,
			Check:   "download headers " + service.DownloadURL,
			Err:     checkDownloadHeaders(service.DownloadURL, config.timeout()),
		})
	}
	return results
}

func checkTCPConnectivity(host, port string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dialer := &net.Dialer{Timeout: timeout}
	connection, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return err
	}
	_ = connection.Close()
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

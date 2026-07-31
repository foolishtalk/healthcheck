# Healthcheck

[![Hourly health check](https://github.com/foolishtalk/healthcheck/actions/workflows/healthcheck.yml/badge.svg)](https://github.com/foolishtalk/healthcheck/actions/workflows/healthcheck.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

English | [简体中文](README.zh-CN.md)

A lightweight domain and download URL monitor powered entirely by GitHub Actions. It requires no server and checks your configured services once per hour by default.

## What it checks

Each configured service receives two checks:

1. **Network reachability** — resolves the host and opens a TCP connection to the URL port (`80` for HTTP and `443` for HTTPS by default).
2. **Download URL availability** — sends an HTTP `HEAD` request and verifies that the final response has a `2xx` status.

The download check only requests response headers. It never sends a `GET` request, reads a response body, or saves a file. Large `.pkg`, `.dmg`, and `.zip` targets are therefore not downloaded, keeping traffic to a minimum.

If a server does not support `HEAD` requests—for example, if it returns `405 Method Not Allowed`—the check fails. The program deliberately does not fall back to `GET`.

## Quick start

### 1. Fork this repository

Click **Fork** in the upper-right corner of this repository to create a copy under your GitHub account.

### 2. Configure your targets

Edit [`services.json`](services.json) on the default branch of your fork:

```json
{
  "timeout_seconds": 10,
  "services": [
    {
      "name": "website",
      "download_url": "https://example.com/"
    },
    {
      "name": "macOS installer",
      "download_url": "https://downloads.example.com/app.dmg"
    },
    {
      "name": "separate TCP host",
      "host": "example.com",
      "download_url": "https://cdn.example.com/app.pkg"
    }
  ]
}
```

Configuration fields:

| Field | Required | Description |
| --- | --- | --- |
| `timeout_seconds` | No | Timeout for each check; defaults to 10 seconds |
| `name` | Yes | Service name used in logs and alert messages |
| `download_url` | Yes | Full URL checked with an HTTP `HEAD` request |
| `host` | No | Host used for the TCP check; derived automatically from `download_url` when omitted |

You do not need to specify `host` when the TCP and download checks use the same domain. A host and port shared by multiple URLs is checked only once per run.

### 3. Enable GitHub Actions

GitHub disables scheduled workflows in public forks by default. In your fork:

1. Open the **Actions** tab and enable workflows.
2. Select **Hourly health check** in the sidebar.
3. Click **Run workflow** to perform an initial manual check.
4. After the manual run succeeds, the workflow will continue automatically every hour.

The schedule is defined in [`.github/workflows/healthcheck.yml`](.github/workflows/healthcheck.yml):

```yaml
schedule:
  - cron: "0 * * * *"
```

GitHub Actions cron expressions use UTC. This schedule runs at the start of every hour, although GitHub may delay scheduled jobs by a few minutes during periods of high load. GitHub may also disable scheduled workflows in public repositories after 60 days without repository activity; you can re-enable the workflow from the Actions tab.

## WeCom alerts (optional)

A failed check is always visible as a failed workflow run. You can also receive alerts through a WeCom bot:

1. Open **Settings → Secrets and variables → Actions** in your repository.
2. Click **New repository secret**.
3. Enter `WECOM_WEBHOOK_URL` as the name.
4. Paste your WeCom bot webhook URL as the value.

The webhook URL is never stored in the configuration file and is not copied when the repository is forked. Health checks continue to work without this secret; only WeCom notifications are skipped.

### Test WeCom without checking production targets

You can verify the Secret and webhook without opening a TCP connection to a target or requesting any monitored URL:

1. Open **Actions → Hourly health check**.
2. Click **Run workflow**.
3. Enable **Send a test WeCom notification without checking targets**.
4. Click **Run workflow** again to start the run.

This mode sends one clearly labeled test message to WeCom and exits. It does not resolve or connect to any monitored host, send a `HEAD` request, or access a download URL. Scheduled runs always use the normal health-check mode.

## Reading the results

Each workflow run prints the result of every check:

```text
[OK]   website              tcp example.com:443
[OK]   website              download headers https://example.com/
all 1 services are healthy
```

If any check fails, the program exits with a non-zero status, marks the workflow run as failed, and sends an alert when a webhook is configured. Every configured service is checked before the program exits, so one result never prevents later targets from being tested.

## Notes

- TCP connectivity verifies the network path and service port without relying on ICMP, which is commonly blocked by servers and firewalls.
- A successful `HEAD` request confirms that the URL is reachable and returns a successful HTTP status; it does not verify the complete file contents.
- Requests originate from GitHub-hosted runners and do not represent connectivity from every region or network provider.
- Use a reasonable schedule to avoid placing unnecessary load on the monitored websites.

## Run locally

Go 1.22.5 or later is required:

```sh
go test ./...
go run . -config services.json
# Send a test notification without checking any target:
go run . -config services.json -test-notification
```

## License

This project is licensed under the [MIT License](LICENSE). You may use, copy, modify, publish, and distribute it as long as the original copyright and license notice are retained.

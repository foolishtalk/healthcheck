# Healthcheck

[![Hourly health check](https://github.com/foolishtalk/healthcheck/actions/workflows/healthcheck.yml/badge.svg)](https://github.com/foolishtalk/healthcheck/actions/workflows/healthcheck.yml)

一个轻量的域名与下载地址健康监测工具。项目完全运行在 GitHub Actions 上，无需购买服务器，默认每小时检查一次。

## 检查内容

每个服务会执行两项检查：

1. **网络连通性**：向目标域名发送 1 个 ICMP ping。
2. **下载地址可用性**：发送 HTTP `HEAD` 请求，检查最终响应是否为 `2xx`。

下载检查只获取响应头，不发送 `GET`，不读取响应正文，也不会保存文件。即使目标是体积很大的 `.pkg`、`.dmg` 或 `.zip` 文件，也不会真正下载，从而尽量减少网站流量。

如果服务器不支持 `HEAD` 请求（例如返回 `405 Method Not Allowed`），检查会直接失败。程序不会回退到 `GET`。

## 快速开始

### 1. Fork 仓库

点击仓库右上角的 **Fork**，将项目复制到自己的 GitHub 账号。

### 2. 配置监测目标

编辑默认分支中的 [`services.json`](services.json)：

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
      "name": "separate ping host",
      "host": "example.com",
      "download_url": "https://cdn.example.com/app.pkg"
    }
  ]
}
```

字段说明：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `timeout_seconds` | 否 | 单项检查超时时间，默认为 10 秒 |
| `name` | 是 | 服务名称，用于日志和告警消息 |
| `download_url` | 是 | 需要通过 `HEAD` 检查的完整地址 |
| `host` | 否 | 需要 ping 的域名；省略时从 `download_url` 自动提取 |

如果 ping 与下载使用同一个域名，不需要配置 `host`。

### 3. 启用 GitHub Actions

GitHub 默认会关闭公开 fork 仓库中的定时工作流。进入你自己的仓库：

1. 打开 **Actions** 页面并启用工作流。
2. 在左侧选择 **Hourly health check**。
3. 点击 **Run workflow** 手动运行一次。
4. 确认检查结果正常后，后续任务会自动每小时运行。

定时配置位于 [`.github/workflows/healthcheck.yml`](.github/workflows/healthcheck.yml)：

```yaml
schedule:
  - cron: "0 * * * *"
```

GitHub Actions 的 cron 使用 UTC。当前配置表示每个整点运行一次，平台繁忙时可能延迟几分钟。公开仓库连续 60 天没有活动时，GitHub 可能自动停用定时工作流，重新启用即可。

## 企业微信告警（可选）

当任一检查失败时，工作流会显示失败。你还可以配置企业微信机器人通知：

1. 打开仓库的 **Settings → Secrets and variables → Actions**。
2. 点击 **New repository secret**。
3. 名称填写 `WECOM_WEBHOOK_URL`。
4. 值填写企业微信机器人的 webhook 地址。

Webhook 地址不会写入配置文件，也不会随 fork 复制。未配置该 Secret 时，健康检查仍会正常运行，只是不发送企业微信通知。

## 如何判断结果

每次运行会在 GitHub Actions 日志中输出各项状态：

```text
[OK]   website              ping example.com
[OK]   website              download headers https://example.com/
all 1 services are healthy
```

只要有一项检查失败，程序就会返回非零状态，使该次 GitHub Actions 运行标红，并在已配置 webhook 时发送告警。程序会检查完所有服务，不会因为前一个服务成功或失败而跳过后续目标。

## 注意事项

- 部分正常网站会主动屏蔽 ICMP ping，此时下载地址可能正常，但 ping 检查仍会失败。
- `HEAD` 成功代表地址可以建立 HTTP 连接并返回成功状态，不代表完整文件内容一定正确。
- GitHub Actions 发出的请求来自 GitHub 托管运行器，并不代表所有地区或运营商的访问情况。
- 建议合理设置检查频率，避免给目标网站造成不必要的请求压力。

## 本地运行

需要 Go 1.22.5 或更高版本：

```sh
go test ./...
go run . -config services.json
```

## License

[MIT](LICENSE)

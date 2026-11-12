# go-shop｜用 Go 寫實戰購物網站

**English** · [繁體中文](#繁體中文)

Companion repo for a YouTube series: **building a production-style e-commerce backend in Go**, aimed at C#/Java backend engineers moving to Go.

- **Incident-driven.** Every episode starts from a production incident (overselling, double payment, lost emails, a Black-Friday-style rush), introduces one Go concept to fix it, and proves the fix with a test or a load test.
- **One tag per episode.** `git checkout ep01` … `ep12`, plus `ep13-pitfalls`. Tags never move.
- **Stack.** `net/http` + chi, PostgreSQL (pgx + sqlc + goose), Redis, outbox + NATS, OpenTelemetry, testcontainers-go, k6, Docker → Azure Container Apps.

```bash
go run ./cmd/api
curl http://localhost:8080/healthz
go test ./...
```

Code: MIT ([LICENSE](LICENSE)). Docs and scripts under `docs/`: CC BY-NC 4.0 ([docs/LICENSE.md](docs/LICENSE.md)).

---

## 繁體中文

繁中 YouTube 教學系列的配套 repo。每集對應一個 git tag（`ep01`…`ep12`、`ep13-pitfalls`），觀眾可以 checkout 到該集的程式碼。

## 定位

- **觀眾**：會 C#／Java、想轉 Go 的後端工程師。對照以 C# 為主、Java 為輔。
- **形式**：事故驅動。每集從一個線上事故開始，帶出一個 Go 概念，最後用測試或壓測證明修好了。
- **範圍**：後端 JSON API 為主。前端 Next.js 只當外殼，影片不細講。

## 技術棧

| 面向 | 選擇 |
|---|---|
| HTTP | `net/http` + chi router，JSON API |
| 專案結構 | `cmd/` + `internal/`，分層 handler → service → repository |
| API 契約 | OpenAPI + oapi-codegen |
| 資料庫 | PostgreSQL + pgx + sqlc，migration 用 goose |
| 快取／狀態 | Redis（session、購物車、限流、搶購預扣） |
| 非同步 | outbox + NATS（或 RabbitMQ）+ worker |
| 金流 | 綠界回呼，import [`github.com/newhand31/twpayhook`](https://github.com/newhand31/twpayhook)（本 repo 只 import 不改它） |
| 可觀測性 | slog + OpenTelemetry + Prometheus |
| 生命週期 | graceful shutdown |
| 測試 | table-driven + testcontainers-go，壓測用 k6 |
| 部署 | 多階段 Dockerfile → docker compose → Azure Container Apps |

細節見 [docs/architecture.md](docs/architecture.md)。

## 分集表

| 集 | 事故 | Go 概念 | C# 對照 |
|---|---|---|---|
| 01 | 老闆說明天開賣 | 專案結構、chi、slog | 沒有 DI 容器 |
| 02 | 商品要存起來 | pgx、sqlc、context、migration | 沒有 ORM |
| 03 | API 格式亂 | OpenAPI、錯誤處理 | 沒有 exception |
| 04 | 有人亂打 API | middleware、session、限流 | 沒有 filter attribute |
| 05 | 購物車不見 | Redis、隱式 interface | 不用宣告 implements |
| 06 | 超賣 | 交易、`FOR UPDATE`、k6 重現 | 沒有 `[Transactional]` |
| 07 | 付兩次錢 | 綠界回呼、冪等鍵 | — |
| 08 | 信沒寄出 | outbox、goroutine | 不是 async/await |
| 09 | 庫存被佔 | 排程、context 逾時取消 | 不是 CancellationToken 的翻版 |
| 10 | 雙 11 打爆 DB | Redis 預扣、channel 背壓 | 不是 Channel&lt;T&gt; 的翻版 |
| 11 | 半夜查不到原因 | OpenTelemetry、metrics | — |
| 12 | 部署斷線 | graceful shutdown、scratch image | — |
| 番外 | 踩坑合集 | nil interface、迴圈變數、slice 共用底層陣列 | — |

完整版（含每集細節）見 [docs/episodes.md](docs/episodes.md)。文案風格見 [docs/style-guide.md](docs/style-guide.md)。

## 執行

```bash
go run ./cmd/api
```

```bash
curl http://localhost:8080/healthz
```

測試：

```bash
go test ./...
```

# go-shop｜用 Go 寫實戰購物網站

**English** · [繁體中文](#繁體中文)

Companion repo for a YouTube series: **building a production-style e-commerce backend in Go**, aimed at C#/Java backend engineers moving to Go.

- **Incident-driven.** Every episode starts from a production incident (overselling, double payment, lost emails, a Black-Friday-style rush), introduces one Go concept to fix it, and proves the fix with a test or a load test.
- **One tag per episode.** `git checkout ep01` … `ep12`, plus `ep13-pitfalls`. Tags never move.
- **Stack.** `net/http` + chi, PostgreSQL (pgx + sqlc + goose), Redis, outbox + NATS, OpenTelemetry, testcontainers-go, k6, Docker → Azure Container Apps.

Run it locally (needs Go and Docker). Since episode 2 products live in PostgreSQL; the API applies migrations on startup.

```bash
docker compose up -d --wait
DATABASE_URL="postgres://goshop:goshop@localhost:5432/goshop?sslmode=disable" REDIS_URL="redis://localhost:6379/0" go run ./cmd/api
curl http://localhost:8080/products
```

PowerShell: `$env:DATABASE_URL = "postgres://goshop:goshop@localhost:5432/goshop?sslmode=disable"; $env:REDIS_URL = "redis://localhost:6379/0"; go run ./cmd/api`.
Since episode 5 the API also needs Redis (`REDIS_URL`; sessions and carts live there). Port 5432 or 6379 already taken? Start with `POSTGRES_PORT=55432` / `REDIS_PORT=56379` and use those ports in the URLs. The credentials are local throwaway defaults.

```bash
go test ./...                       # unit tests, no Docker
go test -tags=integration ./...     # integration tests, starts PostgreSQL via testcontainers (needs Docker)
```

After editing `db/queries/*.sql`, regenerate the repository code with `make sqlc` (runs sqlc in Docker).

The HTTP contract lives in `api/openapi.yaml`; after editing it, regenerate `internal/api/api.gen.go` with `make api` (oapi-codegen, pinned version). Every error the API returns has one shape: `{"code", "message", "field"?}` (a panic is turned into the same shape by the `Recover` middleware, episode 4).

```bash
curl http://localhost:8080/products/1      # one product
curl http://localhost:8080/products/999    # 404 {"code":"product_not_found",...}
curl http://localhost:8080/products/abc    # 400 {"code":"invalid_request","field":"id",...}
```

Since episode 4 every request goes through a middleware chain: request log → panic recovery → rate limit (per client IP, 10 requests/s, burst 20) → session cookie. Flood it and watch the `429`:

```bash
curl -i http://localhost:8080/products     # Set-Cookie: sid=...
for i in $(seq 1 200); do curl -s -o /dev/null -w "%{http_code} " http://localhost:8080/products; done   # 200 ... then 429
```

The proof is a test, not a promise: `go test -tags=integration ./internal/handler` sends 1000 requests from one IP to a real server.

Since episode 5 sessions and carts live in Redis, so any API instance can serve any visitor. The cart is tied to the session cookie:

```bash
curl -c jar -b jar -X PUT -d '{"quantity":2}' http://localhost:8080/cart/items/1   # add 2 of product 1
curl -c jar -b jar http://localhost:8080/cart                                      # the cart, with a total
```

The proof is `go test -tags=integration ./internal/handler`: two API instances behind one cookie jar. With in-process memory the cart vanishes when the "load balancer" switches instance; with Redis it survives.

Since episode 6 `POST /checkout` turns the cart into an order. Stock is checked and taken inside one database transaction with `SELECT ... FOR UPDATE`, so the last monitor can only be sold once (the seed data has exactly one):

```bash
curl -c jar -b jar -X PUT -d '{"quantity":1}' http://localhost:8080/cart/items/3
curl -c jar -b jar -X POST http://localhost:8080/checkout    # 201 the first time, 409 {"code":"out_of_stock"} for everyone after
```

The proof is `go test -tags=integration ./internal/repository`: 20 buyers, 1 monitor. Without the lock more than one buyer is told "yours"; with it exactly one. To watch it on the real server with [k6](https://k6.io), raise the rate limit for the run (a load test comes from one IP), reset the stock, and fire 50 simultaneous buyers:

```bash
RATE_LIMIT_RPS=100000 RATE_LIMIT_BURST=100000 DATABASE_URL=... REDIS_URL=... go run ./cmd/api
docker compose exec postgres psql -U goshop -c "UPDATE products SET stock = 1 WHERE id = 3"
k6 run scripts/k6/oversell.js     # exactly 1 order, 49 times 409
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
| 金流 | 綠界回呼，驗章與解析在本 repo 自己寫（`internal/payment/ecpay`） |
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

需要 Go 和 Docker。第 2 集起商品存在 PostgreSQL，API 啟動時會自動跑 migration。

```bash
docker compose up -d --wait
```

```bash
DATABASE_URL="postgres://goshop:goshop@localhost:5432/goshop?sslmode=disable" REDIS_URL="redis://localhost:6379/0" go run ./cmd/api
```

PowerShell：`$env:DATABASE_URL = "postgres://goshop:goshop@localhost:5432/goshop?sslmode=disable"; $env:REDIS_URL = "redis://localhost:6379/0"; go run ./cmd/api`。
第 5 集起 API 也需要 Redis（`REDIS_URL`；session 與購物車都放在那裡）。5432 或 6379 被佔用了？用 `POSTGRES_PORT=55432`／`REDIS_PORT=56379` 起 compose，兩個 URL 也改成同一個 port。帳密是本機用的預設值，不是密鑰。

```bash
curl http://localhost:8080/products
```

測試：

```bash
go test ./...
```

整合測試（用 testcontainers 起一個 PostgreSQL，需要 Docker）：

```bash
go test -tags=integration ./...
```

改了 `db/queries/*.sql` 之後，用 `make sqlc` 重新產生 repository 程式碼（sqlc 在 Docker 裡跑）。

HTTP 契約寫在 `api/openapi.yaml`；改完用 `make api` 重新產生 `internal/api/api.gen.go`（oapi-codegen，版本固定）。API 回傳的所有錯誤都是同一個形狀：`{"code", "message", "field"?}`（panic 由第 4 集的 `Recover` middleware 轉成同樣的形狀）。

```bash
curl http://localhost:8080/products/1      # 一件商品
curl http://localhost:8080/products/999    # 404 {"code":"product_not_found",...}
curl http://localhost:8080/products/abc    # 400 {"code":"invalid_request","field":"id",...}
```

第 4 集起每個請求都會經過 middleware 鏈：請求紀錄 → panic 復原 → 限流（每個來源 IP 每秒 10 次、突發 20）→ session cookie。狂打看看 `429`：

```bash
curl -i http://localhost:8080/products     # Set-Cookie: sid=...
for i in $(seq 1 200); do curl -s -o /dev/null -w "%{http_code} " http://localhost:8080/products; done   # 200 … 之後 429
```

證明是測試不是口號：`go test -tags=integration ./internal/handler` 從同一個 IP 對真的 server 打 1000 次。

第 5 集起 session 和購物車都放在 Redis，任何一台 API 都能服務任何訪客。購物車綁在 session cookie 上：

```bash
curl -c jar -b jar -X PUT -d '{"quantity":2}' http://localhost:8080/cart/items/1   # 商品 1 放 2 個
curl -c jar -b jar http://localhost:8080/cart                                      # 看購物車與總價
```

證明是 `go test -tags=integration ./internal/handler`：兩台 API 共用一個 cookie jar。狀態放在行程記憶體，「負載平衡器」一換台購物車就空了；放在 Redis 就不會。

第 6 集起 `POST /checkout` 把購物車變成訂單。檢查庫存與扣庫存發生在同一個資料庫交易裡，用 `SELECT ... FOR UPDATE` 鎖住商品那一列，所以最後一台螢幕只賣得出去一次（種子資料正好只有一台）：

```bash
curl -c jar -b jar -X PUT -d '{"quantity":1}' http://localhost:8080/cart/items/3
curl -c jar -b jar -X POST http://localhost:8080/checkout    # 第一次 201，之後每個人都是 409 {"code":"out_of_stock"}
```

證明是 `go test -tags=integration ./internal/repository`：20 個買家搶 1 台螢幕。沒有鎖，不只一個買家被告知「是你的」；有鎖，剛好一個。想用 [k6](https://k6.io) 在真的 server 上看：先把限流調高（壓測都從同一個 IP 來）、把庫存放回 1，再同時送出 50 個買家：

```bash
RATE_LIMIT_RPS=100000 RATE_LIMIT_BURST=100000 DATABASE_URL=... REDIS_URL=... go run ./cmd/api
docker compose exec postgres psql -U goshop -c "UPDATE products SET stock = 1 WHERE id = 3"
k6 run scripts/k6/oversell.js     # 剛好 1 張訂單，49 次 409
```

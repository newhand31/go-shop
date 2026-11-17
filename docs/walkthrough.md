# Walkthrough｜逐集操作與證明

Per-episode commands and the tests that prove each fix. Moved here from the README.

逐集的操作指令與證明測試，從 README 搬來。

## English

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

Since episode 7 the shop accepts ECPay's payment callback at `POST /payments/ecpay/callback`. The signature check (`CheckMacValue`, SHA-256) and the field parsing are written in this repo, in `internal/payment/ecpay`, using only the standard library. ECPay resends a callback until it gets `1|OK`, so the same payment can arrive many times; `payments.trade_no` is `UNIQUE`, and that constraint is the idempotency key. The API needs `ECPAY_MERCHANT_ID`, `ECPAY_HASH_KEY` and `ECPAY_HASH_IV` and refuses to start without them; for the demo set `ECPAY_SANDBOX=1` to use ECPay's published sandbox test credentials (never for real money).

The proof is `go test -tags=integration ./internal/repository`: the same signed callback delivered 10 times at once is recorded exactly once, and a forged or wrong-amount callback records nothing. The signature rules are unit-tested in `go test ./internal/payment/ecpay`.

Since episode 8 checkout also writes a row to the `outbox` table in the same transaction as the order ("send a confirmation email"), and a background goroutine (`service.OutboxDispatcher`, started and stopped in `cmd/api/main.go`) sends it later and retries on failure. The shop has no customer email addresses yet, so the mailer only logs what it would send.

The proof is `go test -tags=integration ./internal/repository -run Outbox`: a failed send is retried until it works, and an order that rolls back leaves no outbox row.

Since episode 9 a background janitor (`service.ExpiryJanitor`, a `time.Ticker` loop started and stopped in `cmd/api/main.go`) releases orders that stayed unpaid for 15 minutes: their stock goes back and the order is stamped `expired_at`. Every sweep runs under `context.WithTimeout`, so a stuck round is cancelled and the next one starts clean.

The proof is `go test ./internal/service -run Expiry` and `go test -tags=integration ./internal/repository -run Expiry`: an expired order is released, a paid one is left alone, and a round that runs out of time is cancelled.

Since episode 10 there is an optional flash-sale route, `POST /flash-sales/{productID}/orders`, switched on by `FLASH_SALE_PRODUCT_ID` and `FLASH_SALE_STOCK` (without them the route does not exist). A Redis Lua script pre-deducts one unit (`repository.Reservations`), a bounded channel plus `select … default` turns away buyers the shop has no room for with a 503 and `Retry-After` (`service.FlashSale`), and a few workers (`FLASH_SALE_WORKERS`, default 4) are the only ones allowed into a connection pool capped by `DB_MAX_CONNS` (default 10). The queue length is `FLASH_SALE_QUEUE` (default 64).

The proof is `go test ./internal/service -run Flash` and `go test -tags=integration ./internal/service -run Integration`: the pre-deduction never oversells, a full queue rejects without touching the database, and busy connections never exceed the worker count.

Since episode 11 every request carries a trace id. `otelhttp` opens the request span (`cmd/api/main.go`), `telemetry.NewLogHandler` copies `trace_id` and `span_id` into each `slog` record logged with a context, `service.OrderService.Checkout` adds one child span (`order.place`), and `GET /metrics` serves Prometheus metrics: `http_requests_total` and `http_request_duration_seconds`, labelled by route pattern, never by raw path. No collector is needed: `OTEL_TRACES_EXPORTER` is `none` (default, trace ids in logs only) or `stdout` (spans as JSON on stderr).

The proof is `go test ./internal/handler -run "TraceID|Metrics"` and `go test ./internal/platform/telemetry`: the request log line carries the same trace id the handler sees, and `/metrics` shows the new series.

Since episode 12 the API shuts down gracefully. `SIGINT` and `SIGTERM` (what `docker stop` sends) stop the listener, running requests get up to `SHUTDOWN_GRACE_SECONDS` (default 25) to finish (`lifecycle.Serve`), and only then the background goroutines are stopped and awaited (`lifecycle.Group`), so a buyer waiting in the flash-sale queue is still served. Redis, the connection pool and the last spans are closed after that. The `Dockerfile` is multi-stage and ends in `FROM scratch` (a static binary, `CGO_ENABLED=0`, no shell, no `HEALTHCHECK`); `docker compose --profile app up --build` runs the whole shop, and `docker stop` should log `stop signal received: requests drained, workers stopped`.

The proof is `go test ./internal/platform/lifecycle ./cmd/api`: a request still running when the stop signal arrives gets its answer before `Serve` returns, new connections are refused, a grace period that runs out cuts the rest, and a buyer queued behind a busy flash-sale worker is still served. One more test reads the `Dockerfile` as text (it does not build an image).

The bonus episode (`ep13-pitfalls`) adds no feature. `internal/pitfalls` holds three Go traps next to their fixes: a nil pointer inside an interface is not a nil interface, a closure over a loop variable sees one shared variable before Go 1.22 (one file is pinned to the old rules with `//go:build go1.21` so both behaviours sit side by side), and `append` on a sub-slice can overwrite the original (`cart[:n:n]` prevents it). Nothing in the shop imports it.

The proof is `go test ./internal/pitfalls -v`: each trap is made to fire once and its fix is checked.

## 繁體中文

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

第 7 集起 API 在 `POST /payments/ecpay/callback` 接綠界的付款回呼。驗章（`CheckMacValue`、SHA-256）與欄位解析都在本 repo 自己寫（`internal/payment/ecpay`），只用標準庫。綠界沒收到 `1|OK` 就會一直重送，所以同一筆付款可能來好幾次；`payments.trade_no` 設成 `UNIQUE`，這條約束就是冪等鍵。API 需要 `ECPAY_MERCHANT_ID`、`ECPAY_HASH_KEY`、`ECPAY_HASH_IV`，沒設就拒絕啟動；示範時設 `ECPAY_SANDBOX=1` 改用綠界公開的 sandbox 測試帳密（不能拿來收真的錢）。

證明是 `go test -tags=integration ./internal/repository`：同一筆簽好章的回呼同時送 10 次，只入帳一次；偽造或金額不符的回呼什麼都不會記。驗章規則的單元測試在 `go test ./internal/payment/ecpay`。

第 8 集起 `POST /checkout` 會在訂單的同一個交易裡多寫一列 `outbox`（「要寄確認信」），再由背景 goroutine（`service.OutboxDispatcher`，在 `cmd/api/main.go` 啟動與停止）稍後寄出，失敗就重試。商店目前沒有客人的 email，所以寄信器只會印出「我會寄什麼」。

證明是 `go test -tags=integration ./internal/repository -run Outbox`：寄信失敗會一直重試到成功；下單回滾的話，不會留下 outbox 那一列。

第 9 集起背景有一個清潔工（`service.ExpiryJanitor`，`time.Ticker` 迴圈，在 `cmd/api/main.go` 啟動與停止），把 15 分鐘沒付款的訂單釋放：庫存還回去，訂單蓋上 `expired_at`。每一輪掃描都包在 `context.WithTimeout` 裡，卡住的那一輪會被取消，下一輪重新開始。

證明是 `go test ./internal/service -run Expiry` 與 `go test -tags=integration ./internal/repository -run Expiry`：逾時的訂單會被釋放、已付款的不動、單輪逾時會被取消。

第 10 集起有一條可選的搶購路由 `POST /flash-sales/{productID}/orders`，用 `FLASH_SALE_PRODUCT_ID` 與 `FLASH_SALE_STOCK` 開啟（沒設就沒有這條路由）。Redis Lua 腳本預扣一件庫存（`repository.Reservations`）；有容量上限的 channel 加 `select … default`，排不下的人當場回 503 與 `Retry-After`（`service.FlashSale`）；只有少數幾個 worker（`FLASH_SALE_WORKERS`，預設 4）能進到上限由 `DB_MAX_CONNS`（預設 10）控制的連線池。排隊長度是 `FLASH_SALE_QUEUE`（預設 64）。

證明是 `go test ./internal/service -run Flash` 與 `go test -tags=integration ./internal/service -run Integration`：預扣不超賣、佇列滿了拒絕且不碰資料庫、忙碌的連線數不超過 worker 數。

第 11 集起每個請求都帶 trace id。`otelhttp` 開出請求的 span（`cmd/api/main.go`），`telemetry.NewLogHandler` 把 `trace_id`、`span_id` 補進每一筆帶 context 的 `slog` 紀錄，`service.OrderService.Checkout` 多開一個子 span（`order.place`），`GET /metrics` 提供 Prometheus 指標：`http_requests_total` 與 `http_request_duration_seconds`，標籤是路由樣式，不是原始路徑。不需要外部 collector：`OTEL_TRACES_EXPORTER` 預設 `none`（只有 log 裡的 trace id），設 `stdout` 則把 span 以 JSON 印到 stderr。

證明是 `go test ./internal/handler -run "TraceID|Metrics"` 與 `go test ./internal/platform/telemetry`：請求紀錄裡的 trace id 和 handler 看到的相同，`/metrics` 看得到新指標。

第 12 集起 API 會優雅關機。`SIGINT` 與 `SIGTERM`（`docker stop` 送的就是它）先關掉監聽，進行中的請求最多有 `SHUTDOWN_GRACE_SECONDS`（預設 25）秒做完（`lifecycle.Serve`），之後才停止背景 goroutine 並等它們結束（`lifecycle.Group`），所以排在搶購佇列裡的客人仍會被服務；然後才關 Redis、連線池並沖出最後的 span。`Dockerfile` 是多階段、最後一階 `FROM scratch`（靜態二進位、`CGO_ENABLED=0`、沒有 shell、沒有 `HEALTHCHECK`）；`docker compose --profile app up --build` 會跑整個商店，`docker stop` 時日誌應出現 `stop signal received: requests drained, workers stopped`。

證明是 `go test ./internal/platform/lifecycle ./cmd/api`：停止訊號到達時還在跑的請求會先收到回應，`Serve` 才回來；新連線被拒絕；寬限時間用完就砍掉剩下的；排在忙碌搶購 worker 後面的買家仍會被服務。另有一個測試把 `Dockerfile` 當文字讀（不會建置 image）。

番外（`ep13-pitfalls`）沒有新功能。`internal/pitfalls` 把三個 Go 的坑和修法放在一起：nil 指標放進 interface 後不等於 nil interface；Go 1.22 以前 closure 抓迴圈變數看到的是同一個變數（其中一個檔案用 `//go:build go1.21` 釘回舊規則，兩種行為並排）；對子 slice 做 `append` 可能蓋掉原本的資料（用 `cart[:n:n]` 避免）。商店本身沒有引用它。

證明是 `go test ./internal/pitfalls -v`：每個坑都會讓它發生一次，並確認修法有效。

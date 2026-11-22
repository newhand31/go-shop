# 有人亂打 API（short04）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep04`
- 主張清單：`docs/claims/ch04.json`（執行 `make claims` 逐條檢查）
- 片長：00:37
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 有人亂打 API。
- `00:02` 事故情境是這樣。同一個 IP，一秒打了 1000 次 products。
- `00:08` 原因是 tag ep03 沒有限流。同一個 IP 送多少請求，API 就處理多少。〔C4-2〕
- `00:17` 修法是加一個 middleware 限流。桶子空了，就回 429，並帶 Retry-After。〔C4-4、C4-6〕
- `00:25` middleware 只是收一個 handler、回傳一個 handler 的函式。〔C4-3〕
- `00:30` 測試證明通過的請求沒有超過上限。完整說明在長片第 4 章。〔C4-1〕

## 這份逐字稿引用的主張

- **C4-1**　同一個 IP 對 `/products` 連續送 1000 個請求：限流器的桶子容量是 10、每秒補 5 個。通過的請求數不超過「容量加這段時間補的數量」，至少有容量那麼多，其餘回 429，而且每個 429 都帶 `Retry-After`。測試 `TestFloodFromOneIPIsMostlyRejected` 起真的 HTTP server，走真的 TCP 驗證。
- **C4-2**　tag `ep03` 的程式碼沒有限流，也沒有 session：`internal/handler/middleware.go`、`internal/platform/ratelimit/ratelimit.go`、`internal/platform/session/memory.go` 都不存在。這三個檔案是這一章新增的。
- **C4-3**　`Recover`、`RateLimit`、`Session` 三個 middleware 的型別都是 `func(http.Handler) http.Handler`：收一個 handler，回傳一個 handler。`handler.New` 用 chi 的 `r.Use` 串起來，順序是 `RequestLog`、`Recover`，然後是呼叫端傳入的 middleware。測試 `TestChainOrder` 證明請求依序進去、回應依相反順序出來：`first in, second in, second out, first out`。
- **C4-4**　`RateLimit` 在限流器回報用完額度時，回 429 與 `api.Error`（code 是 `rate_limited`），加上 `Retry-After` 標頭，然後 `return`，不呼叫 `next`。`Retry-After` 的秒數無條件進位，最少是 1 秒。測試 `TestRateLimit` 的案例 `blocked`（0.3 秒進位成 1）與 `retry-after rounds up`（2.1 秒進位成 3）都通過。
- **C4-6**　限流器是 token bucket：每個 key 一個桶子，桶子有容量上限；每次經過的時間乘以補充速率，補進令牌，但不超過容量；每次請求拿走 1 個令牌。`cmd/api/main.go` 設定每個 IP 每秒補 10 個、容量 20。測試 `TestAllow` 的 4 個案例都通過。

# 半夜查不到原因（short11）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep11`
- 主張清單：`docs/claims/ch11.json`（執行 `make claims` 逐條檢查）
- 片長：00:32
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 半夜查不到原因。
- `00:02` 凌晨兩點，客服說結帳有時候很慢。我們打開 log 查原因。
- `00:09` 原因是 log 沒有請求編號，看不出哪幾行屬於同一個請求。〔C11-1〕
- `00:15` 修法是每個請求開一個 span。logHandler 從 ctx 讀 span，把 trace id 加進每一行 log。〔C11-3、C11-5〕
- `00:23` 指標的 route 標籤用路由樣式，不用原始路徑。〔C11-8〕
- `00:28` 完整說明在長片第 11 章。

## 這份逐字稿引用的主張

- **C11-1**　tag `ep10` 的程式碼還沒有 `telemetry` 套件，也沒有 `observe.go`；整個 `*.go` 裡沒有 `trace_id` 這個字，也沒有任何 `go.opentelemetry.io` 的 import。`ep10` 的 log 用沒有 ctx 的 `logger.Info("request", ...)` 寫出請求紀錄，只有 `method`、`path`、`status`、`duration_ms` 四個欄位，logger 是純 JSON handler。這些都是這一章新增的。
- **C11-3**　`cmd/api/main.go` 用 `otelhttp.NewHandler` 包住整個商店，名稱是 `shop`，所以每個請求在其他中介層之前先開一個 span。`GET /metrics` 掛在同一個 `http.ServeMux` 上、商店旁邊，不經過限流器和 session：限流器與 session 只接在商店的 handler 上，`mux` 建立之後的程式碼不再出現它們。
- **C11-5**　`telemetry.NewLogHandler` 回傳的 `slog.Handler` 包住內層 handler。ctx 裡有有效的 span 時，它在每筆紀錄加上 `trace_id` 與 `span_id` 兩個欄位。`WithAttrs` 與 `WithGroup` 也回傳包過的 `logHandler`，所以 `logger.With` 與 `WithGroup` 之後 trace id 還在。把 `WithAttrs` 改成直接回傳內層 handler，測試 `TestLogHandler/survives_logger.With` 就失敗。
- **C11-8**　`Instrument` 用 chi 比對到的路由樣式（例如 `/products/{id}`）當 `route` 標籤，不用原始路徑，而且在 `next.ServeHTTP` 跑完之後才讀。沒有路由認領的請求，以及 `Mount` 產生的 `/*`，標籤都是 `unmatched`。測試 `TestMetricsExposeTheRoutePatternNotThePath` 打 `/products/1`、`/products/2` 各一次和一次不存在的路徑：`/products/{id}` 的次數是 2，`unmatched` 的 404 次數是 1，`/metrics` 輸出裡沒有原始路徑。

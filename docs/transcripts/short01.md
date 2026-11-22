# 老闆說明天開賣（short01）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep01`
- 主張清單：`docs/claims/ch01.json`（執行 `make claims` 逐條檢查）
- 片長：00:35
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 老闆說明天開賣。
- `00:02` 事故情境是這樣。開賣前一天晚上十一點四十七分，老闆問 API 好了沒。
- `00:10` 原因是起點的程式只有 healthz 端點。它沒有 products 端點。〔C1-1〕
- `00:16` 修法是分成三層，再由 main 手動接線。〔C1-3、C1-4〕
- `00:21` 接線只有三行。少傳一個參數，編譯就失敗。〔C1-4、C1-5〕
- `00:27` 測試證明：healthz 回 200。〔C1-10、C1-11〕
- `00:31` 完整說明在長片第 1 章。

## 這份逐字稿引用的主張

- **C1-1**　起點的 commit `40ed180` 只有一個 `/healthz` 端點：`cmd/api/main.go` 有 `/healthz`，沒有 `/products`；沒有 `internal/service/product.go`、`internal/repository/product.go`，也沒有 `RequestLog`。
- **C1-3**　三層的分工寫在各套件的說明註解：`handler` 是 HTTP 層，`service` 放業務邏輯，`repository` 是資料存取層。依賴方向是 `handler` 到 `service` 到 `repository`：`handler` 只 import `service`；`service` 只 import `repository`；`repository` 不 import `service` 或 `handler`；`service` 不 import `handler`；`handler` 不 import `repository`。
- **C1-4**　`cmd/api/main.go` 的 `main` 用三行手寫的呼叫接線：`repository.NewMemoryProducts()`、`service.NewProductService(repo)`、`handler.New(svc, logger)`。埠號讀環境變數 `PORT`，沒設就用 `8080`。`go.mod` 只有 `chi` 一個依賴，沒有 DI 容器套件，也沒有資料庫或 Redis 的套件。
- **C1-5**　把 `cmd/api/main.go` 裡 `service.NewProductService(repo)` 的參數拿掉，`go build ./cmd/api` 失敗，錯誤訊息含 `not enough arguments in call to service.NewProductService`。
- **C1-10**　`TestRoutes` 是 table-driven 測試，表裡有 6 個情境：`/healthz` 回 200、`/products` 有資料、`/products` 空清單回 `[]`、service 出錯回 500、不存在的路徑回 404、`POST /products` 回 405。測試用 `httptest` 直接呼叫 router，沒有啟動伺服器。`go test ./...` 全部通過。
- **C1-11**　`repository.NewMemoryProducts` 內建 3 件商品：機械式鍵盤 2490 元、人體工學滑鼠 1290 元、27 吋螢幕 5990 元。`GET /products` 回傳的 JSON 每件商品有 `id`、`name`、`price_twd` 三個欄位。

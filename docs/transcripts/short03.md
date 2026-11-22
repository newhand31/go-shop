# API 的錯誤格式不一致（short03）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep03`
- 主張清單：`docs/claims/ch03.json`（執行 `make claims` 逐條檢查）
- 片長：00:54
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` API 的錯誤格式不一致。
- `00:03` 事故情境是這樣。前端說，API 出錯的時候，有的回一行文字，有的什麼都沒回。
- `00:12` 原因是出口不只一個。handler 呼叫 http.Error，路由器也有預設回應。〔C3-1、C3-2〕
- `00:19` 修法是 handler 的錯誤都走 writeError。路由器的 404 與 405 也換成 JSON 回應。errors.Is 檢查哨兵錯誤，找不到就回 404。〔C3-7、C3-9〕
- `00:32` 一個測試檢查八種出錯情況，形狀都相同。〔C3-10〕
- `00:46` 完整說明在長片第 3 章。

## 這份逐字稿引用的主張

- **C3-1**　tag `ep02` 的程式碼只有 `/healthz` 與 `/products` 兩條路由，沒有 API 契約檔 `api/openapi.yaml`。`internal/handler/handler.go` 出錯時呼叫 `http.Error` 回純文字，也沒有替換路由器預設的 404 與 405 回應。
- **C3-2**　路由器與生成程式碼的預設回應不是 JSON。在 tag `ep03` 把替換用的函式拿掉，`TestErrorResponsesShareOneShape` 就失敗：沒有自訂的 404 時，`Content-Type` 是 `text/plain; charset=utf-8`；沒有自訂的 405 時，`Content-Type` 是空的；沒有 `ErrorHandlerFunc` 時，路徑參數格式錯誤的回應也是 `text/plain; charset=utf-8`。
- **C3-7**　`internal/handler/errors.go` 的 `writeError` 是 handler 錯誤與生成程式碼參數錯誤的出口；路由器的 404 與 405 不走這裡，由 `notFound` 與 `methodNotAllowed` 直接呼叫 `writeJSON`。它用 `errors.Is` 檢查 `ErrProductNotFound`，回 404 與 `product_not_found`；用 `errors.As` 取出 `*service.ValidationError` 與生成程式碼的三種參數錯誤，回 400 與 `invalid_request`，並把欄位名稱放進 `field`。測試 `TestErrorResponsesShareOneShape` 的 `id` 不是數字與 `id` 不是正數兩列，檢查狀態碼 400、`code` 是 `invalid_request`、`field` 是 `id`。把 `errors.Is` 換成 `==` 比較後，`TestErrorResponsesShareOneShape` 失敗，訊息是 `status = 500, want 404`。
- **C3-9**　tag `ep03` 的 `internal/handler` 套件沒有任何 `http.Error` 呼叫。`ListProducts` 與 `GetProduct` 出錯時都呼叫 `writeError`。路由器的 404 與 405 換成 `notFound` 與 `methodNotAllowed`，兩者回傳 `http.HandlerFunc`，寫出的 body 是 `api.Error` 的 JSON，並直接呼叫 `writeJSON`。生成程式碼的 `ErrorHandlerFunc` 也呼叫 `writeError`。
- **C3-10**　測試 `TestErrorResponsesShareOneShape` 有 8 列，每列是一種出錯情況：商品不存在、包了兩層的商品不存在、id 不是數字、id 不是正數、列表出錯、單筆出錯、路由不存在、方法不對。每一列都檢查 `Content-Type` 是 JSON、body 能解成沒有多餘欄位的 `api.Error`、`code` 符合預期、`field` 符合預期，`message` 不是空的也不含 `postgres`；方法不對那一列另外檢查 `Allow` 標頭是 `GET`。這個測試通過。

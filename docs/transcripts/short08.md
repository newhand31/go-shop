# 信沒寄出（short08）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep08`
- 主張清單：`docs/claims/ch08.json`（執行 `make claims` 逐條檢查）
- 片長：00:37
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 信沒寄出。
- `00:02` 事故情境是這樣。客人下單成功，卻沒收到確認信。
- `00:08` 原因是下單和寄信是兩次寫入。兩次之間，沒有東西保證都成功。
- `00:16` 修法是把要寄信的事，在同一個交易裡寫成 outbox 的一列。訂單和 outbox 一起提交，或一起回滾。〔C8-2、C8-4〕
- `00:26` 背景 worker 寄信失敗會重試，而且有次數上限。整合測試證明這件事。〔C8-11、C8-12〕
- `00:33` 完整說明在長片第 8 章。

## 這份逐字稿引用的主張

- **C8-2**　`Orders.Place` 在同一個交易裡，用同一個 `q`，先寫訂單與訂單明細，再呼叫 `q.InsertOutbox` 寫一列 outbox，最後才 `tx.Commit`。`orders.go` 沒有寄信的呼叫：裡面沒有 `smtp`、`Mailer`、`SendOrderConfirmation` 或 `.Send...(` 這類字，也沒有 import 任何 `net/` 套件或 `service` 套件。測試 `TestOutbox_PlaceWritesOneRowWithTheOrder` 證明下一張單，outbox 剛好多一列，主題是 `order_confirmation`。
- **C8-4**　交易的 `Commit` 一定失敗時，下一張單之後，`outbox` 與 `orders` 都是 0 列，商品庫存回到原本的 10。測試 `TestOutbox_RolledBackOrderLeavesNoOutboxRow` 證明。
- **C8-11**　寄信器前兩次失敗、第三次成功時，worker 會重試：寄信器被呼叫 3 次，`attempts` 是 3，`last_error` 被清掉，`sent_at` 有值。測試結束前先 `cancel()` 並等 `done`，確認 worker 真的停了才讀最後狀態。再跑一輪，寄信器沒有被呼叫第 4 次。測試 `TestOutbox_FailedSendIsRetriedUntilItWorks` 證明。
- **C8-12**　預設最多嘗試 8 次，每次掃描最多 20 列，每 2 秒掃一次。寄信器永遠失敗、`MaxAttempts` 設成 3 時，跑 6 輪，寄信器只被呼叫 3 次，那一列沒有寄出，`last_error` 是 `smtp down`。測試 `TestOutbox_GivesUpAfterMaxAttemptsAndKeepsTheError` 證明。

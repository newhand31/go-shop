# 庫存被佔（short09）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep09`
- 主張清單：`docs/claims/ch09.json`（執行 `make claims` 逐條檢查）
- 片長：00:44
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 庫存被佔。
- `00:02` 事故情境是這樣。老闆傳來訊息：27 吋螢幕整個週末都買不到。
- `00:09` 原因是 tag ep08 沒有加回庫存的查詢。沒付款的訂單，一直佔著最後 1 件螢幕。〔C9-1、C9-2〕
- `00:18` 修法是定時掃描。逾期沒付款的訂單，庫存在同一個交易裡還回去。〔C9-5、C9-7〕
- `00:25` 每一輪用 context.WithTimeout 設期限。預設 30 秒。卡住的一輪，期限到了就被取消。〔C9-8、C9-9〕
- `00:35` 測試證明：逾期的釋放，已付款的不動。〔C9-10、C9-11〕
- `00:40` 完整說明在長片第 9 章。

## 這份逐字稿引用的主張

- **C9-1**　tag `ep08` 的程式碼沒有把庫存加回去的寫法：沒有 `internal/service/expiry.go`，沒有 `internal/repository/expiry.go`；在 `db`、`internal`、`cmd` 的非測試檔裡，找不到 `stock +`、`IncreaseStock`、`restock` 這類字，也就是沒有加回庫存的 SQL 或函式。`expiry.go` 兩個檔與 `IncreaseStock` 是 tag `ep09` 新增的。
- **C9-2**　種子資料裡「27 吋螢幕」是商品編號 3，庫存 1 件。一張下單後沒有付款的訂單，已經從庫存扣掉那 1 件，螢幕庫存變 0；訂單的 `paid_at` 欄位只有付款時才會被寫入：非測試的 SQL 與 Go 程式裡，只有 `payments.sql` 的 `UPDATE` 寫 `paid_at`（`dbgen` 裡有它產生的同一句），下單用的 `InsertOrder` 只寫 `total_twd`。測試 `TestExpiry_UnpaidOrderIsReleasedAndStockComesBack` 的開頭先確認這個狀態。
- **C9-5**　`Expiry.ReleaseOne` 在同一個交易裡依序做四件事：取出一張逾期訂單、讀它的品項、把每個品項的數量用 `IncreaseStock` 加回庫存、用 `MarkOrderExpired` 把 `expired_at` 設成現在，最後才 `Commit`。每次呼叫只處理一張訂單。
- **C9-7**　`ExpiryJanitor.Run` 用 `time.NewTicker` 建立計時器，並 `defer ticker.Stop()`；每一圈先呼叫 `Sweep`，再用 `select` 等 `ctx.Done()` 或 `ticker.C`，`ctx` 被取消就 `return`。測試 `TestExpiryJanitor_RunStopsWhenContextIsCancelled` 證明 `cancel` 之後 `Run` 會回來。
- **C9-8**　`ExpiryConfig` 有四個欄位：`Interval`、`OrderTTL`、`RoundTimeout`、`Now`。預設值是每 1 分鐘掃一輪、訂單 15 分鐘沒付款算逾期、單輪期限 30 秒、時鐘是 `time.Now`。`Now` 是函式欄位，測試可以換掉。
- **C9-9**　`Sweep` 每一輪用 `context.WithTimeout` 給自己一個期限（預設 30 秒），`defer cancel()`，並把這個帶期限的 `context` 傳給 `ReleaseOne`。錯誤分三種：錯誤是 `context.DeadlineExceeded` 而外層 `ctx` 沒被取消，記警告並結束這一輪；其他錯誤，外層 `ctx` 沒被取消時記錯誤並結束這一輪；找不到逾期訂單就結束這一輪。測試 `TestExpiryJanitor_SweepIsCancelledWhenTheRoundTimesOut` 把期限設成 20 毫秒，卡住的 store 在期限到時被取消，看到 `DeadlineExceeded`，`Sweep` 回傳 0。
- **C9-10**　逾期未付款的訂單會被釋放。測試下一張單買商品 1 的 2 件與最後 1 件螢幕，把時鐘撥快 1 小時後掃一輪：回傳釋放 1 張，商品 1 庫存回到 10，螢幕庫存回到 1，訂單的 `expired_at` 有值。再掃一輪釋放 0 張，商品 1 庫存仍是 10。測試 `TestExpiry_UnpaidOrderIsReleasedAndStockComesBack` 證明。
- **C9-11**　已付款的訂單不會被釋放：用 `Record` 記一筆付款後把時鐘撥快 1 小時，掃一輪釋放 0 張，商品 1 庫存維持 8。剛下單、還沒付款的訂單用真的時鐘掃，也釋放 0 張。測試 `TestExpiry_PaidOrderIsLeftAlone` 與 `TestExpiry_RecentUnpaidOrderIsLeftAlone` 證明。

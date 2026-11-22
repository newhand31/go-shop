# 超賣（short06）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep06`
- 主張清單：`docs/claims/ch06.json`（執行 `make claims` 逐條檢查）
- 片長：00:38
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 超賣。
- `00:02` 事故情境是這樣。27 吋螢幕庫存只有 1 台，卻有 3 張已付款的訂單。
- `00:10` 原因是舊寫法先讀庫存，再判斷，再寫回去。多個買家同時讀到庫存是 1，每個買家都判斷夠賣。〔C6-3〕
- `00:20` 修法是 FOR UPDATE。它把查到的那一列鎖到交易結束，別的交易要等待。〔C6-5、X6-2〕
- `00:27` 測試讓 20 個買家搶 1 台螢幕。結果是 1 個成功，19 個缺貨。〔C6-7〕
- `00:34` 完整說明在長片第 6 章。

## 這份逐字稿引用的主張

- **C6-3**　舊寫法 `naiveOrders` 只存在於測試檔。它在交易裡用一般的 `SELECT stock` 讀庫存（測試檔的註解寫明：即使在交易裡，一般的 `SELECT` 也不上鎖），在 Go 裡判斷，再用 `UPDATE products SET stock = $2` 把讀到的數字減去數量後寫回。測試 `TestOversell_WithoutALockMoreThanTheStockIsSold` 讓 20 個買家搶 1 台螢幕，賣出的數量超過 1，測試通過，代表超賣被重現。
- **C6-5**　修法的鎖是一條 SQL：`LockProduct` 查詢結尾是 `FOR UPDATE`。`Place` 在交易裡先呼叫 `LockProduct`，再用讀到的庫存和數量比較，最後才呼叫 `DecreaseStock` 扣庫存。扣庫存的 SQL 是 `stock = stock - $2`。
- **C6-7**　同樣 20 個買家搶 1 台螢幕，改走真的 `repository.Orders` 之後（測試包了 `slowBegin`，讓每個交易在每次呼叫 `Exec` 前暫停 50 毫秒；`slowTx` 只覆寫 `Exec`，`QueryRow` 不暫停），剛好 1 個買家成功、19 個得到缺貨，資料表裡有 1 張訂單、庫存是 0。測試 `TestOversell_WithForUpdateExactlyTheStockIsSold` 證明。把查詢裡的 `FOR UPDATE` 拿掉，這個測試就失敗，輸出含 `products_stock_check`：資料表的 `CHECK` 擋住負庫存，但有買家收到的是資料庫錯誤，不是缺貨。
- **X6-2**（說明用對照，機器不檢查）　PostgreSQL 的 `SELECT ... FOR UPDATE` 會鎖住查到的列，直到交易結束；別的交易要更新或鎖同一列時要等待。

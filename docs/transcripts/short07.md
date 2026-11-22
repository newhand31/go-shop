# 付兩次錢（short07）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep07`
- 主張清單：`docs/claims/ch07.json`（執行 `make claims` 逐條檢查）
- 片長：00:41
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 付兩次錢。
- `00:02` 財務傳來訊息。訂單 42 號，銀行只進來一筆，營收表上卻有兩筆。
- `00:10` 原因是付款通知會重送。假設先查再新增的寫法，會各記一列。這寫法不在 go-shop 裡。〔C7-7〕
- `00:20` 修法是把 trade_no 宣告為 UNIQUE。寫入用 ON CONFLICT。重複的影響 0 列。〔C7-5〕
- `00:28` 同一份回呼用 10 個 goroutine 同時送出。10 個都回 1|OK，payments 表只有 1 列。〔C7-10〕
- `00:36` 完整的程式碼和測試，在長片的第 7 章。

## 這份逐字稿引用的主張

- **C7-5**　`payments` 表的 `trade_no` 欄位宣告 `UNIQUE`。`InsertPayment` 的 SQL 用 `ON CONFLICT (trade_no) DO NOTHING`，所以重複的 `trade_no` 影響 0 列，不是錯誤。`Record` 在影響 0 列時回傳 `false, nil`。`db/queries/payments.sql` 裡沒有 `SELECT`，也就是沒有先查再寫。遷移檔的註解說明：同一個回呼重送時，`trade_no` 相同。`trade_no` 來自綠界回呼的 `TradeNo` 欄位；`ecpay.go` 的註解說明 `TradeNo` 是綠界的交易編號，每筆付款唯一、重送時相同；回呼沒有 `TradeNo` 就被拒絕。
- **C7-7**　回呼路由對綠界的回應有四種：處理成功回 200 與 `1|OK`；請求內容讀不成表單（例如超過 16 KB 的上限）回 400 與 `0|bad form`；簽章、格式、金額或訂單不對（`ErrInvalidCallback`、`ErrPaymentMismatch`）回 400 與 `0|rejected`；其他錯誤（例如資料庫掛了）回 500 與 `0|error`，不回 `1|OK`。路由程式裡只有這四個 `reply` 呼叫。測試 `TestECPayCallbackRoute` 涵蓋成功、簽章或金額不對、其他錯誤這三類並通過；測試 `TestECPayCallbackRoute_OversizedBodyIsRejected` 證明超過上限的請求回 400。程式註解說明：綠界收到不是 `1|OK` 的回應就會重送。
- **C7-10**　整合測試用真的 PostgreSQL，走完整的 handler、service、repository。`TestPayments_ResentCallbackIsRecordedOnce` 把同一份簽好章的回呼用 10 個 goroutine 同時送出：10 個都收到 200 與 `1|OK`，`payments` 表 1 列，訂單標成已付。`TestPayments_DifferentTradeNoIsNotADuplicate` 證明不同的 `trade_no` 記成 2 列。`TestPayments_UniqueKeyIsEnforcedByTheDatabase` 繞過 service 直接寫入同一個 `trade_no` 兩次，第二次被資料庫以 SQLSTATE 23505 拒絕。

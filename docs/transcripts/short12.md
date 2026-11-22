# 部署時進行中的結帳被切斷（short12）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep12`
- 主張清單：`docs/claims/ch12.json`（執行 `make claims` 逐條檢查）
- 片長：00:40
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 部署時進行中的結帳被切斷。
- `00:03` 部署新版時，一位客人剛按下結帳，畫面變成白的。
- `00:09` 原因是 ep11 只監聽 Ctrl+C，不監聽 SIGTERM，收到取消就呼叫 srv.Close，立刻切斷連線。〔C12-1〕
- `00:18` 修法是用 Shutdown 排空伺服器，進行中的請求做完才回傳。〔C12-2〕
- `00:24` 背景 goroutine 排在後面才停：serveThenStop 先 Serve，再 Stop。〔C12-6〕
- `00:30` 證明：訊號來了，排隊的客人也收到 201。〔C12-6〕
- `00:36` 完整說明在長片第 12 章。

## 這份逐字稿引用的主張

- **C12-1**　tag `ep11` 的 `cmd/api/main.go` 只監聽 `os.Interrupt`，不監聽 `SIGTERM`；收到取消之後直接 `return srv.Close()`。tag `ep12` 的 `cmd/api/main.go` 同時監聽 `os.Interrupt` 與 `syscall.SIGTERM`。
- **C12-2**　`lifecycle.Serve` 收到取消之後呼叫 `srv.Shutdown`，並且等 `Serve` 回傳才結束。測試 `TestServeLetsAnInFlightRequestFinishBeforeReturning` 證明：取消之後，進行中的請求照樣收到完整回應，`Serve` 回傳 `nil`。把 `srv.Shutdown` 換成 `srv.Close`，這個測試就失敗。
- **C12-6**　`serveThenStop` 只有兩個動作，順序是先 `lifecycle.Serve`，再 `workers.Stop`；`run` 呼叫的就是 `serveThenStop`。測試 `TestStopSignalLetsQueuedBuyersFinish`：一個 worker、客人 A 卡在資料庫、客人 B 在佇列，訊號來了之後兩位都收到 201。把這兩行對調，這個測試就失敗。

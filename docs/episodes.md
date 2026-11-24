# 分集表

事故驅動：每集從一個線上事故開始，只引入一個主要 Go 概念，最後用測試或壓測證明修好。
每集完成後在 `main` 打 tag（`ep01`…`ep12`、番外 `ep13-pitfalls`）。
平實口播版的規則見 [plain-style-guide.md](plain-style-guide.md)，各章的主張清單在 [claims/](claims/)，逐字稿在 [transcripts/](transcripts/)。第一版幽默文案的腳本骨架與語氣見 [style-guide.md](style-guide.md)。

> 「C# 對照」欄標 ＊ 的是 repo 建立時補的草案，錄製前再定。

| 集 | 事故 | Go 概念 | C# 對照（Java 為輔） |
|---|---|---|---|
| 01 | **老闆說明天開賣**：從零要有一個能上線的 API | 專案結構（`cmd/`、`internal/`）、chi router、`log/slog` | 沒有 DI 容器：`main` 裡手動 new，取代 `IServiceCollection`／Spring 容器 |
| 02 | **商品要存起來**：記憶體裡的商品重啟就沒了 | pgx、sqlc、`context.Context`、goose migration | 沒有 ORM：寫 SQL 產生程式碼，取代 EF Core／JPA |
| 03 | **API 格式亂**：前端說每支 API 回的錯誤長得不一樣 | OpenAPI + oapi-codegen、`error` 回傳值、`errors.Is/As`、`%w` 包裝 | 沒有 exception：錯誤是值，不是 `try/catch` |
| 04 | **有人亂打 API**：同一個 IP 一秒打一千次 | middleware 鏈、session、限流 | ＊middleware 就是函式包函式，取代 ActionFilter／Servlet Filter |
| 05 | **購物車不見**：換一台機器購物車就空了 | Redis、隱式 interface（使用端定義） | ＊不用寫 `: IRepository`／`implements`，方法對上就滿足 |
| 06 | **超賣**：庫存 1 件賣出 3 件，用 k6 重現 | DB 交易、`SELECT ... FOR UPDATE`、`defer tx.Rollback` | 沒有 `[Transactional]`／`TransactionScope`：交易邊界自己寫 |
| 07 | **付兩次錢**：綠界回呼重送，訂單入帳兩次 | 自寫綠界回呼驗章（`crypto/sha256`、`net/url`）、冪等鍵 | ＊沒有 middleware 魔法幫你去重，冪等鍵自己存 |
| 08 | **信沒寄出**：下單成功但確認信沒寄 | outbox pattern、goroutine | 不是 async/await：goroutine 不回傳 Task，要自己管生命週期 |
| 09 | **庫存被佔**：沒付款的訂單一直佔著庫存 | 排程（`time.Ticker`）、`context.WithTimeout` 逾時取消 | ＊context 取代 `CancellationToken`，但還帶 deadline 與值 |
| 10 | **雙 11 打爆 DB**：搶購瞬間 DB 連線池耗盡 | Redis 預扣、channel 與背壓 | ＊channel 對照 `System.Threading.Channels`／`BlockingQueue` |
| 11 | **半夜查不到原因**：只知道「有時候很慢」 | OpenTelemetry trace、Prometheus metrics、trace id 串 slog | ＊對照 Application Insights／Micrometer，差在要自己接線 |
| 12 | **部署斷線**：每次部署都有請求被砍掉 | graceful shutdown、多階段 Dockerfile、scratch image | ＊對照 `IHostApplicationLifetime`；單一靜態二進位不需要 runtime |
| 番外 | **踩坑合集** | nil interface 不等於 nil、迴圈變數捕捉（Go 1.22 前後）、slice 共用底層陣列 | ＊各自對照 C# 的 null、closure 捕捉、`List<T>` 複製語意 |
| 14 | **前端沒地方付錢**：訂單成立了，前端卻不能開始付款、也查不到訂單付了沒 | 開始付款與查訂單兩支 API（綠界訂單表單自己簽章）、struct 嵌入（`embedding`）把三個 service 接成一個 handler 介面 | ＊嵌入像組合而非繼承：方法被「提升」，不是 `base`／`super` |

第 14 集同時補上前面幾集留下的三個設定缺口，讓多語言系列的契約測試能跑：確認信改走 SMTP（`SMTP_HOST`、`SMTP_PORT`）、訂單逾期時間與清掃間隔可調（`ORDER_TTL_SECONDS`、`EXPIRY_SWEEP_SECONDS`）、接受進來的 `traceparent`（`propagation.TraceContext{}`）。併進 `main` 後打 tag `ep14-pay-and-orders`。

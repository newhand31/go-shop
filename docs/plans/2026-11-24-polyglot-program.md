# 多語言計畫：同一份契約，多個後端與前端系列（英文優先）

- 指揮台：未定（開工的窗填），工作鍵 `newhand31/go-shop:polyglot-program`
- 決策：`state/decisions/2026-11-24-204538-5d23bfb0.md`（英文優先與配音）、`state/decisions/2026-11-24-204539-f51a378a.md`（多語言計畫）
- 使用者點頭：2026-11-24 同意全部排程、自動推進、每系列先行章把關
- tick：on（指揮窗不在時，排程 tick 可以推進本計畫）
- 時盒：沒有期限。同時最多兩條線。

## 現況（2026-11-24 查）

- go-shop 的 13 章程式碼、tag、主張清單都做完。CI 每次 push 跑嚴格主張檢查。
- 影片產線在 `go-shop-video`：分鏡 → 配音 → 字幕 → Remotion。目前是中文配音（edge-tts），字幕燒在畫面上。
- `api/openapi.yaml` 只有 6 支 API（商品、購物車、結帳）。綠界回呼與限時搶購不在契約裡。「開始付款」與「查訂單」兩支 API 不存在，前端走不完付款流程。
- Kokoro 試作完成：第 5 章前 9 段英文口播，三個聲音。只用 CPU，2 分鐘音檔約 30 秒產生。`C#` 要寫成 `C sharp`、`ep05` 要寫成 `ep-05` 才念得對。

## 目標

同一組事故，用不同語言各做一個系列。每個系列的成品是一支分章節的長片，加上推廣它的 Short。任何一個前端接任何一個後端都能跑，由同一套契約測試與 E2E 證明。

## 約束

- 成品形式（2026-11-24 決策）：一個系列上傳一支長片，各章是長片裡的章節（YouTube 章節時間戳）。章不單獨上傳。Short 的用途是推廣長片：每支 Short 講一個事故，結尾指向長片的對應章節。
- 口播結構：長片開場 15 秒內講痛點；每章先跑錯的寫法、再改成正解；每做完一個功能就跑一次看結果；每章最多一個生活比喻（不是 repo 事實，不掛主張編號）；長片結尾依序是回顧、限制、下一步。不放贊助段。
- 英文口播是唯一正本。長片上傳英文與中文兩條 CC，畫面不燒字幕。Short 燒英文字幕，另附中文 CC。
- 配音：Kokoro-82M、聲音 `af_heart`。影片說明欄加一行模型致謝。
- 各系列以自己的語言為主角。語言對照以 C# 為主、Java 為輔。C# 系列對照 Go 版的同一個事故。前端只在有真正對應的概念時才提 C#。
- 一個系列一個 repo。go-shop 既有的 tag 不移動。
- 新 repo 先建成私人的。轉公開、上傳 YouTube 一律由作者決定，排程不自動做。
- 每章只引入一個主要概念。每句 repo 事實都要對應一條主張。
- 公開 repo 不放密鑰、個人資料、本機路徑。

## 你介入的時點

1. 每個系列的先行章做完：看一次，說可以才放行其餘章。
2. 機器擋下某條線，自己修了三輪還不過：那條線停下，寫一筆 todo。其他線繼續。
3. 公開發佈。

## 每章的驗收（機器把關）

| 檢查 | 後端章 | 前端章 |
|---|---|---|
| 該章的 tag 已打 | ✓ | ✓ |
| 契約測試通過到該章的等級 | ✓ | — |
| 主張清單全部 `PASS`（`claimcheck -strict`） | ✓ | ✓ |
| 字幕檢查：口播的主張編號都在清單裡；en 與 zh 句數相同 | ✓ | ✓ |
| Playwright E2E、axe 零違規、截圖差異在門檻內 | — | ✓ |

一個系列的各章都過了，再組長片。長片的驗收：章節時間戳從各章長度算出、第一個是 `0:00`、每章至少 10 秒；英文與中文 SRT 接起來後時間不重疊；每支 Short 的說明欄有長片連結的欄位（網址由作者上傳後填）。

## 契約測試等級

後端的 `chNN` tag 要通過第 N 章以前的所有等級。

| 等級 | 對應章 | 怎麼驗 |
|---|---|---|
| L1 商品 | ch01–02 | Hurl |
| L2 統一錯誤格式 | ch03 | Hurl＋Schemathesis |
| L3 限流 | ch04 | Hurl 連打，預期 429 |
| L4 購物車跟著 session | ch05 | Hurl 帶 cookie |
| L5 不超賣 | ch06 | k6 併發 |
| L6 付款回呼冪等 | ch07 | 同一個回呼送兩次，只入帳一次 |
| L7 確認信一定寄出 | ch08 | 測試用郵件伺服器收信 |
| L8 逾期訂單釋放庫存 | ch09 | 逾期時間設成幾秒 |
| L9 搶購背壓 | ch10 | k6 |
| L10 可觀測性 | ch11 | `/metrics`＋`traceparent` |
| L11 優雅關機 | ch12 | 壓測中送 SIGTERM，請求零失敗 |

## repo

| repo | 內容 | 狀態 |
|---|---|---|
| `go-shop` | Go 系列（本 repo） | 已有 |
| `go-shop-video` | 影片產線，擴充成支援多個 repo 與雙語 | 已有 |
| `shop-contract` | 完整契約、共用 migration、契約測試（Hurl、Schemathesis、k6） | 待建 |
| `cs-shop` | C# 系列：.NET 10、ASP.NET Core Minimal API | 待建 |
| `rs-shop` | Rust 系列：axum＋sqlx | 待建 |
| `shop-web` | 前端 monorepo：`packages/core`（產生的 API client＋TanStack Query 查詢定義）、`packages/tokens`、`packages/e2e`、`apps/next`、`apps/nuxt`、`apps/sveltekit`、`apps/angular` | 待建 |

## 分片

狀態值：`todo`（還沒開始）、`doing`、`gate`（等作者看先行章）、`done`。依賴欄列出要先 `done` 的片。

| id | 內容 | repo | 依賴 | 先行章 | 狀態 |
|---|---|---|---|---|---|
| P0a | 產線雙語化：分鏡 `narration: { en, zh }`、句數檢查、Kokoro 配音、念法對照表、音量正規化、whisper 取時間點、英文與中文 SRT、長片不燒字幕。長片組裝：把各章接成一支、產生章節時間戳與合併後的 SRT。補英文口播規範與口播結構到 `docs/plain-style-guide.md` | go-shop-video、go-shop | — | — | todo |
| P0b | Go 英文版：一支長片（13 章，英文口播與中文 CC）、13 支 Short。改 `CLAUDE.md`、README 的語言說明 | go-shop、go-shop-video | P0a | ch05 | todo |
| P0c | 建 `shop-contract`：完整契約（含付款回呼、搶購）、L1–L11 契約測試，拿 go-shop 跑到全綠 | shop-contract | — | — | todo |
| P0d | go-shop 新增一章：「開始付款」與「查訂單」兩支 API，進契約，打新 tag | go-shop、shop-contract | P0c | — | todo |
| P1 | C# 系列 13 章：程式碼、tag、主張清單、一支長片、Short | cs-shop | P0a、P0c | ch01 | todo |
| P2 | 前端共用篇 4 章（一支長片）：從契約產生 client、設計 token、錯誤碼對照、同一份 E2E | shop-web | P0a、P0d | ch01 | todo |
| P3 | React（Next.js）8 章（一支長片）＋Short | shop-web | P2 | ch01 | todo |
| P4 | Vue（Nuxt）、Svelte（SvelteKit）、Angular 各 8 章（各一支長片）＋Short | shop-web | P3 | 各自的 ch01 | todo |
| P5 | Rust 系列 13 章（一支長片）＋Short | rs-shop | P0a、P0c | ch01 | todo |
| P6 | 對照特輯 4 支：超賣、付兩次、部署斷線各三種語言對照；換後端、前端不用改 | go-shop-video | P1、P4、P5 | 第 1 支 | todo |

前端每個框架的 8 章：開賣頁空白（骨架、路由）、搜尋引擎找不到商品（SSR）、加入購物車沒反應（資料抓取、樂觀更新）、換頁購物車就空了（SSR 轉送 cookie）、錯誤訊息看不懂（表單、欄位錯誤）、結帳按兩次（送出中的狀態）、手機看不到結帳鈕與讀螢幕軟體用不了（元件、無障礙）、首頁好慢與部署（效能、建置、換後端）。

## 開工順序

1. P0a 與 P0c 同時開始（不同 repo）。
2. P0a 完成 → P0b。P0c 完成 → P0d。
3. 之後依 P1、P2、P3、P4、P5、P6 的順序補進空出的線。

## 風險

- 同一個風格問題被複製到每一章：由先行章擋。
- 同一個事故在別的語言沒有對應概念（例：Rust 沒有隱式 interface）：事故不變，概念換成該語言原生的。
- 截圖比對誤判：各框架字型渲染有細微差異，門檻在 P2 定。
- Kokoro 念錯程式名詞：每章配音後用 whisper 轉回文字，與原稿比對。
- 另一個窗宣告了 `docs/plans/2026-11-17-trial-v1.md`：本計畫不動那個檔。

# 多語言計畫：同一份契約，多個後端與前端系列（英文優先）

- 指揮台：`local_2e9951ea-e36b-4957-bf04-350ce6475691`（2026-11-24 開工），工作鍵 `newhand31/go-shop:polyglot-program`
- 決策（示範店面與全端旗艦片）：見 `state/DECISIONS.md` 2026-11-24「前後端同片」那一條
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
- 長度：每章 8–10 分鐘。13 章的長片約 2 小時，8 章的約 1 小時 10 分。字幕檢查時每章配音長度要落在 8–10 分鐘，超出就擋下。
- 口播結構：長片開場 15 秒內講痛點；每章先跑錯的寫法、再改成正解；每做完一個功能就跑一次看結果；每章最多一個生活比喻（不是 repo 事實，不掛主張編號）；長片結尾依序是回顧、限制、下一步。不放贊助段。
- 英文口播是唯一正本。長片上傳英文與中文兩條 CC，畫面不燒字幕。Short 燒英文字幕，另附中文 CC。
- 示範店面（2026-11-24 決策）：後端長片用一個現成的示範店面當驗收畫面，不教它的程式碼。每章開場在店面上重現事故，接著看後端錯的寫法、改成正解、跑契約測試，結尾回到店面確認事故不再發生。Go、C#、Rust 三支長片用同一個店面，證明換後端、前端不用改。店面畫面用 Playwright 自動錄。前端只出現在每章的開場與結尾，口播不解釋前端程式碼；後端長片結尾指向前端系列。限流、可觀測性、優雅關機這類店面上看不出差別的章不硬套，照舊用終端機與壓測圖。
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
4. 全端旗艦片（P7）動工前：長度、排在哪、與 Go 系列和 React 系列重複的內容怎麼處理，都由作者決定。

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
| `shop-contract` | 完整契約、共用 migration、契約測試（Hurl、Schemathesis、k6） | 已建（私人） |
| `cs-shop` | C# 系列：.NET 10、ASP.NET Core Minimal API | 待建 |
| `rs-shop` | Rust 系列：axum＋sqlx | 待建 |
| `shop-web` | 前端 monorepo：`packages/core`（產生的 API client＋TanStack Query 查詢定義）、`packages/tokens`、`packages/e2e`、`apps/next`、`apps/nuxt`、`apps/sveltekit`、`apps/angular` | 已建（私人） |

## 分片

狀態值：`todo`（還沒開始）、`doing`、`gate`（等作者看先行章）、`done`。依賴欄列出要先 `done` 的片。

| id | 內容 | repo | 依賴 | 先行章 | 狀態 |
|---|---|---|---|---|---|
| P0a | 產線雙語化：分鏡 `narration: { en, zh }`、句數檢查、Kokoro 配音、念法對照表、音量正規化、whisper 取時間點、英文與中文 SRT、長片不燒字幕。長片組裝：把各章接成一支、產生章節時間戳與合併後的 SRT。補英文口播規範與口播結構到 `docs/plain-style-guide.md` | go-shop-video、go-shop | — | — | doing |
| P0b | Go 英文版：一支長片（13 章，英文口播與中文 CC）、13 支 Short。改 `CLAUDE.md`、README 的語言說明 | go-shop、go-shop-video | P0a、P0e1 | ch05 | todo |
| P0c | 建 `shop-contract`：完整契約（含付款回呼、搶購）、L1–L11 契約測試，拿 go-shop 跑到全綠 | shop-contract | — | — | done |
| P0d | go-shop 新增一章：「開始付款」與「查訂單」兩支 API，進契約，打新 tag。同一片補 P0c 找到的三個缺口，讓 L7、L8、L10 變 `PASS`：用設定選的 SMTP 寄信、訂單逾期時間與清掃間隔可設定、接受進來的 `traceparent`（既有 tag 不動，改在 `main` 上）。契約測試那邊補四件：被擋的級在缺口補上後要變紅、L10 的缺口只認「追蹤編號不同」這一種失敗、L5／L6／L9／L11 各做一次故意弄壞的驗證、L11 加一個慢請求 | go-shop、shop-contract | P0c | — | todo |
| P0e1 | 示範店面前三段：一個 Next.js 店面，就是 `shop-web` 的 `apps/next` 最小版，先做商品、購物車、結帳（現有的 6 支 API 就夠用），後端網址用設定切換。每個事故一支 Playwright 腳本，自動錄下「事故發生」與「事故不再發生」兩段畫面。產線能把錄好的片段插進分鏡。作者 2026-11-24 同意先做這三段，P0a 或 P0c 的線一空出來就開 | shop-web、go-shop-video | — | — | doing |
| P0e2 | 示範店面後兩段：付款與查訂單，補上對應事故的 Playwright 腳本 | shop-web | P0e1、P0d | — | todo |
| P1 | C# 系列 13 章：程式碼、tag、主張清單、一支長片、Short | cs-shop | P0a、P0c、P0e2 | ch01 | todo |
| P2 | 前端共用篇 4 章（一支長片）：從契約產生 client、設計 token、錯誤碼對照、同一份 E2E | shop-web | P0a、P0d、P0e2 | ch01 | todo |
| P3 | React（Next.js）8 章（一支長片）＋Short | shop-web | P2 | ch01 | todo |
| P4 | Vue（Nuxt）、Svelte（SvelteKit）、Angular 各 8 章（各一支長片）＋Short | shop-web | P3 | 各自的 ch01 | todo |
| P5 | Rust 系列 13 章（一支長片）＋Short | rs-shop | P0a、P0c、P0e2 | ch01 | todo |
| P6 | 對照特輯 4 支：超賣、付兩次、部署斷線各三種語言對照；換後端、前端不用改 | go-shop-video | P1、P4、P5 | 第 1 支 | todo |
| P7 | 全端旗艦片：Go＋Next.js 從頭做到尾的一支全端長片，其餘系列照舊。長度、排在哪、與 Go 系列和 React 系列重複的內容怎麼處理都還沒定，動工前先問作者 | go-shop、shop-web、go-shop-video | P0b、P3 | 動工前問作者 | todo |

前端每個框架的 8 章：開賣頁空白（骨架、路由）、搜尋引擎找不到商品（SSR）、加入購物車沒反應（資料抓取、樂觀更新）、換頁購物車就空了（SSR 轉送 cookie）、錯誤訊息看不懂（表單、欄位錯誤）、結帳按兩次（送出中的狀態）、手機看不到結帳鈕與讀螢幕軟體用不了（元件、無障礙）、首頁好慢與部署（效能、建置、換後端）。

## 執行紀錄

指揮窗每次改狀態時更新。證據以 git 與 gh 實查為準，不採執行者自述。

| id | 執行者 | 分支／工作鍵 | 證據 |
|---|---|---|---|
| P0a | `agent:worker`（2026-11-24 派出） | 產線：`local/go-shop-video:feat/bilingual-pipeline`（該 repo 沒有 remote，只在本機）；口播規範：`newhand31/go-shop:feat/english-narration-guide` | 驗收過，產線分支等合併。產線分支在第 1 輪修正後多 6 個 commit（head `7570c70`）；指揮窗清掉快取重跑 `npm run verify` exit 0，12 段配音與 12 段轉錄都沒走快取。全新 checker 第 2 輪：通過，0 個高、2 個中、3 個低。兩個中等問題是環境變數能放寬檢查（`ALLOW_FAKE_TIMINGS`、`PIPELINE_CONFIG`、`CLAIMS_LOCAL`），排進「產線插入錄影片段」那一片一起修。口播規範的 PR 26 已併（merge commit `cdb24cc`）。舊的 39 集字串分鏡現在會直接報錯、不能配音，要搬成 `{ en, zh }` 才能用，這在 P0b 處理 |
| P0c | `agent:worker`（2026-11-24 派出） | `newhand31/shop-contract:feat/contract-l1-l11` | 完成。PR `newhand31/shop-contract` 1 號已併（merge commit `b1d961c`，作者 2026-11-25 說由指揮窗併）。指揮窗自己跑全跑指令：結束碼 2，L1–L6、L9、L11 `PASS`，L7、L8、L10 `BLOCKED`，沒有 `FAIL`。全新 checker 第 1 輪：通過，0 個高、4 個中、8 個低。被擋的三級與 4 個中等問題併進 P0d |
| P0e1 | `agent:worker`（2026-11-24 派出） | `newhand31/shop-web:feat/storefront-first-three` | 驗收過，等合併。PR `newhand31/shop-web` 1 號，兩輪修正後 head 含 `42a5942`、`13549ed`（repo 是私人的）。指揮窗從全新 clone 跑安裝、`pnpm verify`、`pnpm record ch05` 都 exit 0，跑前跑後 `go-shop` 的 worktree 清單完全相同。全新 checker 第 3 輪：通過，0 個高、1 個中、3 個低。剩下的中等問題：兩個啟動器同時跑時，後到的失敗後會清掉先到的暫時 worktree（只傷啟動器自己的暫存）。第 5 章的事故補丁查過是真的改後端，沒有造假。第 6 章超賣的錄影還沒做，留到 P0b 做到那一章時補。產線插入錄影片段的部分等 P0a 的分支併進產線 `main` 再做 |

- P0e1 的驗收指令：`pnpm verify`（typecheck、lint、build、E2E 對 `go-shop` 的 `main`）exit 0；`pnpm record ch05` 產出事故發生與事故不再發生兩段錄影。
- P0a 的驗收指令：產線分支上 `npm run verify` exit 0；`go-shop` 的 PR 只動 `docs/plain-style-guide.md` 且 `go run ./cmd/claimcheck -strict` 通過。
- P0c 的驗收指令：`shop-contract` 的全跑指令對 `go-shop` 的 `main`，L1–L11 都 `PASS`；`go-shop` 做不到的級標 `BLOCKED` 並記在 `shop-contract` 的 `docs/gaps.md`，由指揮窗決定怎麼補。
- 產線 repo 的未追蹤資料夾 `episodes/measure01/` 已查：是另一個專案留下的試轉檔，沒有別的窗在做，本計畫不動它。
- 畫面與縮圖的風格還沒查證。P0a 不改畫面設計，留到 P0b 的先行章之前查。

## 開工順序

1. P0a 與 P0c 同時開始（不同 repo）。
2. P0a 或 P0c 的線一空出來 → P0e1。P0c 完成 → P0d。P0a 與 P0e1 都完成 → P0b（先行章 ch05 只用到購物車；付款那幾章的店面畫面等 P0e2）。P0d 與 P0e1 都完成 → P0e2。
3. 之後依 P1、P2、P3、P4、P5、P6、P7 的順序補進空出的線。P7 動工前先問作者。

## 風險

- 同一個風格問題被複製到每一章：由先行章擋。
- 同一個事故在別的語言沒有對應概念（例：Rust 沒有隱式 interface）：事故不變，概念換成該語言原生的。
- 截圖比對誤判：各框架字型渲染有細微差異，門檻在 P2 定。
- Kokoro 念錯程式名詞：每章配音後用 whisper 轉回文字，與原稿比對。
- 另一個窗宣告了 `docs/plans/2026-11-17-trial-v1.md`：本計畫不動那個檔。

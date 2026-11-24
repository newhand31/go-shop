# 多語言計畫：同一份契約，多個後端與前端系列（英文優先）

- 指揮台：第四個指揮窗 `local_49776d97-14be-43c4-89c4-23daa6dc0ed8`（2026-11-25 接手）。前三個指揮窗 `local_9ef31a99-8861-49a4-aa06-35dde6b89f39`、`local_ec80beeb-5c15-42d2-9d13-ba4177f78c20`、`local_2e9951ea-e36b-4957-bf04-350ce6475691` 都因 context 過長交接。規劃窗是 `local_cb9273e6-fc32-4611-bbf8-15ccb22061e0`。工作鍵 `newhand31/go-shop:polyglot-program`
- 決策（示範店面與全端旗艦片）：見 `state/DECISIONS.md` 2026-11-24「前後端同片」那一條
- 決策：`state/decisions/2026-11-24-204538-5d23bfb0.md`（英文優先與配音）、`state/decisions/2026-11-24-204539-f51a378a.md`（多語言計畫）
- 使用者點頭：2026-11-24 同意全部排程、自動推進、每系列先行章把關。2026-11-25 同意把各片的點頭欄填成 2026-11-24，P7 除外（動工前先問作者）
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

每片一節，排程 tick 照這個格式讀。狀態只寫裸字：`planned`、`running`、`verifying`、`fixing(n)`、`merging`、`decision`、`accepted`、`handoff`、`dropped`；附註寫在證據欄。等作者看先行章時狀態寫 `decision`。「點頭」欄只由作者填或同意後才填，沒有日期的片 tick 不派。證據以 git 與 gh 實查為準，不採執行者自述。

### P0a 產線雙語化

- id：P0a
- 目標：產線雙語化：分鏡 `narration: { en, zh }`、句數檢查、Kokoro 配音、念法對照表、音量正規化、whisper 取時間點、英文與中文 SRT、長片不燒字幕。長片組裝：把各章接成一支、產生章節時間戳與合併後的 SRT。補英文口播規範與口播結構到 `docs/plain-style-guide.md`
- repo：go-shop-video、go-shop
- 檔案：`go-shop-video` 整個 repo；`go-shop` 的 `docs/plain-style-guide.md`
- 驗收指令：產線分支上 `npm run verify` exit 0；`go-shop` 的 PR 只動 `docs/plain-style-guide.md` 且 `go run ./cmd/claimcheck -strict` 通過。
- 完成的樣子：產線能從 `{ en, zh }` 分鏡做出英文配音、兩條 SRT 與不燒字幕的長片；口播規範補進 `docs/plain-style-guide.md`。
- 依賴：無
- 先行章：無
- 執行者：`agent:worker`（2026-11-24 派出）；分支／工作鍵 產線：`local/go-shop-video:feat/bilingual-pipeline`（該 repo 沒有 remote，只在本機）；口播規範：`newhand31/go-shop:feat/english-narration-guide`
- 點頭：2026-11-24
- 狀態：accepted
- 證據：完成。產線分支 2026-11-25 併進產線 `main`（本機 merge commit `5d7f194`，作者說由指揮窗併；這個 repo 沒有遠端）。併完在產線主 checkout 跑單元測試：46 過、1 不過。不過的是「沒搬的集數配音會失敗且不產生輸出」那一條，它假設 `public/ch05` 不存在，但主 checkout 留有舊的配音輸出；在乾淨的 worktree 會過。這是測試寫法的問題，排進「產線插入錄影片段」那一片修。以下是併之前的驗收紀錄。產線分支在第 1 輪修正後多 6 個 commit（head `7570c70`）；指揮窗清掉快取重跑 `npm run verify` exit 0，12 段配音與 12 段轉錄都沒走快取。全新 checker 第 2 輪：通過，0 個高、2 個中、3 個低。兩個中等問題是環境變數能放寬檢查（`ALLOW_FAKE_TIMINGS`、`PIPELINE_CONFIG`、`CLAIMS_LOCAL`），排進「產線插入錄影片段」那一片一起修。口播規範的 PR 26 已併（merge commit `cdb24cc`）。舊的 39 集字串分鏡現在會直接報錯、不能配音，要搬成 `{ en, zh }` 才能用，這在 P0b 處理

### P0b Go 英文版

- id：P0b
- 目標：Go 英文版：一支長片（13 章，英文口播與中文 CC）、13 支 Short。改 `CLAUDE.md`、README 的語言說明
- repo：go-shop、go-shop-video
- 檔案：`go-shop` 的 `docs/claims/**`、`docs/transcripts/**`、`docs/plain-style-guide.md`、`README.md`、`CLAUDE.md`；`go-shop-video` 整個 repo。先行章這一段不動 `README.md`、`CLAUDE.md`
- 驗收指令：產線分支上 `npm test`、`npm run typecheck`、`npm run verify` 都 exit 0，設了 `ALLOW_FAKE_TIMINGS`、`PIPELINE_CONFIG`、`CLAIMS_LOCAL` 也不能放寬嚴格檢查；ch05 清掉快取後整條建置 exit 0，配音長度 8–10 分鐘，en 與 zh 句數相同，成片裡有店面的兩段錄影，長片版沒有燒字幕；`go-shop` 分支 `go run ./cmd/claimcheck -strict` exit 0。P0b 這一段不動 `api/`、`cmd/`、`internal/`、`docs/episodes.md`。
- 完成的樣子：先行章 ch05 的成片、兩條 SRT、Short、縮圖候選都在，作者看過說可以；之後 13 章接成一支長片，每章都過「每章的驗收」。
- 依賴：P0a,P0e1
- 先行章：ch05
- 執行者：`session:local_fa860364-f52d-4995-abab-ed8e9681a99f`（2026-11-25 派出，sonnet）；分支／工作鍵 `newhand31/go-shop:feat/p0b-english-series`、`local/go-shop-video:feat/p0b-english-series`
- 點頭：2026-11-24
- 狀態：running
- 證據：進行中。2026-11-25 第三個指揮窗實查：執行窗仍卡在它窗裡一個只有作者能答的提問，指揮窗看不到內容、也不代答；產線分支還是只有一個 commit `034de2e`（錄影片段的分鏡、正式章節拒絕測試用開關）；`go-shop` 的分支沒有新 commit、還沒推上 origin，沒有 PR。新指揮窗不等完成通知，用 git 實查分支進度。這一段只做先行章 ch05，做完停下等作者看。同一段做「產線插入錄影片段」、修產線的兩個中等問題與那條只在乾淨目錄會過的單元測試、查證畫面與縮圖風格

### P0c 契約與契約測試

- id：P0c
- 目標：建 `shop-contract`：完整契約（含付款回呼、搶購）、L1–L11 契約測試，拿 go-shop 跑到全綠
- repo：shop-contract
- 檔案：`shop-contract` 整個 repo
- 驗收指令：`shop-contract` 的全跑指令對 `go-shop` 的 `main`，L1–L11 都 `PASS`；`go-shop` 做不到的級標 `BLOCKED` 並記在 `shop-contract` 的 `docs/gaps.md`，由指揮窗決定怎麼補。
- 完成的樣子：完整契約與 L1–L11 契約測試都在 `shop-contract`，對 `go-shop` 跑得出每一級的結果。
- 依賴：無
- 先行章：無
- 執行者：`agent:worker`（2026-11-24 派出）；分支／工作鍵 `newhand31/shop-contract:feat/contract-l1-l11`
- 點頭：2026-11-24
- 狀態：accepted
- 證據：完成。PR `newhand31/shop-contract` 1 號已併（merge commit `b1d961c`，作者 2026-11-25 說由指揮窗併）。指揮窗自己跑全跑指令：結束碼 2，L1–L6、L9、L11 `PASS`，L7、L8、L10 `BLOCKED`，沒有 `FAIL`。全新 checker 第 1 輪：通過，0 個高、4 個中、8 個低。被擋的三級與 4 個中等問題併進 P0d

### P0d 付款與查訂單 API、補契約缺口

- id：P0d
- 目標：go-shop 新增一章：「開始付款」與「查訂單」兩支 API，進契約，打新 tag。同一片補 P0c 找到的三個缺口，讓 L7、L8、L10 變 `PASS`：用設定選的 SMTP 寄信、訂單逾期時間與清掃間隔可設定、接受進來的 `traceparent`（既有 tag 不動，改在 `main` 上）。契約測試那邊補四件：被擋的級在缺口補上後要變紅、L10 的缺口只認「追蹤編號不同」這一種失敗、L5／L6／L9／L11 各做一次故意弄壞的驗證、L11 加一個慢請求
- repo：go-shop、shop-contract
- 檔案：`go-shop` 的 `api/**`、`cmd/**`、`internal/**`、migration 與 sqlc 的路徑、`docs/episodes.md`、`docs/architecture.md`、`go.mod`、`go.sum`、設定範例檔、`Makefile`、`docker-compose*.yml`；`shop-contract` 整個 repo
- 驗收指令：`go-shop` 分支上 `go build ./... && go vet ./... && go test ./... && go run ./cmd/claimcheck -strict` exit 0；既有 tag 的 sha 不變；`shop-contract` 分支的全跑指令對 `go-shop` 的 P0d 分支 exit 0，L1–L11 都 `PASS`、沒有 `BLOCKED`，兩支新 API 的測試有跑到；對 `go-shop` 的 `main`（缺口還沒補）時 L7、L8、L10 不是 `PASS`；L5、L6、L9、L11 的「故意弄壞」驗證有可重跑的做法。檔案分工：P0d 不動 `README.md`、`CLAUDE.md`、`docs/claims/`、`docs/transcripts/`、`docs/plain-style-guide.md`。新 tag 等 PR 併進 `main` 後由指揮窗打。
- 完成的樣子：兩個 repo 各一支 PR 併進主幹；新章的 tag 由指揮窗在 merge commit 上打；L1–L11 全 `PASS`。
- 依賴：P0c
- 先行章：無
- 執行者：`session:local_291204f3-5fc5-4dea-9029-a4b5f237722c`（2026-11-25 派出，sonnet）；分支／工作鍵 `newhand31/go-shop:feat/p0d-pay-and-orders`、`newhand31/shop-contract:feat/p0d-gates`
- 點頭：2026-11-24
- 狀態：accepted
- 證據：完成。newhand31/go-shop#32 已併（merge commit `57b168f`，head `54929d5`），tag `ep14-pay-and-orders` 打在這個 merge commit 上；newhand31/shop-contract#2 已併（merge commit `7373bff`，head `04f15ab`，`go-shop` 併完後重跑 CI 是綠的）。合併依據：作者在規劃窗授權指揮窗審查與合併（由規劃窗轉達），驗收指令過、全新 checker 通過、指揮窗沒有疑慮才併。上一版交接寫「shop-contract#2 沒有新 commit」是過期資訊：`04f15ab` 修了 L6 的兩題。指揮窗 2026-11-25 第 2 輪驗收（全新 clone）：`go build`、`go vet`、`go test`、`claimcheck -strict` 結束碼 0（148 條過）；契約全跑對 P0d 分支結束碼 0，L1–L11 全 `PASS`；對併之前的 `main` 結束碼 1，L7、L8、L10 是 `FAIL`（L2、L6 也紅，因為當時 `main` 沒有兩支新 API）；`scripts/mutation-check.sh` 的五個故意弄壞驗證（5、6、6b、9、11）都被抓到；改動都在檔案清單內，既有 tag 沒動。全新 checker 第 2 輪：通過，0 個高、0 個中、5 個低，三個中等問題都修掉，交易編號的新格式對過綠界官方文件。規劃窗另開的一個獨立 checker 結論相同（0 高、0 中）。兩個 checker 的結論檔只在本機。留下的低等問題記在「其他紀錄」。跑契約測試的注意事項：這台機器的埠 15432 被另一個專案的資料庫占用，要帶 `SC_POSTGRES_PORT=25432`；`SRC_DIR` 要寫成 `C:/…` 的形式，寫 `/c/…` 時 `mutation-check.sh` 匯不出原始碼。以下是第 1 輪的紀錄（當時 go-shop#32 的 head 是 `44a274b`）。指揮窗 2026-11-25 第 1 輪驗收：兩個 head 都在 origin；改動都在檔案清單內；既有 tag 的 sha 沒變；全新 clone 上 `go build`、`go vet`、`go test`、`claimcheck -strict` 結束碼 0（148 條過）；契約全跑對 P0d 分支結束碼 0，L1–L11 全 `PASS`；同一份契約測試對 `go-shop` 的 `main` 結束碼 1，L7（結帳後寄信、信件伺服器斷線後補寄）、L8（三項檢查）、L10（log 裡的追蹤編號）是 `FAIL`，符合「缺口沒補時不是 `PASS`」。全新 checker 第 1 輪：不過，0 個高、3 個中、9 個低。三個中等問題：L6「開始付款兩次」比對簽章，跨秒時正確的後端也會紅；L6「只開始付款不會入帳」的檢查永遠不會失敗；每張付款表單用同一個交易編號，綠界可能拒絕第二次付款（未證實，要查官方文件）。已請執行窗修

### P0e1 示範店面前三段

- id：P0e1
- 目標：示範店面前三段：一個 Next.js 店面，就是 `shop-web` 的 `apps/next` 最小版，先做商品、購物車、結帳（現有的 6 支 API 就夠用），後端網址用設定切換。每個事故一支 Playwright 腳本，自動錄下「事故發生」與「事故不再發生」兩段畫面。產線能把錄好的片段插進分鏡。作者 2026-11-24 同意先做這三段，P0a 或 P0c 的線一空出來就開
- repo：shop-web、go-shop-video
- 檔案：`shop-web` 整個 repo
- 驗收指令：`pnpm verify`（typecheck、lint、build、E2E 對 `go-shop` 的 `main`）exit 0；`pnpm record ch05` 產出事故發生與事故不再發生兩段錄影。
- 完成的樣子：店面能走完商品、購物車、結帳，並錄下第 5 章的兩段畫面。
- 依賴：無
- 先行章：無
- 執行者：`agent:worker`（2026-11-24 派出）；分支／工作鍵 `newhand31/shop-web:feat/storefront-first-three`
- 點頭：2026-11-24
- 狀態：accepted
- 證據：完成。PR `newhand31/shop-web` 1 號已併（merge commit `0b16268`，作者 2026-11-25 說由指揮窗併），兩輪修正後 head 含 `42a5942`、`13549ed`（repo 是私人的）。指揮窗從全新 clone 跑安裝、`pnpm verify`、`pnpm record ch05` 都 exit 0，跑前跑後 `go-shop` 的 worktree 清單完全相同。全新 checker 第 3 輪：通過，0 個高、1 個中、3 個低。剩下的中等問題：兩個啟動器同時跑時，後到的失敗後會清掉先到的暫時 worktree（只傷啟動器自己的暫存）。第 5 章的事故補丁查過是真的改後端，沒有造假。第 6 章超賣的錄影還沒做，留到 P0b 做到那一章時補。產線插入錄影片段的部分等 P0a 的分支併進產線 `main` 再做

### P0e2 示範店面後兩段

- id：P0e2
- 目標：示範店面後兩段：付款與查訂單，補上對應事故的 Playwright 腳本。同一片修 P0e1 留下的中等問題：啟動器拿不到鎖時，收尾不能清掉另一個啟動器的暫時 worktree
- repo：shop-web
- 檔案：`shop-web` 整個 repo
- 驗收指令：全新 clone 上 `pnpm install --frozen-lockfile && pnpm verify` exit 0（E2E 對 `go-shop` 的 `main`，涵蓋付款成功與查訂單三種狀態）；`pnpm test:scripts` exit 0，含啟動器鎖的新測試；`pnpm record ch14`（前端沒地方付錢）與 `pnpm record ch09`（庫存被佔）各 exit 0、各產出事故與修好兩段錄影；`pnpm record ch05` 仍然 exit 0。
- 完成的樣子：店面能走完付款與查訂單，對應事故各有一支錄影腳本；啟動器的鎖問題修掉。
- 依賴：P0e1,P0d
- 先行章：無
- 執行者：`session:local_6a2b2027-1ed9-40b2-9515-ea2543750e80`（2026-11-25 派出，sonnet）；分支／工作鍵 `newhand31/shop-web:feat/p0e2-pay-and-orders`
- 點頭：2026-11-24
- 狀態：running
- 證據：進行中。2026-11-25 第四個指揮窗派出。章號由指揮窗定：第 14 章（付款與查訂單頁面本身；事故段用 tag `ep13-pitfalls` 的後端）與第 9 章（逾期訂單釋放庫存，靠查訂單頁看得到）是必做。第 7 章「付兩次錢」的差別只在入帳紀錄，查訂單 API 只回狀態，店面上可能看不出來；執行窗先評估，看得到才錄，看不到就在 PR 說明原因，不造假。付款不送到綠界的網站，E2E 與錄影改由腳本送簽章正確的回呼。執行窗開在 repo 上層資料夾（app 不讓指揮窗在自動接續的回合把子對話開到別的資料夾），它自己在短路徑開 `shop-web` 的 worktree 做。P0d 執行窗的收尾已用 coord 實查：兩個認領都是 `closed`（原因 `merged`），收尾記錄 `none`

### P1 C# 系列

- id：P1
- 目標：C# 系列 13 章：程式碼、tag、主張清單、一支長片、Short
- repo：cs-shop
- 檔案：`cs-shop` 整個 repo（待建，先建成私人的）
- 驗收指令：每章照「每章的驗收」表的後端章欄；派工前由指揮窗補成可執行指令
- 完成的樣子：該系列每章都過「每章的驗收」，長片過長片的驗收；先行章作者看過說可以。
- 依賴：P0a,P0c,P0e2
- 先行章：ch01
- 執行者：未派
- 點頭：2026-11-24
- 狀態：planned
- 證據：還沒開始

### P2 前端共用篇

- id：P2
- 目標：前端共用篇 4 章（一支長片）：從契約產生 client、設計 token、錯誤碼對照、同一份 E2E
- repo：shop-web
- 檔案：`shop-web` 的 `packages/**`
- 驗收指令：每章照「每章的驗收」表的前端章欄；派工前由指揮窗補成可執行指令
- 完成的樣子：該系列每章都過「每章的驗收」，長片過長片的驗收；先行章作者看過說可以。
- 依賴：P0a,P0d,P0e2
- 先行章：ch01
- 執行者：未派
- 點頭：2026-11-24
- 狀態：planned
- 證據：還沒開始

### P3 React（Next.js）

- id：P3
- 目標：React（Next.js）8 章（一支長片）＋Short
- repo：shop-web
- 檔案：`shop-web` 的 `apps/next/**`、`packages/e2e/**`
- 驗收指令：每章照「每章的驗收」表的前端章欄；派工前由指揮窗補成可執行指令
- 完成的樣子：該系列每章都過「每章的驗收」，長片過長片的驗收；先行章作者看過說可以。
- 依賴：P2
- 先行章：ch01
- 執行者：未派
- 點頭：2026-11-24
- 狀態：planned
- 證據：還沒開始

### P4 Vue、Svelte、Angular

- id：P4
- 目標：Vue（Nuxt）、Svelte（SvelteKit）、Angular 各 8 章（各一支長片）＋Short
- repo：shop-web
- 檔案：`shop-web` 的 `apps/nuxt/**`、`apps/sveltekit/**`、`apps/angular/**`、`packages/e2e/**`
- 驗收指令：每章照「每章的驗收」表的前端章欄；派工前由指揮窗補成可執行指令
- 完成的樣子：該系列每章都過「每章的驗收」，長片過長片的驗收；先行章作者看過說可以。
- 依賴：P3
- 先行章：各自的 ch01
- 執行者：未派
- 點頭：2026-11-24
- 狀態：planned
- 證據：還沒開始

### P5 Rust 系列

- id：P5
- 目標：Rust 系列 13 章（一支長片）＋Short
- repo：rs-shop
- 檔案：`rs-shop` 整個 repo（待建，先建成私人的）
- 驗收指令：每章照「每章的驗收」表的後端章欄；派工前由指揮窗補成可執行指令
- 完成的樣子：該系列每章都過「每章的驗收」，長片過長片的驗收；先行章作者看過說可以。
- 依賴：P0a,P0c,P0e2
- 先行章：ch01
- 執行者：未派
- 點頭：2026-11-24
- 狀態：planned
- 證據：還沒開始

### P6 對照特輯

- id：P6
- 目標：對照特輯 4 支：超賣、付兩次、部署斷線各三種語言對照；換後端、前端不用改
- repo：go-shop-video
- 檔案：`go-shop-video` 的 `episodes/**`
- 驗收指令：產線的 `npm run verify` exit 0；派工前由指揮窗補細項
- 完成的樣子：該系列每章都過「每章的驗收」，長片過長片的驗收；先行章作者看過說可以。
- 依賴：P1,P4,P5
- 先行章：第 1 支
- 執行者：未派
- 點頭：2026-11-24
- 狀態：planned
- 證據：還沒開始

### P7 全端旗艦片

- id：P7
- 目標：全端旗艦片：Go＋Next.js 從頭做到尾的一支全端長片，其餘系列照舊。長度、排在哪、與 Go 系列和 React 系列重複的內容怎麼處理都還沒定，動工前先問作者
- repo：go-shop、shop-web、go-shop-video
- 檔案：動工前跟作者談定範圍後再填
- 驗收指令：動工前跟作者談定範圍後再填
- 完成的樣子：該系列每章都過「每章的驗收」，長片過長片的驗收；先行章作者看過說可以。
- 依賴：P0b,P3
- 先行章：動工前問作者
- 執行者：未派
- 點頭：
- 狀態：planned
- 證據：還沒開始

## 其他紀錄

- 產線 repo 的未追蹤資料夾 `episodes/measure01/` 已查：是另一個專案留下的試轉檔，沒有別的窗在做，本計畫不動它。
- P0d 留下的低等問題（不擋合併，還沒排進任何片）：同一筆訂單開兩個付款分頁可能付兩次，系統只記帳、不示警（`internal/repository/payments.go`），建議寫 warn log 並在契約註明；故意弄壞驗證 6b 在「第二次開始付款回 409」就紅了，沒有單獨證明三條新的入帳檢查有用；訂單編號到 10 位數時交易編號會超過綠界的 20 字元上限，程式沒擋；契約允許交易編號 `T` 後面有大寫字母，`go-shop` 只收小寫；`docs/scripts/ep07.md`（中英）還寫著重複付款「沒處理」；tag 名 `ep14-pay-and-orders` 不在 `CLAUDE.md` 列的 tag 清單裡。
- 畫面與縮圖的風格還沒查證。P0a 不改畫面設計，留到 P0b 的先行章之前查（已排進 P0b 先行章那一段）。

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

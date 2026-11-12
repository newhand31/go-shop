# go-shop 專案規則

教學影片配套 repo：「用 Go 寫實戰購物網站」。觀眾是要轉 Go 的 C#／Java 後端工程師。
定位、技術棧、分集見 `README.md`、`docs/architecture.md`、`docs/episodes.md`。

## Git 工作流（B 類協作 repo）

- `main` 永遠可建置。不在 `main` 直接工作。
- 每件事開 `feat/<short-name>` → PR → `git merge --no-ff` → 刪分支。
- 還原 merge 用 `git revert <merge> -m 1`。
- **每集一個 tag**：該集內容併進 `main` 後，在 merge commit 打 annotated tag `ep01`…`ep12`，番外 `ep13-pitfalls`。tag 打了就不移動；要修就在後面加 commit 並在 README 註明。

## 分層

- `cmd/api/`：組裝（讀設定、建依賴、接線、啟動）。手動注入，不用 DI 容器或 wire。
- `internal/handler/`：HTTP 層。解析請求、呼叫 service、寫回應。不碰 SQL。
- `internal/service/`：業務邏輯與交易邊界。不碰 `http.Request`。
- `internal/repository/`：資料存取（sqlc 產生碼＋薄包裝）。不含業務規則。
- `internal/platform/`：跨層基礎設施（db 連線、redis、logger、otel）。
- 依賴方向只能 handler → service → repository。反向或跨跳都不行。

## Interface 規則

- **interface 定義在使用端**，不在實作端。例：service 需要存商品，就在 `service` 套件宣告 `type ProductStore interface{...}`；repository 只回傳具體型別。
- interface 要小，只列使用端真的用到的方法。
- 不為了「之後可能換」預先抽 interface。測試需要替身時再抽。

## 程式慣例

- 錯誤用回傳值，`fmt.Errorf("...: %w", err)` 包裝，邊界用 `errors.Is/As` 判斷。不用 panic 當流程控制。
- 每個會碰 I/O 的函式第一個參數是 `ctx context.Context`。
- log 一律 `log/slog`，結構化欄位，不用 `fmt.Println`。
- 第三方依賴新增前要能說出理由；能用標準庫就用標準庫。

## 測試

- 一律 `go test ./...`。table-driven 為預設寫法。
- 整合測試用 testcontainers-go，以 build tag `integration` 分開：`go test -tags=integration ./...`。
- 這台開發機沒有 gcc，`-race` 跑不起來；不要把 `-race` 寫成必要步驟（CI 上再開）。

## 教學一致性

- 每集只引入一個主要 Go 概念。程式碼要服務該集的事故劇情，不要提前用到後面集數的技術。
- 文案、腳本照 `docs/style-guide.md`。笑點只對著程式、事故、自己，不對著人。
- 金流 import `github.com/newhand31/twpayhook`，本 repo 只 import 不改它（那是使用者的學習 repo）。

## 公開 repo 與語言

- 本 repo 是公開的。不放密鑰、個人資料、本機路徑；不點名評論真人或其他創作者。
- 程式碼識別字、註解、測試案例名、commit 訊息一律英文（觀眾含英語市場）。
- README 中英雙語；改一邊要同步另一邊。
- 腳本按語言分：中文 `docs/scripts/epNN.md`，英文 `docs/scripts/en/epNN.md`。英文腳本照 `docs/style-guide.md` §6 重寫笑點，不直譯。英文版上獨立的英文頻道，語言對照以 Java 為主、C# 為輔（中文版維持 C# 為主）。
- 授權：程式碼 MIT（`LICENSE`），`docs/` 為 CC BY-NC 4.0（`docs/LICENSE.md`）。

## 決策紀錄

`state/` 由 state-ledger 管理。`state/DECISIONS.md` 是產生的檢視檔，不要手改；新增用 `~/.claude/skills/state-ledger/state-ledger.ps1 add`。

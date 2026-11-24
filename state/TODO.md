# TODO

- [ ] **平實口播版公開前還要決定的三件事**（交接自「go-shop｜平實口播收尾：CI＋第 5 章去行號」）
  - 目標：平實口播版可以公開發佈：配音合法、事故情境有重現、英文版有著落。
  - 現況：主張已有 CI 守著。`.github/workflows/claims.yml` 在每次 push 跑 `go run ./cmd/claimcheck -strict`，runner 上 148 passed、0 failed、0 skipped。
  - 現況：13 章口播都不唸行號，逐字稿與章節時間戳已更新。README（中英）有查證說明。C1-1 用 `{root}` 指第一個 commit，不寫死編號。
  - 現況：三項裡有兩項在 2026-11-24 定案（見 `state/DECISIONS.md`）。一，配音改用本地 Kokoro-82M（Apache-2.0）、聲音 `af_heart`；產線還沒接上，排在計畫的 P0a。三，英文版要做，而且英文優先；排在計畫的 P0b。計畫檔是 `docs/plans/2026-11-24-polyglot-program.md`。
  - 現況：還要使用者決定的剩一項。二，第 4、7、8、10 章的事故情境沒有測試或壓測重現（規範第 2 節）。
  - 現況：影片產線 repo 的分鏡改動在它的 `feat/plain-followups` 分支，還沒併進它的 main。
  - 檔案：`docs/plain-style-guide.md`、`docs/claims/`、`docs/transcripts/`；影片產線 repo 的 `episodes/`、`recordings/`。
  - 完成的樣子：四章的事故情境各有一條掛 `C` 的重現主張，或規範改寫成不要求。配音與英文版的實作由計畫的 P0a、P0b 驗收。
  - 工作鍵：newhand31/go-shop:feat/plain-voice-and-incidents

- [ ] **開新的多語言計畫指揮窗，同時開 P0d 與 P0b**（交接自「go-shop#25｜指揮多語言計畫 P0a＋P0c｜進行中」）
  - 目標：照 `docs/plans/2026-11-24-polyglot-program.md` 繼續推進剩下的分片。作者只在四個時點介入：每個系列的先行章、機器修三輪不過、公開發佈、P7 全端旗艦片動工前。
  - 現況：計畫檔是正本，執行紀錄節有每片的證據。P0a（產線雙語化）、P0c（契約與契約測試）、P0e1（示範店面前三段）都是 `done`，都已併進各自的主幹。
  - 現況：依賴已滿足的下一步是 P0d 與 P0b，同時最多兩條線。P0b 的 ch05 先行章做完要停下請作者看；做到畫面前要先查證畫面與縮圖風格。
  - 現況：P0d 要補的東西已寫進計畫檔：兩支新 API、go-shop 的三個缺口（讓契約測試 L7、L8、L10 從 BLOCKED 變 PASS，既有 tag 不動）、契約測試的四個中等問題。
  - 現況：產線留下兩個中等問題（環境變數可以放寬檢查）和一條只在乾淨工作目錄會過的單元測試，排在「產線插入錄影片段」那一片。舊的 39 集字串分鏡現在不能配音，搬成 `{en, zh}` 排在 P0b。
  - 現況：示範店面留下一個中等問題，還沒排進任何片：啟動器拿不到鎖時，收尾會清掉另一個啟動器的暫時 worktree。第 6 章超賣錄影沒做，留到 P0b。店面的埠是固定的，不能同時跑兩份。
  - 現況：計畫檔標頭的「指揮台」還寫著上一個指揮窗，新窗接手後要改。各 repo 已併的分支都還沒刪。
  - 檔案：`docs/plans/2026-11-24-polyglot-program.md`、`state/decisions/2026-11-24-*.md`、`docs/plain-style-guide.md`；另外三個 repo：影片產線、`shop-contract`、`shop-web`。
  - 完成的樣子：計畫檔每片都是 accepted、handoff 或 dropped。
  - 工作鍵：newhand31/go-shop:polyglot-program

- [x] **開多語言計畫的指揮窗，從 P0a 與 P0c 開始排程**（交接自「go-shop#24｜寫英文優先決策與多語言總計畫｜進行中」）
  - 目標：照 `docs/plans/2026-11-24-polyglot-program.md` 自動推進全部分片。作者只在三個時點介入：每個系列的先行章、機器修三輪不過、公開發佈。
  - 現況：計畫與三條決策已併進 main（PR 24）。形式都定了：英文優先、Kokoro `af_heart`、一個系列一支分章節的長片加 Short 推廣、每章 8–10 分鐘、對照以 C# 為主。所有分片都是 `todo`，還沒有指揮窗。
  - 現況：先開 P0a（產線雙語化，repo `go-shop-video`）與 P0c（建 `shop-contract`，先建成私人 repo），同時最多兩條線。
  - 現況：`go-shop-video` 有一個不是這條線建的未追蹤資料夾 `episodes/measure01/`，動產線前先查有沒有別的窗在做。它的 `feat/plain-followups` 分支還沒併進它的 main。
  - 現況：Kokoro 試作的做法：`kokoro-onnx`（不需 PyTorch）、模型 `kokoro-v1.0.onnx` 與 `voices-v1.0.bin`、`faster-whisper` 的 `small.en` 取字級時間點。`C#` 要念成 `C sharp`、`ep05` 要念成 `ep-05`。試作環境在暫存資料夾，P0a 要重建在 repo 裡。
  - 現況：畫面與縮圖的風格沒有查證（逐字稿看不出來），P0a 做到畫面時再查。
  - 檔案：`docs/plans/2026-11-24-polyglot-program.md`、`state/decisions/2026-11-24-*.md`、`docs/plain-style-guide.md`；影片產線 repo 的 `episodes/`、`src/subtitles.ts`。
  - 完成的樣子：計畫檔標頭填上指揮窗；P0a 與 P0c 的狀態是 `doing` 或之後的值，各有一支開著或已併的 PR。
  - 工作鍵：newhand31/go-shop:polyglot-program

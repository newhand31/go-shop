# TODO

- [ ] **平實口播版公開前還要決定的三件事**（交接自「go-shop｜平實口播收尾：CI＋第 5 章去行號」）
  - 目標：平實口播版可以公開發佈：配音合法、事故情境有重現、英文版有著落。
  - 現況：主張已有 CI 守著。`.github/workflows/claims.yml` 在每次 push 跑 `go run ./cmd/claimcheck -strict`，runner 上 148 passed、0 failed、0 skipped。
  - 現況：13 章口播都不唸行號，逐字稿與章節時間戳已更新。README（中英）有查證說明。C1-1 用 `{root}` 指第一個 commit，不寫死編號。
  - 現況：還沒做的有三項，都要使用者決定。一，配音是暫時的 TTS，公開前要換真人錄音或正式授權的 TTS。二，第 4、7、8、10 章的事故情境沒有測試或壓測重現（規範第 2 節）。三，英文版還沒寫（規範第 6 節）。
  - 現況：影片產線 repo 的分鏡改動在它的 `feat/plain-followups` 分支，還沒併進它的 main。
  - 檔案：`docs/plain-style-guide.md`、`docs/claims/`、`docs/transcripts/`；影片產線 repo 的 `episodes/`、`recordings/`。
  - 完成的樣子：配音來源有書面授權或是真人錄音；四章的事故情境各有一條掛 `C` 的重現主張，或規範改寫成不要求；英文版做或不做有一條決策紀錄。
  - 工作鍵：newhand31/go-shop:feat/plain-voice-and-incidents

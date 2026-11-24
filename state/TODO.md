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

# TODO

- [ ] **把其餘 14 章改成平實口播並補主張清單**（交接自「go-shop#21｜第5章平實口播＋主張清單試作｜待拍板」）
  - 目標：15 章都有平實口播、主張清單、逐字稿；合成一支分章節的長片（約 90 分），每章另出一支 Short。
  - 現況：第 5 章試作完成（PR #21）。長片片段 6 分 20 秒、Short 45 秒；`make claims` 是 11 passed、0 failed、0 skipped、1 reference。等使用者看過試作、說要調哪些規則，才開始批次改。
  - 現況：已定的決策見 `state/DECISIONS.md` 2026-11-23「文案：平實口播＋主張清單」。口播正本是影片產線 repo 的分鏡 JSON，逐字稿由它產生，不手寫。
  - 現況：還沒做的有 `docs/style-guide.md`、README（中英）與 `docs/episodes.md` 更新、長片合成與章節時間戳、英文版。這個 repo 沒有 CI，主張只能在本機跑 `make claims`。
  - 現況：試作留下的小問題：口播唸出的行號沒有機器檢查；事故故事是情境，只由測試重現；X5-1 的 C# 與 Java 寫法沒有附來源；`claimcheck` 的暫存目錄清理沒有用 defer。
  - 檔案：`docs/claims/chNN.json`、`docs/transcripts/`、`cmd/claimcheck/`、`docs/claims/README.md`；影片產線 repo 的 `episodes/chNN/storyboard.json`、`episodes/shortNN/storyboard.json`。
  - 完成的樣子：每章有 `docs/claims/chNN.json` 與 `docs/transcripts/chNN.md`；`go run ./cmd/claimcheck -strict` 結束碼 0；每章的長片片段與 Short 都算得出來，字幕檔引用的主張編號都存在。
  - 工作鍵：newhand31/go-shop:feat/plain-rollout

# TODO

- [ ] **補平實口播版的收尾項目**（交接自「go-shop｜其餘 12 章改平實口播＋主張清單｜待併」）
  - 目標：平實口播版可以公開發佈：配音合法、主張有 CI 守著、13 章規則一致。
  - 現況：13 章都有主張清單（`docs/claims/ch01.json`–`ch13.json`）與逐字稿（`docs/transcripts/`）；`go run ./cmd/claimcheck -strict` 全數通過。長片 1 小時 24 分 55 秒、13 支 Short 都已在影片產線 repo 算出，章節時間戳在 `docs/transcripts/chapters.txt`。
  - 現況：每一章都由全新 context 的檢查員對過 tag 的程式碼，找到的問題已修；修完之後沒有再做第二輪獨立檢查。
  - 現況：還沒做的有五項。一，配音是暫時的 TTS，公開前要換真人錄音或正式授權的 TTS。二，這個 repo 沒有 CI，主張只在本機跑。三，第 5 章口播還唸行號，與 `docs/plain-style-guide.md` 第 2 節不一致。四，第 4、7、8、10 章的事故情境沒有測試或壓測重現。五，英文版還沒寫（規範第 6 節）。
  - 現況：ch01 的 C1-1 引用了一個 commit 編號，歷史改寫後會失效。README（中英）還沒介紹主張清單。
  - 檔案：`docs/plain-style-guide.md`、`docs/claims/`、`docs/transcripts/`、`cmd/claimcheck/`、`README.md`；影片產線 repo 的 `episodes/chNN/storyboard.json`、`episodes/shortNN/storyboard.json`。
  - 完成的樣子：CI 在每次 push 跑 `go run ./cmd/claimcheck -strict` 且結束碼 0；第 5 章口播不唸行號並重算；配音來源有書面授權或是真人錄音；README 有一節說明怎麼查證主張。
  - 工作鍵：newhand31/go-shop:feat/plain-followups

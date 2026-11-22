# 主張清單 / Claims

影片口播裡每一句「這個 repo 是這樣」的話，都在這裡有一條編號。每一條附上檢查方法，任何人都能自己跑。

Every statement the video scripts make about this repo has a numbered entry here. Each entry carries a check that anyone can run.

口播的逐字稿在 `docs/transcripts/`，每一段的行尾標了它對應的主張編號。逐字稿由影片分鏡產生，不是手寫的。

The transcripts are in `docs/transcripts/`; each paragraph ends with the ids of the claims it relies on. They are generated from the video storyboard, not written by hand.

## 怎麼跑 / How to run

```
make claims
```

沒有 `make` 時，跑 `go run ./cmd/claimcheck`。只查一章：`go run ./cmd/claimcheck docs/claims/ch05.json`。

Without `make`, run `go run ./cmd/claimcheck`. One chapter only: `go run ./cmd/claimcheck docs/claims/ch05.json`.

檢查器會把該章的 tag 取出到暫存目錄，在裡面執行指令，跑完刪掉。你的工作目錄不會被改動。

The checker checks out the chapter's tag into a temporary directory, runs the commands there, and removes it. Your working tree is not touched.

## 結果怎麼讀 / Reading the result

| 狀態 Status | 意思 Meaning |
|---|---|
| `PASS` | 這條主張的每個檢查都通過。 Every check of the claim passed. |
| `FAIL` | 至少一個檢查沒通過。口播或程式有一邊錯了。 At least one check failed: the script or the code is wrong. |
| `SKIP` | 檢查需要 Docker，而 Docker 沒在執行。這條**沒有被驗證**。 The check needs Docker and Docker is not running. The claim was **not verified**. |
| `REF` | 說明用的對照（例如 C# 或 Java 的寫法）。只附來源連結，機器不檢查。 A comparison for explanation (C# or Java). It lists a source and is not machine-checked. |

`-strict` 會把 `SKIP` 當成失敗：`go run ./cmd/claimcheck -strict`。

`-strict` treats `SKIP` as a failure.

## 檔案格式 / File format

一章一個檔 `chNN.json`。 One file per chapter.

| 欄位 Field | 說明 Meaning |
|---|---|
| `chapter` | 章節代號，例如 `ch05`。 Chapter id. |
| `tag` | 這一章對應的 git tag。所有檢查都對這個 tag 的程式碼做。 The git tag every check runs against. |
| `claims[].id` | 編號。`C5-1` 是第 5 章第 1 條可檢查的主張，`X5-1` 是第 5 章第 1 條說明用對照。 `C` = checked claim, `X` = reference only. |
| `claims[].kind` | `repo`（機器檢查）或 `reference`（只附來源）。 |
| `claims[].claim` / `claim_en` | 主張本文，中文與英文。 The claim, in Chinese and English. |
| `claims[].checks[]` | 檢查方法，全部通過才算 `PASS`。 All must pass. |
| `claims[].sources[]` | `reference` 主張的來源連結。 Sources of a reference claim. |

檢查有三種。 Three check types:

| `type` | 做什麼 What it does |
|---|---|
| `contains` | 該 tag 的 `file` 含有 `text` 這段字。 `file` at the tag contains `text`. |
| `matches` | 該 tag 的 `file` 符合 `pattern` 這個正規表示式。 `file` at the tag matches the regular expression `pattern`. |
| `run` | 在該 tag 的暫存目錄執行 `cmd`。`exit` 是 `zero`（預設）或 `nonzero`；`output_contains` 是輸出必須含有的字；`edit` 會在執行前先改一處程式碼，用來示範「改掉這裡就編譯失敗」；`requires: ["docker"]` 表示需要 Docker。 Runs `cmd` in the checkout. `edit` changes one piece of code first, to show what breaks without it. |

`cmd` 裡的 `{root}` 代表該 tag 歷史上的第一個 commit，檢查器執行前會換成它的編號。講「起點」的主張用它，不寫死 commit 編號。 In `cmd`, `{root}` stands for the first commit in the tag's history; the checker fills it in before running. Claims about the starting point use it instead of a fixed commit id.

## 範圍 / Scope

主張清單只保證「口播說的事，在這個 tag 的程式碼上成立」。它不保證做法是唯一正解，也不檢查 `reference` 類的主張。

The list guarantees only that what the script says holds for the code at that tag. It does not claim the approach is the only correct one, and it does not check `reference` claims.

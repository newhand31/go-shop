# go-shop｜用 Go 寫實戰購物網站

**English** · [繁體中文](#繁體中文)

Companion repo for a YouTube series: a production-style e-commerce backend in Go, for C#/Java engineers moving to Go. Each episode starts from a production incident, fixes it with one Go concept, and proves the fix with a test.

One tag per episode: `git checkout ep01` … `ep12`, plus `ep13-pitfalls`. Tags never move.

## Run

Needs Go and Docker.

```bash
docker compose --profile app up --build    # whole shop on http://localhost:8080
curl http://localhost:8080/products
```

Or run the API from source:

```bash
docker compose up -d --wait
DATABASE_URL="postgres://goshop:goshop@localhost:5432/goshop?sslmode=disable" REDIS_URL="redis://localhost:6379/0" ECPAY_SANDBOX=1 go run ./cmd/api
```

Ports taken? Set `POSTGRES_PORT` / `REDIS_PORT` / `API_PORT`. Credentials are local throwaway defaults; `ECPAY_SANDBOX=1` uses ECPay's public test credentials.

## Test

```bash
go test ./...                     # unit
go test -tags=integration ./...   # integration (testcontainers, needs Docker)
```

Codegen: `make sqlc` after editing `db/queries/*.sql`; `make api` after editing `api/openapi.yaml`.

## Check what the videos say

Every statement the narration makes about this repo is a numbered claim in [docs/claims/](docs/claims/), and each claim carries a check you can run yourself.

```bash
go run ./cmd/claimcheck -strict                        # every chapter
go run ./cmd/claimcheck -strict docs/claims/ch05.json  # one chapter
```

Needs Go, git and a running Docker. The checker checks out each chapter's tag into a temporary directory and runs the checks there; your working tree is not touched.

- `PASS`: every check of the claim passed. `FAIL`: the narration or the code is wrong.
- `SKIP`: the check needs Docker and Docker is not running, so the claim was not verified. `-strict` counts it as a failure.
- `REF`: a comparison with C# or Java. It lists an official source and is not machine-checked.

To trace one sentence: open the transcript in [docs/transcripts/](docs/transcripts/), read the claim ids at the end of the paragraph (for example `C5-3`), and find that id in `docs/claims/ch05.json`. CI runs the strict check on every push ([workflow](.github/workflows/claims.yml)). File format and scope: [docs/claims/README.md](docs/claims/README.md).

## Docs

- [docs/episodes.md](docs/episodes.md): episode list
- [docs/walkthrough.md](docs/walkthrough.md): per-episode commands and proofs
- [docs/architecture.md](docs/architecture.md): stack and layering

License: code MIT ([LICENSE](LICENSE)); `docs/` CC BY-NC 4.0 ([docs/LICENSE.md](docs/LICENSE.md)).

---

## 繁體中文

繁中 YouTube 教學系列的配套 repo：用 Go 寫實戰電商後端，給想轉 Go 的 C#／Java 工程師。每集從一個線上事故開始，用一個 Go 概念修好，再用測試證明。

每集一個 tag：`git checkout ep01` … `ep12`，番外 `ep13-pitfalls`。tag 打了不移動。

## 執行

需要 Go 和 Docker。

```bash
docker compose --profile app up --build    # 整個商店，http://localhost:8080
curl http://localhost:8080/products
```

或從原始碼跑 API：

```bash
docker compose up -d --wait
DATABASE_URL="postgres://goshop:goshop@localhost:5432/goshop?sslmode=disable" REDIS_URL="redis://localhost:6379/0" ECPAY_SANDBOX=1 go run ./cmd/api
```

port 被佔用？設 `POSTGRES_PORT`／`REDIS_PORT`／`API_PORT`。帳密是本機預設值；`ECPAY_SANDBOX=1` 用綠界公開的測試帳密。

## 測試

```bash
go test ./...                     # 單元測試
go test -tags=integration ./...   # 整合測試（testcontainers，需要 Docker）
```

產生碼：改 `db/queries/*.sql` 後跑 `make sqlc`；改 `api/openapi.yaml` 後跑 `make api`。

## 查證影片說的話

口播裡每一句關於這個 repo 的話，都是 [docs/claims/](docs/claims/) 裡一條有編號的主張。每一條附上檢查方法，你可以自己跑。

```bash
go run ./cmd/claimcheck -strict                        # 全部章節
go run ./cmd/claimcheck -strict docs/claims/ch05.json  # 只查一章
```

需要 Go、git，以及執行中的 Docker。檢查器把該章的 tag 取出到暫存目錄，在裡面跑檢查；你的工作目錄不會被改動。

- `PASS`：這條主張的每個檢查都通過。`FAIL`：口播或程式有一邊錯了。
- `SKIP`：檢查需要 Docker，而 Docker 沒在執行，這條沒有被驗證。`-strict` 把它算成失敗。
- `REF`：與 C# 或 Java 的對照。只附官方來源，機器不檢查。

想追一句話的出處：打開 [docs/transcripts/](docs/transcripts/) 的逐字稿，看那一段行尾的主張編號（例如 `C5-3`），再到 `docs/claims/ch05.json` 找這個編號。CI 在每次 push 跑嚴格檢查（[workflow](.github/workflows/claims.yml)）。檔案格式與範圍見 [docs/claims/README.md](docs/claims/README.md)。

## 文件

- [docs/episodes.md](docs/episodes.md)：分集表
- [docs/walkthrough.md](docs/walkthrough.md)：逐集操作與證明
- [docs/architecture.md](docs/architecture.md)：技術棧與分層

授權：程式碼 MIT（[LICENSE](LICENSE)）；`docs/` 為 CC BY-NC 4.0（[docs/LICENSE.md](docs/LICENSE.md)）。

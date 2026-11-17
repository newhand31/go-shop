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

## 文件

- [docs/episodes.md](docs/episodes.md)：分集表
- [docs/walkthrough.md](docs/walkthrough.md)：逐集操作與證明
- [docs/architecture.md](docs/architecture.md)：技術棧與分層

授權：程式碼 MIT（[LICENSE](LICENSE)）；`docs/` 為 CC BY-NC 4.0（[docs/LICENSE.md](docs/LICENSE.md)）。

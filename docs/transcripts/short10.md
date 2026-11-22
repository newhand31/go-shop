# 雙 11 打爆資料庫（short10）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep10`
- 主張清單：`docs/claims/ch10.json`（執行 `make claims` 逐條檢查）
- 片長：00:44
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 雙 11 打爆資料庫。
- `00:02` 事故情境是這樣。搶購開始後，資料庫連線全滿，結帳請求逾時。這是假設的情境。
- `00:12` 原因是結帳的每個請求，都直接呼叫資料庫。修法第一步，先在 Redis 預扣庫存。賣完的人，不進資料庫。〔C10-13、C10-2、C10-3〕
- `00:23` 第二步，channel 有容量，再加 select default。放不進去，就回傳 ErrBusy。〔C10-5〕
- `00:31` 測試證明：200 個人同時買，有人被拒絕。成交加上被拒絕，剛好 200。〔C10-12〕
- `00:39` 完整說明在長片第 10 章。

## 這份逐字稿引用的主張

- **C10-2**　Redis 預扣用一支 Lua 腳本：腳本裡先 `GET` 計數器，不夠就回傳 0，夠就 `DECRBY`，回傳 1；計數器不存在回傳 -1。`Reserve` 收到 -1 時回傳 `ErrSaleNotOpen`，收到 1 時才算預扣成功。Redis 一次執行一支腳本，檢查和扣除之間沒有別的指令。測試 `TestReservations_Integration_ReserveIsAtomic` 讓 50 個 client 搶 5 個單位，剛好 5 個成功。
- **C10-3**　Redis 計數器是閘門，不是真相：訂單成不成立由資料庫的 `Place` 決定。`Buy` 先呼叫 `Reserve`，賣完就回 `ErrOutOfStock`，不碰資料庫。資料庫失敗時，`Release` 用 `INCRBY` 把預扣的數量還回去。測試 `TestFlashSale_DatabaseFailureGivesTheUnitBack` 證明資料庫失敗後預扣有還回去。
- **C10-5**　`Buy` 把工作放進 channel 用的是 `select` 加 `default`：放得進去就放，放不進去就執行 `default`，還回預扣並回傳 `ErrBusy`。`internal/service/flashsale.go` 裡沒有單獨一行 `f.queue <- job`。
- **C10-12**　庫存 300、2 個 worker、5 個排隊位置、200 個人同時買：有人得到 `ErrBusy`，成交加上被拒絕剛好是 200，Redis 計數器剛好等於庫存減訂單數，被拒絕的人都把預扣還回去。測試 `TestFlashSale_Integration_FullQueueRejectsAndKeepsTheCounterHonest` 證明。
- **C10-13**　tag `ep09` 的 `Checkout` 直接呼叫 `s.store.Place` 寫入資料庫，沒有預扣，也沒有排隊：整個 `*.go` 裡沒有 `Reserve`，`internal/service/order.go` 裡沒有 `chan`。這一章才加上預扣與排隊。

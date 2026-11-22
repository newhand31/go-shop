# 購物車換一台機器就空了（short05）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep05`
- 主張清單：`docs/claims/ch05.json`（執行 `make claims` 逐條檢查）
- 片長：00:45
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 購物車換一台機器就空了。
- `00:03` 客人放了三件商品進購物車。我們部署新版之後，客人的購物車是空的。
- `00:11` 原因是購物車存在 API 程式的記憶體裡。兩台 API 各有一份購物車，客人換到第二台，就看到空的購物車。〔C5-6〕
- `00:22` 修法是把 session 和購物車存進 Redis。兩台 API 共用同一個 Redis，換機器之後購物車還在。〔C5-7〕
- `00:31` 購物車邏輯只依賴這個介面。Go 的型別有這三個方法，就滿足介面，不用寫 implements。〔C5-1、C5-2〕
- `00:40` 完整的程式碼和測試，在長片的第 5 章。

## 這份逐字稿引用的主張

- **C5-1**　`CartStore` 介面宣告在使用它的 `service` 套件，只列三個方法：`Items`、`SetItem`、`RemoveItem`。
- **C5-2**　`repository` 套件沒有 import `service` 套件，`repository.Carts` 沒有任何「實作 `CartStore`」的宣告，仍然滿足 `service.CartStore`。`cmd/api/main.go` 有一行編譯期檢查，整個專案編譯通過。
- **C5-6**　兩台 API 各自把購物車存在行程記憶體時，同一個瀏覽器換到第二台，購物車是空的。測試 `TestCartIsLostWhenStateLivesInProcessMemory` 重現這個事故。
- **C5-7**　兩台 API 共用同一個 Redis 時，在第一台放進購物車的商品，第二台看得到；另一個瀏覽器看不到這台購物車。測試 `TestCartSurvivesSwitchingMachinesWithRedis` 證明。

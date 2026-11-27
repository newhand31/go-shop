# The cart is empty after a restart (short05) transcript

<!-- GENERATED from the video storyboard. Do not edit by hand; change the storyboard and regenerate. -->

- Code: tag `ep05`
- Claims list: `docs/claims/ch05.json` (run `make claims` to check each one)
- Length: 00:37
- Each line is one segment: start time in the video first, claim ids last. The Chinese line under it is the sentence-by-sentence translation.

- `00:00` A customer puts a keyboard in the cart. We restart the API, and the cart is empty.
  - 客人把一個鍵盤放進購物車。我們重新啟動 API，購物車就空了。
- `00:05` The cart lived in the memory of the API. Two API instances kept two separate carts. The customer landed on the second one. [C5-6]
  - 購物車存在 API 的記憶體裡。兩台 API 各有一份購物車。客人剛好連到第二台。
- `00:14` We put the cart in Redis. Two API instances now share one Redis. An item added on the first instance shows up on the second. [C5-7]
  - 我們把購物車放進 Redis。現在兩台 API 共用同一個 Redis。在第一台加入的商品，在第二台也看得到。
- `00:23` The cart logic needs only three methods. They are Items, SetItem, and RemoveItem. A Go type that has them fits, with no implements line. [C5-1, C5-2]
  - 購物車邏輯只需要三個方法。它們是 Items、SetItem、RemoveItem。有這三個方法的 Go 型別就吻合，不用寫 implements。
- `00:33` The full story is in chapter 5 of the long video.
  - 完整的故事，在長片的第 5 章。

## Claims quoted in this transcript

- **C5-1**  `CartStore` 介面宣告在使用它的 `service` 套件，只列三個方法：`Items`、`SetItem`、`RemoveItem`。
- **C5-2**  `repository` 套件沒有 import `service` 套件，`repository.Carts` 沒有任何「實作 `CartStore`」的宣告，仍然滿足 `service.CartStore`。`cmd/api/main.go` 有一行編譯期檢查，整個專案編譯通過。
- **C5-6**  兩台 API 各自把購物車存在行程記憶體時，同一個瀏覽器換到第二台，購物車是空的。測試 `TestCartIsLostWhenStateLivesInProcessMemory` 重現這個事故。
- **C5-7**  兩台 API 共用同一個 Redis 時，在第一台放進購物車的商品，第二台看得到；另一個瀏覽器看不到這台購物車。測試 `TestCartSurvivesSwitchingMachinesWithRedis` 證明。

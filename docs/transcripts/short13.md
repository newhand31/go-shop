# 番外　踩坑合集（short13）逐字稿

<!-- GENERATED：由影片分鏡產生，不要手改。要改口播，改分鏡後重新產生。 -->

- 程式碼：tag `ep13-pitfalls`
- 主張清單：`docs/claims/ch13.json`（執行 `make claims` 逐條檢查）
- 片長：00:32
- 每一行是一段口播。行首是這一段在影片裡的開始時間，行尾是這一段對應的主張編號。

- `00:00` 踩坑合集。
- `00:02` 函式沒有錯誤要回報，呼叫端的 if err != nil 卻成立。
- `00:08` 原因是 interface 有型別與值。型別是 NotFoundError 指標、值是 nil，這個 interface 就不是 nil。〔C13-3、X13-1〕
- `00:17` 修法是成功的路徑直接寫 return nil。〔C13-2〕
- `00:21` 測試證明修好了。另外兩個坑是迴圈變數與 slice 共用底層陣列。〔C13-3〕
- `00:28` 完整說明在長片第 13 章。

## 這份逐字稿引用的主張

- **C13-2**　`findBuggy` 宣告 `var err *NotFoundError`，沒找到商品時才指定值，最後一律 `return err`，所以商品 1 會回傳一個裝著 nil 指標的 `error`。`findFixed` 在沒有錯誤時寫 `return nil`。
- **C13-3**　測試 `TestNilPointerInsideAnErrorIsNotNil` 證明：`findBuggy(1)` 回傳的 `error` 不是 nil，裡面裝的是 nil 的 `*NotFoundError`；`findFixed(1)` 回傳真正的 nil。
- **X13-1**（說明用對照，機器不檢查）　Go 官方資料：interface 值是（型別，值）一對，裝著 nil 指標的 interface 不是 nil（Go FAQ）。Go 1.22 起，`for` 迴圈每一圈建立新的變數；是否採用新規則，由 `go.mod` 的 `go` 版本決定，也可以用 `//go:build` 行逐檔控制（Go 1.22 發行說明）。

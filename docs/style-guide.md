# Go 實戰教學影片｜幽默文案風格指南（v0）

核心觀察：好笑的技術教學，八成不是自創笑話，而是「嚴肅分析中突然插入一個生活化的反差」。

## 1. 手法

| # | 手法 | 機制 | Go 示範句 |
|---|---|---|---|
| 1 | 荒謬自貶 | 拿自己當最離譜的實例 | 我一個人開 4 個 goroutine 搶同一件庫存，貢獻 4 筆訂單。你覺得我很開心嗎？庫存只有 1 件。 |
| 2 | 降級排比 | 一串「本來要…現在只能…」 | 本來要賣 100 件，現在賣出 101 件；本來要鎖一列，現在鎖整張表。 |
| 3 | 諧音玩名字 | 專有名詞念歪 | （名詞諧音，用在 Go 關鍵字或套件名） |
| 4 | 借權威的嘴 | 引標題／名言講重話 | Rob Pike 說：用溝通來共享記憶體。我們的庫存就是沒聽進去才超賣。 |
| 5 | 自拆偽邏輯 | 正經推一個歪理，再說「好像有道理哦」拆掉 | 在 map 外面包一層 sleep，競態就消失了——好像有道理哦——不對，這叫把炸彈藏起來。 |
| 6 | 寓言收束 | 插一個短故事，最後一句硬轉回論點 | 有隻 goroutine 叼著鎖，狐狸叫它開口說話……這就是忘了 defer Unlock。 |
| 7 | 視覺災難比喻 | 把圖表形狀換成畫面 | 這張 CPU 圖就是聖誕樹，每個 goroutine 都掛一顆燈在等鎖。 |
| 8 | 反問拉觀眾 | 替觀眾回答，製造共謀感 | 換成 channel 之後再也不超賣，這樣的程式難道不吸引你嗎？ |

## 2. 每集骨架（10–15 分鐘）

1. 0:00 事故重現（一個小笑點）
2. 時間錨點開場：「現在是雙 11 晚上 8:31，庫存剩 1 件。」（每集固定句式，是品牌）
3. 2–4 分：C#／Java 會怎麼寫，為什麼在 Go 行不通
4. 4–9 分：Go 修法 2–3 段，用「好那我們來看」切段
5. 9–12 分：前後對比、壓測或測試證明
6. 最後 1 分：「第一、第二、第三」三點收尾，加一句固定道別

密度：大笑點每 3–4 分鐘 1 個，放在技術細節密集段之後；小吐槽每分鐘 1 句。

## 3. 語氣

- 句首「那」「好那」「所以」承接；句尾「哦」「嘛」軟化。
- 第一人稱複數「我們」，把觀眾拉到同一邊。
- 先讓步再翻案：「當然……可是……」。
- 自問自答：「這說明什麼？說明……」。

## 4. 不用的笑點

- 性暗示。
- 政治人物與政治笑話。
- 性別、種族、族群刻板印象。
- 真實個人的私生活八卦。
- 嘲諷受害者（被裁員、背鍋的人）。

替代素材：事故、框架、舊版程式、自己當年的 bug、語言之間的差異。

## 5. 腳本硬規則（可直接貼進 prompt）

1. 笑點只能對著程式、事故、我自己，不能對著人。禁止性暗示、政治、刻板印象、真人八卦。
2. 固定骨架：事故重現 → 時間錨點開場 → C#／Java 錯誤寫法 → Go 修法 2–3 段 → 三點收尾。
3. 每 3–4 分鐘一個大笑點，笑完下一句必須連回技術結論。
4. 比喻要視覺化，而且對應真實現象（鎖爭用、記憶體、延遲），不用抽象誇飾。
5. 口語、「我們」、先讓步再翻案；每段至少一次反問。

## 6. 英文版

英文腳本不直譯，用同一個事故重寫笑點。手法 3（諧音）換成英文雙關；其餘七個手法照用。時間錨點與道別各自另訂英文固定句。

### 6.1 固定句（品牌，每集照用）

- **時間錨點**：`It's <時刻>, <離事故多遠>, and <什麼東西著火了>.`
  例：`It's 11:48 p.m., nine hours before launch, and our shop has exactly zero endpoints.`
  規則：時刻用 12 小時制加 a.m./p.m.（英語觀眾的直覺），第二段永遠是倒數或事故後經過的時間，第三段是一個可量的壞狀態（數字優先）。
- **道別**：`Ship it small, ship it tested. See you at the next incident.`
  前一句是本頻道的教學主張，後一句接下一集的事故預告；兩句都不改字。

### 6.2 手法 3 改成英文雙關

中文的諧音靠同音字；英文改用「Go 的關鍵字或套件名剛好也是日常英文」的雙關。做法：

1. 只拿 Go 的關鍵字、內建函式、標準庫或本集用到的套件名下手（`defer`、`panic`、`go`、`range`、`slog`、`chi`、`context`）。
2. 雙關要靠「聽」就懂：配音或 AI 語音唸出來也成立，不靠拼字、大小寫或只在字幕看得到的梗。
3. 說一次就走，不解釋、不重複；一集最多兩個。
4. 說完下一句接回技術點（同 §5 規則 3）。

例句：

- `panic`：「Go has a built-in called `panic`. It's the one time in this whole series I'll tell you: don't panic. You return an error instead.」
- `slog`：「We log with `slog`. Structured logging. Not to be confused with tonight, which is also a slog, but unstructured.」
- `defer`：「`defer` is Go's way of saying 'I'll do it later', and unlike me, it actually does.」

### 6.3 其他七個手法的英文 Go 示範

| # | 手法 | 英文 Go 示範句 |
|---|---|---|
| 1 | 荒謬自貶 | I started four goroutines to buy the last keyboard. All four succeeded. We had one keyboard. |
| 2 | 降級排比 | We planned to sell 100 units; we sold 101. We planned to lock one row; we locked the whole table. We planned to go home; we did not. |
| 4 | 借權威的嘴 | The Go proverb says: don't communicate by sharing memory, share memory by communicating. Our inventory map heard the first half and stopped listening. |
| 5 | 自拆偽邏輯 | If I add a `time.Sleep` before the map write, the race goes away. Sounds reasonable. It isn't. I just hid the race behind a curtain. |
| 6 | 寓言收束 | A goroutine held a mutex in its beak. A fox begged it to sing. It never opened its beak, so it never let go, and every other goroutine waited in that tree forever. Moral: `defer mu.Unlock()`. |
| 7 | 視覺災難比喻 | This CPU graph is a Christmas tree: every goroutine hangs one light on it, and every light is waiting on the same lock. |
| 8 | 反問拉觀眾 | One channel, zero oversells, no lock in sight. Who wouldn't want to be on call for that code at 3 a.m.? |

### 6.4 英文版語氣與對照

- 語氣：短句、主動句、第一人稱複數 `we`；先讓步再翻案用 `Sure, ... But ...`；自問自答用 `What does that tell us? It tells us ...`。
- 語言對照以 Java（Spring Boot）為主、C#（ASP.NET Core）為輔：先講 Spring 的寫法，C# 只在差異明顯時補一句。
- 腳本要同時能給真人錄與 AI 配音：口播句不放括號舞台指示（笑聲、停頓寫進【畫面】或 `[pause]`），程式碼識別字在口播裡照常唸，需要的話在附錄註明發音（例：`chi` 唸 /kaɪ/ 或 /tʃiː/ 擇一後固定）。

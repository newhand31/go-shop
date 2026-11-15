# Bonus episode | Three Go traps that compile cleanly (voice-over draft v0, English channel)

- Target length: about 11 minutes (about 1,350 spoken words (roughly 8 minutes) plus screen-only beats and code on screen; re-time after recording)
- The topic: **three things that look right, compile, and are wrong.** One: a nil pointer inside an interface is not a nil interface (compare Java's `null`, C#'s `null`). Two: the loop variable captured by a closure, before and after Go 1.22 (compare closures in Java lambdas and C# lambdas). Three: slices share their underlying array (compare `ArrayList.subList` and `List<T>.GetRange`).
- Code: tag `ep13-pitfalls` (`git checkout ep13-pitfalls`); the examples live in `internal/pitfalls` and nothing in the shop imports them. The tests are the lesson.
- Rewritten for the English channel per [style-guide.md](../../style-guide.md) §6, not translated from the [Chinese script](../ep13-pitfalls.md). Comparisons lead with Java (Spring Boot), C# (ASP.NET Core) second.
- 【畫面】 lines are editor cues and are never read aloud. `[pause]` is a beat for the voice, human or AI.
- Laugh map (rule 3: a big laugh every 3–4 minutes): 0:00 / 2:30 / 4:30 / 6:30 / 8:15 / 9:30 / 10:45; the longest gap is 2:30 and every laugh sits in the section it is listed under.

---

## 0:00 Cold open (the incident)

【畫面】A test file. `if err != nil { return err }`. The terminal prints the error: `<nil>`.

> A bug report landed this afternoon. It says: the error is not nil, and the error says nil.
> **We returned a 500 whose explanation was "nothing went wrong."** The system was very upset about it.
> That is not a joke. It is the most famous trap in Go, and it is only the first of three tonight.

【畫面】Title card.

---

## 0:50 Time anchor

> It's 11:20 p.m., ten hours before launch, and code review just found three places that look correct and are not.
> What they share: the compiler is happy, most unit tests stay green, and each one is an instinct you brought from another language.
> This is a bonus episode. No new features. We defuse three mines.

---

## 1:40 What Java and C# taught you

【畫面】Three panels. Java: `String s = null; if (s == null)`. Java: `items.forEach(x -> tasks.add(() -> x));`. Java: `List<Integer> head = cart.subList(0, 2); head.add(99);`

> Java taught you that `null` is one thing. You ask `s == null`, you get one answer.
> It taught you that a lambda captures a value. The variable is effectively final, the compiler checks it for you, and nothing surprising can happen.
> And it taught you that a list view and a list copy are different words, and you read the Javadoc to find out which one you got.
> Sure, those instincts work in Java. But Go keeps all three traps open, and says nothing. The compiler will not mention them. **That is not a bug in you. It is a very long day of Java talking.**

---

## 2:30 Trap one: a nil pointer inside an interface is not nil

【畫面】`internal/pitfalls/nilerror.go`.

> Here is a function:

```go
func findBuggy(id int) error {
	var err *NotFoundError
	if id != 1 {
		err = &NotFoundError{ProductID: id}
	}
	return err
}
```

> When the product exists, `err` is still a nil pointer, and we return it. The caller writes `if err != nil` and gets **true**.
> Why? Because an interface value in Go is a box with two slots: a type, and a pointer. We put in the type `*NotFoundError` and a nil pointer. The box is not empty. **It holds a very confident nothing.** A truly nil interface has both slots empty.
> **Think of an empty lunch box in the fridge. Someone asks "is there lunch?" and you answer "there's a box." True. Useless.**
> The fix is boring. When there is no error, say `return nil`, with the literal.

```go
func findFixed(id int) error {
	if id != 1 {
		return &NotFoundError{ProductID: id}
	}
	return nil
}
```

> One rule: **a function that returns `error` returns the literal `nil` on success, never a variable of a concrete type.** The moment a typed pointer becomes an interface, it carries a type with it.
> This trap loves custom error types. You write a `validate()` that returns `*ValidationError`; when all is well you return that variable. The caller receives it as `error`, and you have just walked into it.
> When in doubt, print the value with `%T` and with `%v`. `%T` says `*pitfalls.NotFoundError` and `%v` says `<nil>`. **A type with no value is the fingerprint of this trap.**
> Java and C# do not have this problem, because their `null` is one value. Go has two: "no box" and "empty box."

---

## 4:30 Trap two: the loop variable (before and after Go 1.22)

【畫面】`loopvar_new.go` and `loopvar_old.go` side by side.

> Two files, the same loop:

```go
for i := 0; i < n; i++ {
	fns = append(fns, func() int { return i })
}
```

> Run it three times, collect three closures, call them. What prints?
> On Go 1.22 and later: 0, 1, 2. On Go 1.21 and earlier: **3, 3, 3.**
> The old rule: one `i` for the whole loop, and all three closures hold the same one. The loop ends when `i` reaches 3, so everyone reports 3.
> **My first goroutine loop did this. I started ten workers and all ten processed the last order. Ten workers, one order, and the dashboard looked busy.**
> Since Go 1.22 every iteration gets its own `i`. This repo's `go.mod` says Go 1.27, so you are safe by default.
> The more common version uses goroutines: `for _, o := range orders { go process(o) }`. Before 1.22, ten goroutines often all received the last order. The charm of that era was `o := o` inside the loop, or passing `o` as an argument.
> One detail people miss: the new rule follows the `go 1.22` (or later) line in `go.mod`. Upgrade the toolchain to 1.27 without touching `go.mod` and nothing changes. **Upgrading the compiler is not upgrading the language version.**
> Why bring it up? Two reasons. You will read older code, and `i := i` on its own line is a charm from that era. And **the language version is per file.** `go.mod` sets the default; a `//go:build go1.21` line at the top of one file puts that file back under the old rules. That is how this repo shows both behaviours side by side.
> For the Java readers: a lambda cannot capture a changing `i` at all, the compiler refuses. Go lets you, and so you have to know the rule. And C# readers: C# changed `foreach` to a fresh variable per iteration long ago; its three-part `for` still shares one. Go 1.22 changed the `for`.

---

## 6:30 Trap three: slices share the underlying array

【畫面】`slices.go`. Below it, four boxes: 10, 20, 30, 40, with a frame around the first two.

> Last one. A cart with four items. Take the first two, add one more:

```go
func firstBuggy(cart []int, n, extra int) []int {
	head := cart[:n]
	return append(head, extra)
}
```

> The result is 10, 20, 99. Fine. But the cart is now **10, 20, 99, 40.** Item three was overwritten.
> A slice is not an array. It is a window: a pointer, a length, a capacity. `cart[:2]` is another window onto the same array, length 2, **capacity 4.** `append` sees room and writes into it.
> **It's like borrowing a coworker's monitor to look at two lines, then typing on line three. Their file changed.**
> The fix is to cap the window with a third index:

```go
head := cart[:n:n]
```

> Capacity is now 2. `append` finds no room, allocates a new array and copies. The cart is untouched. Or call `slices.Clone`, which says what you mean.
> The nastier version: you never call `append`, but you hand `head` to another function, and that function does. You cannot tell from its signature whether it writes.
> So before a public function returns a sub-slice, ask whether it should be a `slices.Clone`. And if all you do is read, do not copy: a window is cheap exactly because it does not copy. **The price of cheap is knowing who owns the memory.**
> Java's `subList` is a view too, and it bites differently: `head.add(99)` writes through to the parent list, inserting at index 2 and shifting the rest, where Go silently overwrites item three. C#'s `GetRange` hands you a copy, so there you are safe. Go gives you the window by default. **If you want a copy, you have to say so.**

---

## 8:15 How to catch them before the incident

> Can review catch these? Let's be honest about it.
> Trap one: `go vet` stays quiet; we ran it on this package and it printed nothing. Extra linters can catch some shapes of it, with no guarantee. **The reliable defence is the rule: success means `return nil`.**
> Trap two: upgrade to 1.22 and it is gone. `go vet` also warns about the old shape for goroutines and `defer`.
> Trap three: no tool. You just have to ask, before every `append`: whose capacity is this?
> Or you skip memorising and write a test. Tests are memory that someone else maintains.
> One review habit helps: when the first argument of an `append` is not a slice you just created, ask "whose is this?" It takes five seconds, and the incident review takes an afternoon.

---

## 9:30 Proof: six tests

【畫面】`go test ./internal/pitfalls -v`, six green PASS lines.

> The proof tonight is not a load test. It is six small tests, grouped by trap: each trap is made to fire once, and the fix is checked.
> First pair: `findBuggy(1)` returns a non-nil error holding a nil `*NotFoundError`. `findFixed(1)` returns a real nil.
> Second pair: `counterNew(3)` returns 0, 1, 2. `counterOld(3)` returns 3, 3, 3. The second one lives in the file we pinned to Go 1.21.
> Third pair: after `firstBuggy` the cart reads 10, 20, 99, 40. After `firstFixed` it is unchanged.
> **We left the broken versions in the repo on purpose.** You only learn to spot a trap after you have watched it go off.

---

## 10:45 Three takeaways

> One. An interface is a two-slot box: a type and a pointer. On success, `return nil`.
> Two. Since Go 1.22 every iteration has its own loop variable, and the language version is per file. `i := i` is history, not superstition.
> Three. A slice is a window, not a copy. Before `append`, ask whose capacity it is, and use `[:n:n]` or `slices.Clone` when you need to be alone.
> None of these is a defect in Go. It is Go choosing cheap over safe. If you want safe, you buy it yourself.
> **We defused three mines tonight, and learned that the scariest bugs are the ones the compiler signs off on.**
> Ship it small, ship it tested. See you at the next incident.

---

## Appendix A: hard-rule self-check (tick before recording)

- [ ] Rule 1: jokes aim at code, incidents, and me (this script's targets: an error that says nothing went wrong, an empty lunch box, ten workers on one order, borrowing a monitor, a compiler that signs off). No sexual content, politics, stereotypes or real-person gossip.
- [ ] Rule 2: skeleton complete (cold open 0:00 → time anchor 0:50 → Java/C# instincts 1:40 → three traps 2:30 / 4:30 / 6:30 → catching them 8:15 → proof 9:30 → three takeaways 10:45)
- [ ] Rule 3: big laughs at 0:00 (the error says nothing went wrong), 2:30 (a confident nothing, the lunch box), 4:30 (ten workers, one order), 6:30 (typing on a borrowed monitor), 8:15 (tests are memory someone else maintains), 9:30 (broken versions left in on purpose), 10:45 (the compiler signs off); longest gap 2:30; each laugh is followed by the technical conclusion
- [ ] Rule 4: metaphors map to real mechanics (empty lunch box = typed nil pointer in an interface; borrowed monitor = shared backing array; ten workers on one order = all closures sharing one `i` before 1.22)
- [ ] Rule 5: spoken, "we", concede then overturn (`Sure, those instincts work in Java. But ...`); questions: "What prints?" "So why doesn't this bite in Java?"
- [ ] Style-guide §6.2: no pun this episode; an English pun on `range` or `go` can be added at the loop section if wanted (max two, say it once).

## Appendix B: to confirm

- Laugh timestamps are estimates; re-time against the real recording. About 1,350 spoken words (code excluded), roughly 8 minutes at the series pace; about 3 minutes come from screen and code beats. If the recording runs short, move the later timestamps earlier.
- "`go vet` stays quiet": on this machine `go vet` printed nothing for `internal/pitfalls`. "Extra linters can catch some shapes of it" was not tested with any third-party linter; run one before naming a tool on screen.
- The Java `subList` write-through statement (`add` inserts into the parent and shifts the rest) and the Java lambda statement ("the compiler refuses a capture of a changing variable") and the C# one (`foreach` fresh variable per iteration since C# 5, `for` still shared) are from memory; check both against the official docs before recording.
- Per-file language version: confirmed locally. `counterOld(3)` returns 3, 3, 3 in the file with `//go:build go1.21`, and `counterNew(3)` returns 0, 1, 2.
- `slices.Clone` is standard library (Go 1.21 and later). No new dependencies, no new configuration.
- The sign-off is the fixed line from style-guide §6.1.

## Appendix C: reproduce

```powershell
git checkout ep13-pitfalls
go test ./internal/pitfalls -v
```

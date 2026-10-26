# Task 01: make it compile

**Goal:** `./check.sh` prints `OK`.

**Context:** `main.go` and `cards.go` were written without ever being compiled, against an older Ebiten than the v2.10.5 in `go.mod`. Expect a few small errors.

**Do:**
1. Run `go mod tidy` once.
2. Run `./check.sh`.
3. Fix the first error only, with the smallest edit. Run `./check.sh` again. Repeat.
4. For any error about an Ebiten function, run `go doc github.com/hajimehoshi/ebiten/v2 <Name>` (or the subpackage, for example `.../ebitenutil`, `.../inpututil`) and use what it says.

**Do not:** change game behavior, rename things, restructure files, or run `go get -u`.

**Done when:** `./check.sh` prints `OK`. In your report, list each error and how you fixed it.

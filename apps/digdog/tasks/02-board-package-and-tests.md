# Task 02: board package and tests

**Goal:** move the game rules into a package that does not import Ebiten, so `go test` works on a machine with no graphics libraries. Medium difficulty: if you are stuck after two attempts, stop and report.

**Use the module path from `go.mod`** for imports (it ends in `/apps/digdog` or `/digdog`, whichever is there now).

**Do, in this order, running `./check.sh` after each step:**

1. Create `internal/board/board.go` (`package board`). Move these from `cards.go`: `CardState` with `Buried`, `Revealed`, `Matched`; `Result` with its constants; `Card`; `Board`; `missTicks`; `Pick`; `Reveal`; `Update`; `allMatched`. Keep `Moves` and `Won` as exported fields. The package must not import ebiten, image, or color.
2. Add to the package: `NewBoardFromFaces(cols, rows int, faces []int) *Board` (panics if `len(faces) != cols*rows`), and make `NewBoard` shuffle a deck and call it. Add `Cols() int`, `Rows() int`, `Len() int`, and `Card(i int) Card`.
3. Rename `cards.go` to `draw.go` (`git mv`). Keep `cellW`, `cellH`, `cardW`, `cardH`, `faceColors`, `fillRect` there. Replace the old `Board` methods that depend on screen size with plain functions in `package main`: `boardOrigin(b *board.Board) (int, int)`, `cellAt(b *board.Board, x, y int) int`, `cellCenter(b *board.Board, i int) (float64, float64)`, `drawBoard(screen *ebiten.Image, b *board.Board)`. Same logic as before, using `b.Cols()`, `b.Rows()`, `b.Card(i)`.
4. Update `main.go`: import the board package; `g.board` becomes `*board.Board`; `NewBoard` becomes `board.NewBoard`; `g.board.CellAt(x, y)` becomes `cellAt(g.board, x, y)`; `g.board.CellCenter(i)` becomes `cellCenter(g.board, i)`; `g.board.Draw(screen)` becomes `drawBoard(screen, g.board)`; result constants become `board.ResultMatch` and so on.
5. Create `internal/board/board_test.go` using `NewBoardFromFaces(2, 2, []int{0, 1, 0, 1})`. Test:
   - first pick then reveal returns `ResultFirst` and unlocks the board
   - revealing a matching pair returns `ResultMatch` and both cards become `Matched`
   - revealing a mismatched pair returns `ResultMiss`, the board stays locked, and after `missTicks` calls to `Update` both cards are `Buried` and the board is unlocked
   - `Pick` returns false for an out-of-range index, an already revealed card, and when locked
   - matching the last pair returns `ResultWin` and sets `Won`

**Do not:** change game behavior or add dependencies.

**Done when:** `./check.sh` prints `OK` and it ran the board tests.

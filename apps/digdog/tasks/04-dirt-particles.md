# Task 04: dirt particles while digging

**Goal:** little brown dirt specks fly up from the mound while the dog digs.

**Read:** `main.go` and the file that defines `fillRect` (`cards.go` or `draw.go`).

**Do:**
1. Create `particles.go` (`package main`). Type `Particle` with `x, y, vx, vy float64`, `life int`, `col color.RGBA`. Type `Particles` as a slice of `Particle`, with methods:
   - `Spawn(x, y float64)`: adds one particle with a small random sideways speed (-1 to 1), an upward speed (-2.5 to -1), life 20 to 30 ticks, and a brown color picked from two shades. Cap the total at 200.
   - `Update()`: moves each particle, adds 0.15 to `vy` each tick (gravity), subtracts 1 from `life`, and removes dead ones.
   - `Draw(screen *ebiten.Image)`: draws each as a 2x2 square with the existing `fillRect` helper, using rounded integer positions.
2. In `Game`, add a `particles Particles` field. In `Update`, call `g.particles.Update()`. While `g.digging` is true, call `g.particles.Spawn(...)` twice per tick at the mound: `g.board.CellCenter(g.target)` (or the `cellCenter` function if task 02 is done), moved down 6 pixels.
3. In `Draw`, draw the particles after the board and before the dog.
4. In `reset()`, clear the particles.

**Do not:** add dependencies or sound. Use `math/rand` from the standard library.

**Done when:** `./check.sh` prints `OK`. Say plainly that the look is untested. A human will check it in a browser.

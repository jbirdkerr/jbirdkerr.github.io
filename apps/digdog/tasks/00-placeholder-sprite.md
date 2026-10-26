# Task 00: placeholder sprite

**Goal:** `main.go` embeds `assets/dog.png`, which does not exist yet, so nothing compiles. Create a placeholder.

**Do:**
1. Create `tools/genplaceholder/main.go` (`package main`, standard library only: `image`, `image/color`, `image/png`, `os`).
2. It writes `assets/dog.png`, creating the `assets/` folder if needed.
3. The image is 128x160 pixels: 4 columns by 5 rows of 32x32 frames.
4. Rows and frame counts (leave unused cells fully transparent):
   - row 0 idle, 4 frames, brown
   - row 1 dig, 4 frames, orange
   - row 2 sniff, 3 frames, green
   - row 3 happy, 3 frames, yellow
   - row 4 sad, 2 frames, blue
5. Each used frame: fill with the row color, draw a 1-pixel darker border, and draw (frame index + 1) small white 4x4 squares in a line near the top-left, so frames can be told apart.

**Do not:** touch `main.go`, `cards.go`, or `go.mod`.

**Run:** from `apps/digdog`: `go run ./tools/genplaceholder`

**Done when:** `file assets/dog.png` reports `128 x 160`.

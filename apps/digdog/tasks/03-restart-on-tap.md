# Task 03: tap to restart after winning

**Goal:** after the last pair is matched, a click or tap anywhere restarts the game. Today restarting needs the `R` key, which touch users do not have.

**Read:** `main.go` only.

**Do:**
1. In `Game.Update`, if `g.board.Won` and `pointerPos()` reports a click or tap, call `g.reset()` and return.
2. Change the win message in `Game.Draw` to end with `tap or click to play again`. Keep `press R` out of it, but keep the `R` key working.

**Do not:** change `Board`, the animations, or how picking works.

**Done when:** `./check.sh` prints `OK`. Say plainly that you could not test the click and the win flow. A human will check it in a browser.

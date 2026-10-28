# DigDog 🐶🃏

DigDog is a retro 32-bit-era pixel-art card-matching game where a dog walks to dirt mounds, digs up cards, and matches pairs. Written in Go using **Ebitengine v2**, compiled to WebAssembly, and hosted statically on Cloudflare at [digdog.jbirdkerr.net](https://digdog.jbirdkerr.net).

---

## Gameplay

- Click, tap, or navigate with the keyboard to select dirt mounds.
- Henry the dog walks to the mound, digs it up with flying dirt particles, and reveals the card.
- Dig up two matching cards to clear them from the board. Mismatched pairs stay revealed briefly before being re-buried.
- Consecutive matches build a **combo streak multiplier** (`streak: xN`).
- Use the **Sniff power-up** to temporarily reveal a random buried card.
- When all pairs are matched, press Space, click, or tap anywhere to start a new shuffled round!

---

## Controls

| Action | Keyboard | Mouse / Touch |
| --- | --- | --- |
| **Move Focus** | Arrow keys / `W`, `A`, `S`, `D` | Hover / Click / Tap |
| **Dig Card** | `Space` / `Enter` | Click or tap a mound |
| **Sniff Power-up** | `S` | Click / tap the `[S] SNIFF` HUD button |
| **Toggle Mute** | `M` | Click / tap the `[M] AUDIO` HUD button |
| **Toggle Particles** | `P` | Automatically respects `prefers-reduced-motion` |
| **Restart Game** | `R` (or `Space` after winning) | Click or tap anywhere when won |

---

## Architecture & Code Layout

- **`internal/board/`**: Pure Go rules engine (`Board`, `Card`, `Pick`, `Reveal`, `Update`, `Streak`). Has zero graphics or external dependencies and is 100% testable via standard `go test`.
- **`draw.go`**: Pixel rendering, targeting reticle brackets, card backgrounds, and accessible high-contrast geometric face shapes (diamond, cross, box, triangle, heart, circle) plus dot pips for players with color-vision differences.
- **`audio.go`**: Procedural waveform synthesis for 8-bit sound effects (dig scratch, card flip whoosh, match chime, miss bonk, and win fanfare). Initializes strictly after the first user gesture to comply with browser autoplay policies.
- **`particles.go`**: Particle emitter and physics simulation for dirt flying while digging.
- **`motion_js.go` / `motion_other.go`**: WebAssembly bridge detecting system `prefers-reduced-motion` via `syscall/js`.
- **`web/`**: HTML host page configured for crisp nearest-neighbor integer scaling, mobile responsive layout, and WebAssembly loader.
- **`tools/genplaceholder/`**: Utility generating the 128×160 sprite sheet placeholder.

---

## Local Development & Commands

Run all checks from the `apps/digdog/` directory:

```bash
# Verify formatting, wasm vet, wasm build, and board unit tests
./check.sh

# Compile WebAssembly binary (outputs web/digdog.wasm)
./build-wasm.sh

# Run board unit tests natively
go test -v ./internal/board/...

# Serve locally in browser
cd web && python3 -m http.server 8080
# Open http://localhost:8080
```

---

## Deployment

The game builds in GitHub Actions via `.github/workflows/digdog.yml`:
1. Triggers only on changes to `apps/digdog/**` or the workflow file.
2. Pins Go version directly from `apps/digdog/go.mod`.
3. Compiles `digdog.wasm` and copies the matching `wasm_exec.js` from `$(go env GOROOT)`.
4. Deploys the static `apps/digdog/web/` directory to Cloudflare Pages at `digdog.jbirdkerr.net` using Wrangler.

---

## Licensing & Privacy

- **License:** Licensed under [GNU Affero General Public License v3.0 (AGPL-3.0)](../../LICENSE).
- **Privacy:** DigDog contains zero analytics, trackers, telemetry, external network requests, or third-party cookies.

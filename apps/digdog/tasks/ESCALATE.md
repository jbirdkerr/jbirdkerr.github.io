# Not for the small model

Bring these to a stronger model, or review its output very carefully. They are easy to get subtly wrong and hard to check without a browser or real accounts.

- **Crisp scaling in the browser:** integer scaling and what `Layout` should return. Needs the Ebiten docs and visual testing.
- **Audio:** the Ebiten audio package, synthesized sounds, starting audio only after a user gesture, and a mute toggle.
- **Deployment:** the GitHub Actions workflow, Wrangler config, path filters, and Cloudflare custom domain. Secrets and DNS belong to the owner.
- **Real art:** choices about the sprite sheet, palette, and card faces.
- **Keyboard play and accessibility:** cursor movement, focus indicator, reduced motion.
- **New game rules:** sniff power-up, combo streak, levels, buried bones. These change `Board` and the turn flow together.
- **Any bug you cannot fix in two attempts.**

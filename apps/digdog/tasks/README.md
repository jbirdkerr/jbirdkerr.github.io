# Tasks for a small local model

Work through these in order. One task per session.

## Before each task

1. `git status` is clean, or commit what is there.
2. Start a fresh agent session from `apps/digdog/` so stale context is gone.

## Prompt to paste

> Read AGENTS.md and tasks/NN-name.md. Do only that task. Use the edit tool for existing files. When finished, run ./check.sh and tell me which files you changed and what you could not test.

## After each task

1. Run `./check.sh` yourself. It must print `OK`.
2. Read `git diff`. Look for deleted code you did not expect.
3. Commit. If the result is wrong after two attempts, `git checkout .` and either tighten the task text or move the job to a stronger model.

## Order

| Task | What | Difficulty |
| --- | --- | --- |
| 00 | Generate placeholder `assets/dog.png` | Easy |
| 01 | Make the code compile | Easy to medium |
| 02 | Move board rules into a testable package, add tests | Medium |
| 03 | Tap to restart after winning | Easy |
| 04 | Dirt particles while digging | Medium |

Things that need a human in a browser: how the game looks and feels. Nothing here can check that, so play it after tasks 01, 03, and 04.

See `ESCALATE.md` for work that should not go to a small model.

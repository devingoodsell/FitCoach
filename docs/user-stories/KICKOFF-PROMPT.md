# Claude Code — Kickoff Prompt

Paste the block below into Claude Code from the root of the FitCoach repo. It scaffolds the monorepo and builds **Wave 0 (Foundations)**. Use `BUILD-PLAN.md` for the prompts that follow each subsequent wave.

---

```
You are helping me build an Android + backend "AI Health Coach" app. Read these first, in full:
- docs/user-stories/README.md
- docs/user-stories/v1-mvp.md
- docs/user-stories/BUILD-PLAN.md
- docs/fitness-coach-product-spec.md   (the product spec)

(If user-stories/ and the spec aren't under docs/ yet, move them there as your first step.)

STACK (locked — do not change without asking me):
- Client: Android native, Kotlin + Jetpack Compose, first-class Health Connect.
- Backend: Go + MySQL. The backend holds the Claude API key, assembles Coach Memory into
  prompts, calls Claude, runs a deterministic safety layer, and persists data.
- Monorepo layout: /android, /backend, /docs.

YOUR TASK NOW — Wave 0 (Foundations) only. Implement epics E15, E1, E3 for the MVP:
- E15: Go service skeleton, MySQL schema + migrations, config/secrets handling (Claude API key
  server-side ONLY, never in the client or repo), structured logging, health check, CI.
- E1:  email/password signup, login, logout, password reset, health-data consent capture,
  account+data deletion, multi-device session restore. (Stories E1-S1..S6.)
- E3:  a durable, versioned Coach Memory store in MySQL (profile, goals, schedule, preferences,
  locations, injuries, diet, workout logs, coach notes) with a deterministic prompt-assembly
  function (stubbed model call is fine for now). (Stories E3-S1..S3.)

HOW TO WORK:
1. First produce a short implementation plan and the proposed MySQL schema + API contract for
   these epics. Pause and let me review before you write feature code.
2. Then implement backend first, with unit tests on handlers; scaffold the Android app with the
   auth + consent screens wired to the backend.
3. Reference the story IDs you satisfy in commits/PRs (e.g. "E1-S1, E3-S1").
4. Do NOT build Wave 1+ epics (onboarding, readiness, injuries, engine, etc.) yet.

CONSTRAINTS:
- Secrets never reach the client or the repo.
- Coach Memory schema must be versioned so it can evolve without data loss.
- Account deletion must actually remove backend + local data.

Definition of done for Wave 0: I can sign up and log in; the backend reads/writes a versioned
Coach Memory record tied to my account; consent and account-deletion flows work; CI is green.

Start by reading the docs and giving me the plan + schema + API contract.
```

---

## After Wave 0

Once foundations are green, open **Wave 1** epics (E2, E4, E7, E9, E13, E11) — each in its own `git worktree` and Claude Code session so they run in parallel. Use the per-wave prompt templates in `BUILD-PLAN.md` section 6. Agree the E13↔E7 contraindication interface before starting either.

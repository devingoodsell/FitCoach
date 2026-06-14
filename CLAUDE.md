# CLAUDE.md — Working agreement & repo map for FitCoach

This file is the **first thing to read** before touching the codebase. It tells you where
everything lives so you **navigate to existing files instead of recreating them**, and it sets
the engineering standards every change must meet. Keep it accurate: if you move or add a
top-level package, update the map below in the same PR.

---

## 1. What this project is

FitCoach is an **AI Health Coach**: an Android app + Go backend that generates an injury-aware,
age-aware, recovery-driven workout session when the user taps "Start workout," explains its
reasoning, and runs fully offline once cached. A single concept ties it together — **Coach
Memory**, a durable structured store the model reads on every decision.

- Product spec: [`docs/fitness-coach-product-spec.md`](docs/fitness-coach-product-spec.md)
- User stories & epics: [`docs/user-stories/`](docs/user-stories/)
- Build sequencing (waves, dependencies): [`docs/user-stories/BUILD-PLAN.md`](docs/user-stories/BUILD-PLAN.md)
- Per-epic PR breakdowns: [`docs/epics/`](docs/epics/) — one file per epic (`E1.md` … `E15.md`)

## 2. Stack (LOCKED — do not change without asking the owner)

- **Client:** Android native, **Kotlin + Jetpack Compose**, first-class **Health Connect**.
- **Backend:** **Go + MySQL**. Holds the **Claude/Anthropic API key**, assembles Coach Memory
  into prompts, calls Claude, runs the **deterministic safety layer**, persists data.
- **Repo:** monorepo — `/android`, `/backend`, `/docs`.

Non-negotiable rules:
- **The API key is server-side only.** Never put it in the Android app, in client config, in
  tests, or in the repo. The client calls **our backend**, never Anthropic directly.
- **Coach Memory is versioned** so its schema can evolve without data loss.
- **Account deletion really deletes** backend + local data.
- **Offline-first:** any in-session feature (logging, timers, rest countdowns, autoregulation)
  must work with connectivity disabled.

## 3. Repository map — navigate here before creating files

```
FitCoach/
├── CLAUDE.md                  ← you are here (repo map + working agreement)
├── README.md                  ← project overview
├── .env.example               ← template for backend secrets (real .env is git-ignored)
├── .gitignore
│
├── android/                   ← Android app (Kotlin + Compose)
│   └── app/src/
│       ├── main/
│       │   ├── java/pro/d11l/fitcoach/
│       │   │   ├── core/          shared infra used by every feature
│       │   │   │   ├── network/     Retrofit/Ktor client → OUR backend (no Anthropic calls)
│       │   │   │   ├── auth/        token storage (Android Keystore), session state
│       │   │   │   ├── db/          Room database, DAOs, offline cache + sync queue
│       │   │   │   └── designsystem/  Compose theme, reusable components, disclaimers
│       │   │   ├── di/           manual dependency container + ViewModel factory
│       │   │   ├── feature/       one package per user-facing feature (screens + VMs)
│       │   │   │   ├── auth/         signup, login, password reset             (E1)
│       │   │   │   ├── consent/      health-data consent + manual-mode choice  (E1)
│       │   │   │   ├── onboarding/   adaptive onboarding wizard               (E2)
│       │   │   │   ├── readiness/    readiness display + explanation          (E4)
│       │   │   │   ├── injury/       injury entry/lifecycle/assist            (E7)
│       │   │   │   ├── location/     locations & current-context switch       (E9)
│       │   │   │   ├── session/      start/run workout, log, timers, rests    (E5/E6)
│       │   │   │   ├── diet/         daily targets & guidance                 (E11)
│       │   │   │   └── settings/     edit any user-model field, consent review (E14)
│       │   │   ├── data/          repositories mediating network ↔ Room cache
│       │   │   └── healthconnect/ Health Connect permissions + signal ingestion (E4)
│       │   └── res/             Android resources
│       ├── test/java/          JVM unit tests (ViewModels, readiness math, autoregulation)
│       └── androidTest/java/   instrumented/UI tests
│
├── backend/                   ← Go service + MySQL
│   ├── cmd/server/            main.go — wiring/entrypoint ONLY (no business logic)
│   ├── internal/              all backend logic (not importable outside this module)
│   │   ├── platform/          cross-cutting infra
│   │   │   ├── config/          env/secrets loading (Claude key lives here at runtime)
│   │   │   ├── db/              MySQL connection, tx helpers, migration runner
│   │   │   ├── httpx/           router, middleware, error/JSON helpers
│   │   │   ├── logging/         structured logging (redacts secrets & PII)
│   │   │   └── events/          generation/safety audit-event writer (redacted)  (E15-S3)
│   │   ├── auth/              accounts, sessions, password reset, deletion      (E1)
│   │   ├── consent/           health-data / disclaimer consent capture          (E1)
│   │   ├── disclaimer/        central versioned disclaimer copy (served)        (E13)
│   │   ├── memory/            Coach Memory store + prompt assembly              (E3)
│   │   ├── onboarding/        user-model capture/validation                    (E2)
│   │   ├── readiness/         readiness compute from raw signals               (E4)
│   │   ├── injury/            injury parsing & lifecycle                       (E7)
│   │   ├── location/          locations & equipment / current context         (E9)
│   │   ├── diet/              nutrition target computation                     (E11)
│   │   ├── coaching/          session generation: assemble → call Claude → cache (E5/E8)
│   │   └── safety/            DETERMINISTIC safety layer (no LLM)              (E7-S4/E13)
│   ├── migrations/            ordered, versioned SQL migrations
│   ├── api/                   API contract (OpenAPI) — source of truth for client ↔ server
│   └── (go.mod created in Wave 0)
│
└── docs/
    ├── fitness-coach-product-spec.md
    ├── user-stories/          README.md, v1-mvp.md, v1.1-full-release.md, BUILD-PLAN.md, KICKOFF-PROMPT.md
    └── epics/                 E1.md … E15.md — PR-sized task breakdown per epic
```

> Empty package dirs currently hold a `.gitkeep`; delete it when you add the first real file.

### Quick "where does X go?" guide
- **A new backend endpoint** → handler in the relevant `internal/<domain>/`, route registered via
  `internal/platform/httpx`, wired in `cmd/server`. Update `backend/api/` (the contract) too.
- **A new DB table/column** → a new file in `backend/migrations/` (never edit a shipped migration);
  touch Coach Memory? bump its schema version (see [`E3.md`](docs/epics/E3.md)).
- **A new Android screen** → a package under `feature/`; shared widgets go in `core/designsystem`.
- **Anything that calls Claude** → `backend/internal/coaching` only. The client never does.
- **A safety/contraindication rule** → `backend/internal/safety` (deterministic, unit-tested).

## 4. Engineering hygiene (apply to every change)

**Small PRs.** One coherent slice per PR — ideally one story or sub-task from the epic files.
If a PR touches two epics or grows past a few hundred meaningful lines, split it. Each PR cites
the **story IDs** it satisfies (e.g. "implements E5-S1, E5-S3").

**TDD.** Write the failing test first, then the code to pass it, then refactor. Especially
mandatory for: the `safety` layer (prove unsafe plans are corrected/rejected), readiness math,
autoregulation, and Coach Memory prompt assembly. No untested handler or business rule merges.

**DRY.** Look for an existing helper before writing a new one — the map above tells you where to
look. Shared backend concerns live in `internal/platform`; shared UI in `core/`. Don't copy a
function between packages; lift it to a shared home. **Never recreate a file that already
exists** — search first (`rg`, the repo map), edit in place.

**Idiomatic to the language.**
- *Go:* standard project layout (`cmd/` + `internal/`), small interfaces defined by the consumer,
  `error` returned and wrapped with `%w`, `context.Context` first arg on anything I/O-bound,
  `gofmt`/`go vet` clean, table-driven tests. No global mutable state.
- *Kotlin/Compose:* unidirectional data flow (state down, events up), `ViewModel` + immutable UI
  state, coroutines/`Flow` for async, repositories as the single source of truth, stateless
  composables where possible, `ktlint`/`detekt` clean.

**Other standing rules.**
- API-contract-first: fix/extend `backend/api/` at the start of a wave so backend & Android build
  independently against it.
- Secrets never reach the client or the repo (see §2).
- Disclaimer language ("guidance, not medical advice") appears wherever the body/health is
  involved and is centrally managed — see [`E13.md`](docs/epics/E13.md).
- Conventional, imperative commit subjects; reference story IDs in the body.
- Run formatters/linters and the relevant test suite before declaring a change done; report real
  results — if a test fails or a step was skipped, say so.

## 5. How the work is sequenced

Build in **waves** (see [`BUILD-PLAN.md`](docs/user-stories/BUILD-PLAN.md)):
`Wave 0` foundations (E15→E1→E3) → `Wave 1` inputs (E2, E4, E7, E9, E13, E11) →
`Wave 2` engine (E5, E8) → `Wave 3` execution (E6, E12, E14) → `Wave 4` V1.1.
Critical path: **E15 → E1 → E3 → E5 → E6.** Don't start an epic before its dependencies are
functionally done. Run parallel epics in separate git worktrees so sessions don't collide.

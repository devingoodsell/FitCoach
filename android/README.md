# android — Kotlin + Jetpack Compose app

Native Android client. First-class Health Connect integration. Talks only to the FitCoach
backend (never to Anthropic). Sessions run fully offline once cached.

## Layout (`app/src/main/java/pro/d11l/fitcoach/`)

- `core/` — shared infra: `network` (client → our backend), `auth` (Keystore token storage),
  `db` (Room cache + offline sync queue), `designsystem` (Compose theme, reusable components,
  disclaimer surfaces).
- `feature/` — one package per user-facing feature: `auth`, `onboarding`, `readiness`, `injury`,
  `location`, `session`, `diet`, `settings`. (Feature → epic mapping in [`/CLAUDE.md`](../CLAUDE.md).)
- `data/` — repositories mediating network ↔ Room cache (single source of truth).
- `healthconnect/` — Health Connect permissions and recovery-signal ingestion.

## Conventions

Unidirectional data flow (state down, events up), `ViewModel` + immutable UI state,
coroutines/`Flow` for async, stateless composables where possible, `ktlint`/`detekt` clean.
JVM unit tests in `src/test`, instrumented/UI tests in `src/androidTest`. Any in-session feature
must be verified with connectivity disabled.

> The Gradle project, version catalog, and base theme are created in Wave 0 (E15/E1).

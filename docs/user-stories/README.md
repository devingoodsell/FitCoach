# AI Health Coach — User Stories & Epics

Backlog derived from the Product Spec (Draft v0.1, 2026-06-09). Platform: Android native (Kotlin + Jetpack Compose) plus a backend that holds the Claude API key and runs a deterministic safety layer.

## Files in this folder

| File | Contents |
|---|---|
| `README.md` | This index: epic map, version strategy, conventions, glossary |
| `v1-mvp.md` | V1 (MVP) epics and stories — the buildable core that delivers the wedge |
| `v1.1-full-release.md` | V1.1 (Full Release) epics and stories — the rest of the spec vision |

## Version strategy

The spec locks v1 priorities as: **age-aware (1), injury-aware (2), recovery/rest (3), Equinox classes (4)**, with lightweight diet guidance throughout. We split the full spec across two releases:

- **V1 (MVP)** proves the core loop: durable Coach Memory → readiness from Health Connect → an LLM-generated, injury-respecting, age-aware session at "Start workout" → in-session logging with timers and rest countdowns → a deterministic safety layer. Plus the account/auth and onboarding foundation needed to support it, and lightweight diet targets.
- **V1.1 (Full Release)** completes the vision: Equinox class ingestion and slotting, proactive temporal check-ins, multi-wearable support, advanced healthspan programming, periodization and plateau-driven re-planning, richer privacy controls, and platform observability.

Items deferred to V1.1 are the lower-priority (#4 Equinox) or polish/scale concerns that are not required to demonstrate value in session one. Note: the app's initial audience is the builder's friends and family, so injury identification assist is in the MVP (no legal-review gate); the "not a diagnosis" disclaimer is retained as good practice.

## Epic map

| Epic | Title | V1 | V1.1 |
|---|---|:--:|:--:|
| E1 | Authentication, Accounts & Privacy | ● | ● |
| E2 | Adaptive Onboarding & User Model | ● | ● |
| E3 | Coach Memory | ● | ● |
| E4 | Recovery & Readiness (Health Connect) | ● | ● |
| E5 | Coaching Engine & Session Generation | ● | ● |
| E6 | In-Session Experience | ● | ● |
| E7 | Injury & Condition Management | ● | ● |
| E8 | Healthy-Aging / Longevity Programming | ● | ● |
| E9 | Training Locations & Equipment | ● | ● |
| E10 | Classes (Equinox) | | ● |
| E11 | Diet & Nutrition Guidance | ● | ● |
| E12 | Offline, Sync & Caching | ● | ● |
| E13 | Safety & Compliance Layer | ● | ● |
| E14 | Settings & Profile Management | ● | ● |
| E15 | Backend Platform & Observability | ● | ● |

(● = that epic has stories in that release. Most epics span both: a core slice in MVP, enhancements in V1.1.)

## Conventions

**Story ID format:** `E<epic>-S<n>` (e.g. `E5-S3`). IDs are stable across releases; a story listed only once lives in the file for its release.

**Story shape:** Each story uses the form *As a [persona], I want [capability], so that [benefit]*, followed by acceptance criteria written as testable Given/When/Then or checklist statements.

**Personas:**

- **Trainee** — the end user (experienced lifter caring about performance and longevity; the model generalizes to novices and older users).
- **Returning user** — a Trainee with accumulated Coach Memory and history.
- **Coach (system)** — the LLM-in-the-loop coaching engine acting on the Trainee's behalf.
- **Admin / Operator** — the team running the backend (infra, safety, observability).

**Priority labels** inside each story: `[P0]` must-have for that release, `[P1]` should-have, `[P2]` nice-to-have.

## Glossary

- **Coach Memory** — durable, structured store (profile, goals, history, injuries, diet, coach notes) the model reads on every decision.
- **Readiness** — in-house score computed from raw Health Connect signals (HRV trend, resting HR trend, sleep duration/quality).
- **Autoregulation** — light on-device logic that adjusts the next set based on actual RPE/performance, without calling the LLM.
- **Safety layer** — deterministic, non-LLM validation between the model output and the user that enforces injury contraindications and load/volume bounds.
- **Healthspan / aging block** — programming for bone density, balance, joint/tendon resilience, and high-intensity cardiovascular capacity.
- **Health Connect** — Android's vendor-agnostic health data layer; our single ingestion point for recovery signals.

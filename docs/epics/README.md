# Epics → PR breakdown

One file per epic (`E1.md` … `E15.md`), breaking each epic's user stories into **small,
mergeable PRs**. Use these as the task list when implementing a wave. Source of truth for the
*stories* themselves is [`../user-stories/v1-mvp.md`](../user-stories/v1-mvp.md); source of truth
for *sequencing* is [`../user-stories/BUILD-PLAN.md`](../user-stories/BUILD-PLAN.md).

## Conventions

- **PR ID:** `E<epic>-PR<n>` (e.g. `E5-PR2`). Stable within an epic.
- **Track:** each PR is tagged `[backend]`, `[android]`, `[contract]`, or `[infra]`. Backend and
  Android tracks proceed in parallel once the API contract for the wave is fixed.
- **Each PR lists:** scope, the story IDs it advances, the key tests, and its done criteria.
- **Small by default:** a PR is one coherent slice. Prefer more, smaller PRs over a big one.
- **TDD / DRY / idiomatic:** see [`/CLAUDE.md`](../../CLAUDE.md) §4.

## Wave → epic index

| Wave | Epics | Files |
|------|-------|-------|
| 0 — Foundations | E15, E1, E3 | [E15](E15.md), [E1](E1.md), [E3](E3.md) |
| 1 — Inputs & capture | E2, E4, E7, E9, E13, E11 | [E2](E2.md), [E4](E4.md), [E7](E7.md), [E9](E9.md), [E13](E13.md), [E11](E11.md) |
| 2 — Coaching engine | E5, E8 | [E5](E5.md), [E8](E8.md) |
| 3 — Execution | E6, E12, E14 | [E6](E6.md), [E12](E12.md), [E14](E14.md) |
| 4 — V1.1 | E10 (+ enhancements) | [E10](E10.md) |

**Critical path:** E15 → E1 → E3 → E5 → E6.

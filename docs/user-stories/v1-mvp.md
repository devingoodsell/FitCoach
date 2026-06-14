# V1 (MVP) — User Stories

The buildable core that proves the wedge: durable Coach Memory feeding an LLM that generates an injury-respecting, age-aware, recovery-driven session at the moment the user taps "Start workout," validated by a deterministic safety layer, with in-session logging, timers, and rest countdowns. Plus the account, onboarding, location, and lightweight-diet foundation needed to support it.

See `README.md` for conventions, personas, and the epic map.

---

## E1 — Authentication, Accounts & Privacy

**Goal:** A secure account that anchors cloud-backed Coach Memory across devices, with clear consent for sensitive health data.

### E1-S1 — Create an account `[P0]`
As a **Trainee**, I want to sign up with email and password, so that my profile and history are saved to my account.

- Given valid email and a password meeting strength rules, when I submit, then an account is created and I am signed in.
- Email format and password strength are validated client-side with clear errors.
- Duplicate-email signups are rejected with a non-enumerating message.
- A verification email is sent; unverified accounts can use the app but are prompted to verify.

### E1-S2 — Log in and log out `[P0]`
As a **Trainee**, I want to log in and log out, so that I control access to my data on this device.

- Correct credentials authenticate and restore my session; incorrect credentials show a generic failure.
- Sessions persist across app restarts via secure token storage (Android Keystore-backed).
- Logging out clears local tokens and cached personal data on the device.
- Repeated failed attempts are rate-limited / backed off.

### E1-S3 — Reset a forgotten password `[P0]`
As a **Trainee**, I want to reset my password by email, so that I can recover access.

- Requesting a reset sends a time-limited, single-use link regardless of whether the email exists (no account enumeration).
- A successful reset invalidates existing sessions on other devices.

### E1-S4 — Consent to health data use `[P0]`
As a **Trainee**, I want a clear, explicit consent step before any health data is read or stored, so that I know what I'm sharing and why.

- Before first Health Connect read, I see plain-language disclosure of what is collected (sleep, RHR, HRV), how it's used (readiness, programming), and where it's stored.
- I must affirmatively accept; declining still lets me use manual-only mode.
- The app shows the medical-disclaimer language ("guidance, not medical advice") at consent and it is retrievable later.

### E1-S5 — Delete my account and data `[P0]`
As a **Trainee**, I want to delete my account and associated data, so that I can exercise control over my information.

- Initiating deletion requires confirmation and explains what is removed and the timeline.
- On completion, profile, Coach Memory, history, and ingested health data are deleted from backend storage; local data is wiped.
- I receive confirmation when deletion completes.

### E1-S6 — Stay signed in across my devices `[P0]`
As a **Returning user**, I want my account to work on a new device, so that my Coach Memory and history follow me.

- Logging in on a second device restores profile, Coach Memory, injuries, and history from the cloud.
- No user data is hard-coded; all state is loaded from the account.

---

## E2 — Adaptive Onboarding & User Model

**Goal:** Capture a usable user model quickly, expanding only when relevant, and make every field editable forever.

### E2-S1 — Complete short adaptive onboarding `[P0]`
As a new **Trainee**, I want a short onboarding that only asks deeper questions when relevant, so that I can start training quickly without an interrogation.

- Onboarding captures the minimum to generate session one: age/DOB, biological sex, experience level, goal weighting, schedule, and at least one location with equipment.
- Follow-up questions appear only when prior answers warrant (e.g. injuries flow appears only if I indicate something is bothering me).
- I can skip optional sections and complete them later.
- Estimated time to finish the core path is communicated up front.

### E2-S2 — Enter profile and physiology `[P0]`
As a **Trainee**, I want to enter age/DOB, biological sex, and optional height/weight, so that the coach can program for my body.

- Age can be entered as DOB (preferred) or age; the system derives age for programming.
- Height and weight are optional; weight, if entered, is stored with a timestamp so trends can be tracked later.

### E2-S3 — Set experience level `[P0]`
As a **Trainee**, I want to declare my training age and self-rated level (with optional benchmark lifts), so that the coach calibrates difficulty.

- I can provide training age and a self-rated level (e.g. novice/intermediate/advanced).
- Optionally I can enter a few benchmark lifts; these are stored in Coach Memory.

### E2-S4 — Weight my goals with sliders `[P0]`
As a **Trainee**, I want to weight goals across strength, healthspan, body composition, and performance using sliders, so that the plan reflects my real priorities rather than a single pick.

- Goal weights are captured as a distribution, not a single choice.
- Weights are stored in Coach Memory and feed every planning call.
- I can revisit and re-balance weights at any time.

### E2-S5 — Set my schedule `[P0]`
As a **Trainee**, I want to set days per week, session length, and preferred days/times, so that sessions fit my real life.

- I can specify weekly frequency, target session duration, and preferred training days/times.
- Schedule is stored as editable state and used in session generation.

### E2-S6 — Capture exercise and equipment preferences `[P1]`
As a **Trainee**, I want to record likes, dislikes, hard avoids, and equipment preferences, so that the coach respects what I will and won't do.

- I can mark exercises as preferred, disliked, or hard-avoid.
- Hard-avoids are treated as constraints in planning, distinct from soft dislikes.

### E2-S7 — Capture dietary preferences `[P0]`
As a **Trainee**, I want to record dietary preference (vegan, vegetarian, pescatarian, kosher, etc.) and any supplements/medications, so that diet guidance fits how I eat.

- I can select a dietary pattern and free-text notable supplements/medications.
- These are stored in Coach Memory and used by diet guidance (see E11).

### E2-S8 — Healthy-aging emphases defaulted from age and goals `[P1]`
As a **Trainee**, I want healthy-aging emphases pre-set sensibly from my age and goals but adjustable, so that longevity work starts reasonable without extra effort.

- Defaults for bone/balance, joint/tendon resilience, VO2max/high-intensity, and cardiovascular base are inferred from age and goal weights.
- I can adjust each emphasis; changes persist and feed programming.

---

## E3 — Coach Memory

**Goal:** A durable, structured store the model reads on every decision so it never starts from scratch.

### E3-S1 — Persist a structured Coach Memory `[P0]`
As the **Coach (system)**, I want a durable, structured store of the user's profile, goals, history, injuries, and diet, so that every decision is grounded in accumulated context.

- Coach Memory persists profile, goals, schedule, preferences, locations, injuries, diet, workout logs, and coach notes.
- Memory is cloud-backed and portable across devices (ties to E1-S6).
- The schema is versioned so it can evolve without data loss.

### E3-S2 — Assemble memory into every planning call `[P0]`
As the **Coach (system)**, I want to assemble relevant Coach Memory plus current state into the model prompt, so that the session reflects who the user is.

- Each generation call includes profile, goals, active/managed injuries, recent history, readiness, location/equipment, and relevant preferences.
- Assembly is deterministic and logged for debugging (without leaking secrets).

### E3-S3 — Record session outcomes back to memory `[P0]`
As a **Returning user**, I want my completed sessions recorded to memory, so that future sessions build on what I actually did.

- On session completion, logged exercises (reps, weight, rests, timed work) are written to workout logs in Coach Memory.
- Autoregulation events during the session are captured for later reasoning.

---

## E4 — Recovery & Readiness (Health Connect)

**Goal:** Ingest recovery signals through Health Connect and compute our own readiness, then feed it into programming.

### E4-S1 — Connect Health Connect `[P0]`
As a **Trainee**, I want to connect Android Health Connect, so that the coach can read my sleep, resting heart rate, and HRV.

- The app requests Health Connect permissions for sleep, RHR, and HRV with clear rationale.
- Works with Pixel Watch data at minimum for MVP.
- If permission is denied or no data exists, the app degrades gracefully to manual/no-readiness mode.

### E4-S2 — Compute an in-house readiness score `[P0]`
As the **Coach (system)**, I want to compute readiness from raw signals (overnight HRV trend, resting HR trend, sleep duration and quality), so that we are not locked to a vendor's score.

- Readiness is computed from raw Health Connect signals, not a vendor-provided score.
- The computation handles missing/partial data without crashing and flags low-confidence days.

### E4-S3 — See my readiness with an explanation `[P0]`
As a **Trainee**, I want to see today's readiness and a short explanation, so that I understand why today's session is lighter or harder.

- Readiness is displayed with a plain-language explanation of the main drivers (e.g. "HRV down vs. baseline, short sleep").
- The explanation avoids medical claims.

### E4-S4 — Readiness drives the session `[P0]`
As a **Trainee**, I want readiness to actually change my session, so that a bad night shows up as a smarter, lighter day automatically.

- Readiness is an input to volume, intensity, and exercise selection in the generation call.
- A poor-recovery morning is a re-planning trigger (ties to E5).

---

## E5 — Coaching Engine & Session Generation

**Goal:** Generate the full session at "Start workout," explain it, cache it for offline use, and re-plan on triggers.

### E5-S1 — Generate a full session on demand `[P0]`
As a **Trainee**, I want the coach to generate my full session when I tap "Start workout," so that today's plan reflects my current state in one step.

- One LLM call produces the session at session start, grounded in Coach Memory, readiness, plan position, and current location/equipment.
- Output includes warmup, main work with loads/reps and explicit rest periods, accessory work, and an aging/mobility block.
- The call is server-side (API key never on device).

### E5-S2 — See the coach's reasoning `[P0]`
As a **Trainee**, I want short plain-language reasoning for the key choices, so that I learn instead of just following.

- Each generated session includes brief reasoning for the main programming decisions (intensity, key exercise selection, any injury accommodation).
- Reasoning is concise and free of medical claims.

### E5-S3 — Cache the session for offline use `[P0]`
As a **Trainee**, I want my generated session cached locally, so that I can train fully even with no signal at the gym.

- Once generated, the entire session (all sets, rests, timers, reasoning) is available offline.
- Logging during the session works offline and syncs when connectivity returns (ties to E12).

### E5-S4 — On-device autoregulation `[P0]`
As a **Trainee**, I want the next set adjusted based on how the current set actually went, so that the session adapts without waiting on the server.

- Local logic adjusts the next set toward the target RPE based on logged performance.
- Autoregulation runs offline and requires no LLM call.

### E5-S5 — Re-plan on triggers `[P1]`
As a **Returning user**, I want the coach to re-reason when something material changes, so that the plan stays current.

- The LLM re-plans on the next session by default, or immediately on a trigger: new/changed injury, poor recovery morning, or a context/location change.
- Plateau-driven re-planning is deferred to V1.1.

---

## E6 — In-Session Experience

**Goal:** A clean execution surface: start, log, time, and rest with countdowns.

### E6-S1 — Start and run a workout `[P0]`
As a **Trainee**, I want to start the generated workout and move through it set by set, so that I can execute the plan.

- I can start the session and see the ordered list of warmup, main, accessory, and aging blocks.
- Current exercise, target reps/load, and rest are clearly shown.

### E6-S2 — Log reps and weight `[P0]`
As a **Trainee**, I want to log actual reps and weight per set, so that my real performance is recorded.

- I can record reps and weight for each set, defaulting to the prescribed targets.
- Logged values flow into autoregulation (E5-S4) and Coach Memory (E3-S3).

### E6-S3 — Rest countdown timers `[P0]`
As a **Trainee**, I want an automatic countdown between sets and exercises, so that I rest the prescribed amount.

- After logging a set, a rest countdown for the prescribed duration starts automatically where appropriate.
- I can skip, extend, or pause the rest timer.
- An audible/haptic cue fires at zero.

### E6-S4 — Timers for timed exercises `[P0]`
As a **Trainee**, I want a timer for timed work like planks, so that I hold for the prescribed duration.

- Timed exercises present a count-up or count-down timer with start/stop.
- Completed duration is logged like reps/weight.

### E6-S5 — Mark a session complete `[P0]`
As a **Trainee**, I want to finish and save the session, so that it's recorded and informs future plans.

- Completing the session writes all logged work to Coach Memory.
- Partial sessions can be saved; skipped work is recorded as skipped.

---

## E7 — Injury & Condition Management

**Goal:** Injuries as first-class, structured memory objects that constrain the plan, with a deterministic safety net.

### E7-S1 — Enter an injury in natural language `[P0]`
As a **Trainee**, I want to describe an injury in a freeform box and have it parsed into structure, so that I can tell the coach quickly without filling a form.

- A natural-language entry is parsed into structured slots: region, status, severity, aggravating movements, onset date, notes.
- I can review and correct the parsed result before saving.
- The entry is stored with its date in Coach Memory.

### E7-S2 — Manage injury lifecycle `[P0]`
As a **Trainee**, I want each injury to have a status I can change (active flare / managed / recurring-but-fine / resolved), so that my plan reflects how I'm doing now.

- I can add, edit, or remove injury entries at any time.
- Status changes update programming behavior immediately (a new/changed injury is a re-planning trigger).

### E7-S3 — Injuries constrain the plan `[P0]`
As a **Trainee**, I want active and managed conditions to change what's programmed, so that I train around them rather than into them.

- Active/managed conditions inject contraindications and substitution preferences into every planning call.
- The session reasoning notes any injury-driven substitutions in plain language.

### E7-S4 — Deterministic safety validation `[P0]`
As the **Admin / Operator**, I want a deterministic safety layer to validate every generated plan against hard contraindications and sane load/volume bounds, so that unsafe plans never reach the user regardless of model output.

- Validation runs server-side between the model and the user, independent of the LLM.
- Plans violating a hard injury contraindication or load/volume bound are corrected or regenerated, never shown as-is.
- Validation outcomes are logged for audit (ties to E13).

### E7-S5 — Injury identification assist `[P1]`
As a **Trainee** who doesn't know what's wrong, I want the coach to help narrow it down through guided questions, so that I can describe it usefully — while understanding it's not a diagnosis.

- A guided Q&A helps characterize the issue, always framed with explicit "not a diagnosis, consult a clinician" disclaimers.
- The resulting characterization is stored as a normal injury entry the user confirms before saving.
- The flow respects the same safety and disclaimer language as the rest of the injury model (ties to E13-S1).

---

## E8 — Healthy-Aging / Longevity Programming

**Goal:** Make healthspan work a first-class part of every session, not an afterthought.

### E8-S1 — Aging/mobility block in every session `[P0]`
As a **Trainee**, I want an aging/mobility block included in my session, so that I build the capacities that determine how I age.

- Each generated session includes a healthspan block targeting the user's emphases (bone/balance, joint/tendon, VO2max/high-intensity, cardio base) per their settings and age.
- The block respects active injuries and the safety layer.

### E8-S2 — Age-appropriate programming `[P0]`
As an older **Trainee**, I want my programming to differ from a 28-year-old's, so that the plan fits how my body actually responds.

- Age and healthy-aging emphases measurably influence volume, intensity, recovery, and exercise selection in generation.
- The reasoning surfaces at least one age-aware choice when relevant.

---

## E9 — Training Locations & Equipment

**Goal:** Multiple locations with distinct equipment, and a switchable current context.

### E9-S1 — Manage multiple locations `[P0]`
As a **Trainee**, I want to define multiple training locations each with its own equipment profile, so that sessions match where I actually am.

- I can create locations (e.g. Equinox, home, hotel, outdoors), each with an editable equipment list.
- Locations are stored in Coach Memory.

### E9-S2 — Switch current context `[P0]`
As a **Trainee**, I want to switch my current training context, so that the coach programs for today's equipment.

- I can set a current context (e.g. "traveling this week, hotel gym only").
- The active context's equipment constrains session generation; a context change is a re-planning trigger.

---

## E11 — Diet & Nutrition Guidance (Lightweight)

**Goal:** Simple, preference-aware daily targets and guidance tied to the day's training. No full food logging in MVP.

### E11-S1 — Daily nutrition targets `[P0]`
As a **Trainee**, I want simple daily targets (e.g. protein and calories) based on my profile and goals, so that I know roughly how to eat to support training.

- The app shows target ranges for calories and protein derived from age, sex, weight, goal weights, and the day's training load.
- Targets are framed as guidance with a clear non-medical-advice disclaimer.

### E11-S2 — Preference-aware guidance `[P0]`
As a **Trainee**, I want guidance that respects my dietary pattern, so that suggestions fit how I eat (vegan, vegetarian, pescatarian, kosher, etc.).

- Guidance reflects the dietary preference captured in onboarding (e.g. plant-based protein sources for a vegan).
- No suggestion violates a stated hard dietary constraint.

### E11-S3 — Guidance tied to today's workout `[P1]`
As a **Trainee**, I want a short note on how to eat for the remainder of the day given today's session, so that nutrition supports the work I did.

- After a session, the coach offers a brief, plain-language suggestion for the rest of the day (e.g. emphasize protein on a heavy day).
- Suggestion respects dietary preference and is non-prescriptive.

---

## E12 — Offline, Sync & Caching

**Goal:** Training works without connectivity; data syncs reliably when it returns.

### E12-S1 — Run a session fully offline `[P0]`
As a **Trainee**, I want the cached session and all in-session features to work offline, so that a dead signal at the gym never blocks me.

- A generated and cached session runs end-to-end (logging, timers, rest countdowns, autoregulation) with no connectivity.

### E12-S2 — Sync logged data when back online `[P0]`
As a **Trainee**, I want offline-logged sessions to sync automatically when connectivity returns, so that nothing is lost.

- Logged work queued offline syncs to the backend on reconnect.
- Sync is idempotent; reconnecting does not duplicate records.

---

## E13 — Safety & Compliance Layer

**Goal:** Honest, safe guidance with hard limits and clear disclaimers.

### E13-S1 — Disclaimers wherever the body is involved `[P0]`
As a **Trainee**, I want clear "guidance, not medical advice" disclaimers wherever health is involved, so that expectations are honest.

- Disclaimer language appears at onboarding, in injury flows, in readiness, and in diet guidance.
- Disclaimer text is centrally managed and retrievable in Settings.

### E13-S2 — Enforce load and volume bounds `[P0]`
As the **Admin / Operator**, I want sane load/volume bounds enforced independent of the model, so that no session exceeds safe limits.

- The safety layer caps load/volume against configurable bounds before delivery.
- Bound violations are logged (ties to E7-S4 and E15).

---

## E14 — Settings & Profile Management

**Goal:** Everything captured at onboarding is editable forever.

### E14-S1 — Edit all user-model fields `[P0]`
As a **Trainee**, I want to edit any part of my profile, goals, schedule, preferences, locations, diet, and aging emphases in Settings, so that the coach stays current as I change.

- Every onboarding field is editable post-onboarding.
- Edits persist to Coach Memory and take effect on the next planning call.

### E14-S2 — Review disclaimers and consent state `[P1]`
As a **Trainee**, I want to review the disclaimers I accepted and my health-data consent, so that I understand and can revisit what I agreed to.

- Settings shows current consent state and the medical disclaimer.
- I can revoke health-data consent, which disables ingestion and falls back to manual mode.

---

## E15 — Backend Platform & Observability

**Goal:** A secure, observable backend that holds the key, assembles prompts, calls Claude, validates, and persists.

### E15-S1 — Server-side model orchestration `[P0]`
As the **Admin / Operator**, I want the backend to hold the Claude API key and orchestrate generation, so that secrets never reach the client.

- The API key lives only server-side; the client calls our backend, not Anthropic directly.
- The backend assembles the prompt from Coach Memory + current state, calls the model, and runs the safety layer before returning a session.

### E15-S2 — Persist profile, memory, and history `[P0]`
As the **Admin / Operator**, I want durable cloud storage for profile, Coach Memory, and history, so that progress is portable and recoverable.

- Data is persisted reliably and backed up; restore is possible.
- Storage honors account deletion (E1-S5).

### E15-S3 — Generation and safety logging `[P1]`
As the **Admin / Operator**, I want generation and safety-validation events logged, so that we can debug quality and audit safety.

- Each generation logs inputs (redacted), latency, and outcome; each safety check logs pass/correct/regenerate.
- Logs exclude secrets and are access-controlled.

# Product Spec: AI Health Coach (working title)

> An adaptive strength and longevity coach for Android that programs, adjusts, and explains your training the way a good human coach would: around your age, your injuries, your recovery, and your real schedule.

**Status:** Draft v0.1
**Platform:** Android (native, Kotlin + Jetpack Compose)
**Owner:** (you)
**Last updated:** 2026-06-09

---

## 1. The problem

Today's best training apps are excellent *loggers* with a thin layer of progression logic. Fitbod and its peers track sets and nudge weight up over time, but they have no model of the user as a whole, aging human. Specifically, they fall short in four ways:

1. **No recovery awareness.** They do not look at how you slept, your HRV, or your accumulated fatigue before deciding how hard you should train today.
2. **No real injury handling.** There is no first-class way to tell the app "my sciatica is flaring" and have the entire plan adjust around it, then return to normal as you heal.
3. **No aging lens.** They program a 28-year-old and a 58-year-old roughly the same way. They ignore the things that actually determine whether you stay capable into your 60s and 70s: bone density, balance, joint and tendon resilience, and high-intensity cardiovascular capacity.
4. **No life context.** They cannot reason about the gym membership you already pay for (Equinox classes), so they cannot tell you when a mobility or conditioning class would serve you better than another lifting session.
5. **No diet suggestions.** They do not understand your diet or make recommendations on how to augment it givin your workout routine for the day. 

The gap is not "a better workout planner." The gap is a **coach**: something that reasons about *you* over time and adjusts.

---

## 2. What we are building

A coaching app, not a tracker. On any given day it answers: "Given who you are, what you eat, how you have been training, how recovered you are this morning, what is currently bothering your body, and where you can train today, what should you do, and why?"

It generates each session dynamically, explains its reasoning, adjusts in real time to how the session is going, respects active injuries, layers in the work that protects long-term healthspan, slots in your gym's classes when they serve the plan better than another lift, and make recommendations for how to eat for the remainder of the day. 

Underneath, a single concept ties it together: **Coach Memory**, a durable, structured store the coaching model reads on every decision. It holds your profile, your goals, your training history, your injuries, your diet, what you've eaten, and the coach's own notes. The model never starts from scratch; it always reasons from your accumulated context.

---

## 3. Value proposition

For people who want to train seriously *and* still be strong, mobile, and unbroken decades from now, this is a coach in your pocket that:

- **Adapts to today, not to a static template.** Bad night of sleep shows up as a lighter, smarter session, automatically.
- **Treats your body's history as memory.** Tell it once about your sciatica; it remembers, programs around it, and checks back in as you heal.
- **Coaches for the long game.** It deliberately builds the bone, balance, tendon, and cardiovascular capacity that determine how you age, not just how you look next month.
- **Fits your real life.** Configurable schedule, multiple training locations with different equipment, and awareness of the classes you already pay for.
- **Explains itself.** Every recommendation comes with a reason, so you learn instead of just following.

---

## 4. How we are different

| App / category | What it does well | What it misses that we deliver |
|---|---|---|
| **Fitbod** | Clean lifting log, auto-progression, exercise swaps | No recovery input, no injury model, no aging lens, no class/life context |
| **Whoop / Oura** | Excellent recovery and readiness data | Measures recovery but does not program training from it |
| **Future / Caliber (human coaches)** | Real human accountability and personalization | Expensive, not real-time or offline, not instantly adaptive mid-session |
| **Juggernaut AI / strength templates** | Strong periodization for powerlifting | Narrow goal, no aging or injury intelligence, no recovery loop |
| **Apple Fitness+ / class apps** | Good guided classes | No personalized programming or progression |

Our wedge: we sit at the intersection none of them occupy, **recovery-driven, injury-aware, age-aware programming, delivered by an LLM coach that reasons over durable memory and explains itself**, for a fraction of the cost of a human coach.

---

## 5. Target user

Primary: experienced trainees (roughly 5+ years lifting) who care about both performance and longevity, train at a full gym (often a premium one like Equinox), wear a recovery-capable device, and have the kind of recurring or age-related niggles that generic apps ignore.

The system is designed to **generalize across the full range of users** (novice to advanced, 20s to 60s+, healthy to managing multiple conditions). No user's specifics are hard-coded into the engine; everything is captured in the user model and editable.

---

## 6. Core principles

1. **Coach, not tracker.** Logging is table stakes; reasoning and guidance are the product.
2. **Memory-grounded.** The LLM reasons from durable Coach Memory on every call.
3. **Recovery-driven.** Readiness is an input to programming, not a separate dashboard.
4. **Injury-respecting.** Active conditions constrain the plan; a safety layer enforces hard limits.
5. **Age-aware by design.** Healthspan work is a first-class part of programming, not an afterthought.
6. **Configurable and general.** Schedule, locations, goals, and equipment are user-set state, captured at onboarding and editable forever.
7. **Resilient.** Sessions are generated then cached, so training works even with no signal at the gym.
8. **Vendor-agnostic.** We ingest through Android Health Connect and compute our own readiness, so we are not locked to one wearable.
9. **Honest and safe.** Guidance, not medical advice, with clear disclaimers wherever the body or health is involved.

---

## 7. User model (the data we keep)

All fields captured through **adaptive onboarding** (short, expands only when relevant) and editable in Settings.

**Profile / physiology**
- Age or date of birth, biological sex, optional height and weight (with bodyweight tracking over time)

**Experience**
- Training age, self-rated level and/or a few benchmark lifts

**Goals**
- Weighted across strength, healthspan, body composition, performance (sliders, not a single pick)

**Healthy-aging emphases**
- Bone density and balance, joint and tendon resilience, VO2max / high-intensity capacity, cardiovascular base (defaults inferred from age and goals, user-adjustable)

**Schedule**
- Days per week, session length, preferred days and times

**Workout Logs**
- Details of each workout session with type of exercise, length, and details of reps, weight, rests if appropriate  

**Locations**
- A list of training locations (e.g., Equinox, home, travel/hotel, outdoors), each with its own equipment profile; a "current context" the user can switch ("traveling this week, hotel gym only")

**Data sources**
- Connected wearable via Health Connect; readiness computed by us from raw signals

**Equinox / classes**
- Optional weekly class schedule the user enters; engine recommends slotting around training

**Preferences**
- Exercise likes/dislikes, hard avoids, equipment preferences

**Injuries / conditions** (see section 8)

**Coach notes / history**
- Session history, autoregulation events, plateaus, and the coach's own running notes

**Diet / Food**
- Diet preferences (vegan, vegetarian, pescetarian, kosher, etc), medications, supplements

---

## 8. Injury and condition model

Injuries are **first-class memory objects**, not a free-text note buried in a profile.

- **Guided freeform entry.** Structured slots exist (region, status, severity, aggravating movements, onset date, notes) but the user enters them through a natural language box; the system parses freeform input into structure and stores based on date.
- **Identification assist.** If the user does not know what they have, the coach can help narrow it down through guided questions, always framed with explicit disclaimers that this is not a diagnosis and a clinician should be consulted.
- **Lifecycle.** Each entry has a status (active flare / managed / recurring-but-fine / resolved) and can be added, edited, or removed at any time.
- **Temporal check-ins.** After an appropriate interval since last update, the coach proactively asks how the condition is doing and updates its status, the same way a human coach would remember and follow up.
- **Engine behavior.** Active and managed conditions inject contraindications and substitution preferences into every planning call. A deterministic safety layer validates each generated plan against hard contraindications before it ever reaches the user.

---

## 9. The coaching engine

**Decision-making:** LLM-in-the-loop. A server-side backend holds the Anthropic API key, assembles Coach Memory plus current state into the prompt, calls the model, and validates the result.

**When the LLM runs:** It generates the **full session at the moment the user taps "Start workout"** (one call), grounded in last night's readiness, current injuries, plan position, and today's location/equipment. The plan is **cached so the session runs fully offline.** Inside the session, light local logic handles autoregulation (hit the target RPE, adjust the next set). The LLM re-reasons on the next session, or immediately on a trigger: a new or changed injury, a poor recovery morning, a plateau, or a context change.

**Per-session inputs:** Coach Memory (profile, goals, injuries, history, notes) + computed readiness + plan history + current location and equipment + today's class options (if any).

**Per-session output:** warmup, main work with loads/reps and explicit **rest periods**, accessory work, an aging/mobility block, an optional class suggestion, and short plain-language reasoning for the key choices.

**Per-session interaction:** user can start the workout, log specific exercises with reps and weight, start a timer for timed exercises (such as planks), have countdowns for rests between reps and exercises when appropriate. 

**Safety layer:** deterministic validation between the model and the user that enforces hard injury contraindications and sane load/volume bounds, independent of what the model returns.

---

## 10. Recovery and readiness

- Ingest sleep, resting heart rate, and HRV through **Android Health Connect** (works with Pixel Watch today, Oura/Whoop/Garmin if added later).
- **Compute our own readiness score** from raw signals (overnight HRV trend, resting HR trend, sleep duration and quality) rather than depending on a locked vendor score.
- Readiness feeds directly into the day's session generation (volume, intensity, exercise selection), and is shown to the user with a short explanation.

---

## 11. Architecture summary

- **Client:** Android native, Kotlin + Jetpack Compose. First-class Health Connect access.
- **Backend:** holds the API key, assembles prompts from Coach Memory, calls Claude, runs the safety validation layer, persists data.
- **Data ingestion:** Health Connect as the single, vendor-agnostic source for recovery signals.
- **Sync/storage:** cloud-backed profile, memory, and history so progress is durable and portable across devices.
- **Offline:** generated sessions cached locally; full session runnable without connectivity.

---

## 12. Risks

- **Gyms have no public API for pulling class schedule.** Class data may be user-entered via an uploaded PDF or screen shot.
- **Fitbit/Pixel readiness is partly locked.** Mitigated by computing our own readiness from raw Health Connect signals.
- **LLM cost and latency.** Mitigated by one generation per session at session start plus local autoregulation; needs real-world measurement.
- **Medical and liability surface.** Requires careful disclaimer language and a hard safety layer; injury identification assist in particular needs legal review before shipping.
- **Cold-start quality.** How good is session one before the system has history? Onboarding depth vs. speed tradeoff to validate.

---

## 13. Decisions locked so far

- Android native, Kotlin + Jetpack Compose
- Backend + Claude API (key server-side) with a deterministic safety/validation layer
- Health Connect as the single recovery-data ingestion point; readiness computed in-house
- LLM generates the full session at session start, cached offline, with local RPE autoregulation and trigger-based re-planning
- General, configurable user model captured via adaptive onboarding; no user hard-coded into the engine
- Coach Memory as the durable context the model reads on every call
- Priorities for v1: age-aware (1), injury-aware (2), recovery/rest guidance (3), Equinox classes (4)

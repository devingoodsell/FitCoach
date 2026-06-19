package pro.d11l.fitcoach.feature.session

import pro.d11l.fitcoach.core.network.AgingBlockDto
import pro.d11l.fitcoach.core.network.ReasoningNoteDto
import pro.d11l.fitcoach.core.network.SessionDto
import pro.d11l.fitcoach.core.network.SessionExerciseDto
import pro.d11l.fitcoach.core.network.SessionInputsSummaryDto
import pro.d11l.fitcoach.core.network.SetPrescriptionDto

/** Sample mirroring backend/api/examples/session-sample.json, so tests and tools
 *  can build against the published shape without a running backend. */
internal fun sampleSession(): SessionDto = SessionDto(
    id = "0192f3a0-1c2d-7e00-9abc-0123456789ab",
    generatedAt = "2026-06-16T13:30:00Z",
    schemaVersion = 1,
    model = "claude-opus-4-8",
    inputsSummary = SessionInputsSummaryDto(
        readinessValue = 72,
        readinessConfidence = "high",
        contraindicationCount = 1,
        locationName = "Home gym",
        agingEmphases = listOf("bone_balance", "joint_tendon"),
    ),
    warmup = listOf(
        SessionExerciseDto(
            "Rower easy spin", "row_erg", "full_body",
            listOf(SetPrescriptionDto(type = "time", durationSec = 180, rpeTarget = 3.0, restSec = 0)),
        ),
    ),
    mainWork = listOf(
        SessionExerciseDto(
            "Goblet box squat", "box_squat", "quad",
            listOf(
                SetPrescriptionDto(type = "reps", reps = 8, loadKg = 20.0, rpeTarget = 6.0, restSec = 120),
                SetPrescriptionDto(type = "reps", reps = 8, loadKg = 24.0, rpeTarget = 7.0, restSec = 120),
            ),
            notes = "Box keeps load off the knee.",
        ),
    ),
    accessory = listOf(
        SessionExerciseDto(
            "Half-kneeling cable row", "row", "back",
            listOf(SetPrescriptionDto(type = "reps", reps = 12, loadKg = 20.0, rpeTarget = 7.0, restSec = 75)),
        ),
    ),
    agingBlock = AgingBlockDto(
        emphases = listOf("bone_balance", "joint_tendon"),
        items = listOf(
            SessionExerciseDto(
                "Pogo hops", "low_amplitude_jump", "ankle",
                listOf(SetPrescriptionDto(type = "reps", reps = 15, rpeTarget = 5.0, restSec = 45)),
                notes = "Bone-loading and tendon stiffness.",
            ),
        ),
    ),
    reasoning = listOf(
        ReasoningNoteDto("Held RPE 7-8 with full rest on a strong readiness day.", "intensity"),
        ReasoningNoteDto("At 45, added bone-loading hops and balance work.", "age_aware"),
    ),
    disclaimer = "FitCoach provides general fitness guidance, not medical advice.",
)

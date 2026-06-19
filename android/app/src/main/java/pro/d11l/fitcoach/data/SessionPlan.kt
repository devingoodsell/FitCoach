package pro.d11l.fitcoach.data

import kotlinx.serialization.json.Json
import pro.d11l.fitcoach.core.db.ExerciseEntity
import pro.d11l.fitcoach.core.db.SessionWithExercises
import pro.d11l.fitcoach.core.network.SetPrescriptionDto

/** As-performed state of one set (E6-PR2). Null/false until the set is logged. */
data class LoggedSetState(
    val repsDone: Int? = null,
    val loadKgDone: Double? = null,
    val rpeActual: Double? = null,
    val durationDoneSec: Int? = null,
    val skipped: Boolean = false,
    val completed: Boolean = false,
)

/**
 * One performable set in execution order, carrying its prescription, the exercise
 * it belongs to, and the Room `setId` so logged actuals persist offline. This is
 * the flat unit the player steps through.
 */
data class PlanSet(
    val setId: Long,
    val blockType: String,
    val blockTitle: String,
    val exerciseKey: String,
    val exerciseName: String,
    val movement: String,
    val region: String?,
    val notes: String?,
    val setIndexInExercise: Int,
    val setCountInExercise: Int,
    val prescription: SetPrescriptionDto,
    val logged: LoggedSetState = LoggedSetState(),
)

/**
 * The cached session flattened for the in-session player (E6): ordered steps
 * across warmup -> main -> accessory -> aging, plus the metadata the surface
 * needs to record and sync the result.
 */
data class SessionPlan(
    val sessionId: String,
    val clientSessionId: String,
    val disclaimer: String,
    val steps: List<PlanSet>,
)

private val BLOCK_TITLES = mapOf(
    ExerciseEntity.BLOCK_WARMUP to "Warm-up",
    ExerciseEntity.BLOCK_MAIN to "Main work",
    ExerciseEntity.BLOCK_ACCESSORY to "Accessory",
    ExerciseEntity.BLOCK_AGING to "Healthy-aging block",
)

private val BLOCK_ORDER = listOf(
    ExerciseEntity.BLOCK_WARMUP,
    ExerciseEntity.BLOCK_MAIN,
    ExerciseEntity.BLOCK_ACCESSORY,
    ExerciseEntity.BLOCK_AGING,
)

/** Flattens the cached graph into ordered player steps. */
fun SessionWithExercises.toSessionPlan(json: Json): SessionPlan {
    val steps = mutableListOf<PlanSet>()
    val byBlock = exercises.groupBy { it.exercise.blockType }
    for (block in BLOCK_ORDER) {
        val exercisesInBlock = byBlock[block].orEmpty().sortedBy { it.exercise.orderIndex }
        for (ews in exercisesInBlock) {
            val orderedSets = ews.sets.sortedBy { it.orderIndex }
            orderedSets.forEachIndexed { setIndex, set ->
                steps.add(
                    PlanSet(
                        setId = set.setId,
                        blockType = block,
                        blockTitle = BLOCK_TITLES[block] ?: block,
                        exerciseKey = "${ews.exercise.exerciseId}",
                        exerciseName = ews.exercise.name,
                        movement = ews.exercise.movement,
                        region = ews.exercise.region,
                        notes = ews.exercise.notes,
                        setIndexInExercise = setIndex,
                        setCountInExercise = orderedSets.size,
                        prescription = SetPrescriptionDto(
                            type = set.type,
                            reps = set.reps,
                            loadKg = set.loadKg,
                            rpeTarget = set.rpeTarget,
                            durationSec = set.durationSec,
                            restSec = set.restSec,
                        ),
                        logged = LoggedSetState(
                            repsDone = set.repsDone,
                            loadKgDone = set.loadKgDone,
                            rpeActual = set.rpeActual,
                            durationDoneSec = set.durationDoneSec,
                            skipped = set.skipped,
                            completed = set.completed,
                        ),
                    ),
                )
            }
        }
    }
    return SessionPlan(
        sessionId = session.sessionId,
        clientSessionId = session.clientSessionId,
        disclaimer = session.disclaimer,
        steps = steps,
    )
}

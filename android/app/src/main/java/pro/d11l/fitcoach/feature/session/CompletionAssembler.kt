package pro.d11l.fitcoach.feature.session

import pro.d11l.fitcoach.core.network.LoggedExerciseDto
import pro.d11l.fitcoach.core.network.LoggedSetDto
import pro.d11l.fitcoach.core.network.WorkoutLogData
import pro.d11l.fitcoach.data.PlanSet
import pro.d11l.fitcoach.data.SessionPlan

/**
 * Builds the as-performed payload from the player's steps at completion (E6-PR5).
 * Any set that was not completed (skipped, or never reached when finishing early)
 * is recorded as `skipped`, and the session is marked `partial` if any such set
 * exists — so partial sessions are faithfully recorded and inform future plans.
 * Pure — JVM-tested.
 */
object CompletionAssembler {

    fun build(plan: SessionPlan): WorkoutLogData {
        // exerciseKey is unique per exercise and steps are contiguous, so grouping
        // preserves block order and exercise order.
        val exercises = plan.steps.groupBy { it.exerciseKey }.values.map { steps ->
            val head = steps.first()
            LoggedExerciseDto(
                blockType = head.blockType,
                name = head.exerciseName,
                movement = head.movement,
                sets = steps.map { it.toLoggedSetDto() },
            )
        }
        val partial = plan.steps.any { !it.logged.completed }
        return WorkoutLogData(
            sessionId = plan.sessionId,
            status = if (partial) "partial" else "completed",
            exercises = exercises,
        )
    }

    private fun PlanSet.toLoggedSetDto(): LoggedSetDto = LoggedSetDto(
        type = prescription.type,
        repsDone = logged.repsDone,
        loadKg = logged.loadKgDone,
        rpeActual = logged.rpeActual,
        durationDoneSec = logged.durationDoneSec,
        skipped = !logged.completed,
    )
}

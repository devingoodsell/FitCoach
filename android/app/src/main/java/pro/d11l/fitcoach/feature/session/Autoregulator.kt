package pro.d11l.fitcoach.feature.session

import pro.d11l.fitcoach.core.network.SetPrescriptionDto
import pro.d11l.fitcoach.data.LoggedSetState
import kotlin.math.roundToInt

/**
 * On-device autoregulation (E5-PR6). From how the just-completed set actually went
 * versus its RPE target, nudge the next set's load toward that target — entirely
 * on-device, no LLM, works offline.
 *
 * Effort is read from logged RPE when present; otherwise it's inferred from reps
 * achieved versus prescribed (more reps than asked ⇒ the set was easier than the
 * target ⇒ go heavier; fewer ⇒ harder ⇒ go lighter). The load change is ~3% per
 * RPE point / per rep of deviation, clamped to ±15% so a single set never swings
 * the plan wildly, and rounded to the nearest 0.5 kg.
 */
object Autoregulator {

    const val LOAD_STEP_PER_REP = 0.03
    const val LOAD_STEP_PER_RPE = 0.03
    const val MAX_ADJUST = 0.15
    const val ROUND_INCREMENT_KG = 0.5

    /** Returns [next] with its load adjusted toward [previous]'s RPE target, or unchanged. */
    fun adjust(
        previous: SetPrescriptionDto,
        logged: LoggedSetState,
        next: SetPrescriptionDto,
    ): SetPrescriptionDto {
        val nextLoad = next.loadKg ?: return next // bodyweight: nothing to scale
        if (logged.skipped || !logged.completed) return next
        val factor = effortFactor(previous, logged) ?: return next
        val adjusted = roundToIncrement(nextLoad * factor, ROUND_INCREMENT_KG).coerceAtLeast(0.0)
        return next.copy(loadKg = adjusted)
    }

    private fun effortFactor(previous: SetPrescriptionDto, logged: LoggedSetState): Double? {
        val raw = when {
            logged.rpeActual != null && previous.rpeTarget != null ->
                1 + (previous.rpeTarget - logged.rpeActual) * LOAD_STEP_PER_RPE
            logged.repsDone != null && previous.reps != null ->
                1 + (logged.repsDone - previous.reps) * LOAD_STEP_PER_REP
            else -> return null
        }
        return raw.coerceIn(1 - MAX_ADJUST, 1 + MAX_ADJUST)
    }

    private fun roundToIncrement(value: Double, increment: Double): Double =
        (value / increment).roundToInt() * increment
}

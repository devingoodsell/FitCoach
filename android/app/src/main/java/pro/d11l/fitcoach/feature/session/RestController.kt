package pro.d11l.fitcoach.feature.session

/**
 * Pure rest-countdown state (E6-PR3). The ViewModel ticks this once per second
 * while [running]; the UI fires the audible + haptic cue when [finished] first
 * becomes true. Kept free of Android types so it is JVM-unit-testable.
 */
data class RestState(
    val totalSec: Int,
    val remainingSec: Int,
    val running: Boolean,
) {
    val finished: Boolean get() = remainingSec <= 0
}

object RestController {

    const val EXTEND_STEP_SEC = 15

    fun start(totalSec: Int): RestState =
        RestState(totalSec = totalSec, remainingSec = totalSec, running = totalSec > 0)

    /** One second elapsed. Stops running when it reaches zero. */
    fun tick(state: RestState): RestState {
        if (!state.running || state.finished) return state
        val remaining = (state.remainingSec - 1).coerceAtLeast(0)
        return state.copy(remainingSec = remaining, running = remaining > 0)
    }

    fun pause(state: RestState): RestState = state.copy(running = false)

    fun resume(state: RestState): RestState =
        if (state.finished) state else state.copy(running = true)

    /** Adds time and keeps the countdown going. */
    fun extend(state: RestState, bySec: Int = EXTEND_STEP_SEC): RestState =
        state.copy(
            totalSec = state.totalSec + bySec,
            remainingSec = state.remainingSec + bySec,
            running = true,
        )

    /** Ends the rest immediately. */
    fun skip(state: RestState): RestState = state.copy(remainingSec = 0, running = false)
}

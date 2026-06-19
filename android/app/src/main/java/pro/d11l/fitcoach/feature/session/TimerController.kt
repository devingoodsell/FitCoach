package pro.d11l.fitcoach.feature.session

/**
 * Pure count-up timer for timed exercises like planks (E6-PR4). The ViewModel
 * ticks it once per second while [running]; the elapsed seconds are logged as the
 * completed duration. Free of Android types so it is JVM-unit-testable.
 */
data class TimerState(val elapsedSec: Int, val running: Boolean)

object TimerController {
    fun start(): TimerState = TimerState(elapsedSec = 0, running = true)
    fun tick(state: TimerState): TimerState =
        if (state.running) state.copy(elapsedSec = state.elapsedSec + 1) else state
    fun stop(state: TimerState): TimerState = state.copy(running = false)
    fun resume(state: TimerState): TimerState = state.copy(running = true)
}

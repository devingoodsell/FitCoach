package pro.d11l.fitcoach.feature.session

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.data.LoggedSetState
import pro.d11l.fitcoach.data.PlanSet
import pro.d11l.fitcoach.data.SessionPlan
import pro.d11l.fitcoach.data.SessionRepository
import pro.d11l.fitcoach.data.WorkoutSyncManager
import java.time.Instant

/** Result of finishing a session (E6-PR5). */
data class CompletionState(val saved: Boolean, val syncedNow: Boolean)

data class SessionUiState(
    val loading: Boolean = false,
    val error: String? = null,
    val plan: SessionPlan? = null,
    val stepIndex: Int = 0,
    val draft: SetDraft = SetDraft(),
    val rest: RestState? = null,
    /** Increments each time a rest hits zero, driving the audible + haptic cue. */
    val restCueId: Int = 0,
    val timer: TimerState? = null,
    val finished: Boolean = false,
    val completion: CompletionState? = null,
) {
    val current: PlanSet? get() = plan?.steps?.getOrNull(stepIndex)
    val loggedCount: Int get() = plan?.steps?.count { it.logged.completed || it.logged.skipped } ?: 0
    val totalSteps: Int get() = plan?.steps?.size ?: 0
}

/**
 * Drives the in-session player (E6): generate-or-restore the session offline
 * (E5-PR5/E12-S1), then step through it set by set — logging reps/weight that
 * default to the prescription (E6-PR2) and running a rest countdown after each
 * logged set (E6-PR3). All actions work with no connectivity; the only network
 * touch is the initial generate, which falls back to cache.
 */
class SessionViewModel(
    private val repo: SessionRepository,
    private val sync: WorkoutSyncManager,
    private val now: () -> String = { Instant.now().toString() },
) : ViewModel() {

    private val _state = MutableStateFlow(SessionUiState())
    val state: StateFlow<SessionUiState> = _state.asStateFlow()

    private var restJob: Job? = null
    private var timerJob: Job? = null

    /** Tapping "Start workout": generate (or restore offline) and enter the player. */
    fun start() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            repo.generate()
                .onSuccess { loadPlan() }
                .onFailure { e -> _state.update { it.copy(loading = false, error = e.message) } }
        }
    }

    private suspend fun loadPlan() {
        val plan = repo.plan()
        if (plan == null || plan.steps.isEmpty()) {
            _state.update { it.copy(loading = false, error = "No session to run.") }
            return
        }
        _state.update {
            it.copy(
                loading = false,
                error = null,
                plan = plan,
                stepIndex = 0,
                draft = SessionPlayer.draftFor(plan.steps.first().prescription),
                rest = null,
                timer = null,
                finished = false,
                completion = null,
            )
        }
    }

    fun updateReps(value: String) = _state.update { it.copy(draft = it.draft.copy(reps = value)) }
    fun updateLoad(value: String) = _state.update { it.copy(draft = it.draft.copy(loadKg = value)) }
    fun updateDuration(value: String) = _state.update { it.copy(draft = it.draft.copy(durationSec = value)) }

    /** Records the current set from the draft (defaults to prescribed) and advances, resting if prescribed. */
    fun logCurrentSet() {
        val s = _state.value
        val step = s.current ?: return
        val logged = SessionPlayer.logFrom(step.prescription, s.draft)
        viewModelScope.launch { repo.logSet(step.setId, logged) }
        recordAndAdvance(step, logged, restSec = step.prescription.restSec ?: 0)
    }

    /** Marks the current set skipped (recorded as skipped) and advances without resting. */
    fun skipCurrentSet() {
        val s = _state.value
        val step = s.current ?: return
        val logged = SessionPlayer.skipped()
        viewModelScope.launch { repo.logSet(step.setId, logged) }
        recordAndAdvance(step, logged, restSec = 0)
    }

    private fun recordAndAdvance(step: PlanSet, logged: LoggedSetState, restSec: Int) {
        timerJob?.cancel()
        _state.update { st ->
            val plan = st.plan ?: return@update st
            val steps = plan.steps.toMutableList()
            steps[st.stepIndex] = step.copy(logged = logged)
            val nextIndex = st.stepIndex + 1
            val done = nextIndex >= steps.size
            st.copy(
                plan = plan.copy(steps = steps),
                stepIndex = if (done) st.stepIndex else nextIndex,
                draft = if (done) st.draft else SessionPlayer.draftFor(steps[nextIndex].prescription),
                timer = null,
                finished = done,
            )
        }
        // Rest only when there is a next set to rest before; otherwise clear any panel.
        if (!_state.value.finished && restSec > 0) startRest(restSec) else dismissRest()
    }

    // --- timed-exercise timer (E6-PR4) -------------------------------------

    /** Starts the count-up timer; elapsed seconds populate the duration draft live. */
    fun startTimer() {
        timerJob?.cancel()
        _state.update { it.copy(timer = TimerController.start(), draft = it.draft.copy(durationSec = "0")) }
        runTimerLoop()
    }

    fun stopTimer() {
        timerJob?.cancel()
        _state.update { it.timer?.let { t -> it.copy(timer = TimerController.stop(t)) } ?: it }
    }

    fun resumeTimer() {
        val t = _state.value.timer ?: return
        _state.update { it.copy(timer = TimerController.resume(t)) }
        runTimerLoop()
    }

    private fun runTimerLoop() {
        timerJob?.cancel()
        timerJob = viewModelScope.launch {
            while (isActive) {
                val timer = _state.value.timer ?: break
                if (!timer.running) break
                delay(1000)
                val ticked = TimerController.tick(_state.value.timer ?: return@launch)
                _state.update { it.copy(timer = ticked, draft = it.draft.copy(durationSec = ticked.elapsedSec.toString())) }
            }
        }
    }

    // --- completion (E6-PR5) ------------------------------------------------

    /**
     * Finishes the session: records every set (unlogged work as skipped), queues
     * the result to the durable offline write-queue (E12-PR2), marks the cached
     * session completed, and attempts a sync. Offline-safe — a failed sync leaves
     * the log queued to flush on reconnect; the backend records it into Coach
     * Memory idempotently (E3-PR5).
     */
    fun complete() {
        val plan = _state.value.plan ?: return
        if (_state.value.completion?.saved == true) return
        restJob?.cancel()
        timerJob?.cancel()
        viewModelScope.launch {
            val at = now()
            val payload = CompletionAssembler.build(plan)
            sync.enqueue(plan.clientSessionId, payload, at)
            repo.markCompleted(plan.sessionId, at)
            val result = sync.sync()
            _state.update {
                it.copy(
                    rest = null,
                    timer = null,
                    finished = true,
                    completion = CompletionState(saved = true, syncedNow = result.synced > 0),
                )
            }
        }
    }

    // --- rest countdown (E6-PR3) -------------------------------------------

    private fun startRest(totalSec: Int) {
        restJob?.cancel()
        _state.update { it.copy(rest = RestController.start(totalSec)) }
        runRestLoop()
    }

    fun pauseRest() {
        restJob?.cancel()
        _state.update { it.rest?.let { r -> it.copy(rest = RestController.pause(r)) } ?: it }
    }

    fun resumeRest() {
        val r = _state.value.rest ?: return
        if (r.finished) return
        _state.update { it.copy(rest = RestController.resume(r)) }
        runRestLoop()
    }

    fun extendRest() {
        _state.update { it.rest?.let { r -> it.copy(rest = RestController.extend(r)) } ?: it }
        runRestLoop()
    }

    fun skipRest() {
        restJob?.cancel()
        _state.update { it.rest?.let { r -> it.copy(rest = RestController.skip(r)) } ?: it }
    }

    /** Dismisses a finished rest panel. */
    fun dismissRest() {
        restJob?.cancel()
        _state.update { it.copy(rest = null) }
    }

    private fun runRestLoop() {
        restJob?.cancel()
        restJob = viewModelScope.launch {
            while (isActive) {
                val rest = _state.value.rest ?: break
                if (!rest.running) break
                delay(1000)
                val ticked = RestController.tick(_state.value.rest ?: return@launch)
                _state.update { it.copy(rest = ticked) }
                if (ticked.finished) {
                    _state.update { it.copy(restCueId = it.restCueId + 1) }
                    break
                }
            }
        }
    }

    override fun onCleared() {
        restJob?.cancel()
        timerJob?.cancel()
        super.onCleared()
    }
}

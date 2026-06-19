package pro.d11l.fitcoach.feature.readiness

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.ReadinessDto
import pro.d11l.fitcoach.data.HealthSignalsRepository
import pro.d11l.fitcoach.data.IngestResult
import pro.d11l.fitcoach.data.ReadinessRepository
import java.time.Instant

data class ReadinessUiState(
    val loading: Boolean = true,
    val readiness: ReadinessDto? = null,
    /** Why we don't have a fresh reading (manual / no-readiness fallback). */
    val hint: String? = null,
    /** Non-blocking message (e.g. a sync failure); doesn't hide a usable reading. */
    val error: String? = null,
) {
    /** True when there isn't enough data for a meaningful score (manual fallback). */
    val isUnavailable: Boolean
        get() = readiness?.confidence == "low" && readiness.value == 50 && readiness.drivers.isEmpty()
}

/**
 * Plain-language fallback hints for each non-Uploaded ingest outcome. These explain
 * *why* there's no fresh reading and what to do — never a health/medical claim (E13).
 */
object ReadinessHints {
    const val NO_CONSENT =
        "Health data is off, so we can't read recovery signals. Train in manual mode, or turn " +
            "health data on in Settings to get a readiness score."
    const val UNAVAILABLE =
        "Health Connect isn't available on this device, so we can't read recovery signals. " +
            "You can still train in manual mode."
    const val PERMISSIONS_REQUIRED =
        "Allow FitCoach to read sleep, resting heart rate, and HRV in Health Connect to get a " +
            "readiness score. You can train in manual mode in the meantime."
    const val NO_DATA =
        "No recovery data recorded yet. Wear your device overnight and check back after a few " +
            "nights. You can train in manual mode in the meantime."
    const val SYNC_FAILED =
        "We couldn't sync your latest recovery data. Showing your most recent reading."
}

class ReadinessViewModel(
    private val readinessRepo: ReadinessRepository,
    private val healthSignals: HealthSignalsRepository,
    private val now: () -> Instant = Instant::now,
) : ViewModel() {

    private val _state = MutableStateFlow(ReadinessUiState())
    val state: StateFlow<ReadinessUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null, hint = null) }
        viewModelScope.launch {
            // Pull fresh recovery signals from Health Connect first; the outcome only
            // chooses a fallback hint / non-blocking message — we still render whatever
            // readiness the backend has computed so far.
            val ingest = healthSignals.ingest(now())
            val hint = when (ingest) {
                is IngestResult.Uploaded -> null
                IngestResult.NoConsent -> ReadinessHints.NO_CONSENT
                IngestResult.Unavailable -> ReadinessHints.UNAVAILABLE
                IngestResult.PermissionsRequired -> ReadinessHints.PERMISSIONS_REQUIRED
                IngestResult.NoData -> ReadinessHints.NO_DATA
                is IngestResult.Error -> null
            }
            val ingestError = if (ingest is IngestResult.Error) ReadinessHints.SYNC_FAILED else null

            readinessRepo.today()
                .onSuccess { r ->
                    _state.update { it.copy(loading = false, readiness = r, hint = hint, error = ingestError) }
                }
                .onFailure { e ->
                    _state.update { it.copy(loading = false, hint = hint, error = e.message ?: ingestError) }
                }
        }
    }
}

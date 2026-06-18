package pro.d11l.fitcoach.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.ConsentRecord
import pro.d11l.fitcoach.data.ConsentRepository
import pro.d11l.fitcoach.data.ConsentTypes

data class ConsentReviewUiState(
    val loading: Boolean = true,
    val consents: List<ConsentRecord> = emptyList(),
    val isWorking: Boolean = false,
    val error: String? = null,
) {
    /** The current health-data consent record, if the user ever granted one. */
    val healthData: ConsentRecord? get() = consents.firstOrNull { it.type == ConsentTypes.HEALTH_DATA }

    /** True when health-data ingestion is on (consent present and not revoked). */
    val healthDataActive: Boolean get() = healthData?.isActive == true
}

/**
 * Consent review surface (E14-S2): shows what the user agreed to and lets them
 * revoke health-data consent. Revoking disables readiness ingestion server-side, so
 * the app falls back to manual mode.
 */
class ConsentReviewViewModel(private val repo: ConsentRepository) : ViewModel() {

    private val _state = MutableStateFlow(ConsentReviewUiState())
    val state: StateFlow<ConsentReviewUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            repo.load()
                .onSuccess { records -> _state.update { it.copy(loading = false, consents = records) } }
                .onFailure { e -> _state.update { it.copy(loading = false, error = e.message ?: "Could not load consent state.") } }
        }
    }

    fun revokeHealthData() {
        if (_state.value.isWorking) return
        _state.update { it.copy(isWorking = true, error = null) }
        viewModelScope.launch {
            repo.revoke(ConsentTypes.HEALTH_DATA)
                .onSuccess {
                    _state.update { it.copy(isWorking = false) }
                    load() // reflect the new revoked state
                }
                .onFailure { e -> _state.update { it.copy(isWorking = false, error = e.message ?: "Could not revoke consent.") } }
        }
    }
}

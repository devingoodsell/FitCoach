package pro.d11l.fitcoach.feature.consent

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.designsystem.Disclaimers
import pro.d11l.fitcoach.data.AuthRepository
import pro.d11l.fitcoach.data.AuthResult

data class ConsentUiState(
    val isSubmitting: Boolean = false,
    val error: String? = null,
    val decided: Boolean = false,
    val manualMode: Boolean = false,
)

/** Consent types must match the backend's allowed set (E1-S4). */
private const val TYPE_HEALTH_DATA = "health_data"
private const val TYPE_MEDICAL_DISCLAIMER = "medical_disclaimer"

class ConsentViewModel(private val repo: AuthRepository) : ViewModel() {

    private val _state = MutableStateFlow(ConsentUiState())
    val state: StateFlow<ConsentUiState> = _state.asStateFlow()

    /** Affirmatively allow health-data use; also records disclaimer acknowledgement. */
    fun allowHealthData() = record(includeHealthData = true, manual = false)

    /** Decline health data but proceed in manual mode (still acknowledge disclaimer). */
    fun useManualMode() = record(includeHealthData = false, manual = true)

    private fun record(includeHealthData: Boolean, manual: Boolean) {
        if (_state.value.isSubmitting) return
        _state.update { it.copy(isSubmitting = true, error = null) }
        viewModelScope.launch {
            val disclaimer = repo.recordConsent(TYPE_MEDICAL_DISCLAIMER, Disclaimers.VERSION)
            val health = if (includeHealthData) {
                repo.recordConsent(TYPE_HEALTH_DATA, Disclaimers.VERSION)
            } else {
                AuthResult.Success
            }
            _state.update {
                if (disclaimer is AuthResult.Success && health is AuthResult.Success) {
                    it.copy(isSubmitting = false, decided = true, manualMode = manual)
                } else {
                    it.copy(isSubmitting = false, error = "Could not save your choice. Please try again.")
                }
            }
        }
    }
}

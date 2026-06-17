package pro.d11l.fitcoach.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.ScheduleDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.data.SaveResult

data class EditScheduleUiState(
    val loading: Boolean = true,
    val daysPerWeek: String = "3",
    val sessionLengthMin: String = "60",
    val isSaving: Boolean = false,
    val fieldErrors: Map<String, String> = emptyMap(),
    val error: String? = null,
    val saved: Boolean = false,
)

/** Edits the training schedule post-onboarding (E14-S1). Preferred days are
 *  preserved verbatim from memory (not yet surfaced for editing). */
class EditScheduleViewModel(private val repo: OnboardingRepository) : ViewModel() {

    private val _state = MutableStateFlow(EditScheduleUiState())
    val state: StateFlow<EditScheduleUiState> = _state.asStateFlow()

    private var preferredDays: List<String> = emptyList()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            val sc = repo.loadSchedule()
            preferredDays = sc?.preferredDays ?: emptyList()
            _state.update {
                if (sc == null) {
                    it.copy(loading = false)
                } else {
                    it.copy(
                        loading = false,
                        daysPerWeek = sc.daysPerWeek.toString(),
                        sessionLengthMin = sc.sessionLengthMin.toString(),
                    )
                }
            }
        }
    }

    fun onDaysPerWeek(v: String) =
        _state.update { it.copy(daysPerWeek = v.filter(Char::isDigit), fieldErrors = it.fieldErrors - "days_per_week") }

    fun onSessionLength(v: String) =
        _state.update { it.copy(sessionLengthMin = v.filter(Char::isDigit), fieldErrors = it.fieldErrors - "session_length_min") }

    fun save() {
        val s = _state.value
        if (s.isSaving) return
        val days = s.daysPerWeek.toIntOrNull()
        val len = s.sessionLengthMin.toIntOrNull()
        val errors = buildMap {
            if (days == null || days < 1 || days > 7) put("days_per_week", "Choose 1 to 7 days")
            if (len == null || len < 10 || len > 240) put("session_length_min", "Choose 10 to 240 minutes")
        }
        if (errors.isNotEmpty()) {
            _state.update { it.copy(fieldErrors = errors) }
            return
        }
        val dto = ScheduleDto(daysPerWeek = days!!, sessionLengthMin = len!!, preferredDays = preferredDays)
        _state.update { it.copy(isSaving = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            when (val r = repo.saveSchedule(dto)) {
                is SaveResult.Ok -> _state.update { it.copy(isSaving = false, saved = true) }
                is SaveResult.Invalid -> _state.update { it.copy(isSaving = false, fieldErrors = r.fields) }
                is SaveResult.Error -> _state.update { it.copy(isSaving = false, error = r.message) }
            }
        }
    }
}

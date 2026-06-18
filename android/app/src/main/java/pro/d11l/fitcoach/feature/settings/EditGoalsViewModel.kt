package pro.d11l.fitcoach.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.GoalWeightsDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.data.SaveResult

data class EditGoalsUiState(
    val loading: Boolean = true,
    val strength: Float = 0.5f,
    val healthspan: Float = 0.5f,
    val bodyComp: Float = 0.5f,
    val performance: Float = 0.5f,
    val isSaving: Boolean = false,
    val error: String? = null,
    val saved: Boolean = false,
)

/**
 * Edits the goal-weight distribution post-onboarding (E14-S1). Prefills the
 * normalized weights stored in Coach Memory; the backend re-normalizes on save.
 */
class EditGoalsViewModel(private val repo: OnboardingRepository) : ViewModel() {

    private val _state = MutableStateFlow(EditGoalsUiState())
    val state: StateFlow<EditGoalsUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            val g = repo.loadGoals()
            _state.update {
                if (g == null) {
                    it.copy(loading = false)
                } else {
                    it.copy(
                        loading = false,
                        strength = g.strength.toFloat(),
                        healthspan = g.healthspan.toFloat(),
                        bodyComp = g.bodyComposition.toFloat(),
                        performance = g.performance.toFloat(),
                    )
                }
            }
        }
    }

    fun onStrength(v: Float) = _state.update { it.copy(strength = v) }
    fun onHealthspan(v: Float) = _state.update { it.copy(healthspan = v) }
    fun onBodyComp(v: Float) = _state.update { it.copy(bodyComp = v) }
    fun onPerformance(v: Float) = _state.update { it.copy(performance = v) }

    fun save() {
        val s = _state.value
        if (s.isSaving) return
        if (s.strength + s.healthspan + s.bodyComp + s.performance <= 0f) {
            _state.update { it.copy(error = "Set at least one goal above zero.") }
            return
        }
        val dto = GoalWeightsDto(
            strength = s.strength.toDouble(),
            healthspan = s.healthspan.toDouble(),
            bodyComposition = s.bodyComp.toDouble(),
            performance = s.performance.toDouble(),
        )
        _state.update { it.copy(isSaving = true, error = null) }
        viewModelScope.launch {
            when (val r = repo.saveGoals(dto)) {
                is SaveResult.Ok -> _state.update { it.copy(isSaving = false, saved = true) }
                is SaveResult.Invalid -> _state.update { it.copy(isSaving = false, error = "Could not save goals.") }
                is SaveResult.Error -> _state.update { it.copy(isSaving = false, error = r.message) }
            }
        }
    }
}

package pro.d11l.fitcoach.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.DietPrefsDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.data.SaveResult

/** Diet patterns offered in the editor; kosher/halal are hard dietary constraints
 *  honored by planning, so they're preserved exactly as chosen. */
val DIET_PATTERNS = listOf("omnivore", "vegan", "vegetarian", "pescatarian", "kosher", "halal")

data class EditDietUiState(
    val loading: Boolean = true,
    val pattern: String = "",
    val supplements: String = "",
    val medications: String = "",
    val isSaving: Boolean = false,
    val fieldErrors: Map<String, String> = emptyMap(),
    val error: String? = null,
    val saved: Boolean = false,
)

/** Edits dietary preferences post-onboarding (E14-S1), read by diet guidance (E11). */
class EditDietViewModel(private val repo: OnboardingRepository) : ViewModel() {

    private val _state = MutableStateFlow(EditDietUiState())
    val state: StateFlow<EditDietUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            val d = repo.loadDiet()
            _state.update {
                it.copy(
                    loading = false,
                    pattern = d?.pattern.orEmpty(),
                    supplements = d?.supplements.orEmpty(),
                    medications = d?.medications.orEmpty(),
                )
            }
        }
    }

    fun onPattern(v: String) = _state.update { it.copy(pattern = v, fieldErrors = it.fieldErrors - "pattern") }
    fun onSupplements(v: String) = _state.update { it.copy(supplements = v) }
    fun onMedications(v: String) = _state.update { it.copy(medications = v) }

    fun save() {
        val s = _state.value
        if (s.isSaving) return
        if (s.pattern.isBlank()) {
            _state.update { it.copy(fieldErrors = mapOf("pattern" to "Choose a dietary pattern")) }
            return
        }
        val dto = DietPrefsDto(pattern = s.pattern, supplements = s.supplements, medications = s.medications)
        _state.update { it.copy(isSaving = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            when (val r = repo.saveDiet(dto)) {
                is SaveResult.Ok -> _state.update { it.copy(isSaving = false, saved = true) }
                is SaveResult.Invalid -> _state.update { it.copy(isSaving = false, fieldErrors = r.fields) }
                is SaveResult.Error -> _state.update { it.copy(isSaving = false, error = r.message) }
            }
        }
    }
}

package pro.d11l.fitcoach.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.AgingEmphasesDto
import pro.d11l.fitcoach.core.network.ExperienceDto
import pro.d11l.fitcoach.core.network.ProfileDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.data.SaveResult

data class EditProfileUiState(
    val loading: Boolean = true,
    val sex: String = "",
    val age: String = "",
    val heightCm: String = "",
    val weightKg: String = "",
    val level: String = "",
    val isSaving: Boolean = false,
    val fieldErrors: Map<String, String> = emptyMap(),
    val error: String? = null,
    val saved: Boolean = false,
)

/**
 * Edits the profile/physiology section post-onboarding (E14-S1). Prefills from
 * Coach Memory and writes back through the same onboarding endpoint, so edits take
 * effect on the next planning call. Aging emphases are carried through untouched so
 * a profile save never resets the user's E2-S8 tuning (edited separately in E14-PR2).
 */
class EditProfileViewModel(private val repo: OnboardingRepository) : ViewModel() {

    private val _state = MutableStateFlow(EditProfileUiState())
    val state: StateFlow<EditProfileUiState> = _state.asStateFlow()

    // Preserved verbatim from the loaded profile and re-sent on save.
    private var agingEmphases: AgingEmphasesDto? = null

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            val p = repo.loadProfile()
            agingEmphases = p?.agingEmphases
            _state.update {
                it.copy(
                    loading = false,
                    sex = p?.sex.orEmpty(),
                    age = p?.age?.toString().orEmpty(),
                    heightCm = p?.heightCm?.asFieldText().orEmpty(),
                    weightKg = p?.weightKg?.asFieldText().orEmpty(),
                    level = p?.experience?.level.orEmpty(),
                )
            }
        }
    }

    fun onSex(v: String) = _state.update { it.copy(sex = v, fieldErrors = it.fieldErrors - "sex") }
    fun onAge(v: String) = _state.update { it.copy(age = v.filter(Char::isDigit), fieldErrors = it.fieldErrors - "age") }
    fun onHeight(v: String) = _state.update { it.copy(heightCm = v) }
    fun onWeight(v: String) = _state.update { it.copy(weightKg = v) }
    fun onLevel(v: String) = _state.update { it.copy(level = v, fieldErrors = it.fieldErrors - "experience.level") }

    fun save() {
        val s = _state.value
        if (s.isSaving) return
        val errors = buildMap {
            if (s.sex.isBlank()) put("sex", "Select your sex")
            val age = s.age.toIntOrNull()
            if (age == null || age < 13 || age > 120) put("age", "Enter an age between 13 and 120")
            if (s.level.isBlank()) put("experience.level", "Select your experience level")
        }
        if (errors.isNotEmpty()) {
            _state.update { it.copy(fieldErrors = errors) }
            return
        }
        val dto = ProfileDto(
            age = s.age.toIntOrNull(),
            sex = s.sex,
            heightCm = s.heightCm.toDoubleOrNull(),
            weightKg = s.weightKg.toDoubleOrNull(),
            experience = ExperienceDto(level = s.level),
            agingEmphases = agingEmphases,
        )
        _state.update { it.copy(isSaving = true, error = null, fieldErrors = emptyMap()) }
        viewModelScope.launch {
            when (val r = repo.saveProfile(dto)) {
                is SaveResult.Ok -> _state.update { it.copy(isSaving = false, saved = true) }
                is SaveResult.Invalid -> _state.update { it.copy(isSaving = false, fieldErrors = r.fields) }
                is SaveResult.Error -> _state.update { it.copy(isSaving = false, error = r.message) }
            }
        }
    }
}

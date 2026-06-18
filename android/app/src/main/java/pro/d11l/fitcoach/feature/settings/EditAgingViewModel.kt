package pro.d11l.fitcoach.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.AgingEmphasesDto
import pro.d11l.fitcoach.core.network.ProfileDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.data.SaveResult

data class EditAgingUiState(
    val loading: Boolean = true,
    val boneBalance: Float = 0f,
    val jointTendon: Float = 0f,
    val vo2max: Float = 0f,
    val cardioBase: Float = 0f,
    val isSaving: Boolean = false,
    val error: String? = null,
    val saved: Boolean = false,
)

/**
 * Edits the four aging-emphasis weights (E2-S8 carryover): the onboarding wizard
 * defaults them from age and never surfaced adjusters, so this is the first place a
 * user can override them. They persist through the profile write path, so saving
 * sends the full profile section (preserving sex/age/physiology) with only the
 * emphases changed. Defaults mirror the server's age-based defaults when none are set.
 */
class EditAgingViewModel(private val repo: OnboardingRepository) : ViewModel() {

    private val _state = MutableStateFlow(EditAgingUiState())
    val state: StateFlow<EditAgingUiState> = _state.asStateFlow()

    // The loaded profile, re-sent in full on save so no other field is dropped.
    private var profile: ProfileDto? = null

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            val p = repo.loadProfile()
            profile = p
            if (p == null) {
                _state.update { it.copy(loading = false, error = "Set up your profile first.") }
                return@launch
            }
            val aging = p.agingEmphases ?: defaultAgingEmphases(p.age)
            _state.update {
                it.copy(
                    loading = false,
                    boneBalance = aging.boneBalance.toFloat(),
                    jointTendon = aging.jointTendon.toFloat(),
                    vo2max = aging.vo2max.toFloat(),
                    cardioBase = aging.cardioBase.toFloat(),
                )
            }
        }
    }

    fun onBoneBalance(v: Float) = _state.update { it.copy(boneBalance = v) }
    fun onJointTendon(v: Float) = _state.update { it.copy(jointTendon = v) }
    fun onVo2max(v: Float) = _state.update { it.copy(vo2max = v) }
    fun onCardioBase(v: Float) = _state.update { it.copy(cardioBase = v) }

    fun save() {
        val s = _state.value
        if (s.isSaving) return
        val base = profile
        if (base == null) {
            _state.update { it.copy(error = "Set up your profile first.") }
            return
        }
        val dto = base.copy(
            agingEmphases = AgingEmphasesDto(
                boneBalance = s.boneBalance.toDouble(),
                jointTendon = s.jointTendon.toDouble(),
                vo2max = s.vo2max.toDouble(),
                cardioBase = s.cardioBase.toDouble(),
            ),
        )
        _state.update { it.copy(isSaving = true, error = null) }
        viewModelScope.launch {
            when (val r = repo.saveProfile(dto)) {
                is SaveResult.Ok -> _state.update { it.copy(isSaving = false, saved = true) }
                is SaveResult.Invalid -> _state.update { it.copy(isSaving = false, error = "Could not save emphases.") }
                is SaveResult.Error -> _state.update { it.copy(isSaving = false, error = r.message) }
            }
        }
    }
}

/** Mirrors the backend onboarding.DefaultAgingEmphases age tiers so the sliders show
 *  meaningful starting values when a profile predates the emphases field. */
internal fun defaultAgingEmphases(age: Int?): AgingEmphasesDto = when {
    age == null -> AgingEmphasesDto(boneBalance = 0.15, jointTendon = 0.2, vo2max = 0.35, cardioBase = 0.3)
    age >= 60 -> AgingEmphasesDto(boneBalance = 0.35, jointTendon = 0.3, vo2max = 0.15, cardioBase = 0.2)
    age >= 45 -> AgingEmphasesDto(boneBalance = 0.25, jointTendon = 0.25, vo2max = 0.25, cardioBase = 0.25)
    else -> AgingEmphasesDto(boneBalance = 0.15, jointTendon = 0.2, vo2max = 0.35, cardioBase = 0.3)
}

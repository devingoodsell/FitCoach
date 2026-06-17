package pro.d11l.fitcoach.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.PreferencesDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.data.SaveResult

data class EditPreferencesUiState(
    val loading: Boolean = true,
    val likes: String = "",
    val dislikes: String = "",
    val hardAvoids: String = "",
    val isSaving: Boolean = false,
    val error: String? = null,
    val saved: Boolean = false,
)

/**
 * Edits exercise/equipment preferences post-onboarding (E14-S1). Hard-avoids are
 * planning constraints (distinct from soft dislikes); they round-trip through the
 * form so an edit never silently drops them.
 */
class EditPreferencesViewModel(private val repo: OnboardingRepository) : ViewModel() {

    private val _state = MutableStateFlow(EditPreferencesUiState())
    val state: StateFlow<EditPreferencesUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            val p = repo.loadPreferences()
            _state.update {
                it.copy(
                    loading = false,
                    likes = p?.likes.orEmpty().joinToString(", "),
                    dislikes = p?.dislikes.orEmpty().joinToString(", "),
                    hardAvoids = p?.hardAvoids.orEmpty().joinToString(", "),
                )
            }
        }
    }

    fun onLikes(v: String) = _state.update { it.copy(likes = v) }
    fun onDislikes(v: String) = _state.update { it.copy(dislikes = v) }
    fun onHardAvoids(v: String) = _state.update { it.copy(hardAvoids = v) }

    fun save() {
        val s = _state.value
        if (s.isSaving) return
        val dto = PreferencesDto(
            likes = parseList(s.likes),
            dislikes = parseList(s.dislikes),
            hardAvoids = parseList(s.hardAvoids),
        )
        _state.update { it.copy(isSaving = true, error = null) }
        viewModelScope.launch {
            when (val r = repo.savePreferences(dto)) {
                is SaveResult.Ok -> _state.update { it.copy(isSaving = false, saved = true) }
                is SaveResult.Invalid -> _state.update { it.copy(isSaving = false, error = "Could not save preferences.") }
                is SaveResult.Error -> _state.update { it.copy(isSaving = false, error = r.message) }
            }
        }
    }

    private fun parseList(raw: String): List<String> =
        raw.split(",").map(String::trim).filter(String::isNotEmpty)
}

package pro.d11l.fitcoach.feature.diet

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.DietTargetsDto
import pro.d11l.fitcoach.data.DietRepository

data class DietUiState(
    val loading: Boolean = true,
    val targets: DietTargetsDto? = null,
    val note: String? = null,
    val error: String? = null,
)

class DietViewModel(private val repo: DietRepository) : ViewModel() {

    private val _state = MutableStateFlow(DietUiState())
    val state: StateFlow<DietUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            repo.targets()
                .onSuccess { t -> _state.update { it.copy(loading = false, targets = t) } }
                .onFailure { e -> _state.update { it.copy(loading = false, error = e.message) } }
        }
    }

    fun loadNote(heavy: Boolean) {
        viewModelScope.launch {
            repo.postWorkoutNote(heavy)
                .onSuccess { n -> _state.update { it.copy(note = n.note) } }
                .onFailure { e -> _state.update { it.copy(error = e.message) } }
        }
    }
}

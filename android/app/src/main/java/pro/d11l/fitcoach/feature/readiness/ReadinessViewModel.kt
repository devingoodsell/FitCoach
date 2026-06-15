package pro.d11l.fitcoach.feature.readiness

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.ReadinessDto
import pro.d11l.fitcoach.data.ReadinessRepository

data class ReadinessUiState(
    val loading: Boolean = true,
    val readiness: ReadinessDto? = null,
    val error: String? = null,
) {
    /** True when there isn't enough data for a meaningful score (manual fallback). */
    val isUnavailable: Boolean
        get() = readiness?.confidence == "low" && readiness.value == 50 && readiness.drivers.isEmpty()
}

class ReadinessViewModel(private val repo: ReadinessRepository) : ViewModel() {

    private val _state = MutableStateFlow(ReadinessUiState())
    val state: StateFlow<ReadinessUiState> = _state.asStateFlow()

    init {
        load()
    }

    fun load() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            repo.today()
                .onSuccess { r -> _state.update { it.copy(loading = false, readiness = r) } }
                .onFailure { e -> _state.update { it.copy(loading = false, error = e.message) } }
        }
    }
}

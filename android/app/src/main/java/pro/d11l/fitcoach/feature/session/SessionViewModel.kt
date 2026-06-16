package pro.d11l.fitcoach.feature.session

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.core.network.SessionDto
import pro.d11l.fitcoach.data.SessionRepository

data class SessionUiState(
    val loading: Boolean = false,
    val session: SessionDto? = null,
    val error: String? = null,
)

/**
 * Drives the session-preview scaffold (E5-S3). It requests one generated session
 * from the backend and exposes it for read-only display. Full in-session logging,
 * timers, and offline autoregulation are E6/Wave 3 and build on the same
 * [SessionDto] shape.
 */
class SessionViewModel(private val repo: SessionRepository) : ViewModel() {

    private val _state = MutableStateFlow(SessionUiState())
    val state: StateFlow<SessionUiState> = _state.asStateFlow()

    /** Tapping "Start workout" generates the session. */
    fun start() {
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            repo.generate()
                .onSuccess { s -> _state.update { it.copy(loading = false, session = s) } }
                .onFailure { e -> _state.update { it.copy(loading = false, error = e.message) } }
        }
    }
}

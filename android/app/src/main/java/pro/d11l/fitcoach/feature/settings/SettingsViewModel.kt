package pro.d11l.fitcoach.feature.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.data.AuthRepository
import pro.d11l.fitcoach.data.AuthResult

data class SettingsUiState(
    val confirmingDelete: Boolean = false,
    val deletePassword: String = "",
    val isWorking: Boolean = false,
    val error: String? = null,
    val signedOut: Boolean = false,
)

class SettingsViewModel(private val repo: AuthRepository) : ViewModel() {

    private val _state = MutableStateFlow(SettingsUiState())
    val state: StateFlow<SettingsUiState> = _state.asStateFlow()

    fun startDelete() = _state.update { it.copy(confirmingDelete = true, error = null) }
    fun cancelDelete() = _state.update { it.copy(confirmingDelete = false, deletePassword = "", error = null) }
    fun onDeletePasswordChange(v: String) = _state.update { it.copy(deletePassword = v, error = null) }

    fun logout() {
        if (_state.value.isWorking) return
        _state.update { it.copy(isWorking = true) }
        viewModelScope.launch {
            repo.logout()
            _state.update { it.copy(isWorking = false, signedOut = true) }
        }
    }

    fun confirmDelete() {
        val pw = _state.value.deletePassword
        if (pw.isEmpty()) {
            _state.update { it.copy(error = "Enter your password to confirm.") }
            return
        }
        _state.update { it.copy(isWorking = true, error = null) }
        viewModelScope.launch {
            when (val result = repo.deleteAccount(pw)) {
                is AuthResult.Success -> _state.update { it.copy(isWorking = false, signedOut = true) }
                is AuthResult.Failure -> _state.update {
                    it.copy(
                        isWorking = false,
                        error = if (result.code == "http_401") "Incorrect password." else "Could not delete account.",
                    )
                }
            }
        }
    }
}

package pro.d11l.fitcoach.feature.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import pro.d11l.fitcoach.data.AuthRepository
import pro.d11l.fitcoach.data.AuthResult

enum class AuthMode { Login, Signup }

/** Immutable UI state for the auth screen (state down, events up). */
data class AuthUiState(
    val mode: AuthMode = AuthMode.Login,
    val email: String = "",
    val password: String = "",
    val isSubmitting: Boolean = false,
    val error: String? = null,
    val authenticated: Boolean = false,
)

class AuthViewModel(private val repo: AuthRepository) : ViewModel() {

    private val _state = MutableStateFlow(AuthUiState(authenticated = repo.isLoggedIn()))
    val state: StateFlow<AuthUiState> = _state.asStateFlow()

    fun onEmailChange(value: String) = _state.update { it.copy(email = value, error = null) }
    fun onPasswordChange(value: String) = _state.update { it.copy(password = value, error = null) }
    fun toggleMode() = _state.update {
        it.copy(mode = if (it.mode == AuthMode.Login) AuthMode.Signup else AuthMode.Login, error = null)
    }

    fun submit() {
        val current = _state.value
        validate(current)?.let { msg ->
            _state.update { it.copy(error = msg) }
            return
        }
        _state.update { it.copy(isSubmitting = true, error = null) }
        viewModelScope.launch {
            val result = when (current.mode) {
                AuthMode.Login -> repo.login(current.email, current.password)
                AuthMode.Signup -> repo.signup(current.email, current.password)
            }
            _state.update {
                when (result) {
                    is AuthResult.Success -> it.copy(isSubmitting = false, authenticated = true, password = "")
                    is AuthResult.Failure -> it.copy(isSubmitting = false, error = friendly(it.mode, result))
                }
            }
        }
    }

    private fun validate(state: AuthUiState): String? = when {
        !state.email.contains("@") || !state.email.contains(".") -> "Enter a valid email address."
        state.password.length < MIN_PASSWORD -> "Password must be at least $MIN_PASSWORD characters."
        else -> null
    }

    private fun friendly(mode: AuthMode, failure: AuthResult.Failure): String = when {
        failure.code == "http_401" -> "Incorrect email or password."
        failure.code == "http_409" -> "Could not complete signup."
        failure.code == "http_429" -> "Too many attempts. Please try again later."
        failure.code == "network_error" -> "Network error. Check your connection and try again."
        mode == AuthMode.Signup -> "Could not complete signup."
        else -> "Something went wrong. Please try again."
    }

    private companion object {
        const val MIN_PASSWORD = 10
    }
}

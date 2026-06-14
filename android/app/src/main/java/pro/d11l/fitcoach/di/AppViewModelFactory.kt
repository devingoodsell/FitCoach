package pro.d11l.fitcoach.di

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import pro.d11l.fitcoach.feature.auth.AuthViewModel
import pro.d11l.fitcoach.feature.consent.ConsentViewModel
import pro.d11l.fitcoach.feature.settings.SettingsViewModel

/** Constructs ViewModels with their dependencies from the [AppContainer]. */
class AppViewModelFactory(private val container: AppContainer) : ViewModelProvider.Factory {
    @Suppress("UNCHECKED_CAST")
    override fun <T : ViewModel> create(modelClass: Class<T>): T = when {
        modelClass.isAssignableFrom(AuthViewModel::class.java) ->
            AuthViewModel(container.authRepository) as T
        modelClass.isAssignableFrom(ConsentViewModel::class.java) ->
            ConsentViewModel(container.authRepository) as T
        modelClass.isAssignableFrom(SettingsViewModel::class.java) ->
            SettingsViewModel(container.authRepository) as T
        else -> throw IllegalArgumentException("Unknown ViewModel: ${modelClass.name}")
    }
}

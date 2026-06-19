package pro.d11l.fitcoach.di

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import pro.d11l.fitcoach.feature.auth.AuthViewModel
import pro.d11l.fitcoach.feature.consent.ConsentViewModel
import pro.d11l.fitcoach.feature.diet.DietViewModel
import pro.d11l.fitcoach.feature.injury.InjuryViewModel
import pro.d11l.fitcoach.feature.location.LocationViewModel
import pro.d11l.fitcoach.feature.onboarding.OnboardingViewModel
import pro.d11l.fitcoach.feature.readiness.ReadinessViewModel
import pro.d11l.fitcoach.feature.session.SessionViewModel
import pro.d11l.fitcoach.feature.settings.ConsentReviewViewModel
import pro.d11l.fitcoach.feature.settings.EditAgingViewModel
import pro.d11l.fitcoach.feature.settings.EditDietViewModel
import pro.d11l.fitcoach.feature.settings.EditGoalsViewModel
import pro.d11l.fitcoach.feature.settings.EditPreferencesViewModel
import pro.d11l.fitcoach.feature.settings.EditProfileViewModel
import pro.d11l.fitcoach.feature.settings.EditScheduleViewModel
import pro.d11l.fitcoach.feature.settings.SettingsViewModel

/** Constructs ViewModels with their dependencies from the [AppContainer]. */
class AppViewModelFactory(private val container: AppContainer) : ViewModelProvider.Factory {
    @Suppress("UNCHECKED_CAST")
    override fun <T : ViewModel> create(modelClass: Class<T>): T = when {
        modelClass.isAssignableFrom(AuthViewModel::class.java) ->
            AuthViewModel(container.authRepository) as T
        modelClass.isAssignableFrom(ConsentViewModel::class.java) ->
            ConsentViewModel(container.authRepository) as T
        modelClass.isAssignableFrom(OnboardingViewModel::class.java) ->
            OnboardingViewModel(container.onboardingRepository) as T
        modelClass.isAssignableFrom(LocationViewModel::class.java) ->
            LocationViewModel(container.locationRepository) as T
        modelClass.isAssignableFrom(DietViewModel::class.java) ->
            DietViewModel(container.dietRepository) as T
        modelClass.isAssignableFrom(ReadinessViewModel::class.java) ->
            ReadinessViewModel(container.readinessRepository) as T
        modelClass.isAssignableFrom(InjuryViewModel::class.java) ->
            InjuryViewModel(container.injuryRepository) as T
        modelClass.isAssignableFrom(SessionViewModel::class.java) ->
            SessionViewModel(container.sessionRepository, container.workoutSyncManager) as T
        modelClass.isAssignableFrom(SettingsViewModel::class.java) ->
            SettingsViewModel(container.authRepository) as T
        modelClass.isAssignableFrom(EditProfileViewModel::class.java) ->
            EditProfileViewModel(container.onboardingRepository) as T
        modelClass.isAssignableFrom(EditGoalsViewModel::class.java) ->
            EditGoalsViewModel(container.onboardingRepository) as T
        modelClass.isAssignableFrom(EditScheduleViewModel::class.java) ->
            EditScheduleViewModel(container.onboardingRepository) as T
        modelClass.isAssignableFrom(EditPreferencesViewModel::class.java) ->
            EditPreferencesViewModel(container.onboardingRepository) as T
        modelClass.isAssignableFrom(EditDietViewModel::class.java) ->
            EditDietViewModel(container.onboardingRepository) as T
        modelClass.isAssignableFrom(EditAgingViewModel::class.java) ->
            EditAgingViewModel(container.onboardingRepository) as T
        modelClass.isAssignableFrom(ConsentReviewViewModel::class.java) ->
            ConsentReviewViewModel(container.consentRepository) as T
        else -> throw IllegalArgumentException("Unknown ViewModel: ${modelClass.name}")
    }
}

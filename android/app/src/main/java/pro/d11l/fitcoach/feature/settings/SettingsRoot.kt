package pro.d11l.fitcoach.feature.settings

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.lifecycle.viewmodel.compose.viewModel
import pro.d11l.fitcoach.di.AppViewModelFactory
import pro.d11l.fitcoach.feature.location.LocationScreen
import pro.d11l.fitcoach.feature.location.LocationViewModel

/** Settings sections. */
enum class SettingsRoute { Hub, Profile, Goals, Schedule, Preferences, Diet, Locations, Aging, Disclaimers, Consent }

/**
 * Hosts navigation within Settings as local state, so the app's top-level [Step]
 * keeps a single `Settings` entry (no churn on MainActivity per sub-screen). Each
 * editor is created fresh on entry (keyed by a visit counter) so it reloads current
 * memory and starts with a clean save flag.
 */
@Composable
fun SettingsRoot(factory: AppViewModelFactory, resetKey: Int, onSignedOut: () -> Unit) {
    var route by remember { mutableStateOf(SettingsRoute.Hub) }
    var visit by remember { mutableIntStateOf(0) }

    fun open(target: SettingsRoute) {
        visit++
        route = target
    }
    val back = { route = SettingsRoute.Hub }

    when (route) {
        SettingsRoute.Hub -> {
            // Keyed by resetKey (the app's logout epoch) so a retained signed-out
            // SettingsViewModel never bounces a freshly re-logged-in user.
            val vm: SettingsViewModel = viewModel(key = "settings-hub-$resetKey", factory = factory)
            SettingsHubScreen(viewModel = vm, onSignedOut = onSignedOut, onNavigate = ::open)
        }
        SettingsRoute.Profile -> {
            val vm: EditProfileViewModel = viewModel(key = "profile-$visit", factory = factory)
            EditProfileScreen(vm, onDone = back)
        }
        SettingsRoute.Goals -> {
            val vm: EditGoalsViewModel = viewModel(key = "goals-$visit", factory = factory)
            EditGoalsScreen(vm, onDone = back)
        }
        SettingsRoute.Schedule -> {
            val vm: EditScheduleViewModel = viewModel(key = "schedule-$visit", factory = factory)
            EditScheduleScreen(vm, onDone = back)
        }
        SettingsRoute.Preferences -> {
            val vm: EditPreferencesViewModel = viewModel(key = "prefs-$visit", factory = factory)
            EditPreferencesScreen(vm, onDone = back)
        }
        SettingsRoute.Diet -> {
            val vm: EditDietViewModel = viewModel(key = "diet-$visit", factory = factory)
            EditDietScreen(vm, onDone = back)
        }
        SettingsRoute.Locations -> {
            val vm: LocationViewModel = viewModel(key = "settings-locations-$visit", factory = factory)
            LocationScreen(vm, onBack = back)
        }
        SettingsRoute.Aging -> {
            val vm: EditAgingViewModel = viewModel(key = "aging-$visit", factory = factory)
            EditAgingScreen(vm, onDone = back)
        }
        SettingsRoute.Disclaimers -> DisclaimerScreen(onBack = back)
        SettingsRoute.Consent -> {
            val vm: ConsentReviewViewModel = viewModel(key = "consent-review-$visit", factory = factory)
            ConsentReviewScreen(vm, onBack = back)
        }
    }
}

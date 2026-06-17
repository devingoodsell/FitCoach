package pro.d11l.fitcoach.feature.settings

import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.lifecycle.viewmodel.compose.viewModel
import pro.d11l.fitcoach.di.AppViewModelFactory

/** Editable Settings sections. PR1 ships profile/goals/schedule; later slices add
 *  preferences, diet, locations, aging emphases, and consent. */
enum class SettingsRoute { Hub, Profile, Goals, Schedule }

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
    }
}

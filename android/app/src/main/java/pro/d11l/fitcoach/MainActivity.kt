package pro.d11l.fitcoach

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import pro.d11l.fitcoach.core.designsystem.FitCoachTheme
import pro.d11l.fitcoach.di.AppViewModelFactory
import pro.d11l.fitcoach.feature.auth.AuthScreen
import pro.d11l.fitcoach.feature.auth.AuthViewModel
import pro.d11l.fitcoach.feature.consent.ConsentScreen
import pro.d11l.fitcoach.feature.consent.ConsentViewModel
import pro.d11l.fitcoach.feature.onboarding.OnboardingScreen
import pro.d11l.fitcoach.feature.onboarding.OnboardingViewModel
import pro.d11l.fitcoach.feature.settings.SettingsScreen
import pro.d11l.fitcoach.feature.settings.SettingsViewModel

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val container = (application as FitCoachApp).container
        val factory = AppViewModelFactory(container)
        val startLoggedIn = container.authRepository.isLoggedIn()

        setContent {
            FitCoachTheme {
                Surface(modifier = Modifier.fillMaxSize(), color = MaterialTheme.colorScheme.background) {
                    AppRoot(factory = factory, startLoggedIn = startLoggedIn)
                }
            }
        }
    }
}

private enum class Step { Auth, Consent, Onboarding, Home, Settings }

@Composable
private fun AppRoot(factory: AppViewModelFactory, startLoggedIn: Boolean) {
    var step by remember { mutableStateOf(if (startLoggedIn) Step.Home else Step.Auth) }
    // Bumped on logout/delete so a fresh AuthViewModel is created (no stale state).
    var epoch by remember { mutableIntStateOf(0) }

    when (step) {
        Step.Auth -> {
            val vm: AuthViewModel = viewModel(key = "auth-$epoch", factory = factory)
            val state by vm.state.collectAsStateWithLifecycle()
            LaunchedEffect(state.authenticated) {
                if (state.authenticated) step = Step.Consent
            }
            AuthScreen(vm)
        }
        Step.Consent -> {
            val vm: ConsentViewModel = viewModel(key = "consent-$epoch", factory = factory)
            ConsentScreen(vm) { step = Step.Onboarding }
        }
        Step.Onboarding -> {
            val vm: OnboardingViewModel = viewModel(key = "onboarding-$epoch", factory = factory)
            OnboardingScreen(vm) { step = Step.Home }
        }
        Step.Home -> HomeScreen(onOpenSettings = { step = Step.Settings })
        Step.Settings -> {
            val vm: SettingsViewModel = viewModel(key = "settings-$epoch", factory = factory)
            SettingsScreen(vm, onSignedOut = {
                epoch++
                step = Step.Auth
            })
        }
    }
}

@Composable
private fun HomeScreen(onOpenSettings: () -> Unit) {
    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text("FitCoach", style = MaterialTheme.typography.headlineMedium)
        Text(
            "You're signed in. Onboarding and your first workout arrive in the next milestone.",
            style = MaterialTheme.typography.bodyMedium,
        )
        Button(onClick = onOpenSettings, modifier = Modifier.fillMaxWidth()) {
            Text("Settings")
        }
    }
}

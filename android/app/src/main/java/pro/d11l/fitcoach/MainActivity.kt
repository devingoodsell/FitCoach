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
import pro.d11l.fitcoach.feature.diet.DietScreen
import pro.d11l.fitcoach.feature.diet.DietViewModel
import pro.d11l.fitcoach.feature.location.LocationScreen
import pro.d11l.fitcoach.feature.location.LocationViewModel
import pro.d11l.fitcoach.feature.injury.InjuryScreen
import pro.d11l.fitcoach.feature.injury.InjuryViewModel
import pro.d11l.fitcoach.feature.onboarding.OnboardingScreen
import pro.d11l.fitcoach.feature.onboarding.OnboardingViewModel
import pro.d11l.fitcoach.feature.readiness.ReadinessScreen
import pro.d11l.fitcoach.feature.readiness.ReadinessViewModel
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

private enum class Step { Auth, Consent, Onboarding, Home, Locations, Diet, Readiness, Injuries, Settings }

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
        Step.Home -> HomeScreen(
            onOpenSettings = { step = Step.Settings },
            onOpenLocations = { step = Step.Locations },
            onOpenDiet = { step = Step.Diet },
            onOpenReadiness = { step = Step.Readiness },
            onOpenInjuries = { step = Step.Injuries },
        )
        Step.Locations -> {
            val vm: LocationViewModel = viewModel(key = "locations-$epoch", factory = factory)
            LocationScreen(vm) { step = Step.Home }
        }
        Step.Diet -> {
            val vm: DietViewModel = viewModel(key = "diet-$epoch", factory = factory)
            DietScreen(vm) { step = Step.Home }
        }
        Step.Readiness -> {
            val vm: ReadinessViewModel = viewModel(key = "readiness-$epoch", factory = factory)
            ReadinessScreen(vm) { step = Step.Home }
        }
        Step.Injuries -> {
            val vm: InjuryViewModel = viewModel(key = "injuries-$epoch", factory = factory)
            InjuryScreen(vm) { step = Step.Home }
        }
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
private fun HomeScreen(
    onOpenSettings: () -> Unit,
    onOpenLocations: () -> Unit,
    onOpenDiet: () -> Unit,
    onOpenReadiness: () -> Unit,
    onOpenInjuries: () -> Unit,
) {
    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text("FitCoach", style = MaterialTheme.typography.headlineMedium)
        Text(
            "You're signed in. Your first generated workout arrives in a later milestone.",
            style = MaterialTheme.typography.bodyMedium,
        )
        Button(onClick = onOpenReadiness, modifier = Modifier.fillMaxWidth()) { Text("Readiness") }
        Button(onClick = onOpenInjuries, modifier = Modifier.fillMaxWidth()) { Text("Injuries") }
        Button(onClick = onOpenLocations, modifier = Modifier.fillMaxWidth()) { Text("Locations") }
        Button(onClick = onOpenDiet, modifier = Modifier.fillMaxWidth()) { Text("Nutrition") }
        Button(onClick = onOpenSettings, modifier = Modifier.fillMaxWidth()) { Text("Settings") }
    }
}

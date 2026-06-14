package pro.d11l.fitcoach.feature.onboarding

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.wrapContentWidth
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import pro.d11l.fitcoach.core.designsystem.MedicalDisclaimer

@Composable
fun OnboardingScreen(viewModel: OnboardingViewModel, onComplete: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.completed) {
        if (state.completed) onComplete()
    }

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        when (state.step) {
            OnboardingStep.Intro -> IntroStep(state, viewModel)
            OnboardingStep.Profile -> ProfileStep(state, viewModel)
            OnboardingStep.Goals -> GoalsStep(state, viewModel)
            OnboardingStep.Schedule -> ScheduleStep(state, viewModel)
            OnboardingStep.Diet -> DietStep(state, viewModel)
            OnboardingStep.Preferences -> PreferencesStep(state, viewModel)
            OnboardingStep.Done -> DoneStep(state, viewModel)
        }
        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
    }
}

@Composable
private fun IntroStep(state: OnboardingUiState, vm: OnboardingViewModel) {
    Text("Let's set up your coach", style = MaterialTheme.typography.headlineSmall)
    Text(
        "The core setup takes about $ESTIMATED_MINUTES minutes. You can skip optional " +
            "sections and finish them later.",
        style = MaterialTheme.typography.bodyMedium,
    )
    Row(verticalAlignment = Alignment.CenterVertically) {
        Checkbox(checked = state.somethingBothering, onCheckedChange = vm::onSomethingBothering)
        Text("Something is bothering me (pain or injury)")
    }
    MedicalDisclaimer(modifier = Modifier.fillMaxWidth())
    Button(onClick = vm::start, modifier = Modifier.fillMaxWidth()) { Text("Start") }
}

@Composable
private fun ProfileStep(state: OnboardingUiState, vm: OnboardingViewModel) {
    StepTitle("About you", "Step 1 of 3")
    Text("Biological sex", style = MaterialTheme.typography.labelLarge)
    ChipRow(listOf("male", "female", "other"), state.sex, vm::onSex)
    fieldError(state, "sex")

    OutlinedTextField(
        value = state.age,
        onValueChange = vm::onAge,
        label = { Text("Age") },
        singleLine = true,
        isError = state.fieldErrors.containsKey("age"),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        modifier = Modifier.fillMaxWidth(),
    )
    fieldError(state, "age")

    OutlinedTextField(
        value = state.heightCm,
        onValueChange = vm::onHeight,
        label = { Text("Height cm (optional)") },
        singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        modifier = Modifier.fillMaxWidth(),
    )
    OutlinedTextField(
        value = state.weightKg,
        onValueChange = vm::onWeight,
        label = { Text("Weight kg (optional)") },
        singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        modifier = Modifier.fillMaxWidth(),
    )

    Text("Experience", style = MaterialTheme.typography.labelLarge)
    ChipRow(listOf("novice", "intermediate", "advanced"), state.level, vm::onLevel)
    fieldError(state, "experience.level")

    NavButtons(onBack = vm::back, onNext = vm::submitProfile, nextEnabled = !state.isSubmitting)
}

@Composable
private fun GoalsStep(state: OnboardingUiState, vm: OnboardingViewModel) {
    StepTitle("Your goals", "Step 2 of 3")
    Text("Weight what matters to you. We'll balance these in every session.")
    SliderRow("Strength", state.strength, vm::onStrength)
    SliderRow("Healthspan", state.healthspan, vm::onHealthspan)
    SliderRow("Body composition", state.bodyComp, vm::onBodyComp)
    SliderRow("Performance", state.performance, vm::onPerformance)
    NavButtons(onBack = vm::back, onNext = vm::submitGoals, nextEnabled = !state.isSubmitting)
}

@Composable
private fun ScheduleStep(state: OnboardingUiState, vm: OnboardingViewModel) {
    StepTitle("Your schedule", "Step 3 of 3")
    OutlinedTextField(
        value = state.daysPerWeek,
        onValueChange = vm::onDaysPerWeek,
        label = { Text("Days per week") },
        singleLine = true,
        isError = state.fieldErrors.containsKey("days_per_week"),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        modifier = Modifier.fillMaxWidth(),
    )
    fieldError(state, "days_per_week")
    OutlinedTextField(
        value = state.sessionLengthMin,
        onValueChange = vm::onSessionLength,
        label = { Text("Session length (minutes)") },
        singleLine = true,
        isError = state.fieldErrors.containsKey("session_length_min"),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        modifier = Modifier.fillMaxWidth(),
    )
    fieldError(state, "session_length_min")
    NavButtons(onBack = vm::back, onNext = vm::submitSchedule, nextEnabled = !state.isSubmitting)
}

@Composable
private fun DietStep(state: OnboardingUiState, vm: OnboardingViewModel) {
    StepTitle("Diet (optional)", "")
    Text("Helps us tailor nutrition guidance. You can skip this.")
    ChipRow(
        listOf("omnivore", "vegan", "vegetarian", "pescatarian", "kosher", "halal"),
        state.dietPattern,
        vm::onDietPattern,
    )
    fieldError(state, "pattern")
    OutlinedTextField(
        value = state.supplements,
        onValueChange = vm::onSupplements,
        label = { Text("Supplements / medications (optional)") },
        modifier = Modifier.fillMaxWidth(),
    )
    SkipNavButtons(onBack = vm::back, onSkip = vm::skipDiet, onNext = vm::submitDiet, enabled = !state.isSubmitting)
}

@Composable
private fun PreferencesStep(state: OnboardingUiState, vm: OnboardingViewModel) {
    StepTitle("Preferences (optional)", "")
    Text("Comma-separated. Hard-avoids are treated as constraints, not just dislikes.")
    OutlinedTextField(state.likes, vm::onLikes, label = { Text("Likes") }, modifier = Modifier.fillMaxWidth())
    OutlinedTextField(state.dislikes, vm::onDislikes, label = { Text("Dislikes") }, modifier = Modifier.fillMaxWidth())
    OutlinedTextField(state.hardAvoids, vm::onHardAvoids, label = { Text("Hard avoids") }, modifier = Modifier.fillMaxWidth())
    SkipNavButtons(onBack = vm::back, onSkip = vm::skipPreferences, onNext = vm::submitPreferences, enabled = !state.isSubmitting)
}

@Composable
private fun DoneStep(state: OnboardingUiState, vm: OnboardingViewModel) {
    Text("You're all set", style = MaterialTheme.typography.headlineSmall)
    Text("Your coach now has what it needs to plan your first session.")
    if (state.somethingBothering) {
        Text(
            "You mentioned something is bothering you — you'll be able to add injury details soon.",
            style = MaterialTheme.typography.bodyMedium,
        )
    }
    Button(onClick = vm::finish, modifier = Modifier.fillMaxWidth()) { Text("Finish setup") }
}

// --- shared pieces ---

@Composable
private fun StepTitle(title: String, subtitle: String) {
    Text(title, style = MaterialTheme.typography.headlineSmall)
    if (subtitle.isNotEmpty()) {
        Text(subtitle, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ChipRow(options: List<String>, selected: String, onSelect: (String) -> Unit) {
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        options.forEach { option ->
            FilterChip(
                selected = selected == option,
                onClick = { onSelect(option) },
                label = { Text(option.replaceFirstChar(Char::uppercase)) },
            )
        }
    }
}

@Composable
private fun SliderRow(label: String, value: Float, onChange: (Float) -> Unit) {
    Column {
        Text("$label: ${(value * 100).toInt()}%", style = MaterialTheme.typography.bodyMedium)
        Slider(value = value, onValueChange = onChange, valueRange = 0f..1f)
    }
}

@Composable
private fun fieldError(state: OnboardingUiState, key: String) {
    state.fieldErrors[key]?.let {
        Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
    }
}

@Composable
private fun NavButtons(onBack: () -> Unit, onNext: () -> Unit, nextEnabled: Boolean) {
    Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
        OutlinedButton(onClick = onBack, modifier = Modifier.weight(1f)) { Text("Back") }
        Button(onClick = onNext, enabled = nextEnabled, modifier = Modifier.weight(1f)) { Text("Continue") }
    }
}

@Composable
private fun SkipNavButtons(onBack: () -> Unit, onSkip: () -> Unit, onNext: () -> Unit, enabled: Boolean) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
            OutlinedButton(onClick = onBack, modifier = Modifier.weight(1f)) { Text("Back") }
            Button(onClick = onNext, enabled = enabled, modifier = Modifier.weight(1f)) { Text("Continue") }
        }
        TextButton(onClick = onSkip, modifier = Modifier.wrapContentWidth()) { Text("Skip for now") }
    }
}

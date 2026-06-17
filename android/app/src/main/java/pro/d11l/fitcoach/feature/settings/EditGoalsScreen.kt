package pro.d11l.fitcoach.feature.settings

import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.lifecycle.compose.collectAsStateWithLifecycle

@Composable
fun EditGoalsScreen(viewModel: EditGoalsViewModel, onDone: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.saved) {
        if (state.saved) onDone()
    }

    SettingsEditScaffold(
        title = "Goals",
        isSaving = state.isSaving,
        onBack = onDone,
        onSave = viewModel::save,
        error = state.error,
    ) {
        Text("Weight what matters to you. We'll balance these in every session.")
        SettingsSliderRow("Strength", state.strength, viewModel::onStrength)
        SettingsSliderRow("Healthspan", state.healthspan, viewModel::onHealthspan)
        SettingsSliderRow("Body composition", state.bodyComp, viewModel::onBodyComp)
        SettingsSliderRow("Performance", state.performance, viewModel::onPerformance)
    }
}

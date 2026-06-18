package pro.d11l.fitcoach.feature.settings

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle

@Composable
fun EditDietScreen(viewModel: EditDietViewModel, onDone: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.saved) {
        if (state.saved) onDone()
    }

    SettingsEditScaffold(
        title = "Diet",
        isSaving = state.isSaving,
        onBack = onDone,
        onSave = viewModel::save,
        error = state.error,
    ) {
        Text("Dietary pattern")
        SettingsChipRow(DIET_PATTERNS, state.pattern, viewModel::onPattern)
        SettingsFieldError(state.fieldErrors, "pattern")
        OutlinedTextField(
            value = state.supplements,
            onValueChange = viewModel::onSupplements,
            label = { Text("Supplements (optional)") },
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedTextField(
            value = state.medications,
            onValueChange = viewModel::onMedications,
            label = { Text("Medications (optional)") },
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

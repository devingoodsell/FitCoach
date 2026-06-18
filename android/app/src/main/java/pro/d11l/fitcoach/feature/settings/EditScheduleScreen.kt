package pro.d11l.fitcoach.feature.settings

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.lifecycle.compose.collectAsStateWithLifecycle

@Composable
fun EditScheduleScreen(viewModel: EditScheduleViewModel, onDone: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.saved) {
        if (state.saved) onDone()
    }

    SettingsEditScaffold(
        title = "Schedule",
        isSaving = state.isSaving,
        onBack = onDone,
        onSave = viewModel::save,
        error = state.error,
    ) {
        OutlinedTextField(
            value = state.daysPerWeek,
            onValueChange = viewModel::onDaysPerWeek,
            label = { Text("Days per week") },
            singleLine = true,
            isError = state.fieldErrors.containsKey("days_per_week"),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        SettingsFieldError(state.fieldErrors, "days_per_week")
        OutlinedTextField(
            value = state.sessionLengthMin,
            onValueChange = viewModel::onSessionLength,
            label = { Text("Session length (minutes)") },
            singleLine = true,
            isError = state.fieldErrors.containsKey("session_length_min"),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        SettingsFieldError(state.fieldErrors, "session_length_min")
    }
}

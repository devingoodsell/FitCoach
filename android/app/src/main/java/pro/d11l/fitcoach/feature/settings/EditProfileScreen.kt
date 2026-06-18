package pro.d11l.fitcoach.feature.settings

import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.lifecycle.compose.collectAsStateWithLifecycle

@Composable
fun EditProfileScreen(viewModel: EditProfileViewModel, onDone: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.saved) {
        if (state.saved) onDone()
    }

    SettingsEditScaffold(
        title = "Profile",
        isSaving = state.isSaving,
        onBack = onDone,
        onSave = viewModel::save,
        error = state.error,
    ) {
        Text("Biological sex", style = MaterialTheme.typography.labelLarge)
        SettingsChipRow(listOf("male", "female", "other"), state.sex, viewModel::onSex)
        SettingsFieldError(state.fieldErrors, "sex")

        OutlinedTextField(
            value = state.age,
            onValueChange = viewModel::onAge,
            label = { Text("Age") },
            singleLine = true,
            isError = state.fieldErrors.containsKey("age"),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        SettingsFieldError(state.fieldErrors, "age")

        OutlinedTextField(
            value = state.heightCm,
            onValueChange = viewModel::onHeight,
            label = { Text("Height cm (optional)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedTextField(
            value = state.weightKg,
            onValueChange = viewModel::onWeight,
            label = { Text("Weight kg (optional)") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
            modifier = Modifier.fillMaxWidth(),
        )

        Text("Experience", style = MaterialTheme.typography.labelLarge)
        SettingsChipRow(listOf("novice", "intermediate", "advanced"), state.level, viewModel::onLevel)
        SettingsFieldError(state.fieldErrors, "experience.level")
    }
}

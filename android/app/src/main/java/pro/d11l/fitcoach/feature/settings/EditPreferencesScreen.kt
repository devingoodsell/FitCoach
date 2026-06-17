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
fun EditPreferencesScreen(viewModel: EditPreferencesViewModel, onDone: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.saved) {
        if (state.saved) onDone()
    }

    SettingsEditScaffold(
        title = "Preferences",
        isSaving = state.isSaving,
        onBack = onDone,
        onSave = viewModel::save,
        error = state.error,
    ) {
        Text("Comma-separated. Hard-avoids are treated as constraints, not just dislikes.")
        OutlinedTextField(state.likes, viewModel::onLikes, label = { Text("Likes") }, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(state.dislikes, viewModel::onDislikes, label = { Text("Dislikes") }, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(state.hardAvoids, viewModel::onHardAvoids, label = { Text("Hard avoids") }, modifier = Modifier.fillMaxWidth())
    }
}

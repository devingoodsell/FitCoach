package pro.d11l.fitcoach.feature.consent

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import pro.d11l.fitcoach.core.designsystem.Disclaimers
import pro.d11l.fitcoach.core.designsystem.MedicalDisclaimer

@Composable
fun ConsentScreen(viewModel: ConsentViewModel, onDecided: (manualMode: Boolean) -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.decided) {
        if (state.decided) onDecided(state.manualMode)
    }

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text("Your health data", style = MaterialTheme.typography.headlineSmall)
        Text(Disclaimers.HEALTH_DATA, style = MaterialTheme.typography.bodyMedium)
        MedicalDisclaimer(modifier = Modifier.fillMaxWidth())

        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        Button(
            onClick = viewModel::allowHealthData,
            enabled = !state.isSubmitting,
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Allow health data") }

        OutlinedButton(
            onClick = viewModel::useManualMode,
            enabled = !state.isSubmitting,
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Use manual mode instead") }
    }
}

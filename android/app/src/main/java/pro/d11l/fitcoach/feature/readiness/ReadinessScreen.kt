package pro.d11l.fitcoach.feature.readiness

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import pro.d11l.fitcoach.core.designsystem.MedicalDisclaimer

@Composable
fun ReadinessScreen(viewModel: ReadinessViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text("Today's readiness", style = MaterialTheme.typography.headlineSmall)
        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        when {
            state.loading -> Text("Loading…", style = MaterialTheme.typography.bodyMedium)
            state.isUnavailable || state.readiness == null ->
                Text(
                    "Not enough recovery data yet. Connect a wearable through Health Connect " +
                        "(coming soon) or check back after a few nights of data. You can train in " +
                        "manual mode in the meantime.",
                    style = MaterialTheme.typography.bodyMedium,
                )
            else -> {
                val r = state.readiness!!
                Card(modifier = Modifier.fillMaxWidth()) {
                    Column(modifier = Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("${r.value}/100", style = MaterialTheme.typography.displaySmall)
                        Text("Confidence: ${r.confidence}", style = MaterialTheme.typography.labelMedium)
                        Text(r.explanation, style = MaterialTheme.typography.bodyMedium)
                    }
                }
            }
        }

        MedicalDisclaimer(modifier = Modifier.fillMaxWidth())
        OutlinedButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Back") }
    }
}

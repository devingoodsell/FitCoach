package pro.d11l.fitcoach.feature.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
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
import pro.d11l.fitcoach.core.network.ConsentRecord
import pro.d11l.fitcoach.data.ConsentTypes

@Composable
fun ConsentReviewScreen(viewModel: ConsentReviewViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text("Consent & disclaimers", style = MaterialTheme.typography.headlineSmall)

        MedicalDisclaimer(modifier = Modifier.fillMaxWidth())

        if (state.loading) {
            Text("Loading…")
        } else if (state.consents.isEmpty()) {
            Text("You haven't recorded any consents yet.", style = MaterialTheme.typography.bodyMedium)
        } else {
            state.consents.forEach { ConsentCard(it) }
        }

        if (state.healthDataActive) {
            Text(
                "Revoking health-data consent stops FitCoach from reading sleep, resting heart " +
                    "rate, and HRV. Readiness switches to manual mode; you can re-enable anytime.",
                style = MaterialTheme.typography.bodySmall,
            )
            Button(
                onClick = viewModel::revokeHealthData,
                enabled = !state.isWorking,
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Revoke health-data consent") }
        } else if (state.healthData != null) {
            Text(
                "Health-data consent is revoked — readiness is in manual mode.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.primary,
            )
        }

        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        OutlinedButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Back") }
    }
}

@Composable
private fun ConsentCard(record: ConsentRecord) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(label(record.type), style = MaterialTheme.typography.titleMedium)
            Text("Version ${record.version}", style = MaterialTheme.typography.bodySmall)
            val status = if (record.isActive) "Active" else "Revoked"
            Text(
                status,
                style = MaterialTheme.typography.labelLarge,
                color = if (record.isActive) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.error,
            )
        }
    }
}

private fun label(type: String): String = when (type) {
    ConsentTypes.HEALTH_DATA -> "Health data"
    ConsentTypes.MEDICAL_DISCLAIMER -> "Medical disclaimer"
    else -> type
}

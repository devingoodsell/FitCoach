package pro.d11l.fitcoach.feature.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import pro.d11l.fitcoach.core.designsystem.LocalDisclaimers

/**
 * Read-only view of the current disclaimer copy, retrievable from Settings (E13-PR2
 * / E14-S2). Text is sourced from the app-level [LocalDisclaimers] (the server's
 * GET /disclaimers document, with the bundled fallback offline).
 */
@Composable
fun DisclaimerScreen(onBack: () -> Unit) {
    val disclaimers = LocalDisclaimers.current

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text("Disclaimers", style = MaterialTheme.typography.headlineSmall)
        Text("Version ${disclaimers.version}", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)

        Text("Medical", style = MaterialTheme.typography.labelLarge)
        Text(disclaimers.medical, style = MaterialTheme.typography.bodyMedium)

        Text("Health data", style = MaterialTheme.typography.labelLarge)
        Text(disclaimers.healthData, style = MaterialTheme.typography.bodyMedium)

        OutlinedButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Back") }
    }
}

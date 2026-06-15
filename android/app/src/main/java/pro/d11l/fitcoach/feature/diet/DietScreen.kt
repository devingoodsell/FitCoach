package pro.d11l.fitcoach.feature.diet

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
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
fun DietScreen(viewModel: DietViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text("Nutrition", style = MaterialTheme.typography.headlineSmall)
        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        state.targets?.let { t ->
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text("Daily targets", style = MaterialTheme.typography.titleMedium)
                    if (t.targets.lowConfidence) {
                        Text(
                            "Add your weight and height in your profile for personalized numbers.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.primary,
                        )
                    } else {
                        Text("Calories: ${t.targets.caloriesMin}–${t.targets.caloriesMax} kcal")
                        Text("Protein: ${t.targets.proteinMinG}–${t.targets.proteinMaxG} g")
                    }
                    if (t.pattern.isNotEmpty()) {
                        Text("Pattern: ${t.pattern}", style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
            t.guidance.forEach { line ->
                Text("• $line", style = MaterialTheme.typography.bodyMedium)
            }
        }

        state.note?.let { Text(it, style = MaterialTheme.typography.bodyMedium) }

        MedicalDisclaimer(modifier = Modifier.fillMaxWidth())
        OutlinedButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Back") }
    }
}

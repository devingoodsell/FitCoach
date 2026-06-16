package pro.d11l.fitcoach.feature.injury

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import pro.d11l.fitcoach.core.designsystem.MedicalDisclaimer
import pro.d11l.fitcoach.core.network.InjuryDto

@Composable
fun InjuryScreen(viewModel: InjuryViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text("Injuries & conditions", style = MaterialTheme.typography.headlineSmall)
        MedicalDisclaimer(modifier = Modifier.fillMaxWidth())
        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        state.injuries.forEach { inj ->
            InjuryCard(inj, onStatus = { viewModel.setStatus(inj, it) }, onDelete = { viewModel.delete(inj.id) })
        }
        if (!state.loading && state.injuries.isEmpty()) {
            Text("No injuries recorded.", style = MaterialTheme.typography.bodyMedium)
        }

        if (!state.draftVisible) {
            Text("Describe something that's bothering you", style = MaterialTheme.typography.titleMedium)
            OutlinedTextField(
                value = state.freeText,
                onValueChange = viewModel::onFreeText,
                label = { Text("e.g. my left knee hurts when I squat") },
                modifier = Modifier.fillMaxWidth(),
            )
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
                Button(onClick = viewModel::parse, modifier = Modifier.weight(1f)) { Text("Parse") }
                OutlinedButton(onClick = viewModel::startManual, modifier = Modifier.weight(1f)) { Text("Add manually") }
            }
        } else {
            DraftForm(state, viewModel)
        }

        OutlinedButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Back") }
    }
}

@Composable
private fun DraftForm(state: InjuryUiState, vm: InjuryViewModel) {
    Text("Review & save", style = MaterialTheme.typography.titleMedium)
    if (state.lowConfidenceFields.isNotEmpty()) {
        Text(
            "Please double-check: ${state.lowConfidenceFields.joinToString(", ")}",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.primary,
        )
    }
    OutlinedTextField(state.region, vm::onRegion, label = { Text("Region") }, modifier = Modifier.fillMaxWidth())

    Text("Status", style = MaterialTheme.typography.labelLarge)
    ChipRow(INJURY_STATUSES, state.status, vm::onStatus)
    Text("Severity", style = MaterialTheme.typography.labelLarge)
    ChipRow(SEVERITIES, state.severity, vm::onSeverity)

    OutlinedTextField(state.aggravating, vm::onAggravating, label = { Text("Aggravating movements (comma-separated)") }, modifier = Modifier.fillMaxWidth())
    OutlinedTextField(state.notes, vm::onNotes, label = { Text("Notes") }, modifier = Modifier.fillMaxWidth())

    Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
        OutlinedButton(onClick = vm::cancelDraft, modifier = Modifier.weight(1f)) { Text("Cancel") }
        Button(onClick = vm::saveDraft, enabled = !state.saving, modifier = Modifier.weight(1f)) { Text("Save") }
    }
}

@Composable
private fun InjuryCard(injury: InjuryDto, onStatus: (String) -> Unit, onDelete: () -> Unit) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(injury.region.ifEmpty { "Unspecified region" }, style = MaterialTheme.typography.titleMedium)
            Text("Severity: ${injury.severity}", style = MaterialTheme.typography.bodySmall)
            ChipRow(INJURY_STATUSES, injury.status, onStatus)
            TextButton(onClick = onDelete) { Text("Delete") }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ChipRow(options: List<String>, selected: String, onSelect: (String) -> Unit) {
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        options.forEach { option ->
            FilterChip(
                selected = selected == option,
                onClick = { onSelect(option) },
                label = { Text(option.replace('_', ' ')) },
            )
        }
    }
}

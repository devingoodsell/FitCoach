package pro.d11l.fitcoach.feature.location

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import pro.d11l.fitcoach.core.network.LocationDto

@Composable
fun LocationScreen(viewModel: LocationViewModel, onBack: () -> Unit) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    var name by remember { mutableStateOf("") }
    var equipment by remember { mutableStateOf("") }

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text("Training locations", style = MaterialTheme.typography.headlineSmall)
        state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

        state.locations.forEach { loc ->
            LocationCard(
                location = loc,
                isCurrent = state.current?.locationId == loc.id,
                onMakeCurrent = { viewModel.setCurrent(loc.id, "") },
                onDelete = { viewModel.deleteLocation(loc.id) },
            )
        }
        if (!state.loading && state.locations.isEmpty()) {
            Text("No locations yet. Add one below.", style = MaterialTheme.typography.bodyMedium)
        }

        Text("Add a location", style = MaterialTheme.typography.titleMedium)
        OutlinedTextField(name, { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(
            equipment,
            { equipment = it },
            label = { Text("Equipment (comma-separated)") },
            modifier = Modifier.fillMaxWidth(),
        )
        Button(
            onClick = {
                viewModel.addLocation(name, equipment)
                name = ""
                equipment = ""
            },
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Add location") }

        OutlinedButton(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("Back") }
    }
}

@Composable
private fun LocationCard(
    location: LocationDto,
    isCurrent: Boolean,
    onMakeCurrent: () -> Unit,
    onDelete: () -> Unit,
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(location.name, style = MaterialTheme.typography.titleMedium)
            Text(
                if (location.equipment.isEmpty()) "No equipment listed"
                else location.equipment.joinToString(", "),
                style = MaterialTheme.typography.bodySmall,
            )
            if (isCurrent) {
                Text("Current context", color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.labelMedium)
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                if (!isCurrent) {
                    TextButton(onClick = onMakeCurrent) { Text("Set current") }
                }
                TextButton(onClick = onDelete) { Text("Delete") }
            }
        }
    }
}

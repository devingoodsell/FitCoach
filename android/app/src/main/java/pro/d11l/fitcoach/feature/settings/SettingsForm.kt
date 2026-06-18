package pro.d11l.fitcoach.feature.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

/**
 * Shared layout for a Settings edit screen: a scrolling form with a title, the
 * caller's fields, an optional error, and a Back / Save action row. Kept local to
 * the settings package so each editor stays a thin, stateless composable.
 */
@Composable
internal fun SettingsEditScaffold(
    title: String,
    isSaving: Boolean,
    onBack: () -> Unit,
    onSave: () -> Unit,
    error: String?,
    content: @Composable ColumnScope.() -> Unit,
) {
    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text(title, style = MaterialTheme.typography.headlineSmall)
        content()
        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
            OutlinedButton(onClick = onBack, enabled = !isSaving, modifier = Modifier.weight(1f)) { Text("Back") }
            Button(onClick = onSave, enabled = !isSaving, modifier = Modifier.weight(1f)) { Text("Save") }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
internal fun SettingsChipRow(options: List<String>, selected: String, onSelect: (String) -> Unit) {
    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        options.forEach { option ->
            FilterChip(
                selected = selected == option,
                onClick = { onSelect(option) },
                label = { Text(option.replaceFirstChar(Char::uppercase)) },
            )
        }
    }
}

@Composable
internal fun SettingsSliderRow(label: String, value: Float, onChange: (Float) -> Unit) {
    Column {
        Text("$label: ${(value * 100).toInt()}%", style = MaterialTheme.typography.bodyMedium)
        Slider(value = value, onValueChange = onChange, valueRange = 0f..1f)
    }
}

@Composable
internal fun SettingsFieldError(fieldErrors: Map<String, String>, key: String) {
    fieldErrors[key]?.let {
        Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
    }
}

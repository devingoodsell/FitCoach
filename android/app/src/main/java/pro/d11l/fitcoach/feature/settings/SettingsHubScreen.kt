package pro.d11l.fitcoach.feature.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle

/**
 * Settings landing screen (E14): lists every editable user-model section and the
 * account actions. Navigation between sub-editors is owned by [SettingsRoot]; this
 * screen only emits navigation intents and handles sign-out / account deletion.
 */
@Composable
fun SettingsHubScreen(
    viewModel: SettingsViewModel,
    onSignedOut: () -> Unit,
    onNavigate: (SettingsRoute) -> Unit,
) {
    val state by viewModel.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.signedOut) {
        if (state.signedOut) onSignedOut()
    }

    Column(
        modifier = Modifier.fillMaxSize().padding(24.dp).verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text("Settings", style = MaterialTheme.typography.headlineSmall)

        Text("Your coach", style = MaterialTheme.typography.labelLarge)
        SettingsNavItem("Profile & physiology") { onNavigate(SettingsRoute.Profile) }
        SettingsNavItem("Goals") { onNavigate(SettingsRoute.Goals) }
        SettingsNavItem("Schedule") { onNavigate(SettingsRoute.Schedule) }
        SettingsNavItem("Preferences") { onNavigate(SettingsRoute.Preferences) }
        SettingsNavItem("Diet") { onNavigate(SettingsRoute.Diet) }
        SettingsNavItem("Locations & current context") { onNavigate(SettingsRoute.Locations) }
        SettingsNavItem("Aging emphases") { onNavigate(SettingsRoute.Aging) }

        HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))

        Text("About", style = MaterialTheme.typography.labelLarge)
        SettingsNavItem("Consent & disclaimers") { onNavigate(SettingsRoute.Consent) }
        SettingsNavItem("Disclaimer text") { onNavigate(SettingsRoute.Disclaimers) }

        HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))

        Text("Account", style = MaterialTheme.typography.labelLarge)
        OutlinedButton(
            onClick = viewModel::logout,
            enabled = !state.isWorking,
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Log out") }

        Button(
            onClick = viewModel::startDelete,
            enabled = !state.isWorking,
            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
            modifier = Modifier.fillMaxWidth(),
        ) { Text("Delete account") }
    }

    if (state.confirmingDelete) {
        AlertDialog(
            onDismissRequest = viewModel::cancelDelete,
            title = { Text("Delete account?") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        "This permanently removes your account, Coach Memory, history, and " +
                            "ingested health data from our servers and this device. This cannot be undone.",
                    )
                    OutlinedTextField(
                        value = state.deletePassword,
                        onValueChange = viewModel::onDeletePasswordChange,
                        label = { Text("Confirm password") },
                        singleLine = true,
                        visualTransformation = PasswordVisualTransformation(),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                }
            },
            confirmButton = {
                TextButton(onClick = viewModel::confirmDelete, enabled = !state.isWorking) {
                    Text("Delete")
                }
            },
            dismissButton = {
                TextButton(onClick = viewModel::cancelDelete) { Text("Cancel") }
            },
        )
    }
}

@Composable
private fun SettingsNavItem(label: String, onClick: () -> Unit) {
    OutlinedButton(onClick = onClick, modifier = Modifier.fillMaxWidth()) { Text(label) }
}

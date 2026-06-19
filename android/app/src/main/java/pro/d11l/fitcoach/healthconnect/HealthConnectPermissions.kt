package pro.d11l.fitcoach.healthconnect

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContract
import androidx.compose.runtime.Composable
import androidx.health.connect.client.PermissionController

/**
 * Remembers a launcher that requests the Health Connect read permissions (sleep,
 * RHR, HRV), routing through the system rationale screen declared in the manifest.
 * [onResult] receives the set the user actually granted — empty / partial means
 * the caller should degrade to manual / no-readiness mode (see HealthSignalsRepository).
 *
 * Device-only glue: the dialog flow can't be exercised in a headless build.
 */
@Composable
fun rememberHealthConnectPermissionLauncher(
    onResult: (granted: Set<String>) -> Unit,
): () -> Unit {
    val contract: ActivityResultContract<Set<String>, Set<String>> =
        PermissionController.createRequestPermissionResultContract()
    val launcher = rememberLauncherForActivityResult(contract) { granted -> onResult(granted) }
    return { launcher.launch(HealthConnectSource.PERMISSIONS) }
}

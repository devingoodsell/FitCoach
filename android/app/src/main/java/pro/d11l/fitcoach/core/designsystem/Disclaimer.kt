package pro.d11l.fitcoach.core.designsystem

import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

/**
 * Centrally-managed disclaimer copy (E13-S1). Every body/health-related surface
 * shows this language; keep the single source of truth here so it can be audited
 * and updated in one place.
 */
object Disclaimers {
    const val VERSION = "v1"

    const val MEDICAL =
        "FitCoach provides general fitness guidance, not medical advice. It is not a " +
            "substitute for professional diagnosis or treatment. Consult a qualified " +
            "clinician before starting a program or if you have pain, an injury, or a " +
            "health condition."

    const val HEALTH_DATA =
        "With your permission, FitCoach reads sleep, resting heart rate, and heart-rate " +
            "variability from Health Connect to estimate daily readiness and tailor your " +
            "training. This data is stored with your account and used only for coaching. " +
            "You can decline and use manual mode, and revoke access at any time."
}

/** Inline medical-disclaimer banner for any health-involved screen. */
@Composable
fun MedicalDisclaimer(modifier: Modifier = Modifier) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceVariant,
        modifier = modifier,
    ) {
        Text(
            text = Disclaimers.MEDICAL,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.padding(12.dp),
        )
    }
}

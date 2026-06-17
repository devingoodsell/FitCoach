package pro.d11l.fitcoach.core.designsystem

import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

/**
 * Centrally-managed disclaimer copy (E13-S1). The server (GET /disclaimers) is the
 * source of truth; these bundled constants are the offline / first-frame fallback so
 * the language renders before the fetch completes and when there's no connectivity.
 * Keep them in sync with backend/internal/disclaimer.
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

/** The disclaimer text in effect for this composition, sourced from the backend
 *  (E13-PR2) and provided at the app root. Defaults to the bundled fallback. */
data class DisclaimerText(
    val version: String = Disclaimers.VERSION,
    val medical: String = Disclaimers.MEDICAL,
    val healthData: String = Disclaimers.HEALTH_DATA,
) {
    companion object {
        /** The bundled offline fallback (matches the constants above). */
        val Bundled = DisclaimerText()
    }
}

/** Disclaimer text available to any composable. Provided at the app root from the
 *  fetched [GET /disclaimers] document; falls back to [DisclaimerText.Bundled]. */
val LocalDisclaimers = staticCompositionLocalOf { DisclaimerText.Bundled }

/** Inline medical-disclaimer banner for any health-involved screen. Reads the
 *  current disclaimer text from [LocalDisclaimers], so every surface stays in sync
 *  with the server copy without each call site fetching it. */
@Composable
fun MedicalDisclaimer(modifier: Modifier = Modifier) {
    Surface(
        color = MaterialTheme.colorScheme.surfaceVariant,
        modifier = modifier,
    ) {
        Text(
            text = LocalDisclaimers.current.medical,
            style = MaterialTheme.typography.bodySmall,
            modifier = Modifier.padding(12.dp),
        )
    }
}

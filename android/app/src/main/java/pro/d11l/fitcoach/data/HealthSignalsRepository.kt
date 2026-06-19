package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.HealthSignalsRequest
import pro.d11l.fitcoach.healthconnect.RecoverySignalSource
import pro.d11l.fitcoach.healthconnect.SignalSourceStatus
import java.time.Duration
import java.time.Instant

/**
 * Outcome of an ingestion attempt. Everything except [Uploaded] is a graceful
 * degrade — the app falls back to the manual / no-readiness state (E4-PR6) rather
 * than failing hard.
 */
sealed interface IngestResult {
    /** Samples were read and accepted by the backend. */
    data class Uploaded(val sampleCount: Int) : IngestResult

    /** Health-data consent (E1) is not in force; Health Connect was never touched. */
    data object NoConsent : IngestResult

    /** Health Connect isn't installed/supported on this device. */
    data object Unavailable : IngestResult

    /** The user hasn't granted the read permissions yet. */
    data object PermissionsRequired : IngestResult

    /** Permissions are granted but nothing was recorded in the window. */
    data object NoData : IngestResult

    /** Reading or uploading failed. */
    data class Error(val message: String?) : IngestResult
}

/**
 * Reads recovery signals from Health Connect and uploads them to the backend,
 * which computes its own readiness from them (E4-PR5 / E4-S1).
 *
 * Consent is checked **first** so a read never happens without health-data consent
 * (E1-S4); a consent check that fails or comes back negative is treated as
 * [IngestResult.NoConsent] — we never read health data on an unconfirmed grant.
 * Any other gap (no provider, missing permission, no samples) degrades to a
 * manual / no-readiness outcome instead of throwing.
 */
class HealthSignalsRepository(
    private val source: RecoverySignalSource,
    private val api: FitCoachApi,
    private val consent: ConsentRepository,
    private val lookback: Duration = Duration.ofDays(LOOKBACK_DAYS),
) {

    suspend fun ingest(now: Instant): IngestResult {
        if (!healthConsentActive()) return IngestResult.NoConsent
        if (source.status() != SignalSourceStatus.AVAILABLE) return IngestResult.Unavailable
        if (!source.hasAllPermissions()) return IngestResult.PermissionsRequired

        val samples = runCatching { source.read(now.minus(lookback), now) }
            .getOrElse { return IngestResult.Error(it.message) }
        if (samples.isEmpty()) return IngestResult.NoData

        return runCatching {
            val resp = api.uploadHealthSignals(HealthSignalsRequest(samples))
            if (resp.isSuccessful) {
                IngestResult.Uploaded(samples.size)
            } else {
                IngestResult.Error("upload failed (${resp.code()})")
            }
        }.getOrElse { IngestResult.Error(it.message) }
    }

    private suspend fun healthConsentActive(): Boolean =
        consent.load().getOrDefault(emptyList())
            .any { it.type == ConsentTypes.HEALTH_DATA && it.isActive }

    companion object {
        /** Trailing window read so the backend has history for its rolling baseline. */
        const val LOOKBACK_DAYS = 7L
    }
}

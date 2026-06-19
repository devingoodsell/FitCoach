package pro.d11l.fitcoach.healthconnect

import pro.d11l.fitcoach.core.network.HealthSignalDto
import java.time.Instant

/** Availability of the on-device recovery-signal provider (Health Connect). */
enum class SignalSourceStatus {
    /** Installed and usable. */
    AVAILABLE,

    /** Provider present but needs a Play-store update before it can be used. */
    NOT_INSTALLED,

    /** Health Connect isn't supported on this device at all. */
    NOT_SUPPORTED,
}

/**
 * Consumer-defined seam over the on-device signal provider. The real
 * implementation ([HealthConnectSource]) carries the `androidx.health`
 * dependency and the permission/device concerns; tests substitute a fake so the
 * ingestion + graceful-degrade logic stays JVM-unit-testable.
 */
interface RecoverySignalSource {

    /** Whether Health Connect is installed/usable on this device. */
    suspend fun status(): SignalSourceStatus

    /** True only when **all** read permissions (sleep, RHR, HRV) are granted. */
    suspend fun hasAllPermissions(): Boolean

    /** Reads and aggregates samples in `[start, end]`; empty when nothing recorded. */
    suspend fun read(start: Instant, end: Instant): List<HealthSignalDto>
}

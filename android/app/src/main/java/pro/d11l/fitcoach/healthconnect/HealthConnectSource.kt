package pro.d11l.fitcoach.healthconnect

import android.content.Context
import androidx.health.connect.client.HealthConnectClient
import androidx.health.connect.client.permission.HealthPermission
import androidx.health.connect.client.records.HeartRateVariabilityRmssdRecord
import androidx.health.connect.client.records.RestingHeartRateRecord
import androidx.health.connect.client.records.SleepSessionRecord
import androidx.health.connect.client.request.ReadRecordsRequest
import androidx.health.connect.client.time.TimeRangeFilter
import pro.d11l.fitcoach.core.network.HealthSignalDto
import java.time.Instant
import java.time.ZoneId

/**
 * Health Connect-backed [RecoverySignalSource]. This is the ONLY file in the
 * feature that depends on `androidx.health`: it maps records into the pure
 * [RawSleepSession] / [RawInstantValue] inputs and delegates all aggregation to
 * [SignalAggregation]. It cannot be unit-tested headlessly — it needs a device
 * with the Health Connect provider and granted read permissions (e.g. a Pixel
 * Watch syncing sleep/RHR/HRV).
 */
class HealthConnectSource(
    private val context: Context,
    private val zone: ZoneId = ZoneId.systemDefault(),
) : RecoverySignalSource {

    private val client: HealthConnectClient? by lazy {
        if (HealthConnectClient.getSdkStatus(context) == HealthConnectClient.SDK_AVAILABLE) {
            HealthConnectClient.getOrCreate(context)
        } else {
            null
        }
    }

    override suspend fun status(): SignalSourceStatus =
        when (HealthConnectClient.getSdkStatus(context)) {
            HealthConnectClient.SDK_AVAILABLE -> SignalSourceStatus.AVAILABLE
            HealthConnectClient.SDK_UNAVAILABLE_PROVIDER_UPDATE_REQUIRED -> SignalSourceStatus.NOT_INSTALLED
            else -> SignalSourceStatus.NOT_SUPPORTED
        }

    override suspend fun hasAllPermissions(): Boolean {
        val hc = client ?: return false
        return hc.permissionController.getGrantedPermissions().containsAll(PERMISSIONS)
    }

    override suspend fun read(start: Instant, end: Instant): List<HealthSignalDto> {
        val hc = client ?: return emptyList()
        val range = TimeRangeFilter.between(start, end)

        val sleep = hc.readRecords(ReadRecordsRequest(SleepSessionRecord::class, range)).records
            .map { RawSleepSession(it.startTime, it.endTime) }
        val rhr = hc.readRecords(ReadRecordsRequest(RestingHeartRateRecord::class, range)).records
            .map { RawInstantValue(it.time, it.beatsPerMinute.toDouble()) }
        val hrv = hc.readRecords(ReadRecordsRequest(HeartRateVariabilityRmssdRecord::class, range)).records
            .map { RawInstantValue(it.time, it.heartRateVariabilityMillis) }

        return SignalAggregation.sleepMinutes(sleep, zone) +
            SignalAggregation.restingHr(rhr, zone) +
            SignalAggregation.hrv(hrv, zone)
    }

    companion object {
        /** Read permissions requested with rationale — sleep, resting HR, HRV. */
        val PERMISSIONS: Set<String> = setOf(
            HealthPermission.getReadPermission(SleepSessionRecord::class),
            HealthPermission.getReadPermission(RestingHeartRateRecord::class),
            HealthPermission.getReadPermission(HeartRateVariabilityRmssdRecord::class),
        )
    }
}

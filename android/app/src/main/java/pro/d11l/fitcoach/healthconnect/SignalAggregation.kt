package pro.d11l.fitcoach.healthconnect

import pro.d11l.fitcoach.core.network.HealthSignalDto
import pro.d11l.fitcoach.core.network.HealthSignalKinds
import java.time.Duration
import java.time.Instant
import java.time.ZoneId
import kotlin.math.roundToLong

/** A completed sleep session reduced to its time bounds (Health-Connect-free). */
data class RawSleepSession(val start: Instant, val end: Instant)

/** A point-in-time numeric reading — resting HR or HRV (Health-Connect-free). */
data class RawInstantValue(val time: Instant, val value: Double)

/**
 * Pure record → per-day-sample aggregation. Deliberately free of any
 * `androidx.health` dependency so it is fully unit-testable on the JVM; the
 * on-device reader maps Health Connect records into the [RawSleepSession] /
 * [RawInstantValue] inputs and delegates here.
 */
object SignalAggregation {

    /**
     * Total sleep minutes per day, bucketed by the local date a session **ends**
     * (the morning the user wakes, which is the day readiness pertains to).
     * Zero-length or inverted sessions are dropped.
     */
    fun sleepMinutes(sessions: List<RawSleepSession>, zone: ZoneId): List<HealthSignalDto> =
        sessions
            .filter { it.end.isAfter(it.start) }
            .groupBy { it.end.atZone(zone).toLocalDate() }
            .map { (day, daySessions) ->
                val minutes = daySessions.sumOf { Duration.between(it.start, it.end).toMinutes() }
                HealthSignalDto(HealthSignalKinds.SLEEP_MINUTES, minutes.toDouble(), day.toString())
            }
            .sortedBy { it.day }

    /** Mean resting HR (bpm) per local day. */
    fun restingHr(samples: List<RawInstantValue>, zone: ZoneId): List<HealthSignalDto> =
        meanPerDay(samples, zone, HealthSignalKinds.RHR_BPM)

    /** Mean overnight HRV (RMSSD, ms) per local day. */
    fun hrv(samples: List<RawInstantValue>, zone: ZoneId): List<HealthSignalDto> =
        meanPerDay(samples, zone, HealthSignalKinds.HRV_MS)

    private fun meanPerDay(
        samples: List<RawInstantValue>,
        zone: ZoneId,
        kind: String,
    ): List<HealthSignalDto> =
        samples
            .groupBy { it.time.atZone(zone).toLocalDate() }
            .map { (day, values) ->
                val mean = values.sumOf { it.value } / values.size
                HealthSignalDto(kind, mean.roundTo1Decimal(), day.toString())
            }
            .sortedBy { it.day }

    private fun Double.roundTo1Decimal(): Double = (this * 10).roundToLong() / 10.0
}

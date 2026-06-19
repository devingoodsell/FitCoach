package pro.d11l.fitcoach.healthconnect

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import pro.d11l.fitcoach.core.network.HealthSignalKinds
import java.time.Instant
import java.time.ZoneId

class SignalAggregationTest {

    private val utc = ZoneId.of("UTC")

    @Test
    fun `sleep minutes are summed per wake-day and bucketed by end date`() {
        val sessions = listOf(
            // Main sleep + a daytime nap, both ending on 2026-06-17 (UTC).
            RawSleepSession(Instant.parse("2026-06-16T23:00:00Z"), Instant.parse("2026-06-17T07:00:00Z")),
            RawSleepSession(Instant.parse("2026-06-17T14:00:00Z"), Instant.parse("2026-06-17T14:30:00Z")),
            // Previous night, ending 2026-06-16.
            RawSleepSession(Instant.parse("2026-06-15T23:00:00Z"), Instant.parse("2026-06-16T06:00:00Z")),
        )

        val out = SignalAggregation.sleepMinutes(sessions, utc)

        assertEquals(listOf("2026-06-16", "2026-06-17"), out.map { it.day })
        assertTrue(out.all { it.kind == HealthSignalKinds.SLEEP_MINUTES })
        assertEquals(7 * 60.0, out.first { it.day == "2026-06-16" }.value, 0.0)
        // 30 min nap + 8h sleep, both ending 06-17.
        assertEquals(30 + 8 * 60.0, out.first { it.day == "2026-06-17" }.value, 0.0)
    }

    @Test
    fun `inverted or zero-length sleep sessions are dropped`() {
        val sessions = listOf(
            RawSleepSession(Instant.parse("2026-06-17T07:00:00Z"), Instant.parse("2026-06-16T23:00:00Z")),
            RawSleepSession(Instant.parse("2026-06-17T07:00:00Z"), Instant.parse("2026-06-17T07:00:00Z")),
        )
        assertTrue(SignalAggregation.sleepMinutes(sessions, utc).isEmpty())
    }

    @Test
    fun `resting HR is averaged per day`() {
        val samples = listOf(
            RawInstantValue(Instant.parse("2026-06-17T05:00:00Z"), 50.0),
            RawInstantValue(Instant.parse("2026-06-17T06:00:00Z"), 54.0),
            RawInstantValue(Instant.parse("2026-06-16T06:00:00Z"), 60.0),
        )

        val out = SignalAggregation.restingHr(samples, utc)

        assertEquals(listOf("2026-06-16", "2026-06-17"), out.map { it.day })
        assertTrue(out.all { it.kind == HealthSignalKinds.RHR_BPM })
        assertEquals(52.0, out.first { it.day == "2026-06-17" }.value, 0.0)
        assertEquals(60.0, out.first { it.day == "2026-06-16" }.value, 0.0)
    }

    @Test
    fun `hrv is averaged per day and rounded to one decimal`() {
        val samples = listOf(
            RawInstantValue(Instant.parse("2026-06-17T03:00:00Z"), 40.0),
            RawInstantValue(Instant.parse("2026-06-17T04:00:00Z"), 45.0),
            RawInstantValue(Instant.parse("2026-06-17T05:00:00Z"), 41.0),
        )

        val out = SignalAggregation.hrv(samples, utc)

        assertEquals(1, out.size)
        assertEquals(HealthSignalKinds.HRV_MS, out.first().kind)
        // (40 + 45 + 41) / 3 = 42.0 -> 42.0
        assertEquals(42.0, out.first().value, 0.0)
    }

    @Test
    fun `day bucketing respects the supplied zone`() {
        // 05:00 UTC on 06-17 is 22:00 on 06-16 in America/Los_Angeles.
        val la = ZoneId.of("America/Los_Angeles")
        val samples = listOf(RawInstantValue(Instant.parse("2026-06-17T05:00:00Z"), 55.0))

        assertEquals("2026-06-16", SignalAggregation.restingHr(samples, la).single().day)
        assertEquals("2026-06-17", SignalAggregation.restingHr(samples, utc).single().day)
    }

    @Test
    fun `empty inputs aggregate to empty`() {
        assertTrue(SignalAggregation.sleepMinutes(emptyList(), utc).isEmpty())
        assertTrue(SignalAggregation.restingHr(emptyList(), utc).isEmpty())
        assertTrue(SignalAggregation.hrv(emptyList(), utc).isEmpty())
    }
}

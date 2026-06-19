package pro.d11l.fitcoach.data

import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import pro.d11l.fitcoach.core.network.ConsentRecord
import pro.d11l.fitcoach.core.network.HealthSignalDto
import pro.d11l.fitcoach.core.network.HealthSignalKinds
import pro.d11l.fitcoach.healthconnect.SignalSourceStatus
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.FakeRecoverySignalSource
import java.time.Instant

class HealthSignalsRepositoryTest {

    private val now = Instant.parse("2026-06-18T08:00:00Z")

    private fun repo(api: FakeApi, source: FakeRecoverySignalSource) =
        HealthSignalsRepository(source, api, ConsentRepository(api))

    private fun apiWithHealthConsent(active: Boolean): FakeApi = FakeApi().apply {
        consentList = listOf(
            ConsentRecord(
                type = "health_data",
                version = "v1",
                acceptedAt = "2026-06-01T00:00:00Z",
                revokedAt = if (active) null else "2026-06-10T00:00:00Z",
            ),
        )
    }

    @Test
    fun `no health consent degrades to NoConsent without reading`() = runTest {
        val source = FakeRecoverySignalSource()
        val result = repo(FakeApi(), source).ingest(now) // empty consent list

        assertEquals(IngestResult.NoConsent, result)
        assertFalse("must not touch Health Connect without consent", source.readCalled)
    }

    @Test
    fun `revoked health consent degrades to NoConsent`() = runTest {
        val result = repo(apiWithHealthConsent(active = false), FakeRecoverySignalSource()).ingest(now)
        assertEquals(IngestResult.NoConsent, result)
    }

    @Test
    fun `unavailable provider degrades to Unavailable`() = runTest {
        val source = FakeRecoverySignalSource(statusResult = SignalSourceStatus.NOT_SUPPORTED)
        val result = repo(apiWithHealthConsent(active = true), source).ingest(now)

        assertEquals(IngestResult.Unavailable, result)
        assertFalse(source.readCalled)
    }

    @Test
    fun `denied permissions degrade to PermissionsRequired without reading`() = runTest {
        val source = FakeRecoverySignalSource(permitted = false)
        val result = repo(apiWithHealthConsent(active = true), source).ingest(now)

        assertEquals(IngestResult.PermissionsRequired, result)
        assertFalse(source.readCalled)
    }

    @Test
    fun `no samples degrades to NoData and uploads nothing`() = runTest {
        val api = apiWithHealthConsent(active = true)
        val result = repo(api, FakeRecoverySignalSource(samples = emptyList())).ingest(now)

        assertEquals(IngestResult.NoData, result)
        assertNull(api.lastSignals)
    }

    @Test
    fun `granted with samples uploads and reports the count`() = runTest {
        val api = apiWithHealthConsent(active = true)
        val samples = listOf(
            HealthSignalDto(HealthSignalKinds.SLEEP_MINUTES, 465.0, "2026-06-18"),
            HealthSignalDto(HealthSignalKinds.RHR_BPM, 52.0, "2026-06-18"),
        )
        val result = repo(api, FakeRecoverySignalSource(samples = samples)).ingest(now)

        assertEquals(IngestResult.Uploaded(2), result)
        assertEquals(samples, api.lastSignals?.samples)
    }

    @Test
    fun `upload failure surfaces as Error`() = runTest {
        val api = apiWithHealthConsent(active = true).apply { uploadSignalsError = true }
        val samples = listOf(HealthSignalDto(HealthSignalKinds.HRV_MS, 42.0, "2026-06-18"))
        val result = repo(api, FakeRecoverySignalSource(samples = samples)).ingest(now)

        assertTrue(result is IngestResult.Error)
    }

    @Test
    fun `read failure surfaces as Error and does not upload`() = runTest {
        val api = apiWithHealthConsent(active = true)
        val result = repo(api, FakeRecoverySignalSource(throwOnRead = true)).ingest(now)

        assertTrue(result is IngestResult.Error)
        assertNull(api.lastSignals)
    }
}

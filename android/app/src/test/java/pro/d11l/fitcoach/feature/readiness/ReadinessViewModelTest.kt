package pro.d11l.fitcoach.feature.readiness

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.ConsentRecord
import pro.d11l.fitcoach.core.network.HealthSignalDto
import pro.d11l.fitcoach.core.network.HealthSignalKinds
import pro.d11l.fitcoach.core.network.ReadinessDto
import pro.d11l.fitcoach.data.ConsentRepository
import pro.d11l.fitcoach.data.HealthSignalsRepository
import pro.d11l.fitcoach.data.ReadinessRepository
import pro.d11l.fitcoach.healthconnect.SignalSourceStatus
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.FakeRecoverySignalSource
import pro.d11l.fitcoach.testing.MainDispatcherRule
import java.time.Instant

@OptIn(ExperimentalCoroutinesApi::class)
class ReadinessViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private val now = Instant.parse("2026-06-18T08:00:00Z")

    private val score = ReadinessDto(value = 72, confidence = "high", drivers = listOf("hrv_high"), explanation = "HRV up")

    /** Builds a VM whose ingest is driven by [source]/[api]; the clock is fixed. */
    private fun vm(api: FakeApi, source: FakeRecoverySignalSource = FakeRecoverySignalSource()) =
        ReadinessViewModel(
            ReadinessRepository(api),
            HealthSignalsRepository(source, api, ConsentRepository(api)),
        ) { now }

    /** An API with active health-data consent so ingest can get past the consent gate. */
    private fun apiWithHealthConsent(): FakeApi = FakeApi().apply {
        consentList = listOf(
            ConsentRecord(type = "health_data", version = "v1", acceptedAt = "2026-06-01T00:00:00Z"),
        )
    }

    private val oneSample = listOf(HealthSignalDto(HealthSignalKinds.RHR_BPM, 52.0, "2026-06-18"))

    // --- readiness rendering (independent of ingest) -------------------------------------

    @Test
    fun `loads a scored reading`() = runTest {
        val s = vm(apiWithHealthConsent().apply { readiness = score }, FakeRecoverySignalSource(samples = oneSample))
        advanceUntilIdle()

        assertFalse(s.state.value.loading)
        assertEquals(72, s.state.value.readiness?.value)
        assertFalse(s.state.value.isUnavailable)
    }

    @Test
    fun `neutral low-confidence reading is treated as unavailable`() = runTest {
        val api = apiWithHealthConsent().apply {
            readiness = ReadinessDto(value = 50, confidence = "low", drivers = emptyList(), explanation = "n/a")
        }
        val s = vm(api, FakeRecoverySignalSource(samples = oneSample))
        advanceUntilIdle()
        assertTrue(s.state.value.isUnavailable)
    }

    @Test
    fun `readiness fetch error surfaces`() = runTest {
        val s = vm(apiWithHealthConsent().apply { readinessError = true }, FakeRecoverySignalSource(samples = oneSample))
        advanceUntilIdle()
        assertTrue(s.state.value.error != null)
    }

    // --- IngestResult -> UI state mapping ------------------------------------------------

    @Test
    fun `Uploaded refreshes the score with no fallback hint`() = runTest {
        val s = vm(apiWithHealthConsent().apply { readiness = score }, FakeRecoverySignalSource(samples = oneSample))
        advanceUntilIdle()

        assertEquals(72, s.state.value.readiness?.value)
        assertNull(s.state.value.hint)
        assertNull(s.state.value.error)
    }

    @Test
    fun `NoConsent maps to the no-consent hint`() = runTest {
        // No consent list -> ingest never reads Health Connect.
        val s = vm(FakeApi().apply { readiness = score })
        advanceUntilIdle()
        assertEquals(ReadinessHints.NO_CONSENT, s.state.value.hint)
    }

    @Test
    fun `Unavailable maps to the unavailable hint`() = runTest {
        val s = vm(apiWithHealthConsent(), FakeRecoverySignalSource(statusResult = SignalSourceStatus.NOT_SUPPORTED))
        advanceUntilIdle()
        assertEquals(ReadinessHints.UNAVAILABLE, s.state.value.hint)
    }

    @Test
    fun `PermissionsRequired maps to the permissions hint`() = runTest {
        val s = vm(apiWithHealthConsent(), FakeRecoverySignalSource(permitted = false))
        advanceUntilIdle()
        assertEquals(ReadinessHints.PERMISSIONS_REQUIRED, s.state.value.hint)
    }

    @Test
    fun `NoData maps to the no-data hint`() = runTest {
        val s = vm(apiWithHealthConsent(), FakeRecoverySignalSource(samples = emptyList()))
        advanceUntilIdle()
        assertEquals(ReadinessHints.NO_DATA, s.state.value.hint)
    }

    @Test
    fun `Error surfaces a non-blocking sync message and keeps the last reading`() = runTest {
        val s = vm(apiWithHealthConsent().apply { readiness = score }, FakeRecoverySignalSource(throwOnRead = true))
        advanceUntilIdle()

        assertEquals(ReadinessHints.SYNC_FAILED, s.state.value.error)
        assertNull(s.state.value.hint)
        assertEquals(72, s.state.value.readiness?.value) // non-blocking: reading still shown
    }
}

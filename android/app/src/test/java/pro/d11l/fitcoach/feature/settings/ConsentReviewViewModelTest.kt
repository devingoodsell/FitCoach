package pro.d11l.fitcoach.feature.settings

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.ConsentRecord
import pro.d11l.fitcoach.data.ConsentRepository
import pro.d11l.fitcoach.data.ConsentTypes
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class ConsentReviewViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun api(records: List<ConsentRecord>) = FakeApi().apply { consentList = records }

    @Test
    fun `loads consent state and flags active health data`() = runTest {
        val api = api(
            listOf(
                ConsentRecord(type = ConsentTypes.HEALTH_DATA, version = "v1", acceptedAt = "2026-06-01T00:00:00Z"),
                ConsentRecord(type = ConsentTypes.MEDICAL_DISCLAIMER, version = "v1", acceptedAt = "2026-06-01T00:00:00Z"),
            ),
        )
        val vm = ConsentReviewViewModel(ConsentRepository(api))
        advanceUntilIdle()

        val s = vm.state.value
        assertFalse(s.loading)
        assertEquals(2, s.consents.size)
        assertTrue(s.healthDataActive)
    }

    @Test
    fun `revoking health data switches it to revoked (manual mode)`() = runTest {
        val api = api(listOf(ConsentRecord(type = ConsentTypes.HEALTH_DATA, version = "v1", acceptedAt = "2026-06-01T00:00:00Z")))
        val vm = ConsentReviewViewModel(ConsentRepository(api))
        advanceUntilIdle()
        assertTrue(vm.state.value.healthDataActive)

        vm.revokeHealthData()
        advanceUntilIdle()

        assertEquals(ConsentTypes.HEALTH_DATA, api.lastRevokedType)
        // reloaded state now shows health-data consent as revoked
        assertFalse(vm.state.value.healthDataActive)
        assertTrue(vm.state.value.healthData != null)
    }

    @Test
    fun `load failure surfaces an error`() = runTest {
        val vm = ConsentReviewViewModel(ConsentRepository(FakeApi().apply { consentListError = true }))
        advanceUntilIdle()
        assertTrue(vm.state.value.error != null)
        assertFalse(vm.state.value.loading)
    }
}

package pro.d11l.fitcoach.feature.readiness

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.ReadinessDto
import pro.d11l.fitcoach.data.ReadinessRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class ReadinessViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun vm(api: FakeApi) = ReadinessViewModel(ReadinessRepository(api))

    @Test
    fun `loads a scored reading`() = runTest {
        val api = FakeApi().apply {
            readiness = ReadinessDto(value = 72, confidence = "high", drivers = listOf("hrv_high"), explanation = "HRV up")
        }
        val s = vm(api)
        advanceUntilIdle()

        assertFalse(s.state.value.loading)
        assertEquals(72, s.state.value.readiness?.value)
        assertFalse(s.state.value.isUnavailable)
    }

    @Test
    fun `neutral low-confidence reading is treated as unavailable`() = runTest {
        val api = FakeApi().apply {
            readiness = ReadinessDto(value = 50, confidence = "low", drivers = emptyList(), explanation = "n/a")
        }
        val s = vm(api)
        advanceUntilIdle()
        assertTrue(s.state.value.isUnavailable)
    }

    @Test
    fun `error surfaces`() = runTest {
        val s = vm(FakeApi().apply { readinessError = true })
        advanceUntilIdle()
        assertTrue(s.state.value.error != null)
    }
}

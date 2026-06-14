package pro.d11l.fitcoach.feature.consent

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.data.AuthRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.FakeMemoryCache
import pro.d11l.fitcoach.testing.InMemoryTokenStorage
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class ConsentViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun viewModel(api: FakeApi = FakeApi()): ConsentViewModel =
        ConsentViewModel(AuthRepository(api, InMemoryTokenStorage(), FakeMemoryCache()))

    @Test
    fun `allowing health data records health consent`() = runTest {
        val api = FakeApi()
        val vm = viewModel(api)

        vm.allowHealthData()
        advanceUntilIdle()

        assertTrue(vm.state.value.decided)
        assertFalse(vm.state.value.manualMode)
        assertEquals("health_data", api.lastConsent?.type)
    }

    @Test
    fun `manual mode decides without enabling health data`() = runTest {
        val api = FakeApi()
        val vm = viewModel(api)

        vm.useManualMode()
        advanceUntilIdle()

        assertTrue(vm.state.value.decided)
        assertTrue(vm.state.value.manualMode)
        // Last consent recorded is the medical disclaimer, never health_data.
        assertEquals("medical_disclaimer", api.lastConsent?.type)
    }
}

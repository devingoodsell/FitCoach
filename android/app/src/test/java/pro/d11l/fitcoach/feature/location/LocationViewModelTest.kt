package pro.d11l.fitcoach.feature.location

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.data.LocationRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class LocationViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun vm(api: FakeApi = FakeApi()) = LocationViewModel(LocationRepository(api))

    @Test
    fun `loads locations on init`() = runTest {
        val vm = vm()
        advanceUntilIdle()
        assertTrue(!vm.state.value.loading)
        assertTrue(vm.state.value.locations.isEmpty())
    }

    @Test
    fun `add location reloads list`() = runTest {
        val api = FakeApi()
        val vm = vm(api)
        advanceUntilIdle()

        vm.addLocation("Home Gym", "dumbbells, bench")
        advanceUntilIdle()

        assertEquals(1, vm.state.value.locations.size)
        assertEquals("Home Gym", vm.state.value.locations.first().name)
        assertEquals(listOf("dumbbells", "bench"), api.lastAddedEquipment())
    }

    @Test
    fun `blank name sets error and skips network`() = runTest {
        val api = FakeApi()
        val vm = vm(api)
        advanceUntilIdle()

        vm.addLocation("   ", "")
        advanceUntilIdle()

        assertEquals("Name is required", vm.state.value.error)
        assertTrue(vm.state.value.locations.isEmpty())
    }

    @Test
    fun `set current updates state`() = runTest {
        val api = FakeApi()
        val vm = vm(api)
        advanceUntilIdle()
        vm.addLocation("Hotel", "")
        advanceUntilIdle()
        val id = vm.state.value.locations.first().id

        vm.setCurrent(id, "traveling")
        advanceUntilIdle()

        assertEquals(id, vm.state.value.current?.locationId)
        assertEquals("traveling", api.lastSetCurrent?.note)
    }

    @Test
    fun `load error surfaces`() = runTest {
        val api = FakeApi().apply { locationsError = true }
        val vm = vm(api)
        advanceUntilIdle()
        assertTrue(vm.state.value.error != null)
    }
}

// Small helper for asserting equipment passed via the fake's mutated doc.
private fun FakeApi.lastAddedEquipment(): List<String> =
    locationsDoc.locations.lastOrNull()?.equipment ?: emptyList()

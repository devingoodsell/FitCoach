package pro.d11l.fitcoach.feature.settings

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.encodeToJsonElement
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.NetworkModule
import pro.d11l.fitcoach.core.network.ScheduleDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class EditScheduleViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun apiWithSchedule(s: ScheduleDto) = FakeApi().apply {
        memorySections["schedule"] = NetworkModule.json.encodeToJsonElement(s)
    }

    @Test
    fun `prefills from memory`() = runTest {
        val api = apiWithSchedule(ScheduleDto(daysPerWeek = 4, sessionLengthMin = 45, preferredDays = listOf("mon", "wed")))
        val vm = EditScheduleViewModel(OnboardingRepository(api))
        advanceUntilIdle()
        val s = vm.state.value
        assertEquals("4", s.daysPerWeek)
        assertEquals("45", s.sessionLengthMin)
    }

    @Test
    fun `edit and save preserves preferred days`() = runTest {
        val api = apiWithSchedule(ScheduleDto(daysPerWeek = 4, sessionLengthMin = 45, preferredDays = listOf("mon", "wed")))
        val vm = EditScheduleViewModel(OnboardingRepository(api))
        advanceUntilIdle()

        vm.onDaysPerWeek("5")
        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.saved)
        assertEquals(5, api.lastSchedule?.daysPerWeek)
        assertEquals(listOf("mon", "wed"), api.lastSchedule?.preferredDays)
    }

    @Test
    fun `out-of-range values are rejected`() = runTest {
        val api = apiWithSchedule(ScheduleDto(daysPerWeek = 3, sessionLengthMin = 60))
        val vm = EditScheduleViewModel(OnboardingRepository(api))
        advanceUntilIdle()

        vm.onDaysPerWeek("9")
        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.fieldErrors.containsKey("days_per_week"))
        assertTrue(!vm.state.value.saved)
    }
}

package pro.d11l.fitcoach.feature.settings

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.encodeToJsonElement
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.DietPrefsDto
import pro.d11l.fitcoach.core.network.NetworkModule
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class EditDietViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun apiWithDiet(d: DietPrefsDto) = FakeApi().apply {
        memorySections["diet"] = NetworkModule.json.encodeToJsonElement(d)
    }

    @Test
    fun `prefills pattern and supplements`() = runTest {
        val api = apiWithDiet(DietPrefsDto(pattern = "kosher", supplements = "creatine", medications = "none"))
        val vm = EditDietViewModel(OnboardingRepository(api))
        advanceUntilIdle()
        val s = vm.state.value
        assertEquals("kosher", s.pattern)
        assertEquals("creatine", s.supplements)
        assertEquals("none", s.medications)
    }

    @Test
    fun `save sends updated pattern`() = runTest {
        val api = apiWithDiet(DietPrefsDto(pattern = "omnivore"))
        val vm = EditDietViewModel(OnboardingRepository(api))
        advanceUntilIdle()

        vm.onPattern("vegan")
        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.saved)
        assertEquals("vegan", api.lastDiet?.pattern)
    }

    @Test
    fun `blank pattern blocks save`() = runTest {
        val vm = EditDietViewModel(OnboardingRepository(FakeApi()))
        advanceUntilIdle()

        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.fieldErrors.containsKey("pattern"))
        assertTrue(!vm.state.value.saved)
    }
}

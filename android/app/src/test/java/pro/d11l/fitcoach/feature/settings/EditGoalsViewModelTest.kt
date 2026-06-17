package pro.d11l.fitcoach.feature.settings

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.encodeToJsonElement
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.GoalWeightsDto
import pro.d11l.fitcoach.core.network.NetworkModule
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class EditGoalsViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun apiWithGoals(g: GoalWeightsDto) = FakeApi().apply {
        memorySections["goals"] = NetworkModule.json.encodeToJsonElement(g)
    }

    @Test
    fun `prefills sliders from stored normalized weights`() = runTest {
        val api = apiWithGoals(GoalWeightsDto(strength = 0.4, healthspan = 0.3, bodyComposition = 0.2, performance = 0.1))
        val vm = EditGoalsViewModel(OnboardingRepository(api))
        advanceUntilIdle()
        val s = vm.state.value
        assertTrue(!s.loading)
        assertEquals(0.4f, s.strength)
        assertEquals(0.1f, s.performance)
    }

    @Test
    fun `edit and save sends updated weights`() = runTest {
        val api = apiWithGoals(GoalWeightsDto(0.25, 0.25, 0.25, 0.25))
        val vm = EditGoalsViewModel(OnboardingRepository(api))
        advanceUntilIdle()

        vm.onStrength(0.9f)
        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.saved)
        assertEquals(0.9, api.lastGoals!!.strength, 1e-6)
    }

    @Test
    fun `all-zero goals are rejected`() = runTest {
        val api = apiWithGoals(GoalWeightsDto(0.25, 0.25, 0.25, 0.25))
        val vm = EditGoalsViewModel(OnboardingRepository(api))
        advanceUntilIdle()

        vm.onStrength(0f)
        vm.onHealthspan(0f)
        vm.onBodyComp(0f)
        vm.onPerformance(0f)
        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.error != null)
        assertTrue(!vm.state.value.saved)
    }
}

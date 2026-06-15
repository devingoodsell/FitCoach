package pro.d11l.fitcoach.feature.diet

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.DietTargetsDto
import pro.d11l.fitcoach.core.network.DietTargetsValuesDto
import pro.d11l.fitcoach.data.DietRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class DietViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun vm(api: FakeApi) = DietViewModel(DietRepository(api))

    @Test
    fun `loads targets on init`() = runTest {
        val api = FakeApi().apply {
            dietTargets = DietTargetsDto(
                targets = DietTargetsValuesDto(caloriesMin = 2000, caloriesMax = 2300, proteinMinG = 150, proteinMaxG = 170),
                guidance = listOf("eat protein"),
                pattern = "vegan",
                disclaimer = "guidance not advice",
            )
        }
        val vm = vm(api)
        advanceUntilIdle()

        val s = vm.state.value
        assertTrue(!s.loading)
        assertNotNull(s.targets)
        assertEquals(2000, s.targets?.targets?.caloriesMin)
        assertEquals("vegan", s.targets?.pattern)
    }

    @Test
    fun `load error surfaces`() = runTest {
        val vm = vm(FakeApi().apply { dietError = true })
        advanceUntilIdle()
        assertTrue(vm.state.value.error != null)
    }

    @Test
    fun `loadNote sets note text`() = runTest {
        val vm = vm(FakeApi())
        advanceUntilIdle()
        vm.loadNote(heavy = true)
        advanceUntilIdle()
        assertTrue(vm.state.value.note?.startsWith("heavy:") == true)
    }
}

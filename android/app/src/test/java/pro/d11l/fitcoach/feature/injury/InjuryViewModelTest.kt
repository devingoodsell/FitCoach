package pro.d11l.fitcoach.feature.injury

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.InjuryAssistResponseDto
import pro.d11l.fitcoach.core.network.InjuryDraftDto
import pro.d11l.fitcoach.core.network.InjuryDto
import pro.d11l.fitcoach.data.InjuryRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class InjuryViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun vm(api: FakeApi = FakeApi()) = InjuryViewModel(InjuryRepository(api))

    @Test
    fun `parse populates an editable draft for review`() = runTest {
        val api = FakeApi().apply {
            parseDraft = InjuryDraftDto(
                injury = InjuryDto(region = "left_knee", status = "active_flare", severity = "severe",
                    aggravatingMovements = listOf("squat"), notes = "hurts"),
                lowConfidenceFields = listOf("severity"),
            )
        }
        val vm = vm(api)
        advanceUntilIdle()
        vm.onFreeText("left knee hurts when I squat")
        vm.parse()
        advanceUntilIdle()

        val s = vm.state.value
        assertTrue(s.draftVisible)
        assertEquals("left_knee", s.region)
        assertEquals("severe", s.severity)
        assertEquals("squat", s.aggravating)
        assertEquals(listOf("severity"), s.lowConfidenceFields)
    }

    @Test
    fun `save draft creates injury and reloads`() = runTest {
        val api = FakeApi()
        val vm = vm(api)
        advanceUntilIdle()
        vm.startManual()
        vm.onRegion("shoulder")
        vm.onStatus("managed")
        vm.saveDraft()
        advanceUntilIdle()

        assertEquals("shoulder", api.lastAddedInjury?.region)
        assertEquals(1, vm.state.value.injuries.size)
        assertFalse(vm.state.value.draftVisible)
    }

    @Test
    fun `save requires a region`() = runTest {
        val vm = vm()
        advanceUntilIdle()
        vm.startManual()
        vm.saveDraft() // region blank
        advanceUntilIdle()
        assertTrue(vm.state.value.error != null)
        assertTrue(vm.state.value.draftVisible)
    }

    @Test
    fun `set status updates the injury`() = runTest {
        val api = FakeApi()
        val vm = vm(api)
        advanceUntilIdle()
        vm.startManual(); vm.onRegion("knee"); vm.saveDraft(); advanceUntilIdle()
        val inj = vm.state.value.injuries.first()

        vm.setStatus(inj, "resolved")
        advanceUntilIdle()
        assertEquals("resolved", api.lastUpdatedInjury?.status)
    }

    @Test
    fun `delete removes the injury`() = runTest {
        val api = FakeApi()
        val vm = vm(api)
        advanceUntilIdle()
        vm.startManual(); vm.onRegion("knee"); vm.saveDraft(); advanceUntilIdle()
        val inj = vm.state.value.injuries.first()

        vm.delete(inj.id)
        advanceUntilIdle()
        assertTrue(vm.state.value.injuries.isEmpty())
    }

    @Test
    fun `assist surfaces the disclaimer and question while gathering info`() = runTest {
        val api = FakeApi().apply {
            assistResponse = InjuryAssistResponseDto(
                disclaimer = "This is not a diagnosis; consult a clinician.",
                done = false,
                question = "Where do you feel it?",
                choices = listOf("knee", "shoulder"),
            )
        }
        val vm = vm(api)
        advanceUntilIdle()
        vm.startAssist()
        advanceUntilIdle()

        val s = vm.state.value
        assertTrue(s.assistVisible)
        // The "not a diagnosis" framing is present throughout the assist flow (E13).
        assertEquals("This is not a diagnosis; consult a clinician.", s.assistDisclaimer)
        assertEquals("Where do you feel it?", s.assistQuestion)
        assertEquals(listOf("knee", "shoulder"), s.assistChoices)
        assertFalse(s.draftVisible)
    }

    @Test
    fun `assist completion hands off to a confirmable review-before-save draft`() = runTest {
        val api = FakeApi().apply {
            assistResponse = InjuryAssistResponseDto(
                disclaimer = "This is not a diagnosis; consult a clinician.",
                done = true,
                draft = InjuryDraftDto(
                    injury = InjuryDto(region = "lower_back", status = "managed", severity = "mild"),
                ),
            )
        }
        val vm = vm(api)
        advanceUntilIdle()
        vm.startAssist()
        advanceUntilIdle()

        // Reuses the existing review-before-save draft form — no separate save path.
        val s = vm.state.value
        assertFalse(s.assistVisible)
        assertTrue(s.draftVisible)
        assertEquals("lower_back", s.region)

        vm.saveDraft()
        advanceUntilIdle()
        assertEquals("lower_back", api.lastAddedInjury?.region)
        assertFalse(vm.state.value.draftVisible)
    }
}

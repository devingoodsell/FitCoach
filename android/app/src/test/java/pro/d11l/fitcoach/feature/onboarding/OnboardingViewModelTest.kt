package pro.d11l.fitcoach.feature.onboarding

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.ProfileDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule
import pro.d11l.fitcoach.testing.validationErrorResponse

@OptIn(ExperimentalCoroutinesApi::class)
class OnboardingViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun vm(api: FakeApi = FakeApi()) = OnboardingViewModel(OnboardingRepository(api))

    private fun fillValidProfile(vm: OnboardingViewModel) {
        vm.onSex("male")
        vm.onAge("30")
        vm.onLevel("novice")
    }

    @Test
    fun `intro starts the wizard`() {
        val vm = vm()
        assertEquals(OnboardingStep.Intro, vm.state.value.step)
        vm.start()
        assertEquals(OnboardingStep.Profile, vm.state.value.step)
    }

    @Test
    fun `profile validates locally before any network call`() = runTest {
        val api = FakeApi()
        val vm = vm(api)
        vm.start()
        vm.submitProfile() // all blank
        advanceUntilIdle()

        assertEquals(OnboardingStep.Profile, vm.state.value.step)
        val errs = vm.state.value.fieldErrors
        assertTrue(errs.containsKey("sex") && errs.containsKey("age") && errs.containsKey("experience.level"))
        assertNull("no request should be sent on local failure", api.lastProfile)
    }

    @Test
    fun `valid profile advances to goals and sends dto`() = runTest {
        val api = FakeApi()
        val vm = vm(api)
        vm.start()
        fillValidProfile(vm)
        vm.submitProfile()
        advanceUntilIdle()

        assertEquals(OnboardingStep.Goals, vm.state.value.step)
        assertEquals("male", api.lastProfile?.sex)
        assertEquals(30, api.lastProfile?.age)
        assertEquals("novice", api.lastProfile?.experience?.level)
    }

    @Test
    fun `server field errors surface and block advance`() = runTest {
        val api = FakeApi().apply {
            profileResponse = validationErrorResponse<ProfileDto>(mapOf("age" to "must be between 13 and 120"))
        }
        val vm = vm(api)
        vm.start()
        fillValidProfile(vm)
        vm.submitProfile()
        advanceUntilIdle()

        assertEquals(OnboardingStep.Profile, vm.state.value.step)
        assertEquals("must be between 13 and 120", vm.state.value.fieldErrors["age"])
    }

    @Test
    fun `full core path reaches done`() = runTest {
        val api = FakeApi()
        val vm = vm(api)
        vm.start()
        fillValidProfile(vm)
        vm.submitProfile(); advanceUntilIdle()
        vm.submitGoals(); advanceUntilIdle()
        assertEquals(OnboardingStep.Schedule, vm.state.value.step)
        vm.submitSchedule(); advanceUntilIdle()
        assertEquals(OnboardingStep.Diet, vm.state.value.step)
        vm.skipDiet()
        vm.skipPreferences()
        assertEquals(OnboardingStep.Done, vm.state.value.step)

        vm.finish()
        assertTrue(vm.state.value.completed)
        // Goals were sent; diet/prefs were skipped (never sent).
        assertEquals(3, api.lastSchedule?.daysPerWeek)
        assertNull(api.lastDiet)
        assertNull(api.lastPreferences)
    }

    @Test
    fun `schedule rejects out-of-range values`() = runTest {
        val vm = vm()
        vm.start()
        fillValidProfile(vm)
        vm.submitProfile(); advanceUntilIdle()
        vm.submitGoals(); advanceUntilIdle()
        vm.onDaysPerWeek("9")
        vm.submitSchedule(); advanceUntilIdle()

        assertEquals(OnboardingStep.Schedule, vm.state.value.step)
        assertTrue(vm.state.value.fieldErrors.containsKey("days_per_week"))
    }

    @Test
    fun `diet requires a pattern unless skipped`() = runTest {
        val vm = vm()
        vm.start()
        fillValidProfile(vm)
        vm.submitProfile(); advanceUntilIdle()
        vm.submitGoals(); advanceUntilIdle()
        vm.submitSchedule(); advanceUntilIdle()
        vm.submitDiet() // no pattern chosen
        advanceUntilIdle()

        assertEquals(OnboardingStep.Diet, vm.state.value.step)
        assertTrue(vm.state.value.fieldErrors.containsKey("pattern"))
    }
}

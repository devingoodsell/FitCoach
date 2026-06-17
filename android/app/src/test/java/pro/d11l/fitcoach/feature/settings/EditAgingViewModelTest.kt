package pro.d11l.fitcoach.feature.settings

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.encodeToJsonElement
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.core.network.AgingEmphasesDto
import pro.d11l.fitcoach.core.network.ExperienceDto
import pro.d11l.fitcoach.core.network.NetworkModule
import pro.d11l.fitcoach.core.network.ProfileDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class EditAgingViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun apiWithProfile(p: ProfileDto) = FakeApi().apply {
        memorySections["profile"] = NetworkModule.json.encodeToJsonElement(p)
    }

    @Test
    fun `prefills from stored emphases`() = runTest {
        val aging = AgingEmphasesDto(boneBalance = 0.4, jointTendon = 0.3, vo2max = 0.2, cardioBase = 0.1)
        val api = apiWithProfile(
            ProfileDto(age = 50, sex = "male", experience = ExperienceDto(level = "novice"), agingEmphases = aging),
        )
        val vm = EditAgingViewModel(OnboardingRepository(api))
        advanceUntilIdle()
        val s = vm.state.value
        assertEquals(0.4f, s.boneBalance)
        assertEquals(0.1f, s.cardioBase)
    }

    @Test
    fun `defaults from age when no emphases stored`() = runTest {
        // age >= 60 tier: bone 0.35, joint 0.30, vo2 0.15, cardio 0.20
        val api = apiWithProfile(ProfileDto(age = 67, sex = "female", experience = ExperienceDto(level = "intermediate")))
        val vm = EditAgingViewModel(OnboardingRepository(api))
        advanceUntilIdle()
        val s = vm.state.value
        assertEquals(0.35f, s.boneBalance)
        assertEquals(0.15f, s.vo2max)
    }

    @Test
    fun `save sends full profile with updated emphases preserving other fields`() = runTest {
        val api = apiWithProfile(
            ProfileDto(
                age = 50,
                sex = "male",
                heightCm = 180.0,
                weightKg = 82.0,
                experience = ExperienceDto(level = "advanced"),
                agingEmphases = AgingEmphasesDto(0.25, 0.25, 0.25, 0.25),
            ),
        )
        val vm = EditAgingViewModel(OnboardingRepository(api))
        advanceUntilIdle()

        vm.onVo2max(0.9f)
        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.saved)
        // other profile fields preserved
        assertEquals("male", api.lastProfile?.sex)
        assertEquals(50, api.lastProfile?.age)
        assertEquals("advanced", api.lastProfile?.experience?.level)
        // emphasis updated
        assertEquals(0.9, api.lastProfile?.agingEmphases?.vo2max!!, 1e-6)
    }

    @Test
    fun `missing profile surfaces an error and blocks save`() = runTest {
        val vm = EditAgingViewModel(OnboardingRepository(FakeApi()))
        advanceUntilIdle()
        assertTrue(vm.state.value.error != null)

        vm.save()
        advanceUntilIdle()
        assertTrue(!vm.state.value.saved)
    }
}

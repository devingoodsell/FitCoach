package pro.d11l.fitcoach.feature.settings

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.encodeToJsonElement
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
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
class EditProfileViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private val aging = AgingEmphasesDto(boneBalance = 0.35, jointTendon = 0.3, vo2max = 0.15, cardioBase = 0.2)
    private val stored = ProfileDto(
        age = 58,
        sex = "female",
        heightCm = 168.0,
        weightKg = 65.0,
        experience = ExperienceDto(level = "intermediate"),
        agingEmphases = aging,
    )

    private fun apiWithProfile(p: ProfileDto = stored) = FakeApi().apply {
        memorySections["profile"] = NetworkModule.json.encodeToJsonElement(p)
    }

    private fun vm(api: FakeApi) = EditProfileViewModel(OnboardingRepository(api))

    @Test
    fun `prefills fields from memory`() = runTest {
        val vm = vm(apiWithProfile())
        advanceUntilIdle()
        val s = vm.state.value
        assertTrue(!s.loading)
        assertEquals("female", s.sex)
        assertEquals("58", s.age)
        assertEquals("168", s.heightCm)
        assertEquals("65", s.weightKg)
        assertEquals("intermediate", s.level)
    }

    @Test
    fun `empty memory leaves a blank form`() = runTest {
        val vm = vm(FakeApi())
        advanceUntilIdle()
        val s = vm.state.value
        assertTrue(!s.loading)
        assertEquals("", s.sex)
        assertEquals("", s.age)
    }

    @Test
    fun `edit and save round-trips and preserves aging emphases`() = runTest {
        val api = apiWithProfile()
        val vm = vm(api)
        advanceUntilIdle()

        vm.onAge("59")
        vm.onLevel("advanced")
        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.saved)
        assertEquals(59, api.lastProfile?.age)
        assertEquals("advanced", api.lastProfile?.experience?.level)
        // aging emphases carried through untouched (not reset to age defaults)
        assertEquals(aging, api.lastProfile?.agingEmphases)
    }

    @Test
    fun `invalid age blocks save with field error`() = runTest {
        val api = apiWithProfile()
        val vm = vm(api)
        advanceUntilIdle()

        vm.onAge("5")
        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.fieldErrors.containsKey("age"))
        assertTrue(!vm.state.value.saved)
        // nothing sent (lastProfile is still null since we never reached the network)
        assertNull(api.lastProfile)
    }
}

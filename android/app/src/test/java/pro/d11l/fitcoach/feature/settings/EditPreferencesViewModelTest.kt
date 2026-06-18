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
import pro.d11l.fitcoach.core.network.PreferencesDto
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule

@OptIn(ExperimentalCoroutinesApi::class)
class EditPreferencesViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun apiWithPrefs(p: PreferencesDto) = FakeApi().apply {
        memorySections["preferences"] = NetworkModule.json.encodeToJsonElement(p)
    }

    @Test
    fun `prefills comma-joined lists`() = runTest {
        val api = apiWithPrefs(PreferencesDto(likes = listOf("squat", "rows"), dislikes = listOf("burpees"), hardAvoids = listOf("overhead press")))
        val vm = EditPreferencesViewModel(OnboardingRepository(api))
        advanceUntilIdle()
        val s = vm.state.value
        assertEquals("squat, rows", s.likes)
        assertEquals("overhead press", s.hardAvoids)
    }

    @Test
    fun `save parses lists and preserves hard avoids`() = runTest {
        val api = apiWithPrefs(PreferencesDto(hardAvoids = listOf("overhead press")))
        val vm = EditPreferencesViewModel(OnboardingRepository(api))
        advanceUntilIdle()

        vm.onLikes("squat, deadlift")
        vm.save()
        advanceUntilIdle()

        assertTrue(vm.state.value.saved)
        assertEquals(listOf("squat", "deadlift"), api.lastPreferences?.likes)
        // hard-avoids round-tripped, not dropped
        assertEquals(listOf("overhead press"), api.lastPreferences?.hardAvoids)
    }
}

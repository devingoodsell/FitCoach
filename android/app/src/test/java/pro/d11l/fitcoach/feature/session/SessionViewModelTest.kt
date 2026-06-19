package pro.d11l.fitcoach.feature.session

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceTimeBy
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.data.SessionRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.FakeSessionCache
import pro.d11l.fitcoach.testing.MainDispatcherRule
import pro.d11l.fitcoach.testing.errorResponse
import retrofit2.Response

@OptIn(ExperimentalCoroutinesApi::class)
class SessionViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun fixtures(): Triple<SessionViewModel, FakeApi, FakeSessionCache> {
        val api = FakeApi().apply { sessionResponse = Response.success(sampleSession()) }
        val cache = FakeSessionCache()
        return Triple(SessionViewModel(SessionRepository(api, cache)), api, cache)
    }

    @Test
    fun `start loads the session plan into the player`() = runTest {
        val (vm, _, _) = fixtures()
        vm.start()
        advanceUntilIdle()

        val s = vm.state.value
        assertFalse(s.loading)
        assertNotNull(s.plan)
        assertEquals(5, s.totalSteps) // warmup1 + main2 + accessory1 + aging1
        assertEquals("Rower easy spin", s.current?.exerciseName)
        assertEquals("180", s.draft.durationSec) // timed warmup seeds duration
        assertNull(s.error)
    }

    @Test
    fun `unsafe session surfaces a friendly message`() = runTest {
        val api = FakeApi().apply { sessionResponse = errorResponse(422) }
        val vm = SessionViewModel(SessionRepository(api, FakeSessionCache()))
        vm.start()
        advanceUntilIdle()
        assertNull(vm.state.value.plan)
        assertTrue(vm.state.value.error!!.contains("safe session", ignoreCase = true))
    }

    @Test
    fun `logging a set records actuals defaulting to prescription and advances`() = runTest {
        val (vm, _, cache) = fixtures()
        vm.start(); advanceUntilIdle()

        // Warm-up (timed, no rest) -> log with prescribed default.
        vm.logCurrentSet(); runCurrent()
        assertEquals(1, vm.state.value.loggedCount)
        assertEquals("Goblet box squat", vm.state.value.current?.exerciseName)

        // Main set 1: edit reps/weight then log.
        vm.updateReps("10"); vm.updateLoad("25")
        vm.logCurrentSet(); runCurrent()

        val mainLog = cache.loggedSets.last().second
        assertEquals(10, mainLog.repsDone)
        assertEquals(25.0, mainLog.loadKgDone!!, 0.0)
        assertTrue(mainLog.completed)
        assertEquals(2, vm.state.value.loggedCount)
    }

    @Test
    fun `logging a set with prescribed rest starts a ticking countdown`() = runTest {
        val (vm, _, _) = fixtures()
        vm.start(); advanceUntilIdle()
        vm.logCurrentSet(); runCurrent() // warm-up, rest 0 -> no rest
        assertNull(vm.state.value.rest)

        vm.logCurrentSet(); runCurrent() // main set 1, rest 120
        assertEquals(120, vm.state.value.rest?.remainingSec)
        assertTrue(vm.state.value.rest?.running == true)

        advanceTimeBy(3000); runCurrent()
        assertEquals(117, vm.state.value.rest?.remainingSec)
    }

    @Test
    fun `rest reaching zero fires the cue once`() = runTest {
        val (vm, _, _) = fixtures()
        vm.start(); advanceUntilIdle()
        vm.logCurrentSet(); runCurrent() // warm-up
        vm.logCurrentSet(); runCurrent() // main set 1 -> 120s rest

        advanceTimeBy(120_000); runCurrent()
        assertTrue(vm.state.value.rest?.finished == true)
        assertEquals(1, vm.state.value.restCueId)
    }

    @Test
    fun `skip records the set as skipped without resting`() = runTest {
        val (vm, _, cache) = fixtures()
        vm.start(); advanceUntilIdle()
        vm.logCurrentSet(); runCurrent() // warm-up
        vm.skipCurrentSet(); runCurrent() // main set 1 skipped

        assertTrue(cache.loggedSets.last().second.skipped)
        assertNull(vm.state.value.rest)
    }

    @Test
    fun `playing through every set marks the session finished`() = runTest {
        val (vm, _, _) = fixtures()
        vm.start(); advanceUntilIdle()
        repeat(vm.state.value.totalSteps) { vm.logCurrentSet(); runCurrent() }

        assertTrue(vm.state.value.finished)
        assertEquals(5, vm.state.value.loggedCount)
    }

    @Test
    fun `pause stops the countdown and resume continues it`() = runTest {
        val (vm, _, _) = fixtures()
        vm.start(); advanceUntilIdle()
        vm.logCurrentSet(); runCurrent()
        vm.logCurrentSet(); runCurrent() // 120s rest running

        advanceTimeBy(5000); runCurrent()
        vm.pauseRest(); runCurrent()
        val paused = vm.state.value.rest!!.remainingSec
        advanceTimeBy(5000); runCurrent()
        assertEquals(paused, vm.state.value.rest!!.remainingSec) // frozen while paused

        vm.resumeRest()
        advanceTimeBy(2000); runCurrent()
        assertEquals(paused - 2, vm.state.value.rest!!.remainingSec)
    }
}

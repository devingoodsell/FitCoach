package pro.d11l.fitcoach.feature.session

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.data.SessionRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.MainDispatcherRule
import pro.d11l.fitcoach.testing.errorResponse
import retrofit2.Response

@OptIn(ExperimentalCoroutinesApi::class)
class SessionViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun vm(api: FakeApi) = SessionViewModel(SessionRepository(api))

    @Test
    fun `start loads a generated session`() = runTest {
        val api = FakeApi().apply { sessionResponse = Response.success(sampleSession()) }
        val s = vm(api)
        s.start()
        advanceUntilIdle()

        assertFalse(s.state.value.loading)
        assertNotNull(s.state.value.session)
        assertEquals(1, s.state.value.session?.schemaVersion)
        assertTrue(s.state.value.session!!.agingBlock.items.isNotEmpty())
        assertEquals(null, s.state.value.error)
    }

    @Test
    fun `unsafe session surfaces a friendly message`() = runTest {
        val api = FakeApi().apply { sessionResponse = errorResponse(422) }
        val s = vm(api)
        s.start()
        advanceUntilIdle()

        assertEquals(null, s.state.value.session)
        assertTrue(s.state.value.error!!.contains("safe session", ignoreCase = true))
    }

    @Test
    fun `request error surfaces`() = runTest {
        val s = vm(FakeApi()) // no sessionResponse configured -> 500
        s.start()
        advanceUntilIdle()
        assertTrue(s.state.value.error != null)
    }
}

package pro.d11l.fitcoach.data

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import pro.d11l.fitcoach.feature.session.sampleSession
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.FakeSessionCache
import pro.d11l.fitcoach.testing.errorResponse
import retrofit2.Response

@OptIn(ExperimentalCoroutinesApi::class)
class SessionRepositoryTest {

    private val sample = sampleSession()

    @Test
    fun `generate caches the session under a stable client_session_id`() = runTest {
        val api = FakeApi().apply { sessionResponse = Response.success(sample) }
        val cache = FakeSessionCache()
        val repo = SessionRepository(api, cache) { "fixed-csid" }

        val result = repo.generate()

        assertTrue(result.isSuccess)
        assertEquals("fixed-csid", cache.latest()?.clientSessionId)
        assertEquals(sample.id, cache.latest()?.session?.id)
    }

    @Test
    fun `offline generate falls back to the cached session`() = runTest {
        val cache = FakeSessionCache(CachedSession("c-1", sample, status = "active", completedAt = null))
        val api = FakeApi().apply { generateThrows = true } // simulate no connectivity
        val repo = SessionRepository(api, cache)

        val result = repo.generate()

        assertTrue(result.isSuccess)
        assertEquals(sample.id, result.getOrNull()?.id)
    }

    @Test
    fun `offline generate with no cache surfaces the failure`() = runTest {
        val api = FakeApi().apply { generateThrows = true }
        val repo = SessionRepository(api, FakeSessionCache())

        assertTrue(repo.generate().isFailure)
    }

    @Test
    fun `422 surfaces the friendly message and does not serve a stale cache`() = runTest {
        val cache = FakeSessionCache(CachedSession("c-1", sample, status = "active", completedAt = null))
        val api = FakeApi().apply { sessionResponse = errorResponse(422) }
        val repo = SessionRepository(api, cache)

        val result = repo.generate()

        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("safe session", ignoreCase = true) == true)
    }

    @Test
    fun `cached returns null before generation and the session after`() = runTest {
        val cache = FakeSessionCache()
        val api = FakeApi().apply { sessionResponse = Response.success(sample) }
        val repo = SessionRepository(api, cache) { "csid" }

        assertNull(repo.cached())
        repo.generate()
        assertEquals(sample.id, repo.cached()?.session?.id)
        assertFalse(cache.clearCalled)
    }
}

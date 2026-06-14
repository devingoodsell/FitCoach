package pro.d11l.fitcoach.data

import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.junit.Assert.assertEquals
import org.junit.Test
import pro.d11l.fitcoach.core.network.MemorySection
import pro.d11l.fitcoach.core.network.MemorySections
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.FakeMemoryCache
import pro.d11l.fitcoach.testing.errorResponse
import retrofit2.Response

class MemoryRepositoryTest {

    private fun section(name: String, age: Int) = MemorySection(
        section = name,
        schemaVersion = 1,
        data = JsonObject(mapOf("age" to JsonPrimitive(age))),
    )

    @Test
    fun `sync caches backend memory`() = runTest {
        val api = FakeApi().apply {
            memoryResponse = Response.success(MemorySections(listOf(section("profile", 40))))
        }
        val cache = FakeMemoryCache()
        val repo = MemoryRepository(api, cache)

        val result = repo.sync()

        assertEquals(1, result.size)
        assertEquals("profile", result[0].section)
        // Cache now holds the synced data for offline reads.
        assertEquals(1, repo.cached().size)
    }

    @Test
    fun `sync falls back to cache on network failure`() = runTest {
        val cache = FakeMemoryCache(
            listOf(CachedSection("goals", 1, "{\"strength\":0.5}", null)),
        )
        val api = FakeApi().apply { memoryResponse = errorResponse(500) }
        val repo = MemoryRepository(api, cache)

        val result = repo.sync()

        // Offline-first: returns the previously cached data instead of failing.
        assertEquals(1, result.size)
        assertEquals("goals", result[0].section)
    }
}

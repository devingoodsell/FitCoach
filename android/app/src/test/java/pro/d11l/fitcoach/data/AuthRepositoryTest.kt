package pro.d11l.fitcoach.data

import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import pro.d11l.fitcoach.core.auth.Tokens
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.FakeMemoryCache
import pro.d11l.fitcoach.testing.InMemoryTokenStorage
import pro.d11l.fitcoach.testing.errorResponse

class AuthRepositoryTest {

    @Test
    fun `login persists tokens`() = runTest {
        val storage = InMemoryTokenStorage()
        val repo = AuthRepository(FakeApi(), storage, FakeMemoryCache())

        val result = repo.login("user@example.com", "abcd1234ef")

        assertTrue(result is AuthResult.Success)
        assertTrue(repo.isLoggedIn())
        assertTrue(storage.load() != null)
    }

    @Test
    fun `logout clears tokens and cache`() = runTest {
        val storage = InMemoryTokenStorage(Tokens("a", "r"))
        val cache = FakeMemoryCache()
        val api = FakeApi()
        val repo = AuthRepository(api, storage, cache)

        repo.logout()

        assertNull(storage.load())
        assertTrue(cache.clearCalled)
        assertTrue(api.logoutCalled)
        assertFalse(repo.isLoggedIn())
    }

    @Test
    fun `failed login does not persist a session`() = runTest {
        val storage = InMemoryTokenStorage()
        val repo = AuthRepository(
            FakeApi().apply { loginResponse = errorResponse(401) },
            storage,
            FakeMemoryCache(),
        )

        val result = repo.login("user@example.com", "abcd1234ef")

        assertTrue(result is AuthResult.Failure)
        assertFalse(repo.isLoggedIn())
    }

    @Test
    fun `delete account wipes local state on success`() = runTest {
        val storage = InMemoryTokenStorage(Tokens("a", "r"))
        val cache = FakeMemoryCache()
        val repo = AuthRepository(FakeApi(), storage, cache)

        val result = repo.deleteAccount("abcd1234ef")

        assertTrue(result is AuthResult.Success)
        assertNull(storage.load())
        assertTrue(cache.clearCalled)
    }
}

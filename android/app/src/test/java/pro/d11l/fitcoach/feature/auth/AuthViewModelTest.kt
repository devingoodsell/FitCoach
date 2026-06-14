package pro.d11l.fitcoach.feature.auth

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import pro.d11l.fitcoach.data.AuthRepository
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.FakeMemoryCache
import pro.d11l.fitcoach.testing.InMemoryTokenStorage
import pro.d11l.fitcoach.testing.MainDispatcherRule
import pro.d11l.fitcoach.testing.errorResponse
import pro.d11l.fitcoach.core.network.TokenPair

@OptIn(ExperimentalCoroutinesApi::class)
class AuthViewModelTest {

    @get:Rule
    val mainDispatcher = MainDispatcherRule()

    private fun viewModel(api: FakeApi = FakeApi()): Pair<AuthViewModel, AuthRepository> {
        val repo = AuthRepository(api, InMemoryTokenStorage(), FakeMemoryCache())
        return AuthViewModel(repo) to repo
    }

    @Test
    fun `valid login authenticates`() = runTest {
        val (vm, _) = viewModel()
        vm.onEmailChange("user@example.com")
        vm.onPasswordChange("abcd1234ef")
        vm.submit()
        advanceUntilIdle()

        val state = vm.state.value
        assertTrue(state.authenticated)
        assertFalse(state.isSubmitting)
        assertNull(state.error)
    }

    @Test
    fun `weak password is rejected without a network call`() = runTest {
        val (vm, _) = viewModel()
        vm.onEmailChange("user@example.com")
        vm.onPasswordChange("short")
        vm.submit()
        advanceUntilIdle()

        assertFalse(vm.state.value.authenticated)
        assertNotNull(vm.state.value.error)
    }

    @Test
    fun `invalid email is rejected`() = runTest {
        val (vm, _) = viewModel()
        vm.onEmailChange("not-an-email")
        vm.onPasswordChange("abcd1234ef")
        vm.submit()
        advanceUntilIdle()

        assertFalse(vm.state.value.authenticated)
        assertNotNull(vm.state.value.error)
    }

    @Test
    fun `bad credentials surface a generic error`() = runTest {
        val api = FakeApi().apply { loginResponse = errorResponse<TokenPair>(401) }
        val (vm, _) = viewModel(api)
        vm.onEmailChange("user@example.com")
        vm.onPasswordChange("abcd1234ef")
        vm.submit()
        advanceUntilIdle()

        assertFalse(vm.state.value.authenticated)
        assertEquals("Incorrect email or password.", vm.state.value.error)
    }

    @Test
    fun `signup mode toggles`() {
        val (vm, _) = viewModel()
        assertEquals(AuthMode.Login, vm.state.value.mode)
        vm.toggleMode()
        assertEquals(AuthMode.Signup, vm.state.value.mode)
    }
}

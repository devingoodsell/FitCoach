package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.auth.TokenStorage
import pro.d11l.fitcoach.core.auth.Tokens
import pro.d11l.fitcoach.core.network.ConsentRequest
import pro.d11l.fitcoach.core.network.Credentials
import pro.d11l.fitcoach.core.network.DeleteAccountRequest
import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.RefreshRequest
import pro.d11l.fitcoach.core.network.TokenPair

/** Result of an auth attempt, surfaced to the ViewModel as a sealed outcome. */
sealed interface AuthResult {
    data object Success : AuthResult
    data class Failure(val code: String, val message: String) : AuthResult
}

/**
 * Single source of truth for authentication. Persists sessions via TokenStorage
 * (Keystore-backed) and clears local caches on logout so no personal data lingers
 * on the device (E1-S2).
 */
class AuthRepository(
    private val api: FitCoachApi,
    private val tokenStorage: TokenStorage,
    private val memoryCache: MemoryCache,
) {
    /** True if a session is persisted (survives app restarts). */
    fun isLoggedIn(): Boolean = tokenStorage.load() != null

    suspend fun signup(email: String, password: String): AuthResult =
        authenticate { api.signup(Credentials(email.trim(), password)) }

    suspend fun login(email: String, password: String): AuthResult =
        authenticate { api.login(Credentials(email.trim(), password)) }

    /** Revokes the session server-side (best effort) and wipes all local state. */
    suspend fun logout() {
        val refresh = tokenStorage.load()?.refreshToken
        if (refresh != null) {
            runCatching { api.logout(RefreshRequest(refresh)) }
        }
        tokenStorage.clear()
        memoryCache.clear()
    }

    suspend fun requestPasswordReset(email: String) {
        runCatching { api.requestReset(pro.d11l.fitcoach.core.network.ResetRequest(email.trim())) }
    }

    suspend fun recordConsent(type: String, version: String): AuthResult = runCatchingResult {
        val resp = api.recordConsent(ConsentRequest(type, version))
        if (resp.isSuccessful) AuthResult.Success else resp.toFailure()
    }

    /** Deletes the account server-side, then wipes local state. */
    suspend fun deleteAccount(password: String): AuthResult = runCatchingResult {
        val resp = api.deleteAccount(DeleteAccountRequest(password))
        if (resp.isSuccessful) {
            tokenStorage.clear()
            memoryCache.clear()
            AuthResult.Success
        } else {
            resp.toFailure()
        }
    }

    private suspend fun authenticate(call: suspend () -> retrofit2.Response<TokenPair>): AuthResult =
        runCatchingResult {
            val resp = call()
            val body = resp.body()
            if (resp.isSuccessful && body != null) {
                tokenStorage.save(Tokens(body.accessToken, body.refreshToken))
                AuthResult.Success
            } else {
                resp.toFailure()
            }
        }
}

private inline fun runCatchingResult(block: () -> AuthResult): AuthResult =
    try {
        block()
    } catch (e: Exception) {
        AuthResult.Failure("network_error", e.message ?: "network error")
    }

private fun retrofit2.Response<*>.toFailure(): AuthResult.Failure =
    AuthResult.Failure("http_${code()}", "request failed (${code()})")

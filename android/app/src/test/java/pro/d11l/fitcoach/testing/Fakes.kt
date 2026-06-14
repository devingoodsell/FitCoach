package pro.d11l.fitcoach.testing

import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.ResponseBody.Companion.toResponseBody
import pro.d11l.fitcoach.core.auth.TokenStorage
import pro.d11l.fitcoach.core.auth.Tokens
import pro.d11l.fitcoach.core.network.ConsentList
import pro.d11l.fitcoach.core.network.ConsentRecord
import pro.d11l.fitcoach.core.network.ConsentRequest
import pro.d11l.fitcoach.core.network.Credentials
import pro.d11l.fitcoach.core.network.DeleteAccountRequest
import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.MemorySection
import pro.d11l.fitcoach.core.network.MemorySections
import pro.d11l.fitcoach.core.network.PutSectionRequest
import pro.d11l.fitcoach.core.network.RefreshRequest
import pro.d11l.fitcoach.core.network.ResetRequest
import pro.d11l.fitcoach.core.network.TokenPair
import pro.d11l.fitcoach.data.CachedSection
import pro.d11l.fitcoach.data.MemoryCache
import retrofit2.Response

/** In-memory TokenStorage for tests. */
class InMemoryTokenStorage(private var tokens: Tokens? = null) : TokenStorage {
    override fun save(tokens: Tokens) {
        this.tokens = tokens
    }

    override fun load(): Tokens? = tokens
    override fun clear() {
        tokens = null
    }
}

/** In-memory MemoryCache for tests, tracking whether it was cleared. */
class FakeMemoryCache(private var sections: List<CachedSection> = emptyList()) : MemoryCache {
    var clearCalled = false
        private set

    override suspend fun replaceAll(sections: List<CachedSection>) {
        this.sections = sections
    }

    override suspend fun all(): List<CachedSection> = sections
    override suspend fun clear() {
        clearCalled = true
        sections = emptyList()
    }
}

/** Configurable fake of the backend API. */
class FakeApi : FitCoachApi {
    var tokenPair = TokenPair("access-jwt", "refresh-opaque", "Bearer", 900)
    var signupResponse: Response<TokenPair>? = null
    var loginResponse: Response<TokenPair>? = null
    var memoryResponse: Response<MemorySections> = Response.success(MemorySections())
    var deleteResponse: Response<Unit> = Response.success(Unit)
    var consentResponse: Response<ConsentRecord> = Response.success(ConsentRecord("health_data", "v1"))

    var logoutCalled = false
    var lastConsent: ConsentRequest? = null

    override suspend fun signup(body: Credentials): Response<TokenPair> =
        signupResponse ?: Response.success(tokenPair)

    override suspend fun login(body: Credentials): Response<TokenPair> =
        loginResponse ?: Response.success(tokenPair)

    override suspend fun refresh(body: RefreshRequest): Response<TokenPair> = Response.success(tokenPair)

    override suspend fun logout(body: RefreshRequest): Response<Unit> {
        logoutCalled = true
        return Response.success(Unit)
    }

    override suspend fun requestReset(body: ResetRequest): Response<Unit> = Response.success(Unit)

    override suspend fun listConsent(): Response<ConsentList> = Response.success(ConsentList())

    override suspend fun recordConsent(body: ConsentRequest): Response<ConsentRecord> {
        lastConsent = body
        return consentResponse
    }

    override suspend fun memory(): Response<MemorySections> = memoryResponse

    override suspend fun putSection(section: String, body: PutSectionRequest): Response<MemorySection> =
        Response.success(MemorySection(section, 1, kotlinx.serialization.json.JsonObject(emptyMap())))

    override suspend fun deleteAccount(body: DeleteAccountRequest): Response<Unit> = deleteResponse
}

/** Builds a Retrofit-style error response with the given status code. */
fun <T> errorResponse(code: Int): Response<T> =
    Response.error(code, "{}".toResponseBody("application/json".toMediaTypeOrNull()))

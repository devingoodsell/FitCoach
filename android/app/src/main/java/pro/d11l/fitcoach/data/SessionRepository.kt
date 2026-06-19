package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.ReplanCheckDto
import pro.d11l.fitcoach.core.network.SessionDto
import java.util.UUID

/**
 * Generates a workout session from the backend (E5) and caches it for offline use
 * (E5-PR5). Generation is one server-side Claude call; the app never talks to
 * Anthropic. Once generated, the session is persisted via [SessionCache] so the
 * in-session experience (E6) and on-device autoregulation (E5-PR6) run with no
 * connectivity. A stable, device-minted `client_session_id` is attached on cache
 * and reused by the offline sync queue (E12-PR2) so replays never duplicate.
 */
class SessionRepository(
    private val api: FitCoachApi,
    private val cache: SessionCache,
    private val newClientSessionId: () -> String = { UUID.randomUUID().toString() },
) {

    /**
     * Generates today's session ("Start workout") and caches it offline. 422 maps
     * to a friendly message. Only a true connectivity failure falls back to the
     * cached session (so a dead signal never blocks training, E12-S1); a server
     * refusal (422) still surfaces so the user is prompted to review injuries.
     */
    suspend fun generate(): Result<SessionDto> {
        val resp = try {
            api.generateSession()
        } catch (cancel: kotlinx.coroutines.CancellationException) {
            throw cancel
        } catch (networkError: Exception) {
            return cache.latest()?.session?.let { Result.success(it) }
                ?: Result.failure(networkError)
        }
        return when {
            resp.isSuccessful -> runCatching {
                val session = resp.body() ?: error("empty session response")
                cache.save(session, newClientSessionId())
                session
            }
            resp.code() == 422 ->
                Result.failure(IllegalStateException("We couldn't build a safe session — please review your injuries."))
            else -> Result.failure(IllegalStateException("request failed (${resp.code()})"))
        }
    }

    /** The cached session, if one was generated earlier — the offline read path (E12-S1). */
    suspend fun cached(): CachedSession? = cache.latest()

    /** The cached session flattened into ordered player steps (E6), offline. */
    suspend fun plan(): SessionPlan? = cache.loadPlan()

    /** Persists logged actuals for one set, offline (E6-PR2). */
    suspend fun logSet(setId: Long, logged: LoggedSetState) = cache.logSet(setId, logged)

    /** Marks the cached session completed (E6-PR5). */
    suspend fun markCompleted(sessionId: String, completedAt: String) =
        cache.markCompleted(sessionId, completedAt)

    /** Asks whether a session cached at [since] (RFC 3339) should be regenerated. */
    suspend fun replanCheck(since: String): Result<ReplanCheckDto> = runCatching {
        val resp = api.replanCheck(since)
        resp.body()?.takeIf { resp.isSuccessful }
            ?: error("request failed (${resp.code()})")
    }
}

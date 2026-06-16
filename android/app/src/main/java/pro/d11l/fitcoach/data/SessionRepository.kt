package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.ReplanCheckDto
import pro.d11l.fitcoach.core.network.SessionDto

/**
 * Generates a workout session from the backend (E5). Generation is one
 * server-side Claude call; the app never talks to Anthropic. Offline caching of
 * the returned session (E5-PR5) and on-device autoregulation (E5-PR6) build on
 * top of this against the published session shape.
 */
class SessionRepository(private val api: FitCoachApi) {

    /** Generates today's session ("Start workout"). 422 maps to a friendly message. */
    suspend fun generate(): Result<SessionDto> = runCatching {
        val resp = api.generateSession()
        when {
            resp.isSuccessful -> resp.body() ?: error("empty session response")
            resp.code() == 422 -> error("We couldn't build a safe session — please review your injuries.")
            else -> error("request failed (${resp.code()})")
        }
    }

    /** Asks whether a session cached at [since] (RFC 3339) should be regenerated. */
    suspend fun replanCheck(since: String): Result<ReplanCheckDto> = runCatching {
        val resp = api.replanCheck(since)
        resp.body()?.takeIf { resp.isSuccessful }
            ?: error("request failed (${resp.code()})")
    }
}

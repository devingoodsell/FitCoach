package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.ReadinessDto

/** Reads today's readiness from the backend (E4). Signal ingestion via Health
 *  Connect (E4-PR5) is a separate, device-dependent concern. */
class ReadinessRepository(private val api: FitCoachApi) {

    suspend fun today(): Result<ReadinessDto> = runCatching {
        val resp = api.getReadiness()
        resp.body()?.takeIf { resp.isSuccessful }
            ?: error("request failed (${resp.code()})")
    }
}

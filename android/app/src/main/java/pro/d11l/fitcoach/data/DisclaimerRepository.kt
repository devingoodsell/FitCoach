package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.designsystem.DisclaimerText
import pro.d11l.fitcoach.core.network.FitCoachApi

/**
 * Fetches the central disclaimer copy from the backend (GET /disclaimers, E13-PR1)
 * so the client renders server-owned language instead of hardcoding it (E13-PR2).
 * Any failure falls back to the bundled copy so the disclaimer always shows.
 */
class DisclaimerRepository(private val api: FitCoachApi) {

    suspend fun fetch(): DisclaimerText =
        try {
            val resp = api.getDisclaimers()
            val body = resp.body()
            if (resp.isSuccessful && body != null) {
                DisclaimerText(version = body.version, medical = body.medical, healthData = body.healthData)
            } else {
                DisclaimerText.Bundled
            }
        } catch (_: Exception) {
            DisclaimerText.Bundled
        }
}

package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.network.ConsentRecord
import pro.d11l.fitcoach.core.network.FitCoachApi

/** Consent type identifiers shared with the backend. */
object ConsentTypes {
    const val HEALTH_DATA = "health_data"
    const val MEDICAL_DISCLAIMER = "medical_disclaimer"
}

/**
 * Reads consent state and revokes consents for the Settings review surface (E14-S2).
 * Recording new consent lives on AuthRepository (used at the consent step); this
 * repository covers the review/revoke side.
 */
class ConsentRepository(private val api: FitCoachApi) {

    suspend fun load(): Result<List<ConsentRecord>> = runCatching {
        val resp = api.listConsent()
        if (resp.isSuccessful) {
            resp.body()?.consents ?: emptyList()
        } else {
            error("request failed (${resp.code()})")
        }
    }

    /** Revokes the given consent. Revoking health-data flips it off on the backend,
     *  disabling readiness ingestion (the app falls back to manual mode). */
    suspend fun revoke(type: String): Result<Unit> = runCatching {
        val resp = api.revokeConsent(type)
        if (!resp.isSuccessful) {
            error("request failed (${resp.code()})")
        }
    }
}

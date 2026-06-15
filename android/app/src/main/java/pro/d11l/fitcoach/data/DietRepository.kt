package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.network.DietTargetsDto
import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.PostWorkoutNoteDto

/** Reads nutrition targets and guidance from the backend (E11). */
class DietRepository(private val api: FitCoachApi) {

    suspend fun targets(): Result<DietTargetsDto> = runCatching {
        val resp = api.getDietTargets()
        resp.body()?.takeIf { resp.isSuccessful }
            ?: error("request failed (${resp.code()})")
    }

    suspend fun postWorkoutNote(heavy: Boolean): Result<PostWorkoutNoteDto> = runCatching {
        val resp = api.getPostWorkoutNote(if (heavy) "heavy" else "light")
        resp.body()?.takeIf { resp.isSuccessful }
            ?: error("request failed (${resp.code()})")
    }
}

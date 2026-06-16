package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.InjuriesDocDto
import pro.d11l.fitcoach.core.network.InjuryDraftDto
import pro.d11l.fitcoach.core.network.InjuryDto
import pro.d11l.fitcoach.core.network.ParseInjuryRequest
import retrofit2.Response

/** Mediates injuries against the backend (E7). */
class InjuryRepository(private val api: FitCoachApi) {

    suspend fun load(): Result<InjuriesDocDto> = call { api.getInjuries() }
    suspend fun parse(text: String): Result<InjuryDraftDto> = call { api.parseInjury(ParseInjuryRequest(text)) }
    suspend fun add(injury: InjuryDto): Result<InjuryDto> = call { api.addInjury(injury) }
    suspend fun update(id: String, injury: InjuryDto): Result<InjuryDto> = call { api.updateInjury(id, injury) }

    suspend fun delete(id: String): Result<Unit> = runCatching {
        val resp = api.deleteInjury(id)
        if (!resp.isSuccessful) error("request failed (${resp.code()})")
    }

    private suspend fun <T> call(block: suspend () -> Response<T>): Result<T> = runCatching {
        val resp = block()
        resp.body()?.takeIf { resp.isSuccessful } ?: error("request failed (${resp.code()})")
    }
}

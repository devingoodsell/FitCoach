package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.network.CurrentContextDto
import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.LocationDto
import pro.d11l.fitcoach.core.network.LocationInputDto
import pro.d11l.fitcoach.core.network.LocationsDocDto
import pro.d11l.fitcoach.core.network.SetCurrentContextDto

/** Result wrapper for location operations. */
sealed interface LocationResult<out T> {
    data class Ok<T>(val value: T) : LocationResult<T>
    data class Error(val message: String) : LocationResult<Nothing>
}

/** Mediates locations & current context against the backend (E9). */
class LocationRepository(private val api: FitCoachApi) {

    suspend fun load(): LocationResult<LocationsDocDto> = call { api.getLocations() }

    suspend fun add(name: String, equipment: List<String>): LocationResult<LocationDto> =
        call { api.addLocation(LocationInputDto(name, equipment)) }

    suspend fun update(id: String, name: String, equipment: List<String>): LocationResult<LocationDto> =
        call { api.updateLocation(id, LocationInputDto(name, equipment)) }

    suspend fun delete(id: String): LocationResult<Unit> = call { api.deleteLocation(id) }

    suspend fun setCurrent(locationId: String, note: String): LocationResult<CurrentContextDto> =
        call { api.setCurrentContext(SetCurrentContextDto(locationId, note)) }

    private suspend fun <T> call(block: suspend () -> retrofit2.Response<T>): LocationResult<T> =
        try {
            val resp = block()
            val body = resp.body()
            when {
                resp.isSuccessful && body != null -> LocationResult.Ok(body)
                resp.isSuccessful -> @Suppress("UNCHECKED_CAST") (LocationResult.Ok(Unit as T))
                else -> LocationResult.Error("request failed (${resp.code()})")
            }
        } catch (e: Exception) {
            LocationResult.Error(e.message ?: "network error")
        }
}

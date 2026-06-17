package pro.d11l.fitcoach.data

import kotlinx.serialization.KSerializer
import pro.d11l.fitcoach.core.network.DietPrefsDto
import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.GoalWeightsDto
import pro.d11l.fitcoach.core.network.NetworkModule
import pro.d11l.fitcoach.core.network.PreferencesDto
import pro.d11l.fitcoach.core.network.ProfileDto
import pro.d11l.fitcoach.core.network.ScheduleDto
import pro.d11l.fitcoach.core.network.ValidationErrorDto
import retrofit2.Response

/** Outcome of saving an onboarding section. */
sealed interface SaveResult {
    data object Ok : SaveResult
    /** Server-side validation failed; per-field messages keyed by field name. */
    data class Invalid(val fields: Map<String, String>) : SaveResult
    data class Error(val message: String) : SaveResult
}

/** Writes user-model sections through the backend onboarding endpoints (E2). */
class OnboardingRepository(private val api: FitCoachApi) {

    suspend fun saveProfile(dto: ProfileDto): SaveResult = call { api.saveProfile(dto) }
    suspend fun saveGoals(dto: GoalWeightsDto): SaveResult = call { api.saveGoals(dto) }
    suspend fun saveSchedule(dto: ScheduleDto): SaveResult = call { api.saveSchedule(dto) }
    suspend fun saveDiet(dto: DietPrefsDto): SaveResult = call { api.saveDiet(dto) }
    suspend fun savePreferences(dto: PreferencesDto): SaveResult = call { api.savePreferences(dto) }

    // Prefill reads for Settings editing (E14): decode the current Coach Memory
    // section into its typed DTO. Returns null when the section isn't set yet
    // (HTTP 404), or on any read/parse failure — the edit form then starts blank.
    suspend fun loadProfile(): ProfileDto? = loadSection("profile", ProfileDto.serializer())
    suspend fun loadGoals(): GoalWeightsDto? = loadSection("goals", GoalWeightsDto.serializer())
    suspend fun loadSchedule(): ScheduleDto? = loadSection("schedule", ScheduleDto.serializer())
    suspend fun loadDiet(): DietPrefsDto? = loadSection("diet", DietPrefsDto.serializer())
    suspend fun loadPreferences(): PreferencesDto? = loadSection("preferences", PreferencesDto.serializer())

    private suspend fun <T> loadSection(section: String, serializer: KSerializer<T>): T? =
        try {
            val resp = api.getMemorySection(section)
            val body = resp.body()
            if (resp.isSuccessful && body != null) {
                NetworkModule.json.decodeFromJsonElement(serializer, body.data)
            } else {
                null
            }
        } catch (_: Exception) {
            null
        }

    private suspend fun <T> call(block: suspend () -> Response<T>): SaveResult =
        try {
            val resp = block()
            when {
                resp.isSuccessful -> SaveResult.Ok
                resp.code() == 400 -> SaveResult.Invalid(parseFieldErrors(resp))
                else -> SaveResult.Error("request failed (${resp.code()})")
            }
        } catch (e: Exception) {
            SaveResult.Error(e.message ?: "network error")
        }

    private fun parseFieldErrors(resp: Response<*>): Map<String, String> {
        val raw = resp.errorBody()?.string().orEmpty()
        return runCatching {
            NetworkModule.json.decodeFromString(ValidationErrorDto.serializer(), raw).fields
        }.getOrDefault(emptyMap())
    }
}

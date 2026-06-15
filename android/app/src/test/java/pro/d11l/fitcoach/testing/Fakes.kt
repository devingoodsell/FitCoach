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
import pro.d11l.fitcoach.core.network.CurrentContextDto
import pro.d11l.fitcoach.core.network.DietPrefsDto
import pro.d11l.fitcoach.core.network.DietTargetsDto
import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.GoalWeightsDto
import pro.d11l.fitcoach.core.network.LocationDto
import pro.d11l.fitcoach.core.network.LocationInputDto
import pro.d11l.fitcoach.core.network.LocationsDocDto
import pro.d11l.fitcoach.core.network.PostWorkoutNoteDto
import pro.d11l.fitcoach.core.network.PreferencesDto
import pro.d11l.fitcoach.core.network.ProfileDto
import pro.d11l.fitcoach.core.network.ScheduleDto
import pro.d11l.fitcoach.core.network.SetCurrentContextDto
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

    // onboarding: default to echoing the request back as success
    var profileResponse: Response<ProfileDto>? = null
    var lastProfile: ProfileDto? = null
    var lastGoals: GoalWeightsDto? = null
    var lastSchedule: ScheduleDto? = null
    var lastDiet: DietPrefsDto? = null
    var lastPreferences: PreferencesDto? = null

    // locations: an in-memory doc the fake mutates so reloads reflect writes
    var locationsDoc = LocationsDocDto()
    var lastSetCurrent: SetCurrentContextDto? = null
    var locationsError = false

    // diet
    var dietTargets = DietTargetsDto()
    var postWorkoutNote = PostWorkoutNoteDto(note = "default note", disclaimer = "d")
    var dietError = false

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

    override suspend fun saveProfile(body: ProfileDto): Response<ProfileDto> {
        lastProfile = body
        return profileResponse ?: Response.success(body)
    }

    override suspend fun saveGoals(body: GoalWeightsDto): Response<GoalWeightsDto> {
        lastGoals = body
        return Response.success(body)
    }

    override suspend fun saveSchedule(body: ScheduleDto): Response<ScheduleDto> {
        lastSchedule = body
        return Response.success(body)
    }

    override suspend fun saveDiet(body: DietPrefsDto): Response<DietPrefsDto> {
        lastDiet = body
        return Response.success(body)
    }

    override suspend fun savePreferences(body: PreferencesDto): Response<PreferencesDto> {
        lastPreferences = body
        return Response.success(body)
    }

    override suspend fun getLocations(): Response<LocationsDocDto> =
        if (locationsError) errorResponse(500) else Response.success(locationsDoc)

    override suspend fun addLocation(body: LocationInputDto): Response<LocationDto> {
        val created = LocationDto(id = "loc-${locationsDoc.locations.size + 1}", name = body.name, equipment = body.equipment)
        locationsDoc = locationsDoc.copy(locations = locationsDoc.locations + created)
        return Response.success(created)
    }

    override suspend fun updateLocation(id: String, body: LocationInputDto): Response<LocationDto> {
        val updated = LocationDto(id = id, name = body.name, equipment = body.equipment)
        locationsDoc = locationsDoc.copy(locations = locationsDoc.locations.map { if (it.id == id) updated else it })
        return Response.success(updated)
    }

    override suspend fun deleteLocation(id: String): Response<Unit> {
        locationsDoc = locationsDoc.copy(locations = locationsDoc.locations.filterNot { it.id == id })
        return Response.success(Unit)
    }

    override suspend fun setCurrentContext(body: SetCurrentContextDto): Response<CurrentContextDto> {
        lastSetCurrent = body
        val current = CurrentContextDto(locationId = body.locationId, note = body.note, changedAt = "2026-06-14T12:00:00Z")
        locationsDoc = locationsDoc.copy(currentContext = current)
        return Response.success(current)
    }

    override suspend fun getDietTargets(): Response<DietTargetsDto> =
        if (dietError) errorResponse(500) else Response.success(dietTargets)

    override suspend fun getPostWorkoutNote(intensity: String): Response<PostWorkoutNoteDto> =
        Response.success(postWorkoutNote.copy(note = "$intensity: ${postWorkoutNote.note}"))
}

/** Builds a Retrofit-style error response with the given status code. */
fun <T> errorResponse(code: Int): Response<T> =
    Response.error(code, "{}".toResponseBody("application/json".toMediaTypeOrNull()))

/** Builds a 400 validation error with a fields map, as the backend returns. */
fun <T> validationErrorResponse(fields: Map<String, String>): Response<T> {
    val entries = fields.entries.joinToString(",") { "\"${it.key}\":\"${it.value}\"" }
    val body = "{\"error\":\"validation_failed\",\"message\":\"invalid\",\"fields\":{$entries}}"
    return Response.error(400, body.toResponseBody("application/json".toMediaTypeOrNull()))
}

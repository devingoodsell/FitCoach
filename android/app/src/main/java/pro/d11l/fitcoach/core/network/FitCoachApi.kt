package pro.d11l.fitcoach.core.network

import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.HTTP
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

/**
 * Typed client for OUR backend. The app never calls Anthropic directly; the
 * Claude key lives only server-side.
 */
interface FitCoachApi {

    @POST("auth/signup")
    suspend fun signup(@Body body: Credentials): Response<TokenPair>

    @POST("auth/login")
    suspend fun login(@Body body: Credentials): Response<TokenPair>

    @POST("auth/refresh")
    suspend fun refresh(@Body body: RefreshRequest): Response<TokenPair>

    @POST("auth/logout")
    suspend fun logout(@Body body: RefreshRequest): Response<Unit>

    @POST("auth/reset/request")
    suspend fun requestReset(@Body body: ResetRequest): Response<Unit>

    @GET("disclaimers")
    suspend fun getDisclaimers(): Response<DisclaimerDocDto>

    @GET("consent")
    suspend fun listConsent(): Response<ConsentList>

    @POST("consent")
    suspend fun recordConsent(@Body body: ConsentRequest): Response<ConsentRecord>

    @GET("memory")
    suspend fun memory(): Response<MemorySections>

    @GET("memory/{section}")
    suspend fun getMemorySection(@Path("section") section: String): Response<MemorySection>

    @PUT("memory/{section}")
    suspend fun putSection(
        @Path("section") section: String,
        @Body body: PutSectionRequest,
    ): Response<MemorySection>

    // DELETE with a body: HTTP annotation allows it where @DELETE does not.
    @HTTP(method = "DELETE", path = "account", hasBody = true)
    suspend fun deleteAccount(@Body body: DeleteAccountRequest): Response<Unit>

    @PUT("onboarding/profile")
    suspend fun saveProfile(@Body body: ProfileDto): Response<ProfileDto>

    @PUT("onboarding/goals")
    suspend fun saveGoals(@Body body: GoalWeightsDto): Response<GoalWeightsDto>

    @PUT("onboarding/schedule")
    suspend fun saveSchedule(@Body body: ScheduleDto): Response<ScheduleDto>

    @PUT("onboarding/diet")
    suspend fun saveDiet(@Body body: DietPrefsDto): Response<DietPrefsDto>

    @PUT("onboarding/preferences")
    suspend fun savePreferences(@Body body: PreferencesDto): Response<PreferencesDto>

    @GET("locations")
    suspend fun getLocations(): Response<LocationsDocDto>

    @POST("locations")
    suspend fun addLocation(@Body body: LocationInputDto): Response<LocationDto>

    @PUT("locations/{id}")
    suspend fun updateLocation(@Path("id") id: String, @Body body: LocationInputDto): Response<LocationDto>

    @DELETE("locations/{id}")
    suspend fun deleteLocation(@Path("id") id: String): Response<Unit>

    @PUT("locations/current")
    suspend fun setCurrentContext(@Body body: SetCurrentContextDto): Response<CurrentContextDto>

    @GET("diet/targets")
    suspend fun getDietTargets(): Response<DietTargetsDto>

    @GET("diet/post-workout-note")
    suspend fun getPostWorkoutNote(@Query("intensity") intensity: String): Response<PostWorkoutNoteDto>

    @GET("readiness")
    suspend fun getReadiness(): Response<ReadinessDto>

    @GET("injuries")
    suspend fun getInjuries(): Response<InjuriesDocDto>

    @POST("injuries")
    suspend fun addInjury(@Body body: InjuryDto): Response<InjuryDto>

    @PUT("injuries/{id}")
    suspend fun updateInjury(@Path("id") id: String, @Body body: InjuryDto): Response<InjuryDto>

    @DELETE("injuries/{id}")
    suspend fun deleteInjury(@Path("id") id: String): Response<Unit>

    @POST("injuries/parse")
    suspend fun parseInjury(@Body body: ParseInjuryRequest): Response<InjuryDraftDto>

    // Session generation is the single server-side Claude call; the client never
    // calls Anthropic directly.
    @POST("sessions/generate")
    suspend fun generateSession(): Response<SessionDto>

    @GET("sessions/replan-check")
    suspend fun replanCheck(@Query("since") since: String): Response<ReplanCheckDto>
}

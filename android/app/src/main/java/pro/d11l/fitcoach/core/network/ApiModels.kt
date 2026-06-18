package pro.d11l.fitcoach.core.network

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

/** Request/response DTOs mirroring backend/api/openapi.yaml. */

@Serializable
data class Credentials(val email: String, val password: String)

@Serializable
data class TokenPair(
    @SerialName("access_token") val accessToken: String,
    @SerialName("refresh_token") val refreshToken: String,
    @SerialName("token_type") val tokenType: String,
    @SerialName("expires_in") val expiresIn: Int,
)

@Serializable
data class RefreshRequest(@SerialName("refresh_token") val refreshToken: String)

@Serializable
data class ResetRequest(val email: String)

@Serializable
data class DeleteAccountRequest(val password: String)

@Serializable
data class DisclaimerDocDto(
    val version: String,
    val medical: String,
    @SerialName("health_data") val healthData: String,
)

@Serializable
data class ConsentRequest(val type: String, val version: String)

@Serializable
data class ConsentRecord(
    val type: String,
    val version: String,
    @SerialName("accepted_at") val acceptedAt: String? = null,
    @SerialName("revoked_at") val revokedAt: String? = null,
) {
    /** A consent is in force when it was accepted and not since revoked. */
    val isActive: Boolean get() = revokedAt == null
}

@Serializable
data class ConsentList(val consents: List<ConsentRecord> = emptyList())

@Serializable
data class MemorySection(
    val section: String,
    @SerialName("schema_version") val schemaVersion: Int = 1,
    val data: JsonElement,
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class MemorySections(val sections: List<MemorySection> = emptyList())

@Serializable
data class PutSectionRequest(val data: JsonElement)

@Serializable
data class ApiError(val error: String, val message: String? = null)

/** Onboarding DTOs (mirror the onboarding endpoints in the contract). */

@Serializable
data class BenchmarkLiftDto(
    val name: String,
    @SerialName("one_rep_max_kg") val oneRepMaxKg: Double,
)

@Serializable
data class ExperienceDto(
    @SerialName("training_age_years") val trainingAgeYears: Double? = null,
    val level: String,
    @SerialName("benchmark_lifts") val benchmarkLifts: List<BenchmarkLiftDto> = emptyList(),
)

@Serializable
data class AgingEmphasesDto(
    @SerialName("bone_balance") val boneBalance: Double = 0.0,
    @SerialName("joint_tendon") val jointTendon: Double = 0.0,
    val vo2max: Double = 0.0,
    @SerialName("cardio_base") val cardioBase: Double = 0.0,
)

@Serializable
data class ProfileDto(
    val dob: String? = null,
    val age: Int? = null,
    val sex: String,
    @SerialName("height_cm") val heightCm: Double? = null,
    @SerialName("weight_kg") val weightKg: Double? = null,
    val experience: ExperienceDto,
    // Defaulted from age server-side when null; carried through profile edits so a
    // profile save never silently resets user-tuned aging emphases (E2-S8 / E14-PR2).
    @SerialName("aging_emphases") val agingEmphases: AgingEmphasesDto? = null,
)

@Serializable
data class GoalWeightsDto(
    val strength: Double,
    val healthspan: Double,
    @SerialName("body_composition") val bodyComposition: Double,
    val performance: Double,
)

@Serializable
data class ScheduleDto(
    @SerialName("days_per_week") val daysPerWeek: Int,
    @SerialName("session_length_min") val sessionLengthMin: Int,
    @SerialName("preferred_days") val preferredDays: List<String> = emptyList(),
)

@Serializable
data class DietPrefsDto(
    val pattern: String,
    val supplements: String = "",
    val medications: String = "",
)

@Serializable
data class PreferencesDto(
    val likes: List<String> = emptyList(),
    val dislikes: List<String> = emptyList(),
    @SerialName("hard_avoids") val hardAvoids: List<String> = emptyList(),
)

@Serializable
data class ValidationErrorDto(
    val error: String,
    val message: String? = null,
    val fields: Map<String, String> = emptyMap(),
)

/** Locations DTOs (mirror the locations endpoints in the contract). */

@Serializable
data class LocationDto(
    val id: String = "",
    val name: String,
    val equipment: List<String> = emptyList(),
)

@Serializable
data class LocationInputDto(
    val name: String,
    val equipment: List<String> = emptyList(),
)

@Serializable
data class CurrentContextDto(
    @SerialName("location_id") val locationId: String,
    val note: String = "",
    @SerialName("changed_at") val changedAt: String? = null,
)

@Serializable
data class LocationsDocDto(
    val locations: List<LocationDto> = emptyList(),
    @SerialName("current_context") val currentContext: CurrentContextDto? = null,
)

@Serializable
data class SetCurrentContextDto(
    @SerialName("location_id") val locationId: String,
    val note: String = "",
)

/** Diet DTOs (mirror the diet endpoints in the contract). */

@Serializable
data class DietTargetsValuesDto(
    @SerialName("calories_min") val caloriesMin: Int = 0,
    @SerialName("calories_max") val caloriesMax: Int = 0,
    @SerialName("protein_min_g") val proteinMinG: Int = 0,
    @SerialName("protein_max_g") val proteinMaxG: Int = 0,
    @SerialName("low_confidence") val lowConfidence: Boolean = false,
)

@Serializable
data class DietTargetsDto(
    val targets: DietTargetsValuesDto = DietTargetsValuesDto(),
    val guidance: List<String> = emptyList(),
    val pattern: String = "",
    val disclaimer: String = "",
)

@Serializable
data class PostWorkoutNoteDto(
    val note: String = "",
    val disclaimer: String = "",
)

/** Readiness DTO (mirrors GET /readiness in the contract). */
@Serializable
data class ReadinessDto(
    val value: Int = 50,
    val confidence: String = "low",
    val drivers: List<String> = emptyList(),
    val explanation: String = "",
)

/** Injury DTOs (mirror the injuries endpoints in the contract). */

@Serializable
data class InjuryDto(
    val id: String = "",
    val region: String,
    val status: String,
    val severity: String = "",
    @SerialName("aggravating_movements") val aggravatingMovements: List<String> = emptyList(),
    @SerialName("onset_date") val onsetDate: String? = null,
    val notes: String = "",
)

@Serializable
data class InjuriesDocDto(
    val injuries: List<InjuryDto> = emptyList(),
    @SerialName("changed_at") val changedAt: String? = null,
)

@Serializable
data class InjuryDraftDto(
    val injury: InjuryDto = InjuryDto(region = "", status = "active_flare"),
    @SerialName("low_confidence_fields") val lowConfidenceFields: List<String> = emptyList(),
)

@Serializable
data class ParseInjuryRequest(val text: String)

/** Session DTOs (mirror the Session schema in the contract — the generated
 *  workout the client renders, caches offline, and autoregulates against). */

@Serializable
data class SessionDto(
    val id: String,
    @SerialName("generated_at") val generatedAt: String,
    @SerialName("schema_version") val schemaVersion: Int,
    val model: String? = null,
    @SerialName("inputs_summary") val inputsSummary: SessionInputsSummaryDto? = null,
    val warmup: List<SessionExerciseDto> = emptyList(),
    @SerialName("main_work") val mainWork: List<SessionExerciseDto> = emptyList(),
    val accessory: List<SessionExerciseDto> = emptyList(),
    @SerialName("aging_block") val agingBlock: AgingBlockDto,
    val reasoning: List<ReasoningNoteDto> = emptyList(),
    @SerialName("safety_findings") val safetyFindings: List<SafetyFindingDto> = emptyList(),
    val disclaimer: String,
)

@Serializable
data class SessionInputsSummaryDto(
    @SerialName("readiness_value") val readinessValue: Int? = null,
    @SerialName("readiness_confidence") val readinessConfidence: String? = null,
    @SerialName("contraindication_count") val contraindicationCount: Int? = null,
    @SerialName("location_name") val locationName: String? = null,
    @SerialName("aging_emphases") val agingEmphases: List<String> = emptyList(),
)

@Serializable
data class SessionExerciseDto(
    val name: String,
    val movement: String,
    val region: String? = null,
    val sets: List<SetPrescriptionDto> = emptyList(),
    val notes: String? = null,
)

@Serializable
data class SetPrescriptionDto(
    val type: String,
    val reps: Int? = null,
    @SerialName("load_kg") val loadKg: Double? = null,
    @SerialName("rpe_target") val rpeTarget: Double? = null,
    @SerialName("duration_sec") val durationSec: Int? = null,
    @SerialName("rest_sec") val restSec: Int? = null,
)

@Serializable
data class AgingBlockDto(
    val emphases: List<String> = emptyList(),
    val items: List<SessionExerciseDto> = emptyList(),
)

@Serializable
data class ReasoningNoteDto(
    val text: String,
    val tag: String? = null,
)

@Serializable
data class SafetyFindingDto(
    val rule: String,
    val action: String,
    val detail: String? = null,
)

/** Re-plan check DTO (mirrors GET /sessions/replan-check). */
@Serializable
data class ReplanCheckDto(
    @SerialName("replan_needed") val replanNeeded: Boolean,
    val reasons: List<String> = emptyList(),
)

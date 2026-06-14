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
data class ConsentRequest(val type: String, val version: String)

@Serializable
data class ConsentRecord(
    val type: String,
    val version: String,
    @SerialName("accepted_at") val acceptedAt: String? = null,
)

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

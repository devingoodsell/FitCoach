package pro.d11l.fitcoach.data

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Test
import pro.d11l.fitcoach.core.db.SessionWithExercises
import pro.d11l.fitcoach.core.network.SessionDto
import java.io.File

/**
 * The normalized cache graph is a lossless representation of the rendered session
 * shape: decomposing the committed sample into session/exercise/set rows and
 * reconstructing it preserves blocks, order, prescriptions, and reasoning.
 */
class SessionCacheMappingTest {

    private val json = Json { ignoreUnknownKeys = true }
    private val strict = Json { ignoreUnknownKeys = false }

    @Test
    fun `round-trips the committed sample through the cache graph`() {
        val original = sample()

        val (entity, exercises) = original.toCacheGraph("csid-1", json)
        val reloaded = SessionWithExercises(entity, exercises).toSessionDto(json)

        assertEquals(original.id, reloaded.id)
        assertEquals(original.generatedAt, reloaded.generatedAt)
        assertEquals(original.schemaVersion, reloaded.schemaVersion)
        assertEquals(original.disclaimer, reloaded.disclaimer)

        // Block membership and order preserved.
        assertEquals(original.warmup.map { it.name }, reloaded.warmup.map { it.name })
        assertEquals(original.mainWork.map { it.name }, reloaded.mainWork.map { it.name })
        assertEquals(original.accessory.map { it.name }, reloaded.accessory.map { it.name })
        assertEquals(original.agingBlock.emphases, reloaded.agingBlock.emphases)
        assertEquals(original.agingBlock.items.map { it.name }, reloaded.agingBlock.items.map { it.name })

        // Ancillary lists preserved verbatim via the JSON columns.
        assertEquals(original.reasoning, reloaded.reasoning)
        assertEquals(original.safetyFindings, reloaded.safetyFindings)
        assertEquals(original.inputsSummary, reloaded.inputsSummary)

        // Per-set prescriptions preserved on a representative set.
        assertEquals(original.mainWork.first().sets, reloaded.mainWork.first().sets)
    }

    @Test
    fun `client_session_id and active status carried on the entity`() {
        val (entity, _) = sample().toCacheGraph("csid-stable", json)
        assertEquals("csid-stable", entity.clientSessionId)
        assertEquals("active", entity.status)
    }

    private fun sample(): SessionDto {
        val text = repoFile("backend/api/examples/session-sample.json").readText()
        return strict.decodeFromString(SessionDto.serializer(), text)
    }

    private fun repoFile(relative: String): File {
        var dir: File? = File(System.getProperty("user.dir"))
        repeat(6) {
            val candidate = File(dir, relative)
            if (candidate.exists()) return candidate
            dir = dir?.parentFile
        }
        error("could not locate $relative from ${System.getProperty("user.dir")}")
    }
}

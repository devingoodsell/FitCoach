package pro.d11l.fitcoach.core.network

import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * Handshake test: the committed backend sample (backend/api/examples/session-sample.json)
 * decodes STRICTLY into the client DTOs. Strict decoding (ignoreUnknownKeys =
 * false) means a field added to the published contract that the client doesn't
 * model fails here — keeping client and server in lockstep on the session shape.
 */
class SessionContractTest {

    private val strict = Json { ignoreUnknownKeys = false }

    @Test
    fun `committed sample decodes into the session DTOs`() {
        val json = repoFile("backend/api/examples/session-sample.json").readText()
        val session = strict.decodeFromString(SessionDto.serializer(), json)

        assertEquals(1, session.schemaVersion)
        assertTrue("warmup present", session.warmup.isNotEmpty())
        assertTrue("main work present", session.mainWork.isNotEmpty())
        assertTrue("aging block present (E8-S1)", session.agingBlock.items.isNotEmpty())
        assertTrue("aging emphases present", session.agingBlock.emphases.isNotEmpty())
        assertTrue("per-set prescriptions present", session.mainWork.first().sets.isNotEmpty())
        assertTrue("age-aware reasoning note", session.reasoning.any { it.tag == "age_aware" })
        assertTrue("disclaimer present", session.disclaimer.isNotBlank())
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

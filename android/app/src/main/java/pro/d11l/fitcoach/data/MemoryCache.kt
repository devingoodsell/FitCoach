package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.db.MemorySectionDao
import pro.d11l.fitcoach.core.db.MemorySectionEntity

/** A cached Coach Memory section (domain type, decoupled from Room). */
data class CachedSection(
    val section: String,
    val schemaVersion: Int,
    val dataJson: String,
    val updatedAt: String?,
)

/**
 * Local cache of Coach Memory. Interface so repositories are unit-testable with
 * an in-memory fake; the production impl wraps the Room DAO.
 */
interface MemoryCache {
    suspend fun replaceAll(sections: List<CachedSection>)
    suspend fun all(): List<CachedSection>
    suspend fun clear()
}

/** Room-backed [MemoryCache]. */
class RoomMemoryCache(private val dao: MemorySectionDao) : MemoryCache {
    override suspend fun replaceAll(sections: List<CachedSection>) {
        dao.clear()
        dao.upsertAll(
            sections.map { MemorySectionEntity(it.section, it.schemaVersion, it.dataJson, it.updatedAt) },
        )
    }

    override suspend fun all(): List<CachedSection> =
        dao.all().map { CachedSection(it.section, it.schemaVersion, it.dataJson, it.updatedAt) }

    override suspend fun clear() = dao.clear()
}

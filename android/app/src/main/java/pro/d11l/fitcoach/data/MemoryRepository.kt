package pro.d11l.fitcoach.data

import pro.d11l.fitcoach.core.network.FitCoachApi

/**
 * Mediates Coach Memory between our backend and the local Room cache. Reads are
 * cache-backed so memory is available offline; sync() refreshes from the backend
 * (e.g. on login / fresh-device restore, E1-S6) and updates the cache.
 */
class MemoryRepository(
    private val api: FitCoachApi,
    private val cache: MemoryCache,
) {
    /**
     * Pulls memory from the backend and replaces the cache. On network failure it
     * returns the cached copy so the caller still has data offline.
     */
    suspend fun sync(): List<CachedSection> {
        return try {
            val resp = api.memory()
            val body = resp.body()
            if (resp.isSuccessful && body != null) {
                val sections = body.sections.map {
                    CachedSection(it.section, it.schemaVersion, it.data.toString(), it.updatedAt)
                }
                cache.replaceAll(sections)
                sections
            } else {
                cache.all()
            }
        } catch (_: Exception) {
            cache.all()
        }
    }

    /** Offline-first read of whatever is cached locally. */
    suspend fun cached(): List<CachedSection> = cache.all()
}

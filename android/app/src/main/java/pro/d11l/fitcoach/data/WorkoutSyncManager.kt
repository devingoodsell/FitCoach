package pro.d11l.fitcoach.data

import kotlinx.coroutines.CancellationException
import kotlinx.serialization.json.Json
import pro.d11l.fitcoach.core.db.WorkoutOutboxDao
import pro.d11l.fitcoach.core.db.WorkoutOutboxEntity
import pro.d11l.fitcoach.core.network.FitCoachApi
import pro.d11l.fitcoach.core.network.WorkoutLogData
import pro.d11l.fitcoach.core.network.WorkoutLogRequest
import java.time.Instant

/** Outcome of a [WorkoutSyncManager.sync] pass. */
data class SyncResult(val synced: Int, val failed: Int)

/**
 * Durable offline write-queue for completed sessions (E12-PR2). Completion
 * enqueues the as-performed session locally (works offline); [sync] flushes the
 * queue to the backend on reconnect. Idempotent end-to-end: each row is keyed by
 * the stable `client_session_id`, so enqueuing twice never duplicates locally and
 * the backend upserts on the same key, so a replay (e.g. a lost success response)
 * never duplicates server-side. Per-row failures stay queued for safe retry,
 * independent of the rows that succeed.
 */
class WorkoutSyncManager(
    private val api: FitCoachApi,
    private val outbox: WorkoutOutboxDao,
    private val json: Json,
    private val now: () -> String = { Instant.now().toString() },
) {

    /** Queues (or replaces) the log for [clientSessionId]; safe with no connectivity. */
    suspend fun enqueue(clientSessionId: String, payload: WorkoutLogData, performedAt: String) {
        outbox.upsert(
            WorkoutOutboxEntity(
                clientSessionId = clientSessionId,
                performedAt = performedAt,
                dataJson = json.encodeToString(WorkoutLogData.serializer(), payload),
                createdAt = now(),
            ),
        )
    }

    /** Number of logs awaiting sync. */
    suspend fun pendingCount(): Int = outbox.count()

    /**
     * Flushes every queued log to the backend. Accepted rows are removed;
     * failed rows are kept (attempt counter bumped) for the next pass. One row's
     * failure never blocks the others.
     */
    suspend fun sync(): SyncResult {
        var synced = 0
        var failed = 0
        for (entry in outbox.pending()) {
            val data = json.decodeFromString(WorkoutLogData.serializer(), entry.dataJson)
            val request = WorkoutLogRequest(entry.clientSessionId, entry.performedAt, data)
            val response = try {
                api.recordWorkout(request)
            } catch (cancel: CancellationException) {
                throw cancel
            } catch (networkError: Exception) {
                outbox.markFailed(entry.clientSessionId, networkError.message)
                failed++
                continue
            }
            if (response.isSuccessful) {
                outbox.delete(entry.clientSessionId)
                synced++
            } else {
                outbox.markFailed(entry.clientSessionId, "http ${response.code()}")
                failed++
            }
        }
        return SyncResult(synced, failed)
    }
}

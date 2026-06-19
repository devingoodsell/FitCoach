package pro.d11l.fitcoach.core.db

import androidx.room.ColumnInfo
import androidx.room.Entity
import androidx.room.PrimaryKey

/**
 * Durable offline write-queue for completed sessions (E12-PR2). One row per
 * logical session, keyed by the stable, device-minted `client_session_id` so
 * enqueuing the same session twice never creates a duplicate locally, and the
 * backend (which upserts on the same key) never duplicates on replay. Rows are
 * deleted once the backend accepts them.
 */
@Entity(tableName = "workout_outbox")
data class WorkoutOutboxEntity(
    @PrimaryKey @ColumnInfo(name = "client_session_id") val clientSessionId: String,
    @ColumnInfo(name = "performed_at") val performedAt: String,
    /** Serialized WorkoutLogData — the as-performed session payload. */
    @ColumnInfo(name = "data_json") val dataJson: String,
    @ColumnInfo(name = "created_at") val createdAt: String,
    @ColumnInfo(name = "attempt_count") val attemptCount: Int = 0,
    @ColumnInfo(name = "last_error") val lastError: String? = null,
)

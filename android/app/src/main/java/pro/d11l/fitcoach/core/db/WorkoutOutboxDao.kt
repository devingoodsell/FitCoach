package pro.d11l.fitcoach.core.db

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query

@Dao
interface WorkoutOutboxDao {

    /** Enqueue (or replace) a queued log. Replace-on-conflict keeps it idempotent per session. */
    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsert(entry: WorkoutOutboxEntity)

    @Query("SELECT * FROM workout_outbox ORDER BY created_at")
    suspend fun pending(): List<WorkoutOutboxEntity>

    @Query("SELECT COUNT(*) FROM workout_outbox")
    suspend fun count(): Int

    @Query("DELETE FROM workout_outbox WHERE client_session_id = :clientSessionId")
    suspend fun delete(clientSessionId: String)

    @Query(
        "UPDATE workout_outbox SET attempt_count = attempt_count + 1, last_error = :error " +
            "WHERE client_session_id = :clientSessionId",
    )
    suspend fun markFailed(clientSessionId: String, error: String?)
}

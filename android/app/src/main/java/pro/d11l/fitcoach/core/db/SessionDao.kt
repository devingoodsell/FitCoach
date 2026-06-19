package pro.d11l.fitcoach.core.db

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.Query
import androidx.room.Transaction

/**
 * DAO for the offline session cache. Writes the normalized graph in one
 * transaction; reads the single current session with its exercises and sets.
 */
@Dao
abstract class SessionDao {

    @Insert
    abstract suspend fun insertSession(session: SessionEntity)

    @Insert
    abstract suspend fun insertExercise(exercise: ExerciseEntity): Long

    @Insert
    abstract suspend fun insertSets(sets: List<SetEntity>)

    /** Removes the cached session; FK cascade clears its exercises and sets. */
    @Query("DELETE FROM sessions")
    abstract suspend fun clearSessions()

    @Query("UPDATE sessions SET status = :status, completed_at = :completedAt WHERE session_id = :sessionId")
    abstract suspend fun updateSessionStatus(sessionId: String, status: String, completedAt: String?)

    /** Records logged actuals for one set (E6-PR2/PR4). */
    @Query(
        "UPDATE session_sets SET reps_done = :repsDone, load_kg_done = :loadKgDone, " +
            "rpe_actual = :rpeActual, duration_done_sec = :durationDoneSec, " +
            "skipped = :skipped, completed = :completed WHERE set_id = :setId",
    )
    abstract suspend fun updateSetLog(
        setId: Long,
        repsDone: Int?,
        loadKgDone: Double?,
        rpeActual: Double?,
        durationDoneSec: Int?,
        skipped: Boolean,
        completed: Boolean,
    )

    @Query("SELECT COUNT(*) FROM session_exercises")
    abstract suspend fun exerciseCount(): Int

    @Query("SELECT COUNT(*) FROM session_sets")
    abstract suspend fun setCount(): Int

    @Transaction
    @Query("SELECT * FROM sessions ORDER BY generated_at DESC LIMIT 1")
    abstract suspend fun latest(): SessionWithExercises?

    @Transaction
    @Query("SELECT * FROM sessions WHERE client_session_id = :clientSessionId")
    abstract suspend fun byClientSessionId(clientSessionId: String): SessionWithExercises?

    /**
     * Replaces any cached session with [session] and its [exercises]. The session
     * is treated as the single current plan, so prior cache is cleared first.
     * Exercise/set ids are assigned by Room here; incoming ids are ignored.
     */
    @Transaction
    open suspend fun replace(session: SessionEntity, exercises: List<ExerciseWithSets>) {
        clearSessions()
        insertSession(session)
        for (ews in exercises) {
            val exerciseId = insertExercise(ews.exercise.copy(exerciseId = 0, sessionId = session.sessionId))
            if (ews.sets.isNotEmpty()) {
                insertSets(ews.sets.map { it.copy(setId = 0, exerciseId = exerciseId) })
            }
        }
    }
}

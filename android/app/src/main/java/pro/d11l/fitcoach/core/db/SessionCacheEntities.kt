package pro.d11l.fitcoach.core.db

import androidx.room.ColumnInfo
import androidx.room.Embedded
import androidx.room.Entity
import androidx.room.ForeignKey
import androidx.room.Index
import androidx.room.PrimaryKey
import androidx.room.Relation

/**
 * Offline cache of a generated workout session (E5-PR5), normalized into three
 * tables — session → exercises → sets — so the player (E6) and on-device
 * autoregulation (E5-PR6) can read prescriptions and write logged actuals per set
 * with no connectivity. Ancillary, render-only lists (reasoning, inputs summary,
 * safety findings, aging emphases) are kept as JSON columns on the session row
 * rather than their own tables.
 *
 * `clientSessionId` is a stable, device-minted idempotency key reused by the
 * offline sync queue (E12-PR2) so replays never duplicate the server record.
 */
@Entity(tableName = "sessions")
data class SessionEntity(
    @PrimaryKey @ColumnInfo(name = "session_id") val sessionId: String,
    @ColumnInfo(name = "client_session_id") val clientSessionId: String,
    @ColumnInfo(name = "generated_at") val generatedAt: String,
    @ColumnInfo(name = "schema_version") val schemaVersion: Int,
    val model: String?,
    val disclaimer: String,
    @ColumnInfo(name = "inputs_summary_json") val inputsSummaryJson: String?,
    @ColumnInfo(name = "reasoning_json") val reasoningJson: String,
    @ColumnInfo(name = "safety_findings_json") val safetyFindingsJson: String,
    @ColumnInfo(name = "aging_emphases_json") val agingEmphasesJson: String,
    /** Lifecycle: "active" until the session is finished; set on completion (E6-PR5). */
    @ColumnInfo(name = "status") val status: String = STATUS_ACTIVE,
    @ColumnInfo(name = "completed_at") val completedAt: String? = null,
) {
    companion object {
        const val STATUS_ACTIVE = "active"
        const val STATUS_COMPLETED = "completed"
    }
}

/** One exercise within a session block (warmup/main/accessory/aging), in display order. */
@Entity(
    tableName = "session_exercises",
    foreignKeys = [
        ForeignKey(
            entity = SessionEntity::class,
            parentColumns = ["session_id"],
            childColumns = ["session_id"],
            onDelete = ForeignKey.CASCADE,
        ),
    ],
    indices = [Index("session_id")],
)
data class ExerciseEntity(
    @PrimaryKey(autoGenerate = true) @ColumnInfo(name = "exercise_id") val exerciseId: Long = 0,
    @ColumnInfo(name = "session_id") val sessionId: String,
    @ColumnInfo(name = "block_type") val blockType: String,
    @ColumnInfo(name = "order_index") val orderIndex: Int,
    val name: String,
    val movement: String,
    val region: String?,
    val notes: String?,
) {
    companion object {
        const val BLOCK_WARMUP = "warmup"
        const val BLOCK_MAIN = "main"
        const val BLOCK_ACCESSORY = "accessory"
        const val BLOCK_AGING = "aging"
    }
}

/**
 * One set: the prescribed targets (from the generated plan) plus the logged
 * actuals (filled in during the session, offline). Defaults leave the set
 * unlogged until the player records it.
 */
@Entity(
    tableName = "session_sets",
    foreignKeys = [
        ForeignKey(
            entity = ExerciseEntity::class,
            parentColumns = ["exercise_id"],
            childColumns = ["exercise_id"],
            onDelete = ForeignKey.CASCADE,
        ),
    ],
    indices = [Index("exercise_id")],
)
data class SetEntity(
    @PrimaryKey(autoGenerate = true) @ColumnInfo(name = "set_id") val setId: Long = 0,
    @ColumnInfo(name = "exercise_id") val exerciseId: Long,
    @ColumnInfo(name = "order_index") val orderIndex: Int,
    // Prescribed targets.
    val type: String,
    val reps: Int?,
    @ColumnInfo(name = "load_kg") val loadKg: Double?,
    @ColumnInfo(name = "rpe_target") val rpeTarget: Double?,
    @ColumnInfo(name = "duration_sec") val durationSec: Int?,
    @ColumnInfo(name = "rest_sec") val restSec: Int?,
    // Logged actuals (E6-PR2/PR4); null/false until the set is performed.
    @ColumnInfo(name = "reps_done") val repsDone: Int? = null,
    @ColumnInfo(name = "load_kg_done") val loadKgDone: Double? = null,
    @ColumnInfo(name = "rpe_actual") val rpeActual: Double? = null,
    @ColumnInfo(name = "duration_done_sec") val durationDoneSec: Int? = null,
    @ColumnInfo(name = "skipped") val skipped: Boolean = false,
    @ColumnInfo(name = "completed") val completed: Boolean = false,
)

/** An exercise with its sets, ordered. Doubles as the read relation and the write holder. */
data class ExerciseWithSets(
    @Embedded val exercise: ExerciseEntity,
    @Relation(parentColumn = "exercise_id", entityColumn = "exercise_id")
    val sets: List<SetEntity>,
)

/** A full cached session with its exercises (each with sets). */
data class SessionWithExercises(
    @Embedded val session: SessionEntity,
    @Relation(entity = ExerciseEntity::class, parentColumn = "session_id", entityColumn = "session_id")
    val exercises: List<ExerciseWithSets>,
)

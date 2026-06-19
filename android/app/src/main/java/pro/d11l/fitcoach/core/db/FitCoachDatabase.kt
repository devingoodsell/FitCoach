package pro.d11l.fitcoach.core.db

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase
import androidx.room.migration.Migration
import androidx.sqlite.db.SupportSQLiteDatabase

@Database(
    entities = [
        MemorySectionEntity::class,
        SessionEntity::class,
        ExerciseEntity::class,
        SetEntity::class,
    ],
    version = 2,
    exportSchema = true,
)
abstract class FitCoachDatabase : RoomDatabase() {
    abstract fun memorySectionDao(): MemorySectionDao
    abstract fun sessionDao(): SessionDao

    companion object {
        /**
         * v1 → v2 (E5-PR5): add the offline session cache (session → exercises →
         * sets). The existing `memory_sections` table is untouched. Foreign keys
         * cascade so clearing a session removes its children.
         */
        val MIGRATION_1_2 = object : Migration(1, 2) {
            override fun migrate(db: SupportSQLiteDatabase) {
                db.execSQL(
                    """
                    CREATE TABLE IF NOT EXISTS `sessions` (
                        `session_id` TEXT NOT NULL,
                        `client_session_id` TEXT NOT NULL,
                        `generated_at` TEXT NOT NULL,
                        `schema_version` INTEGER NOT NULL,
                        `model` TEXT,
                        `disclaimer` TEXT NOT NULL,
                        `inputs_summary_json` TEXT,
                        `reasoning_json` TEXT NOT NULL,
                        `safety_findings_json` TEXT NOT NULL,
                        `aging_emphases_json` TEXT NOT NULL,
                        `status` TEXT NOT NULL,
                        `completed_at` TEXT,
                        PRIMARY KEY(`session_id`)
                    )
                    """.trimIndent(),
                )
                db.execSQL(
                    """
                    CREATE TABLE IF NOT EXISTS `session_exercises` (
                        `exercise_id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
                        `session_id` TEXT NOT NULL,
                        `block_type` TEXT NOT NULL,
                        `order_index` INTEGER NOT NULL,
                        `name` TEXT NOT NULL,
                        `movement` TEXT NOT NULL,
                        `region` TEXT,
                        `notes` TEXT,
                        FOREIGN KEY(`session_id`) REFERENCES `sessions`(`session_id`)
                            ON UPDATE NO ACTION ON DELETE CASCADE
                    )
                    """.trimIndent(),
                )
                db.execSQL(
                    "CREATE INDEX IF NOT EXISTS `index_session_exercises_session_id` " +
                        "ON `session_exercises` (`session_id`)",
                )
                db.execSQL(
                    """
                    CREATE TABLE IF NOT EXISTS `session_sets` (
                        `set_id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
                        `exercise_id` INTEGER NOT NULL,
                        `order_index` INTEGER NOT NULL,
                        `type` TEXT NOT NULL,
                        `reps` INTEGER,
                        `load_kg` REAL,
                        `rpe_target` REAL,
                        `duration_sec` INTEGER,
                        `rest_sec` INTEGER,
                        `reps_done` INTEGER,
                        `load_kg_done` REAL,
                        `rpe_actual` REAL,
                        `duration_done_sec` INTEGER,
                        `skipped` INTEGER NOT NULL,
                        `completed` INTEGER NOT NULL,
                        FOREIGN KEY(`exercise_id`) REFERENCES `session_exercises`(`exercise_id`)
                            ON UPDATE NO ACTION ON DELETE CASCADE
                    )
                    """.trimIndent(),
                )
                db.execSQL(
                    "CREATE INDEX IF NOT EXISTS `index_session_sets_exercise_id` " +
                        "ON `session_sets` (`exercise_id`)",
                )
            }
        }

        fun create(context: Context): FitCoachDatabase =
            Room.databaseBuilder(context, FitCoachDatabase::class.java, "fitcoach.db")
                .addMigrations(MIGRATION_1_2)
                .build()
    }
}

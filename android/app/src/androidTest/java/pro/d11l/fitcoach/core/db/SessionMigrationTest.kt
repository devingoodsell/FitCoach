package pro.d11l.fitcoach.core.db

import androidx.room.testing.MigrationTestHelper
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import org.junit.Assert.assertEquals
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith

/**
 * Validates the v1 → v2 migration (E5-PR5): the existing `memory_sections` data
 * survives and the new session cache tables are created, matching the schema Room
 * generates for version 2.
 */
@RunWith(AndroidJUnit4::class)
class SessionMigrationTest {

    @get:Rule
    val helper = MigrationTestHelper(
        InstrumentationRegistry.getInstrumentation(),
        FitCoachDatabase::class.java,
    )

    @Test
    fun migrate1To2KeepsMemoryAndAddsSessionTables() {
        helper.createDatabase(TEST_DB, 1).apply {
            execSQL(
                "INSERT INTO memory_sections (section, schemaVersion, dataJson, updatedAt) " +
                    "VALUES ('profile', 1, '{\"age\":40}', '2026-06-19T08:00:00Z')",
            )
            close()
        }

        val db = helper.runMigrationsAndValidate(
            TEST_DB,
            2,
            true,
            FitCoachDatabase.MIGRATION_1_2,
        )

        db.query("SELECT COUNT(*) FROM memory_sections").use {
            it.moveToFirst()
            assertEquals(1, it.getInt(0))
        }
        // New tables exist and are empty.
        db.query("SELECT COUNT(*) FROM sessions").use {
            it.moveToFirst()
            assertEquals(0, it.getInt(0))
        }
        db.query("SELECT COUNT(*) FROM session_exercises").use {
            it.moveToFirst()
            assertEquals(0, it.getInt(0))
        }
        db.query("SELECT COUNT(*) FROM session_sets").use {
            it.moveToFirst()
            assertEquals(0, it.getInt(0))
        }
        db.close()
    }

    @Test
    fun migrate2To3AddsWorkoutOutbox() {
        helper.createDatabase(TEST_DB, 2).apply {
            execSQL(
                "INSERT INTO memory_sections (section, schemaVersion, dataJson, updatedAt) " +
                    "VALUES ('profile', 1, '{}', null)",
            )
            close()
        }

        val db = helper.runMigrationsAndValidate(
            TEST_DB,
            3,
            true,
            FitCoachDatabase.MIGRATION_2_3,
        )

        // Existing data preserved; the new write-queue table exists and is empty.
        db.query("SELECT COUNT(*) FROM memory_sections").use {
            it.moveToFirst()
            assertEquals(1, it.getInt(0))
        }
        db.query("SELECT COUNT(*) FROM workout_outbox").use {
            it.moveToFirst()
            assertEquals(0, it.getInt(0))
        }
        db.close()
    }

    private companion object {
        const val TEST_DB = "migration-test"
    }
}

package pro.d11l.fitcoach.core.db

import android.content.Context
import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class WorkoutOutboxDaoTest {

    private lateinit var db: FitCoachDatabase
    private lateinit var dao: WorkoutOutboxDao

    @Before
    fun setUp() {
        val ctx = ApplicationProvider.getApplicationContext<Context>()
        db = Room.inMemoryDatabaseBuilder(ctx, FitCoachDatabase::class.java).build()
        dao = db.workoutOutboxDao()
    }

    @After
    fun tearDown() = db.close()

    private fun entry(id: String) = WorkoutOutboxEntity(
        clientSessionId = id,
        performedAt = "2026-06-19T07:55:00Z",
        dataJson = "{\"session_id\":\"s\"}",
        createdAt = "2026-06-19T08:00:00Z",
    )

    @Test
    fun upsertIsIdempotentOnClientSessionId() = runBlocking {
        dao.upsert(entry("csid-A"))
        dao.upsert(entry("csid-A"))
        assertEquals(1, dao.count())
    }

    @Test
    fun pendingReturnsAllQueuedRows() = runBlocking {
        dao.upsert(entry("csid-A"))
        dao.upsert(entry("csid-B"))
        assertEquals(2, dao.pending().size)
    }

    @Test
    fun deleteRemovesAcceptedRow() = runBlocking {
        dao.upsert(entry("csid-A"))
        dao.delete("csid-A")
        assertEquals(0, dao.count())
    }

    @Test
    fun markFailedBumpsAttemptCount() = runBlocking {
        dao.upsert(entry("csid-A"))
        dao.markFailed("csid-A", "http 500")
        val row = dao.pending().first()
        assertEquals(1, row.attemptCount)
        assertEquals("http 500", row.lastError)
    }
}

package pro.d11l.fitcoach.data

import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Test
import pro.d11l.fitcoach.core.network.LoggedExerciseDto
import pro.d11l.fitcoach.core.network.LoggedSetDto
import pro.d11l.fitcoach.core.network.WorkoutLogData
import pro.d11l.fitcoach.testing.FakeApi
import pro.d11l.fitcoach.testing.FakeWorkoutOutboxDao

@OptIn(ExperimentalCoroutinesApi::class)
class WorkoutSyncManagerTest {

    private val json = Json { encodeDefaults = true; ignoreUnknownKeys = true }

    private fun manager(api: FakeApi, dao: FakeWorkoutOutboxDao) =
        WorkoutSyncManager(api, dao, json, now = { "2026-06-19T08:00:00Z" })

    private fun payload(sessionId: String) = WorkoutLogData(
        sessionId = sessionId,
        status = "completed",
        exercises = listOf(
            LoggedExerciseDto(
                blockType = "main",
                name = "Back Squat",
                movement = "squat",
                sets = listOf(LoggedSetDto(type = "reps", repsDone = 5, loadKg = 60.0, rpeActual = 7.0)),
            ),
        ),
    )

    @Test
    fun `enqueuing the same session twice keeps one queued row`() = runTest {
        val dao = FakeWorkoutOutboxDao()
        val mgr = manager(FakeApi(), dao)

        mgr.enqueue("csid-A", payload("sess-A"), performedAt = "2026-06-19T07:55:00Z")
        mgr.enqueue("csid-A", payload("sess-A"), performedAt = "2026-06-19T07:56:00Z")

        assertEquals(1, dao.count())
    }

    @Test
    fun `sync flushes queued logs and clears them`() = runTest {
        val api = FakeApi()
        val dao = FakeWorkoutOutboxDao()
        val mgr = manager(api, dao)
        mgr.enqueue("csid-A", payload("sess-A"), "2026-06-19T07:55:00Z")

        val result = mgr.sync()

        assertEquals(SyncResult(synced = 1, failed = 0), result)
        assertEquals(0, dao.count())
        assertEquals(1, api.recordedWorkouts.size)
        assertEquals(1, api.recordWorkoutPostCount)
        assertEquals("sess-A", api.recordedWorkouts["csid-A"]?.data?.sessionId)
    }

    @Test
    fun `replaying the same session never duplicates server-side`() = runTest {
        val api = FakeApi()
        val dao = FakeWorkoutOutboxDao()
        val mgr = manager(api, dao)

        mgr.enqueue("csid-A", payload("sess-A"), "2026-06-19T07:55:00Z")
        mgr.sync()
        // Re-completion (or a lost success response) re-queues the same session.
        mgr.enqueue("csid-A", payload("sess-A"), "2026-06-19T07:55:00Z")
        mgr.sync()

        assertEquals(2, api.recordWorkoutPostCount) // posted twice
        assertEquals(1, api.recordedWorkouts.size) // but stored once (idempotent key)
        assertEquals(0, dao.count())
    }

    @Test
    fun `partial failure keeps the failed row queued and retries safely`() = runTest {
        val api = FakeApi().apply { failWorkoutFor = setOf("csid-B") }
        val dao = FakeWorkoutOutboxDao()
        val mgr = manager(api, dao)
        mgr.enqueue("csid-A", payload("sess-A"), "2026-06-19T07:55:00Z")
        mgr.enqueue("csid-B", payload("sess-B"), "2026-06-19T07:56:00Z")

        val first = mgr.sync()
        assertEquals(SyncResult(synced = 1, failed = 1), first)
        assertEquals(1, dao.count()) // B remains
        assertEquals(setOf("csid-A"), api.recordedWorkouts.keys)

        // Connectivity recovers for B; a second pass flushes it without re-duplicating A.
        api.failWorkoutFor = emptySet()
        val second = mgr.sync()
        assertEquals(SyncResult(synced = 1, failed = 0), second)
        assertEquals(0, dao.count())
        assertEquals(setOf("csid-A", "csid-B"), api.recordedWorkouts.keys)
    }

    @Test
    fun `network exception leaves the row queued for the next pass`() = runTest {
        val api = FakeApi().apply { recordWorkoutThrows = true }
        val dao = FakeWorkoutOutboxDao()
        val mgr = manager(api, dao)
        mgr.enqueue("csid-A", payload("sess-A"), "2026-06-19T07:55:00Z")

        assertEquals(SyncResult(synced = 0, failed = 1), mgr.sync())
        assertEquals(1, dao.count())

        api.recordWorkoutThrows = false
        assertEquals(SyncResult(synced = 1, failed = 0), mgr.sync())
        assertEquals(0, dao.count())
    }
}

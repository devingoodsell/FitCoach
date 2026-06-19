package pro.d11l.fitcoach.core.db

import android.content.Context
import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith

@RunWith(AndroidJUnit4::class)
class SessionDaoTest {

    private lateinit var db: FitCoachDatabase
    private lateinit var dao: SessionDao

    @Before
    fun setUp() {
        val ctx = ApplicationProvider.getApplicationContext<Context>()
        db = Room.inMemoryDatabaseBuilder(ctx, FitCoachDatabase::class.java).build()
        dao = db.sessionDao()
    }

    @After
    fun tearDown() = db.close()

    private fun session(id: String) = SessionEntity(
        sessionId = id,
        clientSessionId = "csid-$id",
        generatedAt = "2026-06-19T08:00:00Z",
        schemaVersion = 1,
        model = "claude-opus-4-8",
        disclaimer = "Guidance, not medical advice.",
        inputsSummaryJson = null,
        reasoningJson = "[]",
        safetyFindingsJson = "[]",
        agingEmphasesJson = "[\"bone_balance\"]",
    )

    private fun mainExercise() = ExerciseWithSets(
        exercise = ExerciseEntity(
            sessionId = "ignored", // replace() rebinds to the session id
            blockType = ExerciseEntity.BLOCK_MAIN,
            orderIndex = 0,
            name = "Back Squat",
            movement = "squat",
            region = "legs",
            notes = null,
        ),
        sets = listOf(
            SetEntity(exerciseId = 0, orderIndex = 0, type = "reps", reps = 5, loadKg = 60.0, rpeTarget = 7.0, durationSec = null, restSec = 120),
            SetEntity(exerciseId = 0, orderIndex = 1, type = "reps", reps = 5, loadKg = 60.0, rpeTarget = 8.0, durationSec = null, restSec = 120),
        ),
    )

    @Test
    fun replaceAndReadBackPreservesGraph() = runBlocking {
        dao.replace(session("s1"), listOf(mainExercise()))

        val loaded = dao.latest()!!
        assertEquals("s1", loaded.session.sessionId)
        assertEquals("csid-s1", loaded.session.clientSessionId)
        assertEquals(1, loaded.exercises.size)
        val sets = loaded.exercises.first().sets.sortedBy { it.orderIndex }
        assertEquals(2, sets.size)
        assertEquals(5, sets[0].reps)
        assertEquals(60.0, sets[0].loadKg!!, 0.0)
        assertEquals(120, sets[0].restSec)
        // Logged actuals default to unlogged.
        assertFalse(sets[0].completed)
        assertFalse(sets[0].skipped)
        assertNull(sets[0].repsDone)
    }

    @Test
    fun replaceClearsPriorSession() = runBlocking {
        dao.replace(session("s1"), listOf(mainExercise()))
        dao.replace(session("s2"), listOf(mainExercise()))

        val loaded = dao.latest()!!
        assertEquals("s2", loaded.session.sessionId)
        assertEquals(1, dao.exerciseCount()) // prior session's rows gone
        assertEquals(2, dao.setCount())
    }

    @Test
    fun updateSetLogRecordsActuals() = runBlocking {
        dao.replace(session("s1"), listOf(mainExercise()))
        val setId = dao.latest()!!.exercises.first().sets.first().setId

        dao.updateSetLog(
            setId = setId,
            repsDone = 6,
            loadKgDone = 62.5,
            rpeActual = 7.5,
            durationDoneSec = null,
            skipped = false,
            completed = true,
        )

        val set = dao.latest()!!.exercises.first().sets.first { it.setId == setId }
        assertEquals(6, set.repsDone)
        assertEquals(62.5, set.loadKgDone!!, 0.0)
        assertEquals(true, set.completed)
    }

    @Test
    fun clearCascadesToChildren() = runBlocking {
        dao.replace(session("s1"), listOf(mainExercise()))
        dao.clearSessions()

        assertNull(dao.latest())
        assertEquals(0, dao.exerciseCount())
        assertEquals(0, dao.setCount())
    }
}

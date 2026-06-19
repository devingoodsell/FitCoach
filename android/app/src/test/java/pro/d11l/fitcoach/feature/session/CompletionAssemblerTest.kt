package pro.d11l.fitcoach.feature.session

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import pro.d11l.fitcoach.core.network.SetPrescriptionDto
import pro.d11l.fitcoach.data.LoggedSetState
import pro.d11l.fitcoach.data.PlanSet
import pro.d11l.fitcoach.data.SessionPlan

class CompletionAssemblerTest {

    private fun step(key: String, name: String, logged: LoggedSetState) = PlanSet(
        setId = 0,
        blockType = "main",
        blockTitle = "Main work",
        exerciseKey = key,
        exerciseName = name,
        movement = "squat",
        region = null,
        notes = null,
        setIndexInExercise = 0,
        setCountInExercise = 1,
        prescription = SetPrescriptionDto(type = "reps", reps = 8, loadKg = 20.0),
        logged = logged,
    )

    private fun plan(vararg steps: PlanSet) =
        SessionPlan(sessionId = "sess-1", clientSessionId = "csid-1", disclaimer = "d", steps = steps.toList())

    @Test
    fun `all completed sets produce a completed status grouped by exercise`() {
        val data = CompletionAssembler.build(
            plan(
                step("1", "Squat", LoggedSetState(repsDone = 8, loadKgDone = 20.0, completed = true)),
                step("1", "Squat", LoggedSetState(repsDone = 7, loadKgDone = 20.0, completed = true)),
                step("2", "Row", LoggedSetState(repsDone = 12, completed = true)),
            ),
        )
        assertEquals("completed", data.status)
        assertEquals(2, data.exercises.size) // grouped by exercise
        assertEquals(2, data.exercises.first().sets.size)
        assertFalse(data.exercises.first().sets.first().skipped)
    }

    @Test
    fun `any unlogged or skipped set makes the session partial and marks the set skipped`() {
        val data = CompletionAssembler.build(
            plan(
                step("1", "Squat", LoggedSetState(repsDone = 8, completed = true)),
                step("2", "Row", LoggedSetState(skipped = true, completed = false)),
                step("3", "Plank", LoggedSetState()), // never reached
            ),
        )
        assertEquals("partial", data.status)
        assertTrue(data.exercises[1].sets.first().skipped)
        assertTrue(data.exercises[2].sets.first().skipped)
    }
}

package pro.d11l.fitcoach.feature.session

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import pro.d11l.fitcoach.core.network.SetPrescriptionDto

class SessionPlayerTest {

    private val repsSet = SetPrescriptionDto(type = "reps", reps = 8, loadKg = 20.0, rpeTarget = 7.0, restSec = 120)
    private val timeSet = SetPrescriptionDto(type = "time", durationSec = 180, rpeTarget = 3.0, restSec = 0)

    @Test
    fun `draft seeds from the prescription`() {
        val d = SessionPlayer.draftFor(repsSet)
        assertEquals("8", d.reps)
        assertEquals("20", d.loadKg)
        assertEquals("", d.durationSec)

        assertEquals("180", SessionPlayer.draftFor(timeSet).durationSec)
    }

    @Test
    fun `logging a blank draft falls back to the prescribed targets`() {
        val logged = SessionPlayer.logFrom(repsSet, SetDraft())
        assertEquals(8, logged.repsDone)
        assertEquals(20.0, logged.loadKgDone!!, 0.0)
        assertTrue(logged.completed)
        assertFalse(logged.skipped)
    }

    @Test
    fun `logging an edited draft uses the entered values`() {
        val logged = SessionPlayer.logFrom(repsSet, SetDraft(reps = "10", loadKg = "22.5"))
        assertEquals(10, logged.repsDone)
        assertEquals(22.5, logged.loadKgDone!!, 0.0)
    }

    @Test
    fun `invalid input defaults to the prescription`() {
        val logged = SessionPlayer.logFrom(repsSet, SetDraft(reps = "abc", loadKg = ""))
        assertEquals(8, logged.repsDone)
        assertEquals(20.0, logged.loadKgDone!!, 0.0)
    }

    @Test
    fun `skipped set is recorded but not completed`() {
        val logged = SessionPlayer.skipped()
        assertTrue(logged.skipped)
        assertFalse(logged.completed)
    }

    @Test
    fun `prescription formats as a single line`() {
        assertEquals("8 reps · 20 kg · RPE 7 · rest 120s", formatPrescription(repsSet))
        assertEquals("180s · RPE 3 · rest 0s", formatPrescription(timeSet))
    }
}

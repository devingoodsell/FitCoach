package pro.d11l.fitcoach.feature.session

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import pro.d11l.fitcoach.core.network.SetPrescriptionDto
import pro.d11l.fitcoach.data.LoggedSetState

/**
 * On-device autoregulation (E5-PR6): from the just-logged set's performance vs its
 * RPE target, nudge the next set's load toward that target. Pure, offline, no LLM.
 */
class AutoregulatorTest {

    private val prev = SetPrescriptionDto(type = "reps", reps = 8, loadKg = 20.0, rpeTarget = 7.0)
    private val next = SetPrescriptionDto(type = "reps", reps = 8, loadKg = 20.0, rpeTarget = 7.0)

    private fun logged(reps: Int? = null, rpe: Double? = null, skipped: Boolean = false) =
        LoggedSetState(repsDone = reps, rpeActual = rpe, skipped = skipped, completed = !skipped)

    @Test
    fun `more reps than prescribed raises the next load`() {
        // 2 reps over target -> +6% -> 20 * 1.06 = 21.2 -> 21.0 (round 0.5)
        val adjusted = Autoregulator.adjust(prev, logged(reps = 10), next)
        assertEquals(21.0, adjusted.loadKg!!, 0.0)
    }

    @Test
    fun `fewer reps than prescribed lowers the next load`() {
        // 2 reps under -> -6% -> 20 * 0.94 = 18.8 -> 19.0
        val adjusted = Autoregulator.adjust(prev, logged(reps = 6), next)
        assertEquals(19.0, adjusted.loadKg!!, 0.0)
    }

    @Test
    fun `hitting the prescription exactly leaves the next load unchanged`() {
        assertEquals(20.0, Autoregulator.adjust(prev, logged(reps = 8), next).loadKg!!, 0.0)
    }

    @Test
    fun `a skipped set does not autoregulate`() {
        assertEquals(20.0, Autoregulator.adjust(prev, logged(reps = 0, skipped = true), next).loadKg!!, 0.0)
    }

    @Test
    fun `bodyweight next set (no load) is returned unchanged`() {
        val bodyweight = next.copy(loadKg = null)
        assertNull(Autoregulator.adjust(prev, logged(reps = 12), bodyweight).loadKg)
    }

    @Test
    fun `logged RPE overrides rep inference`() {
        // Hit the reps but at RPE 9 vs target 7 -> too hard -> -6% -> 19.0
        val adjusted = Autoregulator.adjust(prev, logged(reps = 8, rpe = 9.0), next)
        assertEquals(19.0, adjusted.loadKg!!, 0.0)
    }

    @Test
    fun `adjustment is clamped to plus or minus fifteen percent`() {
        // 12 reps over would be +36%, clamp to +15% -> 20 * 1.15 = 23.0
        val adjusted = Autoregulator.adjust(prev, logged(reps = 20), next)
        assertEquals(23.0, adjusted.loadKg!!, 0.0)
    }

    @Test
    fun `result rounds to the nearest half kilogram`() {
        // next load 21, 1 rep over -> +3% -> 21 * 1.03 = 21.63 -> 21.5
        val adjusted = Autoregulator.adjust(prev, logged(reps = 9), next.copy(loadKg = 21.0))
        assertEquals(21.5, adjusted.loadKg!!, 0.0)
    }
}

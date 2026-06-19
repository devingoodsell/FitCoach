package pro.d11l.fitcoach.feature.session

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class TimerControllerTest {

    @Test
    fun `start runs from zero`() {
        val s = TimerController.start()
        assertEquals(0, s.elapsedSec)
        assertTrue(s.running)
    }

    @Test
    fun `tick accumulates while running and freezes when stopped`() {
        var s = TimerController.start()
        s = TimerController.tick(s)
        s = TimerController.tick(s)
        assertEquals(2, s.elapsedSec)

        s = TimerController.stop(s)
        assertFalse(s.running)
        assertEquals(2, TimerController.tick(s).elapsedSec) // no change while stopped

        assertTrue(TimerController.resume(s).running)
    }
}

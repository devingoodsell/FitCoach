package pro.d11l.fitcoach.feature.session

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class RestControllerTest {

    @Test
    fun `start runs a positive countdown`() {
        val s = RestController.start(60)
        assertEquals(60, s.totalSec)
        assertEquals(60, s.remainingSec)
        assertTrue(s.running)
        assertFalse(s.finished)
    }

    @Test
    fun `start with zero is already finished and not running`() {
        val s = RestController.start(0)
        assertTrue(s.finished)
        assertFalse(s.running)
    }

    @Test
    fun `tick decrements and stops at zero`() {
        var s = RestController.start(2)
        s = RestController.tick(s)
        assertEquals(1, s.remainingSec)
        assertTrue(s.running)
        s = RestController.tick(s)
        assertEquals(0, s.remainingSec)
        assertFalse(s.running)
        assertTrue(s.finished)
        // Ticking a finished rest is a no-op.
        assertEquals(s, RestController.tick(s))
    }

    @Test
    fun `pause freezes and resume restarts`() {
        val running = RestController.start(30)
        val paused = RestController.pause(running)
        assertFalse(paused.running)
        assertEquals(30, RestController.tick(paused).remainingSec) // frozen
        assertTrue(RestController.resume(paused).running)
    }

    @Test
    fun `extend adds time and keeps running`() {
        val s = RestController.extend(RestController.start(30))
        assertEquals(45, s.totalSec)
        assertEquals(45, s.remainingSec)
        assertTrue(s.running)
    }

    @Test
    fun `skip ends immediately`() {
        val s = RestController.skip(RestController.start(30))
        assertEquals(0, s.remainingSec)
        assertTrue(s.finished)
        assertFalse(s.running)
    }
}

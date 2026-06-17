package pro.d11l.fitcoach.data

import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import pro.d11l.fitcoach.core.designsystem.DisclaimerText
import pro.d11l.fitcoach.core.network.DisclaimerDocDto
import pro.d11l.fitcoach.testing.FakeApi
import org.junit.Test

class DisclaimerRepositoryTest {

    @Test
    fun `fetch returns server copy`() = runTest {
        val api = FakeApi().apply {
            disclaimerDoc = DisclaimerDocDto(version = "v2", medical = "med", healthData = "hd")
        }
        val text = DisclaimerRepository(api).fetch()
        assertEquals("v2", text.version)
        assertEquals("med", text.medical)
        assertEquals("hd", text.healthData)
    }

    @Test
    fun `fetch falls back to bundled copy on failure`() = runTest {
        val api = FakeApi().apply { disclaimerError = true }
        val text = DisclaimerRepository(api).fetch()
        assertEquals(DisclaimerText.Bundled, text)
    }
}

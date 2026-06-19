package pro.d11l.fitcoach.feature.session

import android.provider.Settings
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.room.Room
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import kotlinx.coroutines.runBlocking
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import pro.d11l.fitcoach.BuildConfig
import pro.d11l.fitcoach.core.auth.TokenStorage
import pro.d11l.fitcoach.core.auth.Tokens
import pro.d11l.fitcoach.core.db.FitCoachDatabase
import pro.d11l.fitcoach.core.network.NetworkModule
import pro.d11l.fitcoach.data.RoomSessionCache
import pro.d11l.fitcoach.data.SessionRepository

/**
 * E12-PR1 — proves the in-session path runs end-to-end with no connectivity. With
 * the device in real airplane mode, a previously-cached session (E5-PR5) is served
 * from Room and rendered set-by-set; the one network attempt at "Start workout"
 * fails fast and falls back to cache, and nothing in the rendered session loop
 * depends on the backend.
 */
@RunWith(AndroidJUnit4::class)
class OfflineSessionTest {

    @get:Rule
    val compose = createComposeRule()

    private val context = ApplicationProvider.getApplicationContext<android.content.Context>()
    private lateinit var db: FitCoachDatabase

    private val noTokens = object : TokenStorage {
        override fun save(tokens: Tokens) = Unit
        override fun load(): Tokens? = null
        override fun clear() = Unit
    }

    @Before
    fun goOffline() {
        // In-memory DB keeps the cache isolated from any real app data.
        db = Room.inMemoryDatabaseBuilder(context, FitCoachDatabase::class.java).build()
        setAirplaneMode(true)
        assertTrue("airplane mode should be on", airplaneModeOn())
    }

    @After
    fun goOnline() {
        setAirplaneMode(false)
        db.close()
    }

    @Test
    fun generatedSessionRunsEndToEndOffline() {
        val sample = sampleSession()
        val cache = RoomSessionCache(db.sessionDao(), NetworkModule.json)
        runBlocking { cache.save(sample, "csid-offline") }

        // Real network stack pointed at the backend; unreachable in airplane mode.
        val api = NetworkModule.create(BuildConfig.BACKEND_BASE_URL, noTokens)
        val repo = SessionRepository(api, cache)

        // The one allowed network touch fails (offline) and falls back to cache.
        val result = runBlocking { repo.generate() }
        assertTrue("offline generate should succeed from cache", result.isSuccess)
        assertEquals(sample.id, result.getOrNull()?.id)

        val vm = SessionViewModel(repo)
        compose.setContent { MaterialTheme { SessionScreen(vm, onBack = {}) } }

        // Enter the player from cache with radios off.
        compose.onNodeWithText("Start workout").performClick()
        compose.waitUntil(timeoutMillis = 15_000) {
            compose.onAllNodesWithText("Rower easy spin").fetchSemanticsNodes().isNotEmpty()
        }
        compose.onNodeWithText("Rower easy spin").assertIsDisplayed()
        compose.onNodeWithText("Log set").assertIsDisplayed()

        // Logging a set (offline write) advances to the next exercise.
        compose.onNodeWithText("Log set").performClick()
        compose.waitUntil(timeoutMillis = 15_000) {
            compose.onAllNodesWithText("Goblet box squat").fetchSemanticsNodes().isNotEmpty()
        }
        compose.onNodeWithText("Goblet box squat").assertIsDisplayed()
    }

    @Test
    fun cachedReadIsPurelyLocal() {
        val sample = sampleSession()
        val cache = RoomSessionCache(db.sessionDao(), NetworkModule.json)
        runBlocking { cache.save(sample, "csid-offline") }

        // Reading the cached session (the in-session data source) touches no network.
        val cached = runBlocking { cache.latest() }
        assertEquals(sample.id, cached?.session?.id)
        assertEquals("csid-offline", cached?.clientSessionId)
        assertEquals(sample.mainWork.first().name, cached?.session?.mainWork?.first()?.name)
    }

    // --- airplane-mode helpers ----------------------------------------------

    private fun setAirplaneMode(on: Boolean) {
        val state = if (on) "enable" else "disable"
        shell("cmd connectivity airplane-mode $state")
        // Belt-and-suspenders for older images that ignore the connectivity cmd.
        shell("svc wifi ${if (on) "disable" else "enable"}")
        shell("svc data ${if (on) "disable" else "enable"}")
        // Allow the radios to settle.
        repeat(20) {
            if (airplaneModeOn() == on) return
            Thread.sleep(100)
        }
    }

    private fun airplaneModeOn(): Boolean =
        Settings.Global.getInt(context.contentResolver, Settings.Global.AIRPLANE_MODE_ON, 0) == 1

    private fun shell(command: String) {
        val auto = InstrumentationRegistry.getInstrumentation().uiAutomation
        auto.executeShellCommand(command).use { pfd ->
            java.io.FileInputStream(pfd.fileDescriptor).use { it.readBytes() }
        }
    }
}

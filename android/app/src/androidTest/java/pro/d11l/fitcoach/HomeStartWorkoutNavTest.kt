package pro.d11l.fitcoach

import android.content.Context
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createEmptyComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ActivityScenario
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.After
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import pro.d11l.fitcoach.core.auth.KeystoreTokenStorage
import pro.d11l.fitcoach.core.auth.Tokens

/**
 * Activity-level glue test (PR6): proves the real app navigation — not the screen
 * in isolation — wires Home's "Start workout" through AppRoot's step switch and
 * the AppViewModelFactory to the session surface. Seeds a logged-in token so the
 * app starts on Home, then launches MainActivity.
 */
@RunWith(AndroidJUnit4::class)
class HomeStartWorkoutNavTest {

    @get:Rule
    val compose = createEmptyComposeRule()

    private val context = ApplicationProvider.getApplicationContext<Context>()
    private val tokenStorage = KeystoreTokenStorage(context)

    @Before
    fun signIn() {
        // isLoggedIn() reads token storage; a present token starts AppRoot on Home.
        tokenStorage.save(Tokens(accessToken = "test-access", refreshToken = "test-refresh"))
    }

    @After
    fun signOut() {
        tokenStorage.clear()
    }

    @Test
    fun startWorkoutFromHomeOpensTheSessionScreen() {
        ActivityScenario.launch(MainActivity::class.java).use {
            // On Home: the only "Start workout" button is Home's nav button.
            compose.onNodeWithText("Start workout").performClick()

            // The session surface ("Today's workout") is unique to the session screen,
            // so seeing it proves the factory built the VM and AppRoot rendered it.
            compose.waitUntil(timeoutMillis = 10_000) {
                compose.onAllNodesWithText("Today's workout").fetchSemanticsNodes().isNotEmpty()
            }
            compose.onNodeWithText("Today's workout").assertIsDisplayed()
        }
    }
}

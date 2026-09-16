package tech.shatrov.familyfinances

import android.app.Application
import android.net.Uri
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.core.app.ApplicationProvider
import mockwebserver3.MockWebServer
import org.junit.After
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.theme.AppTheme
import java.io.File

private const val WAIT_MS = 5_000L

/** Маршрут импорта в корне: share до входа открывается экраном распознавания после бутстрапа. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class AppRootImportTest {
    @get:Rule
    val composeRule = createComposeRule()

    private lateinit var server: MockWebServer
    private val app = ApplicationProvider.getApplicationContext<Application>()
    private val res = app.resources

    @Before
    fun start() {
        server = MockWebServer()
        server.start()
    }

    @After
    fun stop() {
        server.close()
    }

    @Test
    fun shareBeforeLoginOpensRecognizeAfterBootstrap() {
        val graph = AppGraph(ApiGraph(server.url("/").toString(), FakeTokenVault()))
        // Нечитаемый URI: экран распознавания откажет сам, до платного вызова.
        graph.imports.offer(listOf(Uri.fromFile(File(app.filesDir, "missing.png"))))
        server.enqueueJson(200, LOGIN_OK)
        server.enqueueJson(200, ME_OK)
        server.enqueueJson(200, FAMILY_OK)
        composeRule.setContent { AppTheme { AppRoot(graph) } }

        composeRule.onNodeWithText(res.getString(R.string.login_email)).performTextInput("admin@test.com")
        composeRule.onNodeWithText(res.getString(R.string.login_password)).performTextInput("password-123")
        composeRule.onNodeWithText(res.getString(R.string.login_submit)).performClick()

        val refused = res.getString(R.string.recognize_no_images)
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(refused).fetchSemanticsNodes().isNotEmpty() }
        composeRule.onNodeWithText(res.getString(R.string.recognize_title)).assertExists()
        assertNull(graph.imports.pending.value)
    }
}

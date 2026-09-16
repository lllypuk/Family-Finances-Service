package tech.shatrov.familyfinances

import android.app.Application
import android.net.Uri
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performTextInput
import androidx.test.core.app.ApplicationProvider
import mockwebserver3.Dispatcher
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import mockwebserver3.RecordedRequest
import org.junit.After
import org.junit.Assert.assertEquals
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

/** Маршрут импорта в корне: share до входа открывается после бутстрапа, открытую форму не прерывает. */
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

    @Test
    fun shareDuringTransactionFormWaits() {
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse = when (request.url.encodedPath) {
                "/api/v1/me" -> json(ME_OK)
                "/api/v1/family" -> json(FAMILY_OK)
                "/api/v1/stats/summary" -> json(STATS_EMPTY)
                "/api/v1/categories" -> json(CATEGORIES_OK)
                else -> MockResponse.Builder().code(404).build()
            }
        }
        val graph = AppGraph(ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())))
        composeRule.setContent { AppTheme { AppRoot(graph) } }

        val add = res.getString(R.string.home_add_transaction)
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(add).fetchSemanticsNodes().isNotEmpty() }
        composeRule.onNodeWithText(add).performClick()
        val title = res.getString(R.string.transaction_new_title)
        composeRule.onNodeWithText(title).assertIsDisplayed()

        val id = graph.imports.offer(listOf(Uri.fromFile(File(app.filesDir, "missing.png"))))
        composeRule.waitForIdle()

        composeRule.onNodeWithText(title).assertIsDisplayed()
        composeRule.onAllNodesWithText(res.getString(R.string.recognize_title)).assertCountEquals(0)
        assertEquals(id, graph.imports.pending.value)
    }

    private fun json(body: String) = MockResponse.Builder()
        .body(body)
        .setHeader("Content-Type", "application/json")
        .build()
}

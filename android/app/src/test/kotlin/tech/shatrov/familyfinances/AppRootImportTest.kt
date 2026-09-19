package tech.shatrov.familyfinances

import android.app.Application
import android.net.Uri
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.StateRestorationTester
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
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.auth.TokenVault
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.recognize.ImportFiles
import tech.shatrov.familyfinances.ui.recognize.ImportJournal
import tech.shatrov.familyfinances.ui.recognize.ImportJournalStore
import java.io.File
import java.util.UUID
import java.util.concurrent.TimeUnit

private const val WAIT_MS = 5_000L

/**
 * Маршрут импорта в корне: share до входа открывается после бутстрапа, открытую форму не прерывает,
 * импорт с журналом переживает смерть процесса, выход журнал удаляет.
 */
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
        val graph = graph(FakeTokenVault())
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
        val graph = graph()
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

    @Test
    fun processDeathOnRecognizeWithJournalReturnsThere() {
        serveHome()
        var graph = graph()
        val restoration = StateRestorationTester(composeRule)
        restoration.setContent { AppTheme { AppRoot(graph) } }
        openRecognize(graph)

        graph = graph()
        restoration.emulateSavedInstanceStateRestore()

        val refused = res.getString(R.string.recognize_no_images)
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(refused).fetchSemanticsNodes().isNotEmpty() }
        composeRule.onNodeWithText(res.getString(R.string.recognize_title)).assertExists()
    }

    @Test
    fun processDeathOnRecognizeWithoutJournalGoesHome() {
        serveHome()
        var graph = graph()
        val restoration = StateRestorationTester(composeRule)
        restoration.setContent { AppTheme { AppRoot(graph) } }
        openRecognize(graph)
        graph.journals.deleteAll()

        graph = graph()
        restoration.emulateSavedInstanceStateRestore()

        val add = res.getString(R.string.home_add_transaction)
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(add).fetchSemanticsNodes().isNotEmpty() }
        composeRule.onAllNodesWithText(res.getString(R.string.recognize_title)).assertCountEquals(0)
    }

    // Плашка ведёт на экран без `offer`: модель поднимается из журнала, прерванный вызов сама не повторяет.
    @Test
    fun resumeFromHomeRestoresImportFromJournal() {
        serveHome()
        val graph = graph()
        graph.journals.write(
            ImportJournal(
                importId = UUID.randomUUID().toString(),
                updatedAt = System.currentTimeMillis(),
                images = emptyList(),
                dropped = 0,
                recognizing = true,
                result = null,
                accountId = null,
                rows = emptyList(),
                savedCount = 0,
            ),
        )
        composeRule.setContent { AppTheme { AppRoot(graph) } }

        val resume = res.getString(R.string.home_import_resume)
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(resume).fetchSemanticsNodes().isNotEmpty() }
        composeRule.onNodeWithText(resume).performClick()

        val interrupted = res.getString(R.string.recognize_error_interrupted)
        composeRule.waitUntil(WAIT_MS) {
            composeRule.onAllNodesWithText(interrupted).fetchSemanticsNodes().isNotEmpty()
        }
        assertNull(graph.imports.pending.value)
        val paths = generateSequence { server.takeRequest(0, TimeUnit.SECONDS) }.map { it.url.encodedPath }
        assertTrue(paths.none { it == "/api/v1/transactions/recognize" })
    }

    @Test
    fun signOutDeletesImportJournals() {
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse =
                MockResponse.Builder().code(500).body(INTERNAL_ERROR).build()
        }
        val graph = graph()
        graph.journals.write(
            ImportJournal(
                importId = UUID.randomUUID().toString(),
                updatedAt = System.currentTimeMillis(),
                images = emptyList(),
                dropped = 0,
                recognizing = false,
                result = null,
                accountId = null,
                rows = emptyList(),
                savedCount = 0,
            ),
        )
        assertTrue(importRoot().exists())
        composeRule.setContent { AppTheme { AppRoot(graph) } }

        val signOut = res.getString(R.string.sign_out)
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(signOut).fetchSemanticsNodes().isNotEmpty() }
        composeRule.onNodeWithText(signOut).performClick()
        composeRule.waitUntil(WAIT_MS) {
            composeRule.onAllNodesWithText(res.getString(R.string.login_submit)).fetchSemanticsNodes().isNotEmpty()
        }

        assertFalse(importRoot().exists())
    }

    private fun graph(vault: TokenVault = FakeTokenVault(liveToken())) =
        AppGraph(ApiGraph(server.url("/").toString(), vault), ImportJournalStore(ImportFiles.filesRoot(app)))

    private fun importRoot() = ImportFiles.filesRoot(app)

    private fun serveHome() {
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse = when (request.url.encodedPath) {
                "/api/v1/me" -> json(ME_OK)
                "/api/v1/family" -> json(FAMILY_OK)
                "/api/v1/stats/summary" -> json(STATS_EMPTY)
                "/api/v1/categories" -> json(CATEGORIES_OK)
                "/api/v1/accounts" -> json(ACCOUNTS_EMPTY)
                else -> MockResponse.Builder().code(404).build()
            }
        }
    }

    // Нечитаемый URI: журнал пишется после подготовки, а распознавание отказывает до платного вызова.
    private fun openRecognize(graph: AppGraph) {
        val add = res.getString(R.string.home_add_transaction)
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(add).fetchSemanticsNodes().isNotEmpty() }
        val id = graph.imports.offer(listOf(Uri.fromFile(File(app.filesDir, "missing.png"))))
        val refused = res.getString(R.string.recognize_no_images)
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(refused).fetchSemanticsNodes().isNotEmpty() }
        assertTrue(File(importRoot(), "$id/journal.json").exists())
    }

    private fun json(body: String) = MockResponse.Builder()
        .body(body)
        .setHeader("Content-Type", "application/json")
        .build()
}

package tech.shatrov.familyfinances

import android.app.Application
import android.net.Uri
import androidx.activity.ComponentActivity
import androidx.compose.ui.test.assertCountEquals
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.junit4.v2.createAndroidComposeRule
import androidx.compose.ui.test.onAllNodesWithContentDescription
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.compose.ui.test.performScrollTo
import androidx.compose.ui.test.performTextInput
import androidx.test.core.app.ApplicationProvider
import mockwebserver3.Dispatcher
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import mockwebserver3.RecordedRequest
import org.junit.After
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.ui.recognize.ImportFiles
import tech.shatrov.familyfinances.ui.recognize.ImportJournalStore
import java.io.File
import java.util.concurrent.CopyOnWriteArrayList

private const val WAIT_MS = 5_000L

/** Справочник без вкладки: вход из настроек, «назад» туда же, уход помечает устаревшими списки. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class AppRootCategoriesTest {
    @get:Rule
    val composeRule = createAndroidComposeRule<ComponentActivity>()

    private lateinit var server: MockWebServer
    private val app = ApplicationProvider.getApplicationContext<Application>()
    private val res = app.resources
    private val requests = CopyOnWriteArrayList<RecordedRequest>()

    @Before
    fun start() {
        server = MockWebServer()
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse {
                requests += request
                return when (request.url.encodedPath) {
                    "/api/v1/me" -> json(ME_OK)
                    "/api/v1/family" -> json(FAMILY_OK)
                    "/api/v1/stats/summary" -> json(STATS_SUMMARY_RANGE_OK)
                    "/api/v1/stats/monthly" -> json(STATS_MONTHLY_OK)
                    "/api/v1/categories" -> json(CATEGORIES_OK)
                    "/api/v1/transactions" -> json(TRANSACTIONS_PAGE_1)
                    "/api/v1/accounts" -> json(ACCOUNTS_OK)
                    else -> MockResponse.Builder().code(404).build()
                }
            }
        }
        server.start()
    }

    @After
    fun stop() {
        server.close()
    }

    @Test
    fun navBarHasNoCategoriesTab() {
        composeRule.setContent { AppTheme { AppRoot(graph()) } }
        waitFor(res.getString(R.string.transactions_title))

        composeRule.onNode(hasText(res.getString(R.string.categories_title)) and hasClickAction())
            .assertDoesNotExist()
    }

    @Test
    fun backFromCategoriesReturnsToSettingsAndRefreshesHome() {
        composeRule.setContent { AppTheme { AppRoot(graph()) } }
        val settings = res.getString(R.string.settings_title)
        composeRule.waitUntil(WAIT_MS) {
            composeRule.onAllNodesWithContentDescription(settings).fetchSemanticsNodes().isNotEmpty()
        }
        composeRule.onNodeWithContentDescription(settings).performClick()
        waitFor(res.getString(R.string.settings_profile))

        composeRule.onNodeWithText(res.getString(R.string.categories_title)).performScrollTo().performClick()
        waitFor(res.getString(R.string.categories_income))
        composeRule.onNodeWithContentDescription(res.getString(R.string.categories_add)).assertExists()

        composeRule.onNodeWithContentDescription(res.getString(R.string.back)).performClick()
        waitFor(res.getString(R.string.settings_profile))

        val before = homeSummaryCount()
        pressBack()
        composeRule.waitUntil(WAIT_MS) { homeSummaryCount() > before }
    }

    // Справочник поверх формы: store форм не чистится, модель та же, а список категорий перечитан узко.
    @Test
    fun catalogOverFormKeepsDraftAndReloadsCategories() {
        val graph = graph()
        composeRule.setContent { AppTheme { AppRoot(graph) } }
        val tab = res.getString(R.string.transactions_title)
        waitFor(tab)
        composeRule.onNode(hasText(tab) and hasClickAction()).performClick()
        val add = res.getString(R.string.transactions_add)
        composeRule.waitUntil(WAIT_MS) {
            composeRule.onAllNodesWithContentDescription(add).fetchSemanticsNodes().isNotEmpty()
        }
        composeRule.onAllNodesWithContentDescription(add)[0].performClick()
        val manage = res.getString(R.string.categories_manage)
        waitFor(manage)
        composeRule.onNodeWithText(res.getString(R.string.transaction_amount)).performTextInput("1234")

        composeRule.onNodeWithText(manage).performScrollTo().performClick()
        waitFor(res.getString(R.string.categories_income))
        // Share поверх справочника, открытого с формы, ждёт, как на самой форме.
        graph.imports.offer(listOf(Uri.fromFile(File(app.filesDir, "missing.png"))))
        composeRule.waitForIdle()
        composeRule.onAllNodesWithText(res.getString(R.string.recognize_title)).assertCountEquals(0)
        val before = categoriesCount()

        composeRule.onNodeWithContentDescription(res.getString(R.string.back)).performClick()
        waitFor(manage)

        composeRule.onNode(hasText("1234", substring = true)).assertExists()
        composeRule.waitUntil(WAIT_MS) { categoriesCount() > before }
    }

    private fun categoriesCount() = requests.count { it.url.encodedPath == "/api/v1/categories" }

    private fun waitFor(text: String) {
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(text).fetchSemanticsNodes().isNotEmpty() }
    }

    private fun pressBack() {
        composeRule.runOnUiThread { composeRule.activity.onBackPressedDispatcher.onBackPressed() }
        composeRule.waitForIdle()
    }

    private fun homeSummaryCount() = requests.count {
        it.url.encodedPath == "/api/v1/stats/summary" && it.url.queryParameter("from") == null
    }

    private fun graph() = AppGraph(
        ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
        ImportJournalStore(ImportFiles.filesRoot(app)),
    )

    private fun json(body: String) = MockResponse.Builder()
        .body(body)
        .setHeader("Content-Type", "application/json")
        .build()
}

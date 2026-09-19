package tech.shatrov.familyfinances

import android.app.Application
import android.net.Uri
import androidx.activity.ComponentActivity
import androidx.compose.ui.test.SemanticsNodeInteraction
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.hasClickAction
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isEnabled
import androidx.compose.ui.test.junit4.StateRestorationTester
import androidx.compose.ui.test.junit4.v2.createAndroidComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onLast
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import mockwebserver3.Dispatcher
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import mockwebserver3.RecordedRequest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
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

/** «Обзор» в корне: вход с «Главной», расшифровка и возврат, свой флаг устаревания, share, смерть процесса. */
@RunWith(RobolectricTestRunner::class)
// Высокий экран: категории стоят под итогами и графиком, ленивый список ниже края их не создаёт.
@Config(sdk = [ROBOLECTRIC_SDK], qualifiers = "w411dp-h1600dp")
class AppRootOverviewTest {
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
                val path = request.url.encodedPath
                return when {
                    path == "/api/v1/me" -> json(ME_OK)

                    path == "/api/v1/family" -> json(FAMILY_OK)

                    path == "/api/v1/stats/summary" -> json(STATS_SUMMARY_RANGE_OK)

                    path == "/api/v1/stats/monthly" -> json(STATS_MONTHLY_OK)

                    path == "/api/v1/categories" -> json(CATEGORIES_OK)

                    path == "/api/v1/accounts" -> json(ACCOUNTS_EMPTY)

                    path == "/api/v1/users" -> json(USERS_OK)

                    path == "/api/v1/transactions" -> json(TRANSACTIONS_PAGE_1)

                    path == "/api/v1/transactions/$COFFEE_ID" && request.method == "DELETE" ->
                        MockResponse.Builder().code(204).build()

                    path == "/api/v1/transactions/$COFFEE_ID" -> json(TRANSACTION_OK)

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
    fun backFromDrillReturnsToOverviewWithSamePeriod() {
        composeRule.setContent { AppTheme { AppRoot(graph()) } }
        openOverview()
        composeRule.onNodeWithText(res.getString(R.string.overview_three_months)).performClick()
        composeRule.waitForIdle()
        waitFor("Продукты")
        composeRule.onNodeWithText("Продукты").performClick()
        waitFor("Кофе")

        val drill = requests.last { it.url.encodedPath == "/api/v1/transactions" }.url
        assertEquals(GROCERIES_ID, drill.queryParameter("category_id"))
        val summary = requests.last { it.url.encodedPath == "/api/v1/stats/summary" }.url
        assertEquals(summary.queryParameter("from"), drill.queryParameter("date_from"))
        assertEquals(summary.queryParameter("to"), drill.queryParameter("date_to"))

        pressBack()
        waitFor(res.getString(R.string.overview_three_months))
        composeRule.onNodeWithText(res.getString(R.string.overview_three_months)).assertIsSelected()
    }

    @Test
    fun transactionsTabAfterDrillOpensWithoutFilter() {
        composeRule.setContent { AppTheme { AppRoot(graph()) } }
        openOverview()
        waitFor("Продукты")
        composeRule.onNodeWithText("Продукты").performClick()
        waitFor("Кофе")

        val before = requests.size
        tab(R.string.transactions_title).performClick()
        composeRule.waitUntil(WAIT_MS) {
            requests.drop(before).any { it.url.encodedPath == "/api/v1/transactions" }
        }

        val list = requests.drop(before).last { it.url.encodedPath == "/api/v1/transactions" }.url
        assertNull(list.queryParameter("category_id"))
        assertNull(list.queryParameter("date_from"))
    }

    // «Обзор» гасит свой флаг, а не общий с «Главной»: та после него всё ещё перечитывает итоги.
    @Test
    fun homeStillRefreshesAfterOverviewConsumedItsFlag() {
        composeRule.setContent { AppTheme { AppRoot(graph()) } }
        openOverview()
        waitFor("Продукты")
        composeRule.onNodeWithText("Продукты").performClick()
        waitFor("Кофе")
        composeRule.onNodeWithText("Кофе").performClick()
        val delete = res.getString(R.string.transaction_delete)
        composeRule.waitUntil(WAIT_MS) {
            composeRule.onAllNodes(hasText(delete) and isEnabled()).fetchSemanticsNodes().isNotEmpty()
        }
        composeRule.onNodeWithText(delete).performClick()
        waitFor(res.getString(R.string.transaction_delete_confirm))
        composeRule.onAllNodesWithText(delete).onLast().performClick()
        waitFor("Хлеб")

        val beforeOverview = monthlyCount()
        pressBack()
        composeRule.waitUntil(WAIT_MS) { monthlyCount() > beforeOverview }
        composeRule.waitForIdle()

        val beforeHome = homeSummaryCount()
        composeRule.onNodeWithContentDescription(res.getString(R.string.back)).performClick()
        composeRule.waitUntil(WAIT_MS) { homeSummaryCount() > beforeHome }
    }

    @Test
    fun shareOnOverviewOpensRecognize() {
        val graph = graph()
        composeRule.setContent { AppTheme { AppRoot(graph) } }
        openOverview()

        graph.imports.offer(listOf(Uri.fromFile(File(app.filesDir, "missing.png"))))

        waitFor(res.getString(R.string.recognize_no_images))
        composeRule.onNodeWithText(res.getString(R.string.recognize_title)).assertExists()
    }

    @Test
    fun processDeathOnOverviewReturnsThereWithPeriod() {
        var graph = graph()
        val restoration = StateRestorationTester(composeRule)
        restoration.setContent { AppTheme { AppRoot(graph) } }
        openOverview()
        composeRule.onNodeWithText(res.getString(R.string.overview_year)).performClick()
        composeRule.waitForIdle()

        graph = graph()
        restoration.emulateSavedInstanceStateRestore()

        waitFor(res.getString(R.string.overview_year))
        composeRule.onNodeWithText(res.getString(R.string.overview_year)).assertIsSelected()
        assertNotNull(graph.session.value)
    }

    private fun openOverview() {
        val entry = res.getString(R.string.overview_title)
        waitFor(entry)
        composeRule.onNodeWithText(entry).performClick()
        waitFor(res.getString(R.string.overview_this_month))
    }

    private fun waitFor(text: String) {
        composeRule.waitUntil(WAIT_MS) { composeRule.onAllNodesWithText(text).fetchSemanticsNodes().isNotEmpty() }
    }

    private fun tab(title: Int): SemanticsNodeInteraction =
        composeRule.onNode(hasText(res.getString(title)) and hasClickAction())

    private fun pressBack() {
        composeRule.runOnUiThread { composeRule.activity.onBackPressedDispatcher.onBackPressed() }
        composeRule.waitForIdle()
    }

    private fun monthlyCount() = requests.count { it.url.encodedPath == "/api/v1/stats/monthly" }

    // Сводка «Главной» уходит без границ, у «Обзора» они есть всегда.
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

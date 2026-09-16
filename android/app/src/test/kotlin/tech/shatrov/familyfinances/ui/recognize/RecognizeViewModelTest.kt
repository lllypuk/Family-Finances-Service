package tech.shatrov.familyfinances.ui.recognize

import android.app.Application
import android.graphics.Bitmap
import android.net.Uri
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.ViewModelStore
import androidx.lifecycle.viewmodel.initializer
import androidx.lifecycle.viewmodel.viewModelFactory
import androidx.test.core.app.ApplicationProvider
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import mockwebserver3.RecordedRequest
import mockwebserver3.SocketEffect
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import tech.shatrov.familyfinances.CATEGORIES_OK
import tech.shatrov.familyfinances.COFFEE_ID
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.ImportStore
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.SALARY_ID
import tech.shatrov.familyfinances.TRANSACTION_OK
import tech.shatrov.familyfinances.VALIDATION_ERROR
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.transactions.TransactionField
import java.io.File
import java.time.LocalDate
import java.util.UUID
import java.util.concurrent.TimeUnit

/** Шавуха с похожей, зарплата без даты и строка со всеми блокировками разом. */
private const val RECOGNIZE_OK = """
{"data":{"items":[
{"source":0,"amount_minor":30000,"currency":"RUB","type":"expense","date":"2026-09-14","date_assumed":false,
"description":"Шавуха","category_id":"$GROCERIES_ID",
"similar":[{"id":"$COFFEE_ID","date":"2026-09-14","description":"шавуха"}]},
{"source":1,"amount_minor":150000,"currency":null,"type":"income","date":null,"date_assumed":false,
"description":"Зарплата","category_id":"$SALARY_ID","similar":[]},
{"source":null,"amount_minor":999,"currency":"USD","type":"expense","date":"2026-09-10","date_assumed":true,
"description":"x","category_id":null,"similar":[]}],
"incomplete":true,"model":"gemma4:31b"},
"meta":{"request_id":"r-70","timestamp":"2026-09-16T10:00:00Z","version":"v0.5.0"}}
"""

private const val UNAVAILABLE = """{"error":{"code":"RECOGNITION_UNAVAILABLE","message":"недоступно"}}"""

private const val RECOGNIZE_PATH = "/api/v1/transactions/recognize"
private const val TRANSACTIONS_PATH = "/api/v1/transactions"

// Нативная графика нужна ImportFiles: без неё Robolectric не кодирует JPEG.
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class RecognizeViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var app: Application
    private lateinit var store: ImportStore
    private lateinit var importId: UUID
    private lateinit var model: RecognizeViewModel

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        server.start()
        app = ApplicationProvider.getApplicationContext()
        store = ImportStore()
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private fun shot(name: String): Uri {
        val dir = File(app.filesDir, "src").apply { mkdirs() }
        val bitmap = Bitmap.createBitmap(64, 128, Bitmap.Config.ARGB_8888)
        return Uri.fromFile(
            File(dir, name).apply {
                outputStream().use { bitmap.compress(Bitmap.CompressFormat.PNG, 100, it) }
            },
        )
    }

    private fun junk(): Uri = Uri.fromFile(File(app.filesDir, "junk.txt").apply { writeText("не картинка") })

    private fun newModel() = RecognizeViewModel(
        app,
        ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
        store,
        importId,
        "RUB",
    )

    private fun createModel(uris: List<Uri> = listOf(shot("a.png"), junk(), shot("b.png"))) {
        importId = store.offer(uris)
        model = newModel()
    }

    private suspend fun reviewed(): RecognizeUiState = model.state.first {
        (it.phase as? RecognizePhase.Review)?.saving == false || it.phase is RecognizePhase.Failure
    }

    private fun rows(): List<RecognizedRow> = (model.state.value.phase as RecognizePhase.Review).rows

    private fun requests(): List<RecordedRequest> = (1..server.requestCount).map { server.takeRequest() }

    private fun RecordedRequest.text(): String = body?.utf8().orEmpty()

    private suspend fun recognized(): List<RecognizedRow> {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, RECOGNIZE_OK)
        createModel()
        reviewed()
        return rows()
    }

    @Test
    fun rowsComeFromReadyImagesAndSecondShareWaits() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueue(
            MockResponse.Builder()
                .body(RECOGNIZE_OK)
                .setHeader("Content-Type", "application/json")
                .headersDelay(500, TimeUnit.MILLISECONDS)
                .build(),
        )
        createModel()

        model.state.first { it.phase is RecognizePhase.Recognizing }
        val next = store.offer(listOf(shot("c.png")))
        assertNull(store.pending.value)
        assertTrue(store.waiting.value)

        val state = reviewed()
        assertEquals(next, store.pending.value)
        assertTrue(state.incomplete)
        assertTrue(state.images[1] is ImportImage.Failed)
        val rows = rows()
        // Нечитаемая картинка не уходила: `source = 1` — это третья из предложенных.
        assertEquals(listOf(0, 2, null), rows.map { it.image })
        assertEquals(3, rows.map { it.draft }.toSet().size)
        assertTrue(rows.all { it.included })
        assertEquals("шавуха", rows[0].similarTo.single().description)
        assertTrue(rows[2].currencyMismatch)
        assertEquals(1, state.toSave)

        val sent = requests()
        assertEquals(listOf("/api/v1/categories", RECOGNIZE_PATH), sent.map { it.url.encodedPath })
        assertEquals(2, Regex("name=\"images\"").findAll(sent[1].text()).count())
    }

    @Test
    fun unavailableIsOneCallAndRetriedOnlyOnTap() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(503, UNAVAILABLE)
        createModel()

        val failed = reviewed().phase as RecognizePhase.Failure
        assertTrue(failed.retryable)
        assertEquals(UiError.Resource(R.string.recognize_error_unavailable), failed.error)
        assertEquals(2, server.requestCount)

        server.enqueueJson(200, RECOGNIZE_OK)
        model.retry()
        reviewed()

        // Справочник уже есть: повтор — только сам вызов модели.
        assertEquals(
            listOf("/api/v1/categories", RECOGNIZE_PATH, RECOGNIZE_PATH),
            requests().map {
                it.url.encodedPath
            },
        )
    }

    @Test
    fun importClaimedElsewhereFailsWithoutRequests() = runTest {
        importId = store.offer(listOf(shot("a.png")))
        store.claim(importId)
        model = newModel()

        val failed = model.state.value.phase as RecognizePhase.Failure
        assertFalse(failed.retryable)
        assertEquals(0, server.requestCount)
    }

    @Test
    fun blockedRowsUnlockOnlyByExplicitEdits() = runTest {
        val (shawarma, salary, foreign) = recognized().map { it.draft }

        model.onDateChange(salary, LocalDate.parse("2026-09-01"))
        assertTrue(rows()[1].savable)

        model.onDateChange(foreign, LocalDate.parse("2026-09-10"))
        model.onCategoryChange(foreign, UUID.fromString(GROCERIES_ID))
        model.onDescriptionChange(foreign, "Книга")
        assertFalse(rows()[2].dateAssumed)
        // Валюту строки не поправить: `POST` её не принимает.
        assertFalse(rows()[2].savable)

        model.onAmountChange(shawarma, 35000)
        assertTrue(rows()[0].similarTo.isEmpty())
        model.onTypeChange(shawarma, TransactionType.income)
        assertNull(rows()[0].categoryId)
        assertFalse(rows()[0].savable)

        model.onIncludedChange(salary, false)
        assertEquals(0, model.state.value.toSave)
    }

    @Test
    fun disconnectAfterSendRetriesSameDraftAndTakesServerFields() = runTest {
        val draft = recognized()[0].draft
        server.enqueue(
            MockResponse.Builder()
                .code(201)
                .body(TRANSACTION_OK)
                .setHeader("Content-Type", "application/json")
                .onResponseBody(SocketEffect.ShutdownConnection)
                .build(),
        )

        model.save()
        reviewed()
        assertTrue(rows()[0].status is RowStatus.Failed)
        assertEquals(0, model.state.value.savedCount)

        model.onDescriptionChange(draft, "Шавуха большая")
        server.enqueueJson(200, TRANSACTION_OK)
        model.retryRow(draft)
        reviewed()

        val row = rows()[0]
        assertEquals(RowStatus.Saved, row.status)
        assertEquals("Кофе", row.description)
        assertEquals(150000L, row.amountMinor)
        assertEquals(1, model.state.value.savedCount)

        val posts = requests().filter { it.url.encodedPath == TRANSACTIONS_PATH }
        assertEquals(2, posts.size)
        assertTrue(posts.all { it.text().contains("\"id\":\"$draft\"") })

        model.onDescriptionChange(draft, "Правка после сохранения")
        assertEquals("Кофе", rows()[0].description)
    }

    @Test
    fun validationErrorLandsUnderRowField() = runTest {
        val draft = recognized()[0].draft
        server.enqueueJson(422, VALIDATION_ERROR)

        model.save()
        reviewed()

        val row = rows()[0]
        assertEquals(RowStatus.Pending, row.status)
        assertEquals(mapOf(TransactionField.AMOUNT to "должно быть больше нуля"), row.fieldErrors)

        model.onAmountChange(draft, 30100)
        assertTrue(rows()[0].fieldErrors.isEmpty())
    }

    @Test
    fun partialBatchFailureResendsOnlyUnsavedRows() = runTest {
        val (shawarma, salary) = recognized().map { it.draft }
        model.onDateChange(salary, LocalDate.parse("2026-09-01"))
        server.enqueueJson(201, TRANSACTION_OK)
        server.enqueueJson(500, INTERNAL_ERROR)

        model.save()
        reviewed()
        assertEquals(RowStatus.Saved, rows()[0].status)
        assertTrue(rows()[1].status is RowStatus.Failed)
        assertEquals(1, model.state.value.savedCount)
        assertEquals(1, model.state.value.toSave)

        server.enqueueJson(201, TRANSACTION_OK)
        model.save()
        reviewed()

        val posts = requests().filter { it.url.encodedPath == TRANSACTIONS_PATH }.map { it.text() }
        assertEquals(3, posts.size)
        assertTrue(posts[0].contains("\"id\":\"$shawarma\""))
        assertTrue(posts[1].contains("\"id\":\"$salary\""))
        assertTrue(posts[2].contains("\"id\":\"$salary\""))
        assertEquals(2, model.state.value.savedCount)
    }

    @Test
    fun clearingModelDiscardsFilesAndReleasesHold() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, RECOGNIZE_OK)
        importId = store.offer(listOf(shot("a.png")))
        val owner = ViewModelStore()
        model =
            ViewModelProvider(owner, viewModelFactory { initializer { newModel() } })[RecognizeViewModel::class.java]
        reviewed()
        val dir = File(app.cacheDir, "import/$importId")
        assertTrue(dir.exists())

        owner.clear()

        assertFalse(dir.exists())
        val next = store.offer(emptyList())
        assertEquals(next, store.pending.value)
    }
}

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
import tech.shatrov.familyfinances.ACCOUNTS_EMPTY
import tech.shatrov.familyfinances.ACCOUNTS_OK
import tech.shatrov.familyfinances.CARD_ACCOUNT_ID
import tech.shatrov.familyfinances.CATEGORIES_NESTED
import tech.shatrov.familyfinances.CATEGORIES_OK
import tech.shatrov.familyfinances.COFFEE_ID
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.GROCERIES_ID
import tech.shatrov.familyfinances.INTERNAL_ERROR
import tech.shatrov.familyfinances.ImportStore
import tech.shatrov.familyfinances.MemoryLastAccountStore
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.SALARY_ID
import tech.shatrov.familyfinances.TRANSACTION_OK
import tech.shatrov.familyfinances.VALIDATION_ERROR
import tech.shatrov.familyfinances.core.api.ApiGraph
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

private const val RECOGNIZE_EMPTY = """
{"data":{"items":[],"incomplete":false,"model":"gemma4:31b"},
"meta":{"request_id":"r-71","timestamp":"2026-09-16T10:00:00Z","version":"v0.5.0"}}
"""

private const val UNAVAILABLE = """{"error":{"code":"RECOGNITION_UNAVAILABLE","message":"недоступно"}}"""

private const val UPLOAD_TIMEOUT = """{"error":{"code":"REQUEST_TIMEOUT","message":"upload timed out"}}"""

private const val ROW_INVALID = """
{"error":{"code":"VALIDATION_ERROR","message":"Проверьте поля","details":[
{"field":"amount_minor","message":"должно быть больше нуля","code":"gt"},
{"field":"description","message":"слишком коротко","code":"min"}]}}
"""

private const val RECOGNIZE_PATH = "/api/v1/transactions/recognize"
private const val TRANSACTIONS_PATH = "/api/v1/transactions"

// Справочники и распознавание — то, что `recognized()` уже отправил.
private const val RECOGNIZED_REQUESTS = 3
private const val REQUEST_WAIT_SECONDS = 20L
private const val REQUEST_POLL_MILLIS = 10L

// Нативная графика нужна ImportFiles: без неё Robolectric не кодирует JPEG.
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class RecognizeViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var app: Application
    private lateinit var store: ImportStore
    private lateinit var journals: ImportJournalStore
    private lateinit var importId: UUID
    private lateinit var model: RecognizeViewModel
    private var owner = ViewModelStore()
    private val lastAccount = MemoryLastAccountStore()

    // Запись журнала на нём же: к следующей строке теста она уже на диске.
    private val main = UnconfinedTestDispatcher()

    @Before
    fun start() {
        Dispatchers.setMain(main)
        server = MockWebServer()
        server.start()
        app = ApplicationProvider.getApplicationContext()
        store = ImportStore()
        journals = ImportJournalStore(ImportFiles.filesRoot(app))
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

    private fun newModel(): RecognizeViewModel = ViewModelProvider(
        owner,
        viewModelFactory {
            initializer {
                RecognizeViewModel(
                    app,
                    ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
                    store,
                    journals,
                    importId,
                    "RUB",
                    lastAccount,
                    main,
                )
            }
        },
    )["recognize-$importId", RecognizeViewModel::class.java]

    /** Смерть процесса: модель и память уходят, на диске остаётся то, что успело записаться. */
    private fun restart() {
        owner.clear()
        owner = ViewModelStore()
        store = ImportStore()
        journals = ImportJournalStore(ImportFiles.filesRoot(app))
        model = newModel()
    }

    private fun importDir() = File(app.filesDir, "import/$importId")

    private fun createModel(uris: List<Uri> = listOf(shot("a.png"), junk(), shot("b.png"))) {
        importId = store.offer(uris)
        model = newModel()
    }

    private suspend fun reviewed(): RecognizeUiState = model.state.first {
        (it.phase as? RecognizePhase.Review)?.saving == false || it.phase is RecognizePhase.Failure
    }

    private fun rows(): List<RecognizedRow> = (model.state.value.phase as RecognizePhase.Review).rows

    /** Модель отпускает импорт уже после публикации состояния: `pending` дожидаются, а не читают сразу. */
    private suspend fun pending(): UUID? = store.pending.first { it != null }

    /** Пока запрос не дошёл до сервера, отложенный ответ из очереди достанется следующему. */
    private fun awaitRequests(count: Int) {
        val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(REQUEST_WAIT_SECONDS)
        while (server.requestCount < count) {
            check(System.nanoTime() < deadline) { "сервер получил ${server.requestCount} из $count запросов" }
            Thread.sleep(REQUEST_POLL_MILLIS)
        }
    }

    private fun requests(): List<RecordedRequest> = (1..server.requestCount).map { server.takeRequest() }

    private fun RecordedRequest.text(): String = body?.utf8().orEmpty()

    private suspend fun recognized(): List<RecognizedRow> {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        server.enqueueJson(200, RECOGNIZE_OK)
        createModel()
        reviewed()
        return rows()
    }

    @Test
    fun rowsComeFromReadyImagesAndSecondShareWaits() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
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
        assertNull(store.pending.value)
        assertTrue(store.waiting.value)
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
        assertEquals(listOf("/api/v1/categories", "/api/v1/accounts", RECOGNIZE_PATH), sent.map { it.url.encodedPath })
        assertEquals(2, Regex("name=\"images\"").findAll(sent[2].text()).count())

        server.enqueueJson(201, TRANSACTION_OK)
        model.save()
        reviewed()
        assertEquals(next, pending())
    }

    @Test
    fun reviewWithoutWaitingShareIsReplaceable() = runTest {
        recognized()

        val next = store.offer(listOf(shot("c.png")))

        assertEquals(next, pending())
    }

    @Test
    fun slowUploadIsRetryable() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        server.enqueueJson(408, UPLOAD_TIMEOUT)
        createModel()

        val failed = reviewed().phase as RecognizePhase.Failure
        assertTrue(failed.retryable)
        assertEquals(UiError.Resource(R.string.recognize_error_upload_timeout), failed.error)
    }

    @Test
    fun rejectedRequestIsNotRetried() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        server.enqueueJson(422, VALIDATION_ERROR)
        createModel()

        val failed = reviewed().phase as RecognizePhase.Failure
        assertFalse(failed.retryable)

        model.retry()
        assertEquals(3, server.requestCount)
    }

    @Test
    fun unavailableIsOneCallAndRetriedOnlyOnTap() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        server.enqueueJson(503, UNAVAILABLE)
        createModel()

        val failed = reviewed().phase as RecognizePhase.Failure
        assertTrue(failed.retryable)
        assertEquals(UiError.Resource(R.string.recognize_error_unavailable), failed.error)
        assertEquals(3, server.requestCount)

        server.enqueueJson(200, RECOGNIZE_OK)
        model.retry()
        reviewed()

        // Справочник уже есть: повтор — только сам вызов модели.
        assertEquals(
            listOf("/api/v1/categories", "/api/v1/accounts", RECOGNIZE_PATH, RECOGNIZE_PATH),
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

        val failed = reviewed().phase as RecognizePhase.Failure
        assertFalse(failed.retryable)
        assertEquals(0, server.requestCount)
    }

    @Test
    fun blockedRowsUnlockOnlyByExplicitEdits() = runTest {
        val (shawarma, salary, foreign) = recognized().map { it.draft }

        model.onDateChange(salary, LocalDate.parse("2026-09-01"))
        assertTrue(rows()[1].savable)
        val salaryDescription = rows()[1].description
        model.onDescriptionChange(salary, " З ")
        assertFalse(rows()[1].savable)
        model.onDescriptionChange(salary, salaryDescription)
        assertTrue(rows()[1].savable)

        model.onDateChange(foreign, LocalDate.parse("2026-09-10"))
        model.onCategoryChange(foreign, UUID.fromString(GROCERIES_ID))
        model.onDescriptionChange(foreign, "Книга")
        assertFalse(rows()[2].dateAssumed)
        // Валюту строки не поправить: `POST` её не принимает.
        assertFalse(rows()[2].savable)

        model.onDateChange(shawarma, LocalDate.parse("2026-09-13"))
        assertTrue(rows()[0].similarTo.isEmpty())

        model.onIncludedChange(salary, false)
        assertEquals(1, model.state.value.toSave)
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
        server.enqueueJson(422, ROW_INVALID)

        model.save()
        reviewed()

        val row = rows()[0]
        assertEquals(RowStatus.Pending, row.status)
        assertEquals(
            mapOf(
                TransactionField.AMOUNT to "должно быть больше нуля",
                TransactionField.DESCRIPTION to "слишком коротко",
            ),
            row.fieldErrors,
        )

        model.onDescriptionChange(draft, "Шавуха большая")
        assertEquals(setOf(TransactionField.AMOUNT), rows()[0].fieldErrors.keys)
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
    fun waitingShareKeepsFailedRowsUntilRetried() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
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
        reviewed()
        val salary = rows()[1].draft
        model.onDateChange(salary, LocalDate.parse("2026-09-01"))
        server.enqueueJson(201, TRANSACTION_OK)
        server.enqueueJson(500, INTERNAL_ERROR)

        model.save()
        reviewed()
        assertTrue(rows()[1].status is RowStatus.Failed)
        assertNull(store.pending.value)
        assertTrue(store.waiting.value)

        server.enqueueJson(201, TRANSACTION_OK)
        model.retryRow(salary)
        reviewed()
        assertEquals(next, pending())
    }

    @Test
    fun waitingShareKeepsRowsRejectedUnderFields() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
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
        reviewed()
        val shawarma = rows()[0].draft
        server.enqueueJson(422, ROW_INVALID)

        model.save()
        reviewed()
        assertEquals(RowStatus.Pending, rows()[0].status)
        assertNull(store.pending.value)
        assertTrue(store.waiting.value)

        model.onDescriptionChange(shawarma, "Шавуха большая")
        server.enqueueJson(201, TRANSACTION_OK)
        model.save()
        reviewed()
        assertEquals(next, pending())
    }

    @Test
    fun clearingModelKeepsJournalAndReleasesHold() = runTest {
        recognized()
        assertTrue(File(importDir(), "journal.json").exists())

        owner.clear()

        assertTrue(File(importDir(), "journal.json").exists())
        val next = store.offer(emptyList())
        assertEquals(next, store.pending.value)
    }

    @Test
    fun abandonDeletesImportDirectory() = runTest {
        recognized()

        model.abandon()
        owner.clear()

        assertFalse(importDir().exists())
    }

    @Test
    fun savingEverySavableRowClosesJournalAndKeepsPreviews() = runTest {
        val (_, salary, foreign) = recognized().map { it.draft }
        model.onIncludedChange(salary, false)
        model.onIncludedChange(foreign, false)
        server.enqueueJson(201, TRANSACTION_OK)
        model.save()
        reviewed()

        assertTrue(File(importDir(), "journal.json").exists())

        model.onIncludedChange(salary, true)
        model.onDateChange(salary, LocalDate.parse("2026-09-01"))
        server.enqueueJson(201, TRANSACTION_OK)
        model.save()
        reviewed()

        assertFalse(File(importDir(), "journal.json").exists())
        assertTrue(File(importDir(), "0.jpg").exists())
    }

    @Test
    fun unwrittenCheckpointSkipsPaidCall() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        importId = store.offer(listOf(shot("a.png")))
        File(importDir(), "journal.json.tmp").mkdirs()
        model = newModel()

        val failed = reviewed().phase as RecognizePhase.Failure
        assertTrue(failed.retryable)
        assertEquals(UiError.Resource(R.string.recognize_error_journal), failed.error)
        assertEquals(listOf("/api/v1/categories", "/api/v1/accounts"), requests().map { it.url.encodedPath })
    }

    @Test
    fun emptyAnswerLeavesNoJournal() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        server.enqueueJson(200, RECOGNIZE_EMPTY)
        createModel()

        assertTrue(reviewed().phase is RecognizePhase.Review)
        assertFalse(File(importDir(), "journal.json").exists())
    }

    @Test
    fun importWithoutReadyImagesLeavesNoJournal() = runTest {
        createModel(listOf(junk()))

        val failed = reviewed().phase as RecognizePhase.Failure
        assertEquals(UiError.Resource(R.string.recognize_no_images), failed.error)
        assertFalse(File(importDir(), "journal.json").exists())
    }

    @Test
    fun newImportDeletesOtherJournals() = runTest {
        recognized()
        val first = importDir()
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        server.enqueueJson(200, RECOGNIZE_OK)

        createModel(listOf(shot("c.png")))
        reviewed()

        assertFalse(first.exists())
        assertTrue(File(importDir(), "journal.json").exists())
    }

    @Test
    fun deathBeforeCallRecognizesOnRestore() = runTest {
        server.enqueueJson(500, INTERNAL_ERROR)
        createModel()
        assertTrue(reviewed().phase is RecognizePhase.Failure)

        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        server.enqueueJson(200, RECOGNIZE_OK)
        restart()
        reviewed()

        assertEquals(3, rows().size)
        assertTrue(model.state.value.images[1] is ImportImage.Failed)
        assertEquals(listOf(0, 2, null), rows().map { it.image })
        assertEquals(
            listOf("/api/v1/categories", "/api/v1/categories", "/api/v1/accounts", RECOGNIZE_PATH),
            requests().map { it.url.encodedPath },
        )
    }

    @Test
    fun deathDuringCallIsRetriedOnlyOnTap() = runTest {
        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        server.enqueueJson(503, UNAVAILABLE)
        createModel()
        reviewed()

        restart()

        val failed = reviewed().phase as RecognizePhase.Failure
        assertTrue(failed.retryable)
        assertEquals(UiError.Resource(R.string.recognize_error_interrupted), failed.error)
        assertEquals(3, server.requestCount)
        assertNull(store.pending.value)

        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        server.enqueueJson(200, RECOGNIZE_OK)
        model.retry()
        reviewed()
        assertEquals(3, rows().size)
    }

    @Test
    fun restoredAnswerKeepsEditsWithoutPaidCall() = runTest {
        val (shawarma, salary) = recognized().map { it.draft }
        model.onDescriptionChange(shawarma, "Шавуха большая")
        model.onDateChange(salary, LocalDate.parse("2026-09-01"))
        model.onIncludedChange(salary, false)
        model.onAccountChange(UUID.fromString(CARD_ACCOUNT_ID))
        val before = rows()
        val served = server.requestCount

        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        restart()
        val state = reviewed()

        assertEquals(before, rows())
        assertTrue(state.incomplete)
        assertEquals(UUID.fromString(CARD_ACCOUNT_ID), state.accountId)
        assertEquals(
            listOf("/api/v1/categories", "/api/v1/accounts"),
            requests().drop(served).map { it.url.encodedPath },
        )
    }

    @Test
    fun restoredCatalogFailureRetriesWithoutPaidCall() = runTest {
        recognized()
        val served = server.requestCount

        server.enqueueJson(403, VALIDATION_ERROR)
        restart()
        val failed = reviewed().phase as RecognizePhase.Failure
        assertTrue(failed.retryable)

        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        model.retry()
        reviewed()

        assertEquals(3, rows().size)
        assertTrue(requests().drop(served).none { it.url.encodedPath == RECOGNIZE_PATH })
    }

    @Test
    fun restoredAccountThatIsGoneIsDropped() = runTest {
        recognized()
        model.onAccountChange(UUID.fromString(CARD_ACCOUNT_ID))

        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_EMPTY)
        restart()

        assertNull(reviewed().accountId)
    }

    @Test
    fun interruptedSaveResendsSameDraftAndTakesServerFields() = runTest {
        val draft = recognized()[0].draft
        server.enqueue(
            MockResponse.Builder()
                .body(TRANSACTION_OK)
                .setHeader("Content-Type", "application/json")
                .headersDelay(30, TimeUnit.SECONDS)
                .build(),
        )
        model.save()
        model.state.first { state ->
            (state.phase as RecognizePhase.Review).rows[0].status == RowStatus.Saving
        }
        awaitRequests(RECOGNIZED_REQUESTS + 1)

        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        restart()
        reviewed()
        assertEquals(RowStatus.Failed(UiError.Resource(R.string.recognize_row_interrupted)), rows()[0].status)
        assertEquals(draft, rows()[0].draft)

        server.enqueueJson(200, TRANSACTION_OK)
        model.retryRow(draft)
        reviewed()

        val row = rows()[0]
        assertEquals(RowStatus.Saved, row.status)
        assertEquals("Кофе", row.description)
        assertEquals(150000L, row.amountMinor)
        val posts = requests().filter { it.url.encodedPath == TRANSACTIONS_PATH }
        assertEquals(2, posts.size)
        assertTrue(posts.all { it.text().contains("\"id\":\"$draft\"") })
    }

    @Test
    fun restoredSavedRowKeepsServerFieldsAndLocksAccount() = runTest {
        val draft = recognized()[0].draft
        server.enqueueJson(200, TRANSACTION_OK)
        model.retryRow(draft)
        reviewed()

        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        restart()
        val state = reviewed()

        val row = rows()[0]
        assertEquals(RowStatus.Saved, row.status)
        assertEquals("Кофе", row.description)
        assertEquals(150000L, row.amountMinor)
        assertEquals(1, state.savedCount)
        assertTrue(state.accountLocked)
    }

    // `null` в журнале с ответом — выбор пользователя, а не повод взять прошлый счёт.
    @Test
    fun restoredNoAccountDoesNotFallBackToLast() = runTest {
        lastAccount.write(UUID.fromString(CARD_ACCOUNT_ID))
        recognized()
        model.onAccountChange(null)

        server.enqueueJson(200, CATEGORIES_OK)
        server.enqueueJson(200, ACCOUNTS_OK)
        restart()

        assertNull(reviewed().accountId)
    }

    @Test
    fun missingJournalIsLostImport() = runTest {
        importId = UUID.randomUUID()
        model = newModel()

        val failed = reviewed().phase as RecognizePhase.Failure
        assertEquals(UiError.Resource(R.string.recognize_import_lost), failed.error)
        assertEquals(0, server.requestCount)
        assertNull(store.pending.value)
    }

    @Test
    fun batchGoesWithLastAccountAndLocksIt() = runTest {
        lastAccount.write(UUID.fromString(CARD_ACCOUNT_ID))
        recognized()
        assertEquals(UUID.fromString(CARD_ACCOUNT_ID), model.state.value.accountId)

        server.enqueueJson(201, TRANSACTION_OK)
        model.save()
        reviewed()

        val posts = requests().filter { it.url.encodedPath == TRANSACTIONS_PATH }.map { it.text() }
        assertTrue(posts.isNotEmpty())
        assertTrue(posts.all { it.contains("\"account_id\":\"$CARD_ACCOUNT_ID\"") })
        model.onAccountChange(null)
        assertEquals(UUID.fromString(CARD_ACCOUNT_ID), model.state.value.accountId)
    }

    @Test
    fun chosenAccountIsRemembered() = runTest {
        recognized()
        assertNull(model.state.value.accountId)

        model.onAccountChange(UUID.fromString(CARD_ACCOUNT_ID))
        server.enqueueJson(201, TRANSACTION_OK)
        model.save()
        reviewed()

        assertEquals(UUID.fromString(CARD_ACCOUNT_ID), lastAccount.read())
    }

    // Только справочник: счёт уже выбран, а платный вызов не повторяется.
    @Test
    fun reloadCategoriesKeepsRowsAndAccountWithoutPaidCall() = runTest {
        val shawarma = recognized().first().draft
        model.onDescriptionChange(shawarma, "Шавуха большая")
        model.onAccountChange(UUID.fromString(CARD_ACCOUNT_ID))
        val before = rows()
        val served = server.requestCount

        server.enqueueJson(200, CATEGORIES_NESTED)
        model.reloadCategories().join()
        val state = model.state.value

        assertEquals(3, state.categories.size)
        assertEquals(before, rows())
        assertEquals(UUID.fromString(CARD_ACCOUNT_ID), state.accountId)
        assertEquals(listOf("/api/v1/categories"), requests().drop(served).map { it.url.encodedPath })
    }

    @Test
    fun reloadCategoriesFailureKeepsPreviousList() = runTest {
        recognized()

        server.enqueueJson(500, INTERNAL_ERROR)
        model.reloadCategories().join()

        assertEquals(2, model.state.value.categories.size)
        assertTrue(model.state.value.phase is RecognizePhase.Review)
    }
}

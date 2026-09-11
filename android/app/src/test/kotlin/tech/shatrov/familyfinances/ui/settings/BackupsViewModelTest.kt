package tech.shatrov.familyfinances.ui.settings

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
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
import tech.shatrov.familyfinances.BACKUPS_OK
import tech.shatrov.familyfinances.BACKUPS_TRUNCATED
import tech.shatrov.familyfinances.BACKUP_CREATED
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import java.time.ZoneId

private const val BACKUP_GONE = """
{"error":{"code":"NOT_FOUND","message":"backup not found"},
"meta":{"request_id":"r-52","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0"}}
"""

private const val NBSP = "\u00A0"

private const val OLD_BACKUP = "backup_20260910_100000123.db"

/** Бэкапы: одна страница, создание с неизвестным исходом и удаление файла. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class BackupsViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: BackupsViewModel

    @Before
    fun start() {
        Dispatchers.setMain(UnconfinedTestDispatcher())
        server = MockWebServer()
        server.start()
    }

    @After
    fun stop() {
        server.close()
        Dispatchers.resetMain()
    }

    private fun create() {
        model = BackupsViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            ZoneId.of("Europe/Moscow"),
        )
    }

    private suspend fun ready(): BackupsUiState.Ready =
        model.state.first { it is BackupsUiState.Ready && !it.busy } as BackupsUiState.Ready

    @Test
    fun listIsAskedAsOnePageAndFormattedInFamilyZone() = runTest {
        server.enqueueJson(200, BACKUPS_OK)
        create()
        val state = ready()

        val request = server.takeRequest()
        assertEquals("/api/v1/backups", request.url.encodedPath)
        assertEquals("200", request.url.queryParameter("limit"))
        assertEquals(2, state.rows.size)
        assertFalse(state.truncated)

        val newest = state.rows.first()
        assertEquals("backup_20260911_100000123.db", newest.name)
        assertEquals("12,3${NBSP}МБ", newest.size)
        assertEquals("11 сентября 2026, 13:00", newest.created)
    }

    @Test
    fun truncationIsVisibleByTotal() = runTest {
        server.enqueueJson(200, BACKUPS_TRUNCATED)
        create()

        assertTrue(ready().truncated)
    }

    @Test
    fun createSendsPostAndRereadsTheList() = runTest {
        server.enqueueJson(200, BACKUPS_OK)
        create()
        ready()

        server.enqueueJson(201, BACKUP_CREATED)
        server.enqueueJson(200, BACKUPS_OK)
        model.onCreate()
        val state = ready()

        assertNull(state.error)
        server.takeRequest()
        val created = server.takeRequest()
        assertEquals("POST", created.method)
        assertEquals("/api/v1/backups", created.url.encodedPath)
        assertEquals("GET", server.takeRequest().method)
    }

    /** Файл мог появиться — повтор только руками, поэтому список перечитывается с текстом. */
    @Test
    fun brokenCreateAnswerKeepsUnknownResultOverTheRereadList() = runTest {
        server.enqueueJson(200, BACKUPS_OK)
        create()
        ready()

        server.enqueueJson(200, "<html>прокси съел ответ</html>")
        server.enqueueJson(200, BACKUPS_OK)
        model.onCreate()
        val state = ready()

        assertEquals(UiError.Resource(R.string.settings_backup_unknown), state.error)
        assertEquals(3, server.requestCount)
    }

    @Test
    fun deleteSendsRequestAndRereadsTheList() = runTest {
        server.enqueueJson(200, BACKUPS_OK)
        create()
        ready()

        server.enqueue(MockResponse.Builder().code(204).build())
        server.enqueueJson(200, BACKUPS_OK)
        model.onDelete(OLD_BACKUP)
        ready()

        server.takeRequest()
        val deleted = server.takeRequest()
        assertEquals("DELETE", deleted.method)
        assertEquals("/api/v1/backups/$OLD_BACKUP", deleted.url.encodedPath)
        assertEquals("GET", server.takeRequest().method)
    }

    @Test
    fun goneBackupOnlyRereadsTheList() = runTest {
        server.enqueueJson(200, BACKUPS_OK)
        create()
        ready()

        server.enqueueJson(404, BACKUP_GONE)
        server.enqueueJson(200, BACKUPS_OK)
        model.onDelete(OLD_BACKUP)
        val state = ready()

        assertNull(state.error)
        assertEquals(3, server.requestCount)
    }
}

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
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.SESSIONS_OK
import tech.shatrov.familyfinances.SESSIONS_TRUNCATED
import tech.shatrov.familyfinances.SESSION_CURRENT_ID
import tech.shatrov.familyfinances.SESSION_OTHER_ID
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import java.time.ZoneId
import java.util.UUID

private const val SESSION_GONE = """
{"error":{"code":"SESSION_NOT_FOUND","message":"session not found"},
"meta":{"request_id":"r-41","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0"}}
"""

/** Сессии: одна страница, отзыв чужой и исчезнувшая сессия как повод перечитать список. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class SessionsViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: SessionsViewModel

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
        model = SessionsViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            ZoneId.of("Europe/Moscow"),
        )
    }

    private suspend fun ready(): SessionsUiState.Ready =
        model.state.first { it is SessionsUiState.Ready && it.revoking == null } as SessionsUiState.Ready

    @Test
    fun listIsAskedAsOnePageAndFormattedInFamilyZone() = runTest {
        server.enqueueJson(200, SESSIONS_OK)
        create()
        val state = ready()

        val request = server.takeRequest()
        assertEquals("/api/v1/auth/sessions", request.url.encodedPath)
        assertEquals("200", request.url.queryParameter("limit"))
        assertEquals(2, state.rows.size)
        assertFalse(state.truncated)

        val current = state.rows.first()
        assertTrue(current.current)
        assertEquals("Pixel 8", current.deviceName)
        assertEquals("10 сентября 2026, 10:30", current.created)
        assertEquals("11 сентября 2026, 12:00", current.lastUsed)
        // Имени у старой сессии нет: подстановка «Без имени» — забота экрана.
        assertNull(state.rows[1].deviceName)
    }

    @Test
    fun truncationIsVisibleByTotal() = runTest {
        server.enqueueJson(200, SESSIONS_TRUNCATED)
        create()

        assertTrue(ready().truncated)
    }

    @Test
    fun revokeSendsDeleteAndRereadsTheList() = runTest {
        server.enqueueJson(200, SESSIONS_OK)
        create()
        ready()

        server.enqueue(MockResponse.Builder().code(204).build())
        server.enqueueJson(200, SESSIONS_OK)
        model.onRevoke(UUID.fromString(SESSION_OTHER_ID))
        ready()

        server.takeRequest()
        val deleted = server.takeRequest()
        assertEquals("DELETE", deleted.method)
        assertEquals("/api/v1/auth/sessions/$SESSION_OTHER_ID", deleted.url.encodedPath)
        assertEquals("GET", server.takeRequest().method)
    }

    @Test
    fun goneSessionOnlyRereadsTheList() = runTest {
        server.enqueueJson(200, SESSIONS_OK)
        create()
        ready()

        server.enqueueJson(404, SESSION_GONE)
        server.enqueueJson(200, SESSIONS_OK)
        model.onRevoke(UUID.fromString(SESSION_OTHER_ID))
        val state = ready()

        assertNull(state.error)
        assertEquals(3, server.requestCount)
    }

    @Test
    fun currentSessionIsNotRevocable() = runTest {
        server.enqueueJson(200, SESSIONS_OK)
        create()
        ready()

        model.onRevoke(UUID.fromString(SESSION_CURRENT_ID))

        assertEquals(1, server.requestCount)
        assertFalse(model.state.value.busy)
    }
}

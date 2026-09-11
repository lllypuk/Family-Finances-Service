package tech.shatrov.familyfinances.ui.settings

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import mockwebserver3.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.ADMIN_ID
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.USERS_OK
import tech.shatrov.familyfinances.USERS_TRUNCATED
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import java.util.UUID

/** Список пользователей: одна страница, своя запись помечена, усечение видно по `total`. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class UsersViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: UsersViewModel

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
        model = UsersViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            UUID.fromString(ADMIN_ID),
        )
    }

    private suspend fun ready(): UsersUiState.Ready =
        model.state.first { it is UsersUiState.Ready } as UsersUiState.Ready

    @Test
    fun listIsAskedAsOnePageAndMarksSelf() = runTest {
        server.enqueueJson(200, USERS_OK)
        create()
        val state = ready()

        val request = server.takeRequest()
        assertEquals("/api/v1/users", request.url.encodedPath)
        assertEquals("200", request.url.queryParameter("limit"))
        assertEquals(2, state.rows.size)
        assertFalse(state.truncated)

        val self = state.rows.first()
        assertTrue(self.self)
        assertTrue(self.admin)
        assertTrue(self.active)
        assertEquals("Админ Тест", self.name)
        assertEquals("admin@test.com", self.email)
        assertFalse(state.rows[1].self)
        assertFalse(state.rows[1].admin)
    }

    @Test
    fun truncationIsVisibleByTotal() = runTest {
        server.enqueueJson(200, USERS_TRUNCATED)
        create()

        assertTrue(ready().truncated)
    }
}

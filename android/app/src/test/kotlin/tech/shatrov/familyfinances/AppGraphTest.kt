package tech.shatrov.familyfinances

import kotlinx.coroutines.test.runTest
import mockwebserver3.Dispatcher
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import mockwebserver3.RecordedRequest
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import tech.shatrov.familyfinances.core.api.ApiGraph

/** Перечитка сессии из настроек и точечные обновления без второго бутстрапа. */
class AppGraphTest {
    private lateinit var server: MockWebServer
    private lateinit var graph: AppGraph

    @Before
    fun start() {
        server = MockWebServer()
        server.start()
        graph = AppGraph(ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())))
    }

    @After
    fun stop() {
        server.close()
    }

    private fun json(body: String): MockResponse = MockResponse.Builder()
        .code(200)
        .body(body)
        .setHeader("Content-Type", "application/json")
        .build()

    private suspend fun bootstrap() {
        server.enqueueJson(200, ME_OK)
        server.enqueueJson(200, FAMILY_OK)
        assertNull(graph.bootstrap())
    }

    // Обрыв или отказ сервера не повод показать настройки без семьи и роли.
    @Test
    fun refreshFailureKeepsSession() = runTest {
        bootstrap()
        val before = graph.session.value
        server.enqueueJson(500, INTERNAL_ERROR)

        assertNotNull(graph.refreshSession())

        assertEquals(before, graph.session.value)
    }

    // Пока перечитка в пути, сессию мог опубликовать кто-то ещё: её ответ уже устарел.
    @Test
    fun lateRefreshKeepsNewerPublication() = runTest {
        var onFamily: (() -> Unit)? = null
        server.dispatcher = object : Dispatcher() {
            override fun dispatch(request: RecordedRequest): MockResponse =
                if (request.url.encodedPath.endsWith("/family")) {
                    onFamily?.invoke()
                    json(FAMILY_OK)
                } else {
                    json(ME_OK)
                }
        }
        assertNull(graph.bootstrap())
        val renamed = requireNotNull(graph.session.value).user.copy(firstName = "Новый")
        onFamily = { graph.update(renamed) }

        assertNull(graph.refreshSession())

        assertEquals("Новый", graph.session.value?.user?.firstName)
    }

    @Test
    fun refreshPublishesFreshSession() = runTest {
        bootstrap()
        server.enqueueJson(200, MEMBER_OK)
        server.enqueueJson(200, FAMILY_OK)

        assertNull(graph.refreshSession())

        assertEquals("member@test.com", graph.session.value?.user?.email)
    }

    @Test
    fun updateReplacesUserOnly() = runTest {
        bootstrap()
        val before = requireNotNull(graph.session.value)

        graph.update(before.user.copy(email = "new@test.com"))

        val after = requireNotNull(graph.session.value)
        assertEquals("new@test.com", after.user.email)
        assertEquals(before.family, after.family)
    }

    @Test
    fun updateReplacesFamilyOnly() = runTest {
        bootstrap()
        val before = requireNotNull(graph.session.value)

        graph.update(before.family.copy(timezone = "Asia/Tbilisi"))

        val after = requireNotNull(graph.session.value)
        assertEquals("Asia/Tbilisi", after.family.timezone)
        assertEquals(before.user, after.user)
    }

    // Без сессии обновлять нечего: ответ пришёл уже после выхода.
    @Test
    fun updateWithoutSessionIsIgnored() = runTest {
        bootstrap()
        val user = requireNotNull(graph.session.value).user
        server.enqueue(MockResponse.Builder().code(204).build())
        graph.signOut()

        graph.update(user)

        assertNull(graph.session.value)
    }
}

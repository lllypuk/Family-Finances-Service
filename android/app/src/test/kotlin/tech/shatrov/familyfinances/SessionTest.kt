package tech.shatrov.familyfinances

import kotlinx.coroutines.test.runTest
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.auth.SessionToken
import java.time.OffsetDateTime

/** Бутстрап сессии: роль и валюта для остальных экранов и решение «вход или главная». */
class SessionTest {
    private lateinit var server: MockWebServer
    private lateinit var vault: FakeTokenVault
    private lateinit var graph: AppGraph

    @Before
    fun start() {
        server = MockWebServer()
        server.start()
        vault = FakeTokenVault(liveToken())
        graph = AppGraph(ApiGraph(server.url("/").toString(), vault))
    }

    @After
    fun stop() {
        server.close()
    }

    @Test
    fun bootstrapKeepsRoleAndCurrency() = runTest {
        server.enqueueJson(200, ME_OK)
        server.enqueueJson(200, FAMILY_OK)

        assertTrue(graph.bootstrap())

        val session = requireNotNull(graph.session.value)
        assertEquals("RUB", session.currency)
        assertTrue(session.isAdmin)
        assertEquals("/api/v1/me", server.takeRequest().url.encodedPath)
        assertEquals("/api/v1/family", server.takeRequest().url.encodedPath)
    }

    @Test
    fun memberIsNotAdmin() = runTest {
        server.enqueueJson(200, MEMBER_OK)
        server.enqueueJson(200, FAMILY_OK)

        assertTrue(graph.bootstrap())

        assertFalse(requireNotNull(graph.session.value).isAdmin)
    }

    // Токен протух на сервере: интерцептор чистит хранилище, корень уводит на вход.
    @Test
    fun unauthorizedBootstrapReturnsToLogin() = runTest {
        server.enqueueJson(401, """{"error":{"code":"UNAUTHORIZED","message":"токен истёк"}}""")

        assertFalse(graph.bootstrap())

        assertNull(graph.session.value)
        assertNull(vault.read())
    }

    @Test
    fun networkFailureReturnsToLogin() = runTest {
        server.close()

        assertFalse(graph.bootstrap())

        assertNull(graph.session.value)
    }

    // Просроченный токен на старте: запроса не будет, сразу экран входа.
    @Test
    fun expiredTokenIsNotLive() {
        vault.write(SessionToken("t-old", OffsetDateTime.now().minusMinutes(1)))

        assertFalse(graph.hasLiveToken())
    }

    @Test
    fun liveTokenOpensHome() {
        assertTrue(graph.hasLiveToken())
    }

    @Test
    fun emptyVaultIsNotLive() {
        vault.clear()

        assertFalse(graph.hasLiveToken())
    }

    @Test
    fun signOutRevokesSessionAndClearsVault() = runTest {
        server.enqueueJson(200, ME_OK)
        server.enqueueJson(200, FAMILY_OK)
        graph.bootstrap()
        server.enqueue(MockResponse.Builder().code(204).build())

        graph.signOut()

        repeat(2) { server.takeRequest() }
        assertEquals("/api/v1/auth/logout", server.takeRequest().url.encodedPath)
        assertNull(vault.read())
        assertNull(graph.session.value)
    }

    // Отказ сервера не повод оставить токен на телефоне: выход всё равно состоялся.
    @Test
    fun signOutClearsVaultWhenServerUnreachable() = runTest {
        server.close()

        graph.signOut()

        assertNull(vault.read())
    }
}

package tech.shatrov.familyfinances.core.api

import kotlinx.coroutines.test.runTest
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import tech.shatrov.familyfinances.core.api.auth.InMemoryTokenVault

private const val RECOGNIZE_OK = """
{"data":{"items":[],"incomplete":false,"model":"gemma4:31b"},
"meta":{"request_id":"r-1","timestamp":"2026-09-16T10:00:00Z"}}
"""

class ApiGraphTest {
    @get:Rule
    val folder = TemporaryFolder()

    private lateinit var server: MockWebServer
    private lateinit var graph: ApiGraph

    @Before
    fun start() {
        server = MockWebServer()
        server.start()
        graph = ApiGraph(server.url("/").toString(), InMemoryTokenVault())
    }

    @After
    fun stop() {
        server.close()
    }

    @Test
    fun recognizeSendsEveryFileAsImagesPart() = runTest {
        server.enqueue(
            MockResponse.Builder()
                .body(RECOGNIZE_OK)
                .setHeader("Content-Type", "application/json")
                .build(),
        )
        val first = folder.newFile("a.jpg").apply { writeText("первая картинка") }
        val second = folder.newFile("b.jpg").apply { writeText("вторая картинка") }

        val result = graph.recognize(listOf(first, second))

        val request = server.takeRequest()
        val body = request.body!!.utf8()
        assertEquals("/api/v1/transactions/recognize", request.url.encodedPath)
        assertTrue(request.headers["Content-Type"]!!.startsWith("multipart/form-data"))
        assertTrue(body.contains("Content-Disposition: form-data; name=\"images\"; filename=\"a.jpg\""))
        assertTrue(body.contains("Content-Disposition: form-data; name=\"images\"; filename=\"b.jpg\""))
        assertEquals(2, Regex("Content-Type: image/jpeg").findAll(body).count())
        assertTrue(body.contains("первая картинка"))
        assertTrue(body.contains("вторая картинка"))
        assertEquals("gemma4:31b", result.`data`.model)
    }
}

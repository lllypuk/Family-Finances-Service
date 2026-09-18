package tech.shatrov.familyfinances.ui.networth

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
import tech.shatrov.familyfinances.FLAT_ID
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.HOLDINGS_OK
import tech.shatrov.familyfinances.HOLDING_NAME_EXISTS_ERROR
import tech.shatrov.familyfinances.HOLDING_OK
import tech.shatrov.familyfinances.HOLDING_VALUE_OK
import tech.shatrov.familyfinances.MORTGAGE_ID
import tech.shatrov.familyfinances.OLD_CAR_ID
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.HoldingSide
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import java.time.LocalDate
import java.util.UUID

/** Форма позиции: создание с клиентским `id`, правка diff-ом, архив с нулевым снимком, удаление. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class HoldingEditViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: HoldingEditViewModel
    private val today = LocalDate.of(2026, 9, 18)
    private val draft = UUID.fromString("cccccccc-cccc-cccc-cccc-cccccccccccc")

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

    private fun createModel(
        id: String? = null,
        side: HoldingSide = HoldingSide.asset,
        isAdmin: Boolean = true,
    ) {
        model = HoldingEditViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            id?.let(UUID::fromString),
            draft,
            side,
            { today },
            isAdmin,
        )
    }

    private suspend fun open(id: String): HoldingEditUiState {
        server.enqueueJson(200, HOLDINGS_OK)
        createModel(id)
        return model.state.first { !it.loading }.also { server.takeRequest() }
    }

    private suspend fun settled(): HoldingEditUiState = model.state.first { !it.submitting }

    @Test
    fun createSendsDraftSideAndKind() = runTest {
        createModel(side = HoldingSide.liability)
        assertEquals("mortgage", model.state.value.kind)

        model.onNameChange("  Ипотека ")
        model.onKindChange(HoldingKind.LOAN)
        server.enqueueJson(201, HOLDING_OK)
        model.onSubmit()

        assertTrue(settled().done)
        val request = server.takeRequest()
        assertEquals("POST", request.method)
        val body = request.body?.utf8().orEmpty()
        assertTrue(body, body.contains("\"id\":\"$draft\""))
        assertTrue(body, body.contains("\"name\":\"Ипотека\""))
        assertTrue(body, body.contains("\"side\":\"liability\""))
        assertTrue(body, body.contains("\"kind\":\"loan\""))
    }

    @Test
    fun sideSwitchResetsKindToItsList() = runTest {
        createModel()
        model.onKindChange(HoldingKind.PROPERTY)

        model.onSideChange(HoldingSide.liability)

        assertEquals(HoldingSide.liability, model.state.value.side)
        assertEquals("mortgage", model.state.value.kind)
    }

    @Test
    fun nameTakenIsTranslated() = runTest {
        createModel()
        model.onNameChange("Квартира")
        server.enqueueJson(409, HOLDING_NAME_EXISTS_ERROR)

        model.onSubmit()

        val state = settled()
        assertFalse(state.done)
        assertEquals(UiError.Resource(R.string.holding_error_name_exists), state.error)
    }

    @Test
    fun editFindsHoldingInListAndSendsOnlyChanges() = runTest {
        val state = open(MORTGAGE_ID)
        assertEquals("Ипотека", state.name)
        assertEquals(HoldingSide.liability, state.side)
        assertFalse(state.canSubmit)

        model.onNameChange("Ипотека ВТБ")
        server.enqueueJson(200, HOLDING_OK)
        model.onSubmit()

        assertTrue(settled().done)
        val request = server.takeRequest()
        assertEquals("PUT", request.method)
        assertEquals("""{"name":"Ипотека ВТБ"}""", request.body?.utf8())
    }

    @Test
    fun sideCannotChangeOnExistingHolding() = runTest {
        open(FLAT_ID)

        model.onSideChange(HoldingSide.liability)

        assertEquals(HoldingSide.asset, model.state.value.side)
    }

    @Test
    fun archiveWithValueWritesZeroSnapshotFirst() = runTest {
        assertTrue(open(FLAT_ID).archiveNeedsZero)
        server.enqueueJson(200, HOLDING_VALUE_OK)
        server.enqueueJson(200, HOLDING_OK)

        model.onToggleArchive(zeroFirst = true)

        assertTrue(settled().done)
        val zero = server.takeRequest()
        assertEquals("PUT", zero.method)
        assertEquals("/api/v1/holdings/$FLAT_ID/values/2026-09-18", zero.url.encodedPath)
        assertEquals("""{"value_minor":0}""", zero.body?.utf8())
        val archive = server.takeRequest()
        assertEquals("/api/v1/holdings/$FLAT_ID", archive.url.encodedPath)
        assertEquals("""{"is_archived":true}""", archive.body?.utf8())
    }

    @Test
    fun archiveFailureAfterZeroSnapshotMarksChanged() = runTest {
        open(FLAT_ID)
        server.enqueueJson(200, HOLDING_VALUE_OK)
        server.enqueueJson(409, HOLDING_NAME_EXISTS_ERROR)

        model.onToggleArchive(zeroFirst = true)

        val state = settled()
        assertFalse(state.done)
        assertTrue(state.changed)
    }

    @Test
    fun archiveOnlySkipsSnapshot() = runTest {
        open(FLAT_ID)
        server.enqueueJson(200, HOLDING_OK)

        model.onToggleArchive(zeroFirst = false)

        assertTrue(settled().done)
        assertEquals("/api/v1/holdings/$FLAT_ID", server.takeRequest().url.encodedPath)
        assertEquals(2, server.requestCount)
    }

    @Test
    fun archivedWithZeroNeedsNoSnapshotAndUnarchives() = runTest {
        val state = open(OLD_CAR_ID)
        assertFalse(state.archiveNeedsZero)
        server.enqueueJson(200, HOLDING_OK)

        model.onToggleArchive(zeroFirst = true)

        assertTrue(settled().done)
        val request = server.takeRequest()
        assertEquals("""{"is_archived":false}""", request.body?.utf8())
        assertEquals(2, server.requestCount)
    }

    @Test
    fun deleteIsAdminOnly() = runTest {
        server.enqueueJson(200, HOLDINGS_OK)
        createModel(FLAT_ID, isAdmin = false)
        model.state.first { !it.loading }

        model.onDelete()

        assertFalse(model.state.value.canDelete)
        assertEquals(1, server.requestCount)
    }

    @Test
    fun holdingGoneFromListClosesForm() = runTest {
        server.enqueueJson(200, HOLDINGS_OK)
        createModel("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb9")

        assertTrue(model.state.first { !it.loading }.done)
    }
}

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
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.ADMIN_ID
import tech.shatrov.familyfinances.FORBIDDEN_ERROR
import tech.shatrov.familyfinances.FakeTokenVault
import tech.shatrov.familyfinances.MEMBER_ID
import tech.shatrov.familyfinances.MEMBER_OK
import tech.shatrov.familyfinances.ME_OK
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.USER_ADMIN_OK
import tech.shatrov.familyfinances.USER_INACTIVE_OK
import tech.shatrov.familyfinances.core.api.ApiGraph
import tech.shatrov.familyfinances.core.api.Role
import tech.shatrov.familyfinances.enqueueJson
import tech.shatrov.familyfinances.liveToken
import tech.shatrov.familyfinances.ui.UiError
import java.util.UUID

private const val LAST_ADMIN = """
{"error":{"code":"LAST_ADMIN","message":"last active admin"},
"meta":{"request_id":"r-50","timestamp":"2026-09-11T10:00:00Z","version":"v0.1.0"}}
"""

/** Понижение админа до участника: ответ `PATCH` по своей записи. */
private val SELF_MEMBER_OK = ME_OK.replace(""""role":"admin"""", """"role":"member"""")

/** Форма пользователя: `PATCH` с одним полем, перевод `409` и перечитка после ответа. */
@OptIn(ExperimentalCoroutinesApi::class)
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class UserEditViewModelTest {
    private lateinit var server: MockWebServer
    private lateinit var model: UserEditViewModel

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

    private fun create(
        id: String?,
        selfId: String = ADMIN_ID,
    ) {
        model = UserEditViewModel(
            ApiGraph(server.url("/").toString(), FakeTokenVault(liveToken())),
            id?.let(UUID::fromString),
            UUID.fromString(selfId),
        )
    }

    private suspend fun settled(): UserEditUiState = model.state.first { !it.loading && !it.submitting }

    @Test
    fun roleGoesInItsOwnPatch() = runTest {
        server.enqueueJson(200, MEMBER_OK)
        create(MEMBER_ID)
        settled()
        server.takeRequest()

        server.enqueueJson(200, USER_ADMIN_OK)
        server.enqueueJson(200, USER_ADMIN_OK)
        model.onToggleRole()
        val state = settled()

        val patch = server.takeRequest()
        assertEquals("PATCH", patch.method)
        assertEquals("/api/v1/users/$MEMBER_ID", patch.url.encodedPath)
        assertEquals("""{"role":"admin"}""", patch.body?.utf8().orEmpty())
        // После ответа запись перечитывается: активность мог изменить второй шаг того же запроса.
        assertEquals("GET", server.takeRequest().method)
        assertEquals(Role.admin, state.currentRole)
        assertEquals(Role.admin, state.saved?.role)
    }

    @Test
    fun lastAdminConflictIsTranslatedByCode() = runTest {
        server.enqueueJson(200, MEMBER_OK)
        create(MEMBER_ID)
        settled()
        server.takeRequest()

        server.enqueueJson(409, LAST_ADMIN)
        server.enqueueJson(200, MEMBER_OK)
        model.onToggleRole()

        assertEquals(UiError.Resource(R.string.settings_error_last_admin), settled().error)
    }

    @Test
    fun failedDeactivationRereadsTheUser() = runTest {
        server.enqueueJson(200, MEMBER_OK)
        create(MEMBER_ID)
        settled()
        server.takeRequest()

        server.enqueueJson(409, LAST_ADMIN)
        // Отказ описывает не применённый шаг, а запись на сервере уже другая.
        server.enqueueJson(200, USER_INACTIVE_OK)
        model.onToggleActive()
        val state = model.state.first { !it.submitting && !it.active }

        assertEquals("""{"is_active":false}""", server.takeRequest().body?.utf8().orEmpty())
        assertEquals("GET", server.takeRequest().method)
        assertFalse(state.active)
        assertNotNull(state.error)
    }

    @Test
    fun ownDowngradeLeavesForTheRoot() = runTest {
        server.enqueueJson(200, ME_OK)
        create(ADMIN_ID)
        settled()
        server.takeRequest()

        server.enqueueJson(200, SELF_MEMBER_OK)
        model.onToggleRole()
        val state = settled()

        assertEquals("""{"role":"member"}""", server.takeRequest().body?.utf8().orEmpty())
        // Перечитки нет: `/users` этому токену уже не отвечает.
        assertEquals(2, server.requestCount)
        assertEquals(UserEditExit.Root, state.exit)
        assertEquals(Role.member, state.saved?.role)
    }

    @Test
    fun ownRecordHidesDeactivation() = runTest {
        server.enqueueJson(200, ME_OK)
        create(ADMIN_ID)
        val state = settled()

        assertTrue(state.self)
        assertFalse(state.ownActions)
    }

    /** Форма разблокируется только после перечитки: иначе она затёрла бы уже начатый ввод. */
    @Test
    fun saveSendsOnlyTheChangedFieldAndRereads() = runTest {
        server.enqueueJson(200, MEMBER_OK)
        create(MEMBER_ID)
        settled()
        server.takeRequest()

        model.onFirstNameChange(" Участник ")
        assertTrue(model.state.value.canSubmit)

        val renamed = MEMBER_OK.replace(""""first_name":"Член"""", """"first_name":"Участник"""")
        server.enqueueJson(200, renamed)
        server.enqueueJson(200, renamed)
        model.onSubmit()
        val state = settled()

        val put = server.takeRequest()
        assertEquals("PUT", put.method)
        assertEquals("/api/v1/users/$MEMBER_ID", put.url.encodedPath)
        assertEquals("""{"first_name":"Участник"}""", put.body?.utf8().orEmpty())
        assertEquals("GET", server.takeRequest().method)
        assertEquals("Участник", state.firstName)
        assertNull(state.changes)
    }

    /** `PATCH` роли пишет не поля формы: начатый ввод он стирать не должен. */
    @Test
    fun roleChangeKeepsTheUnsavedInput() = runTest {
        server.enqueueJson(200, MEMBER_OK)
        create(MEMBER_ID)
        settled()
        server.takeRequest()

        model.onFirstNameChange("Участник")
        server.enqueueJson(200, USER_ADMIN_OK)
        server.enqueueJson(200, USER_ADMIN_OK)
        model.onToggleRole()
        val state = settled()

        assertEquals("Участник", state.firstName)
        assertEquals(Role.admin, state.currentRole)
        assertTrue(state.canSubmit)
    }

    /** Роль сняли с другого телефона: `/users` этому токену уже не отвечает — уходим в корень. */
    @Test
    fun forbiddenLeavesForTheRoot() = runTest {
        server.enqueueJson(403, FORBIDDEN_ERROR)
        create(MEMBER_ID)
        val state = model.state.first { !it.loading }

        assertEquals(UserEditExit.Root, state.exit)
        assertEquals(UiError.Resource(R.string.settings_forbidden), state.loadError)
    }

    @Test
    fun creationSendsTheWholeBody() = runTest {
        create(null)
        model.onEmailChange(" new@test.com ")
        model.onFirstNameChange("Новый")
        model.onLastNameChange("Пользователь")
        model.onPasswordChange("short")
        assertFalse(model.state.value.canSubmit)

        model.onPasswordChange("Password12")
        assertTrue(model.state.value.canSubmit)

        server.enqueueJson(201, MEMBER_OK)
        model.onSubmit()
        val state = settled()

        val request = server.takeRequest()
        assertEquals("POST", request.method)
        assertEquals("/api/v1/users", request.url.encodedPath)
        assertEquals(
            """{"email":"new@test.com","password":"Password12","first_name":"Новый",""" +
                """"last_name":"Пользователь","role":"member"}""",
            request.body?.utf8().orEmpty(),
        )
        assertEquals(UserEditExit.List, state.exit)
        assertNull(state.loadError)
    }
}

package tech.shatrov.familyfinances.ui.settings

import android.content.Context
import androidx.compose.ui.test.hasAnyAncestor
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.isDialog
import androidx.compose.ui.test.junit4.v2.createComposeRule
import androidx.compose.ui.test.onAllNodesWithText
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.performClick
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import tech.shatrov.familyfinances.SESSION_OTHER_ID
import tech.shatrov.familyfinances.theme.AppTheme
import java.util.UUID

private val CURRENT = SessionRow(
    id = UUID.randomUUID(),
    deviceName = "Pixel 8",
    created = "10 сентября 2026, 10:30",
    lastUsed = "11 сентября 2026, 12:00",
    current = true,
)

private val OTHER = SessionRow(
    id = UUID.fromString(SESSION_OTHER_ID),
    deviceName = null,
    created = "1 августа 2026, 15:00",
    lastUsed = "1 сентября 2026, 15:00",
    current = false,
)

/** Сессии: отзыв необратим, поэтому идёт через подтверждение, а у текущей его нет вовсе. */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class SessionsScreenTest {
    @get:Rule
    val composeRule = createComposeRule()

    private val res = ApplicationProvider.getApplicationContext<Context>().resources

    private fun show(
        state: SessionsUiState = SessionsUiState.Ready(rows = listOf(CURRENT, OTHER), total = 2),
        onRevoke: (UUID) -> Unit = {},
    ) {
        composeRule.setContent {
            AppTheme {
                SessionsScreen(state = state, onRetry = {}, onRevoke = onRevoke, onBack = {})
            }
        }
    }

    @Test
    fun revokeWaitsForConfirmation() {
        var revoked: UUID? = null
        show(onRevoke = { revoked = it })

        // Кнопка отзыва одна: у текущей сессии её нет, поэтому ошибиться строкой нельзя.
        composeRule.onNodeWithText(res.getString(R.string.settings_session_revoke)).performClick()
        assertNull(revoked)

        composeRule.onNodeWithText(res.getString(R.string.settings_session_revoke_confirm)).assertExists()
        val confirm = hasText(res.getString(R.string.settings_session_revoke)) and hasAnyAncestor(isDialog())
        composeRule.onNode(confirm).performClick()
        assertEquals(OTHER.id, revoked)
    }

    @Test
    fun currentSessionHasNoRevokeButton() {
        show(SessionsUiState.Ready(rows = listOf(CURRENT), total = 1))

        composeRule.onNodeWithText(res.getString(R.string.settings_session_current)).assertExists()
        assertEquals(
            0,
            composeRule.onAllNodesWithText(res.getString(R.string.settings_session_revoke)).fetchSemanticsNodes().size,
        )
    }

    @Test
    fun truncationIsShown() {
        show(SessionsUiState.Ready(rows = listOf(CURRENT), total = 5))

        composeRule.onNodeWithText(res.getString(R.string.settings_list_truncated, 1, 5)).assertExists()
    }
}

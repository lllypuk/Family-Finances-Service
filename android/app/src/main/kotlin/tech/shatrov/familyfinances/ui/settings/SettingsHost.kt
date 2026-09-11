package tech.shatrov.familyfinances.ui.settings

import androidx.activity.compose.BackHandler
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.res.stringResource
import androidx.lifecycle.ViewModelStoreOwner
import tech.shatrov.familyfinances.AppGraph
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.Session
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.toUiError

/**
 * Хост настроек: свой `when` по [SettingsPage] и своя «назад». `AppRoot` знает только
 * `AppScreen.Settings` и живёт без девяти веток.
 *
 * @param models store моделей подразделов; чистит его `AppRoot` на каждом переходе.
 * @param onSignedOut «Выйти» подтверждён: сам выход делает `AppRoot` — его scope переживает
 *   смену экрана, а этот умрёт вместе с хостом.
 */
@Composable
fun SettingsHost(
    graph: AppGraph,
    session: Session,
    models: ViewModelStoreOwner,
    page: SettingsPage,
    onPageChange: (SettingsPage) -> Unit,
    onLeave: () -> Unit,
    onSessionChanged: (Session, Session) -> Unit,
    onUsersChanged: () -> Unit,
    onSignedOut: () -> Unit,
) {
    // Пока страница отправляет мутацию, уходить нельзя: очистка store отменила бы корутину,
    // а сервер запись уже мог принять.
    var submitting by remember(page) { mutableStateOf(false) }
    val leave = {
        if (!submitting) {
            val parent = page.parent()
            if (parent == null) onLeave() else onPageChange(parent)
        }
    }

    BackHandler { leave() }

    when (page) {
        is SettingsPage.Root -> SettingsRootPage(
            graph = graph,
            session = session,
            page = page,
            onOpen = onPageChange,
            onSessionChanged = onSessionChanged,
            onBack = leave,
            onSignedOut = onSignedOut,
        )

        else -> Centered {
            Text(page.slug)
            TextButton(onClick = leave) { Text(stringResource(R.string.back)) }
        }
    }
}

@Composable
private fun SettingsRootPage(
    graph: AppGraph,
    session: Session,
    page: SettingsPage.Root,
    onOpen: (SettingsPage) -> Unit,
    onSessionChanged: (Session, Session) -> Unit,
    onBack: () -> Unit,
    onSignedOut: () -> Unit,
) {
    var refreshing by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<UiError?>(null) }
    var attempt by remember { mutableIntStateOf(0) }

    // Роль, валюту и зону мог сменить второй телефон; отказ оставляет прежнюю сессию на экране.
    LaunchedEffect(page.visit, attempt) {
        refreshing = true
        error = null
        val before = graph.session.value
        val failure = graph.refreshSession()
        val after = graph.session.value
        error = failure?.toUiError()
        refreshing = false
        if (failure == null && before != null && after != null && before != after) {
            onSessionChanged(before, after)
        }
    }

    SettingsRootScreen(
        session = session,
        refreshing = refreshing,
        error = error,
        onOpen = onOpen,
        onRetry = { attempt++ },
        onBack = onBack,
        onSignOut = onSignedOut,
    )
}

/** «Назад»: из формы пользователя — в список, из подраздела — в корень, из корня — на главную. */
private fun SettingsPage.parent(): SettingsPage? = when (this) {
    is SettingsPage.Root -> null
    is SettingsPage.UserEdit, is SettingsPage.UserPassword -> SettingsPage.Users()
    else -> SettingsPage.Root()
}

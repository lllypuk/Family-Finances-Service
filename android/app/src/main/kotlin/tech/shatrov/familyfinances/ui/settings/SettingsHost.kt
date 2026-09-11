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
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
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

        is SettingsPage.Profile -> ProfilePage(
            graph = graph,
            session = session,
            models = models,
            page = page,
            onSubmitting = { submitting = it },
            onSessionChanged = onSessionChanged,
            onDone = leave,
        )

        is SettingsPage.Password -> PasswordPage(
            graph = graph,
            models = models,
            page = page,
            onSubmitting = { submitting = it },
            onBack = leave,
        )

        else -> Centered {
            Text(page.slug)
            TextButton(onClick = leave) { Text(stringResource(R.string.back)) }
        }
    }
}

/**
 * Профиль: имя и почта. После `200` сессия обновляется здесь — модель о графе не знает, а
 * `onSessionChanged` тот же, что у перечитки в корне.
 */
@Composable
private fun ProfilePage(
    graph: AppGraph,
    session: Session,
    models: ViewModelStoreOwner,
    page: SettingsPage.Profile,
    onSubmitting: (Boolean) -> Unit,
    onSessionChanged: (Session, Session) -> Unit,
    onDone: () -> Unit,
) {
    val model: ProfileViewModel = viewModel(viewModelStoreOwner = models, key = page.modelKey) {
        ProfileViewModel(graph.api, session.user)
    }
    val state by model.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.submitting) { onSubmitting(state.submitting) }
    LaunchedEffect(state.saved) {
        val saved = state.saved ?: return@LaunchedEffect
        graph.update(saved)
        onSessionChanged(session, session.copy(user = saved))
        onDone()
    }

    ProfileScreen(
        state = state,
        onEmailChange = model::onEmailChange,
        onFirstNameChange = model::onFirstNameChange,
        onLastNameChange = model::onLastNameChange,
        onSubmit = model::onSubmit,
        onBack = onDone,
    )
}

/** Смена своего пароля: успех оставляет экран открытым — на нём и написано, что случилось. */
@Composable
private fun PasswordPage(
    graph: AppGraph,
    models: ViewModelStoreOwner,
    page: SettingsPage.Password,
    onSubmitting: (Boolean) -> Unit,
    onBack: () -> Unit,
) {
    val model: PasswordViewModel = viewModel(viewModelStoreOwner = models, key = page.modelKey) {
        PasswordViewModel(graph.api)
    }
    val state by model.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.submitting) { onSubmitting(state.submitting) }

    PasswordScreen(
        state = state,
        onCurrentChange = model::onCurrentChange,
        onNewChange = model::onNewChange,
        onRepeatChange = model::onRepeatChange,
        onSubmit = model::onSubmit,
        onBack = onBack,
    )
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

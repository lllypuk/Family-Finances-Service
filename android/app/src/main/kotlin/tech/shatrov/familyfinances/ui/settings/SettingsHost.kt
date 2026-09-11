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
import java.util.UUID

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
    // Заход в форму, которому есть что сказать про заданный пароль: модель страницы пароля к
    // этому моменту уже уничтожена вместе со store, а хост переход переживает.
    var passwordSetFor by remember { mutableStateOf<UUID?>(null) }
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

        is SettingsPage.Users -> UsersPage(
            graph = graph,
            session = session,
            models = models,
            page = page,
            onOpen = onPageChange,
            onBack = leave,
        )

        is SettingsPage.UserEdit -> UserEditPage(
            graph = graph,
            session = session,
            models = models,
            page = page,
            passwordSet = passwordSetFor == page.visit,
            onSubmitting = { submitting = it },
            onOpen = onPageChange,
            onSessionChanged = onSessionChanged,
            onUsersChanged = onUsersChanged,
            onBack = leave,
        )

        is SettingsPage.UserPassword -> UserPasswordPage(
            graph = graph,
            models = models,
            page = page,
            onSubmitting = { submitting = it },
            onDone = { target ->
                passwordSetFor = target.visit
                onPageChange(target)
            },
            onBack = leave,
        )

        is SettingsPage.Family -> FamilyPage(
            graph = graph,
            session = session,
            models = models,
            page = page,
            onSubmitting = { submitting = it },
            onSessionChanged = onSessionChanged,
            onDone = leave,
        )

        is SettingsPage.Sessions -> SessionsPage(
            graph = graph,
            session = session,
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

/**
 * Семья: название, валюта и таймзона. Ответ `PUT` меняет зону и валюту всех экранов, поэтому
 * идёт в сессию тем же `onSessionChanged`, что и перечитка в корне.
 */
@Composable
private fun FamilyPage(
    graph: AppGraph,
    session: Session,
    models: ViewModelStoreOwner,
    page: SettingsPage.Family,
    onSubmitting: (Boolean) -> Unit,
    onSessionChanged: (Session, Session) -> Unit,
    onDone: () -> Unit,
) {
    val model: FamilyViewModel = viewModel(viewModelStoreOwner = models, key = page.modelKey) {
        FamilyViewModel(graph.api, session.family)
    }
    val state by model.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.submitting) { onSubmitting(state.submitting) }
    LaunchedEffect(state.saved) {
        val saved = state.saved ?: return@LaunchedEffect
        graph.update(saved)
        onSessionChanged(session, session.copy(family = saved))
        onDone()
    }

    FamilyScreen(
        state = state,
        onNameChange = model::onNameChange,
        onCurrencyChange = model::onCurrencyChange,
        onTimezoneChange = model::onTimezoneChange,
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

/** Сессии: список с отзывом чужих. Даты считаются в зоне семьи, поэтому модель её и получает. */
@Composable
private fun SessionsPage(
    graph: AppGraph,
    session: Session,
    models: ViewModelStoreOwner,
    page: SettingsPage.Sessions,
    onSubmitting: (Boolean) -> Unit,
    onBack: () -> Unit,
) {
    val model: SessionsViewModel = viewModel(viewModelStoreOwner = models, key = page.modelKey) {
        SessionsViewModel(graph.api, session.zone)
    }
    val state by model.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.busy) { onSubmitting(state.busy) }

    SessionsScreen(
        state = state,
        onRetry = model::refresh,
        onRevoke = model::onRevoke,
        onBack = onBack,
    )
}

/** Список пользователей: правки в нём нет, уходить можно всегда. */
@Composable
private fun UsersPage(
    graph: AppGraph,
    session: Session,
    models: ViewModelStoreOwner,
    page: SettingsPage.Users,
    onOpen: (SettingsPage) -> Unit,
    onBack: () -> Unit,
) {
    val model: UsersViewModel = viewModel(viewModelStoreOwner = models, key = page.modelKey) {
        UsersViewModel(graph.api, session.user.id)
    }
    val state by model.state.collectAsStateWithLifecycle()

    UsersScreen(
        state = state,
        onRetry = model::refresh,
        onAdd = { onOpen(SettingsPage.UserEdit(null)) },
        onOpen = { onOpen(SettingsPage.UserEdit(it)) },
        onBack = onBack,
    )
}

/**
 * Форма пользователя. Ответ про свою запись идёт в сессию, про чужую — в флаг устаревания
 * списка: имя автора операции берётся оттуда.
 */
@Composable
private fun UserEditPage(
    graph: AppGraph,
    session: Session,
    models: ViewModelStoreOwner,
    page: SettingsPage.UserEdit,
    passwordSet: Boolean,
    onSubmitting: (Boolean) -> Unit,
    onOpen: (SettingsPage) -> Unit,
    onSessionChanged: (Session, Session) -> Unit,
    onUsersChanged: () -> Unit,
    onBack: () -> Unit,
) {
    val model: UserEditViewModel = viewModel(viewModelStoreOwner = models, key = page.modelKey) {
        UserEditViewModel(graph.api, page.id, session.user.id)
    }
    val state by model.state.collectAsStateWithLifecycle()
    val target = page.id

    LaunchedEffect(state.submitting) { onSubmitting(state.submitting) }
    LaunchedEffect(state.saved) {
        val saved = state.saved ?: return@LaunchedEffect
        if (saved.id == session.user.id) {
            graph.update(saved)
            onSessionChanged(session, session.copy(user = saved))
        } else {
            onUsersChanged()
        }
    }
    LaunchedEffect(state.exit) {
        when (state.exit) {
            UserEditExit.List -> onBack()

            // Себя понизили: `/users` этому токену больше не отвечает.
            UserEditExit.Root -> onOpen(SettingsPage.Root())

            null -> Unit
        }
    }

    UserEditScreen(
        state = state,
        passwordSet = passwordSet,
        onEmailChange = model::onEmailChange,
        onFirstNameChange = model::onFirstNameChange,
        onLastNameChange = model::onLastNameChange,
        onPasswordChange = model::onPasswordChange,
        onRoleChange = model::onRoleChange,
        onSubmit = model::onSubmit,
        onToggleRole = model::onToggleRole,
        onToggleActive = model::onToggleActive,
        onSetPassword = { if (target != null) onOpen(SettingsPage.UserPassword(target)) },
        onRetry = model::load,
        onBack = onBack,
    )
}

/** Установка пароля пользователю: успех уводит обратно в форму, там же и сообщение. */
@Composable
private fun UserPasswordPage(
    graph: AppGraph,
    models: ViewModelStoreOwner,
    page: SettingsPage.UserPassword,
    onSubmitting: (Boolean) -> Unit,
    onDone: (SettingsPage.UserEdit) -> Unit,
    onBack: () -> Unit,
) {
    val model: UserPasswordViewModel = viewModel(viewModelStoreOwner = models, key = page.modelKey) {
        UserPasswordViewModel(graph.api, page.id)
    }
    val state by model.state.collectAsStateWithLifecycle()

    LaunchedEffect(state.submitting) { onSubmitting(state.submitting) }
    LaunchedEffect(state.done) {
        if (state.done) onDone(SettingsPage.UserEdit(page.id))
    }

    UserPasswordScreen(
        state = state,
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

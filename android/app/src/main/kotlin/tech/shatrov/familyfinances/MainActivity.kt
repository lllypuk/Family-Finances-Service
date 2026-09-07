package tech.shatrov.familyfinances

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.categories.CategoriesScreen
import tech.shatrov.familyfinances.ui.categories.CategoriesViewModel
import tech.shatrov.familyfinances.ui.categories.CategoryEditScreen
import tech.shatrov.familyfinances.ui.home.HomeScreen
import tech.shatrov.familyfinances.ui.home.HomeViewModel
import tech.shatrov.familyfinances.ui.login.LoginScreen
import tech.shatrov.familyfinances.ui.login.LoginViewModel
import tech.shatrov.familyfinances.ui.transactions.TransactionEditScreen
import tech.shatrov.familyfinances.ui.transactions.TransactionEditViewModel
import tech.shatrov.familyfinances.ui.transactions.TransactionsScreen
import tech.shatrov.familyfinances.ui.transactions.TransactionsViewModel

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val graph = (application as FamilyFinancesApp).graph
        setContent {
            AppTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    // С targetSdk 37 система рисует контент под своими панелями: без отступа
                    // верхний ряд каждого экрана уезжает под статус-бар.
                    Box(modifier = Modifier.windowInsetsPadding(WindowInsets.safeDrawing)) {
                        AppRoot(graph)
                    }
                }
            }
        }
    }
}

@Composable
fun AppRoot(graph: AppGraph) {
    var screen by rememberSaveable(stateSaver = AppScreenSaver) {
        mutableStateOf(if (graph.hasLiveToken()) AppScreen.Loading else AppScreen.Login)
    }
    // Список и главная переживают правку формы вместе со своими моделями, поэтому
    // перечитываются по возврату, а не при каждом заходе.
    var listStale by rememberSaveable { mutableStateOf(false) }
    var homeStale by rememberSaveable { mutableStateOf(false) }
    val session by graph.session.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()

    // Смерть процесса возвращает сохранённый экран, но не сессию: без роли и валюты главной
    // нечего показывать, поэтому бутстрап прогоняется заново.
    LaunchedEffect(screen, session) {
        if (session == null && screen != AppScreen.Loading && screen != AppScreen.Login) {
            screen = AppScreen.Loading
        }
    }

    LaunchedEffect(Unit) {
        graph.api.sessionExpired.collect { screen = AppScreen.Login }
    }

    when (val current = screen) {
        AppScreen.Loading -> {
            LaunchedEffect(Unit) {
                screen = if (graph.bootstrap()) AppScreen.Home else AppScreen.Login
            }
            Centered { Text(stringResource(R.string.loading)) }
        }

        AppScreen.Login -> {
            val model: LoginViewModel = viewModel { LoginViewModel(graph.api) }
            val state by model.state.collectAsStateWithLifecycle()
            // Токен уже записан хранилищем: дальше бутстрап, а не сразу главная. Флаг снимается
            // тут же — модель переживает выход, и невзведённый обратно флаг увёл бы следующий
            // заход на вход в бесконечный круг «вход → бутстрап → вход».
            LaunchedEffect(state.signedIn) {
                if (state.signedIn) {
                    model.onNavigated()
                    screen = AppScreen.Loading
                }
            }
            LoginScreen(
                state = state,
                onEmailChange = model::onEmailChange,
                onPasswordChange = model::onPasswordChange,
                onSubmit = model::onSubmit,
            )
        }

        // Модели ключуются по вошедшему: они переживают выход, и после входа другим
        // пользователем прежняя роль показала бы ему чужие действия.
        AppScreen.Home -> WithSession(session) { active ->
            val model: HomeViewModel = viewModel(key = "home-${active.user.id}") {
                HomeViewModel(graph.api, active.currency)
            }
            val home by model.state.collectAsStateWithLifecycle()
            LaunchedEffect(homeStale) {
                if (homeStale) {
                    model.refresh()
                    homeStale = false
                }
            }
            HomeScreen(
                state = home,
                onRetry = model::refresh,
                onOpenTransactions = { screen = AppScreen.Transactions },
                onOpenCategories = { screen = AppScreen.Categories },
                onSignOut = {
                    scope.launch {
                        graph.signOut()
                        screen = AppScreen.Login
                    }
                },
            )
        }

        AppScreen.Transactions -> WithSession(session) { active ->
            val model: TransactionsViewModel = viewModel(key = "transactions-${active.user.id}") {
                TransactionsViewModel(graph.api, active)
            }
            val transactions by model.state.collectAsStateWithLifecycle()
            LaunchedEffect(listStale) {
                if (listStale) {
                    model.refresh()
                    listStale = false
                }
            }
            BackHandler { screen = AppScreen.Home }
            TransactionsScreen(
                state = transactions,
                onBack = { screen = AppScreen.Home },
                onRetry = model::refresh,
                onFiltersChange = model::onFiltersChange,
                onLoadMore = model::loadMore,
                onCreate = { screen = AppScreen.TransactionEdit(null) },
                onOpen = { screen = AppScreen.TransactionEdit(it) },
            )
        }

        AppScreen.Categories -> WithSession(session) { active ->
            val model: CategoriesViewModel = viewModel(key = "categories-${active.user.id}") {
                CategoriesViewModel(graph.api, active.isAdmin)
            }
            val categories by model.state.collectAsStateWithLifecycle()
            val editor by model.editor.collectAsStateWithLifecycle()
            val form = editor
            BackHandler { if (form == null) screen = AppScreen.Home else model.onDismiss() }
            if (form == null) {
                CategoriesScreen(
                    state = categories,
                    onBack = { screen = AppScreen.Home },
                    onRetry = model::refresh,
                    onAdd = model::onAdd,
                    onOpen = model::onOpen,
                )
            } else {
                CategoryEditScreen(
                    state = form,
                    onNameChange = model::onNameChange,
                    onTypeChange = model::onTypeChange,
                    onColorChange = model::onColorChange,
                    onIconChange = model::onIconChange,
                    onParentChange = model::onParentChange,
                    onSubmit = model::onSubmit,
                    onDelete = model::onDelete,
                    onBack = model::onDismiss,
                )
            }
        }

        is AppScreen.TransactionEdit -> WithSession(session) {
            // Ключ по черновику: без него следующий заход на форму достался бы модели прошлого,
            // уже сохранённого, и экран сразу закрылся бы.
            val model: TransactionEditViewModel = viewModel(key = "edit-${current.draft}") {
                TransactionEditViewModel(graph.api, current.id, current.draft)
            }
            val edit by model.state.collectAsStateWithLifecycle()
            LaunchedEffect(edit.done) {
                if (edit.done) {
                    listStale = true
                    homeStale = true
                    screen = AppScreen.Transactions
                }
            }
            BackHandler { screen = AppScreen.Transactions }
            TransactionEditScreen(
                state = edit,
                onAmountChange = model::onAmountChange,
                onTypeChange = model::onTypeChange,
                onCategoryChange = model::onCategoryChange,
                onDateChange = model::onDateChange,
                onDescriptionChange = model::onDescriptionChange,
                onSubmit = model::onSubmit,
                onDelete = model::onDelete,
                onBack = { screen = AppScreen.Transactions },
            )
        }
    }
}

/** Сессия гаснет на выходе раньше, чем сменится экран: без валюты и роли рисовать нечего. */
@Composable
private fun WithSession(
    session: Session?,
    content: @Composable (Session) -> Unit,
) {
    if (session == null) {
        Centered { Text(stringResource(R.string.loading)) }
    } else {
        content(session)
    }
}

@Composable
private fun Centered(content: @Composable () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_3, Alignment.CenterVertically),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        content()
    }
}

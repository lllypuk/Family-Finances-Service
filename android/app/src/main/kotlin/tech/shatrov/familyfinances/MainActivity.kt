package tech.shatrov.familyfinances

import android.app.Application
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawing
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.material3.Button
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.core.content.IntentCompat
import androidx.core.view.WindowCompat
import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelStore
import androidx.lifecycle.ViewModelStoreOwner
import androidx.lifecycle.compose.LifecycleResumeEffect
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import tech.shatrov.familyfinances.core.api.net.ApiFailure
import tech.shatrov.familyfinances.theme.AppTheme
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.ThemeMode
import tech.shatrov.familyfinances.theme.isDark
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.AppNavBar
import tech.shatrov.familyfinances.ui.AppTab
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.UiError
import tech.shatrov.familyfinances.ui.budgets.BudgetEditScreen
import tech.shatrov.familyfinances.ui.budgets.BudgetEditViewModel
import tech.shatrov.familyfinances.ui.budgets.BudgetsScreen
import tech.shatrov.familyfinances.ui.budgets.BudgetsViewModel
import tech.shatrov.familyfinances.ui.categories.CategoriesScreen
import tech.shatrov.familyfinances.ui.categories.CategoriesViewModel
import tech.shatrov.familyfinances.ui.categories.CategoryEditScreen
import tech.shatrov.familyfinances.ui.home.HomeScreen
import tech.shatrov.familyfinances.ui.home.HomeViewModel
import tech.shatrov.familyfinances.ui.login.LoginScreen
import tech.shatrov.familyfinances.ui.login.LoginViewModel
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.networth.HoldingEditScreen
import tech.shatrov.familyfinances.ui.networth.HoldingEditViewModel
import tech.shatrov.familyfinances.ui.networth.HoldingHistoryScreen
import tech.shatrov.familyfinances.ui.networth.HoldingHistoryUiState
import tech.shatrov.familyfinances.ui.networth.HoldingHistoryViewModel
import tech.shatrov.familyfinances.ui.networth.NetWorthScreen
import tech.shatrov.familyfinances.ui.networth.NetWorthViewModel
import tech.shatrov.familyfinances.ui.networth.ValueSheet
import tech.shatrov.familyfinances.ui.overview.OverviewScreen
import tech.shatrov.familyfinances.ui.overview.OverviewViewModel
import tech.shatrov.familyfinances.ui.recognize.RecognizeScreen
import tech.shatrov.familyfinances.ui.recognize.RecognizeViewModel
import tech.shatrov.familyfinances.ui.recognize.rememberImportLaunchers
import tech.shatrov.familyfinances.ui.reconciliation.ReconciliationScreen
import tech.shatrov.familyfinances.ui.reconciliation.ReconciliationViewModel
import tech.shatrov.familyfinances.ui.reconciliation.gapCorrection
import tech.shatrov.familyfinances.ui.settings.SettingsHost
import tech.shatrov.familyfinances.ui.settings.SettingsPage
import tech.shatrov.familyfinances.ui.toUiError
import tech.shatrov.familyfinances.ui.transactions.TransactionEditScreen
import tech.shatrov.familyfinances.ui.transactions.TransactionEditViewModel
import tech.shatrov.familyfinances.ui.transactions.TransactionFilters
import tech.shatrov.familyfinances.ui.transactions.TransactionsScreen
import tech.shatrov.familyfinances.ui.transactions.TransactionsViewModel
import java.time.LocalDate
import java.time.YearMonth

class MainActivity : ComponentActivity() {
    private val graph: AppGraph
        get() = (application as FamilyFinancesApp).graph

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Пересозданная активити получает тот же intent: картинки уже предложены до поворота.
        if (savedInstanceState == null) offerShared(intent)
        when (graph.theme.mode.value) {
            ThemeMode.Light -> setTheme(R.style.Theme_FamilyFinances_Light)
            ThemeMode.Dark -> setTheme(R.style.Theme_FamilyFinances_Dark)
            ThemeMode.System -> Unit
        }
        setContent {
            val mode by graph.theme.mode.collectAsStateWithLifecycle()
            val dark = mode.isDark()
            // Живое переключение: XML-атрибуты панелей читаются только при создании окна.
            SideEffect {
                WindowCompat.getInsetsController(window, window.decorView).run {
                    isAppearanceLightStatusBars = !dark
                    isAppearanceLightNavigationBars = !dark
                }
            }
            AppTheme(mode) {
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

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        offerShared(intent)
    }

    // Запуск из «недавних» повторяет исходный intent целиком, а права на его URI уже истекли.
    private fun offerShared(intent: Intent) {
        if (intent.flags and Intent.FLAG_ACTIVITY_LAUNCHED_FROM_HISTORY != 0) return
        val uris = sharedImages(intent)
        if (uris.isNotEmpty()) graph.imports.offer(uris)
        intent.removeExtra(Intent.EXTRA_STREAM)
        intent.clipData = null
    }
}

/** `SEND` и `SEND_MULTIPLE` кладут URI и в `EXTRA_STREAM`, и в `ClipData` — отправители делают по-разному. */
internal fun sharedImages(intent: Intent): List<Uri> {
    if (intent.action != Intent.ACTION_SEND && intent.action != Intent.ACTION_SEND_MULTIPLE) return emptyList()
    val stream = if (intent.action == Intent.ACTION_SEND) {
        listOfNotNull(IntentCompat.getParcelableExtra(intent, Intent.EXTRA_STREAM, Uri::class.java))
    } else {
        IntentCompat.getParcelableArrayListExtra(intent, Intent.EXTRA_STREAM, Uri::class.java).orEmpty()
    }
    val clip = intent.clipData?.let { data -> (0 until data.itemCount).mapNotNull { data.getItemAt(it).uri } }.orEmpty()
    return (stream + clip).distinct()
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
    var budgetsStale by rememberSaveable { mutableStateOf(false) }
    var netWorthStale by rememberSaveable { mutableStateOf(false) }
    // Свой, а не `homeStale`: флаг гасит прочитавший, и «Обзор» оставил бы «Главную» со старыми итогами.
    var overviewStale by rememberSaveable { mutableStateOf(false) }
    // Формы держат справочник категорий в своих моделях и перечитывают его по возврату из справочника.
    var categoriesStale by rememberSaveable { mutableStateOf(false) }
    // Распознавание уводит с расшифровки мимо её выхода, а модель списка фильтр помнит.
    var dropDrill by rememberSaveable { mutableStateOf(false) }
    // Модели экранов лежат в store активити и переживают выход. Ключа по пользователю мало:
    // повторный вход тем же человеком показал бы данные прошлой сессии и не перечитал бы их.
    var epoch by rememberSaveable { mutableIntStateOf(0) }
    val session by graph.session.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    // Модель формы живёт в своём store: ключ у неё свой на каждый заход, а store активити
    // отдаёт брошенные модели только вместе с активити.
    val forms: ScopedModels = viewModel(key = "forms") { ScopedModels() }
    // Справочник, открытый с формы, её черновик не убивает: модель ждёт возврата в store.
    val onForm = screen.isForm() || (screen as? AppScreen.Categories)?.back?.isForm() == true
    LaunchedEffect(onForm) { if (!onForm) forms.viewModelStore.clear() }
    // Свой store: модели подразделов настроек чистятся на каждом переходе, а модели форм — нет.
    val settings: ScopedModels = viewModel(key = "settings") { ScopedModels() }
    val inSettings = screen is AppScreen.Settings
    LaunchedEffect(inSettings) { if (!inSettings) settings.viewModelStore.clear() }

    // Экран, куда вернуться после бутстрапа; `Loading` — никуда.
    var resumeTo by rememberSaveable(stateSaver = AppScreenSaver) { mutableStateOf<AppScreen>(AppScreen.Loading) }

    // Смерть процесса возвращает сохранённый экран, но не сессию: без роли и валюты главной
    // нечего показывать, поэтому бутстрап прогоняется заново.
    LaunchedEffect(screen, session) {
        val current = screen
        if (session == null && current != AppScreen.Loading && current != AppScreen.Login) {
            resumeTo = if (resumable(graph, current)) current else AppScreen.Loading
            screen = AppScreen.Loading
        }
    }

    LaunchedEffect(Unit) {
        graph.api.sessionExpired.collect {
            screen = AppScreen.Login
            withContext(Dispatchers.IO) { graph.journals.deleteAll() }
        }
    }

    val pending by graph.imports.pending.collectAsStateWithLifecycle()
    val importLaunchers = rememberImportLaunchers { graph.imports.offer(it) }
    // Формы и настройки импорт ждёт: уход с них бросил бы введённое или убил бы отправку.
    // Занятый экран распознавания `pending` не публикует — это политика `ImportStore`.
    LaunchedEffect(pending, session, screen) {
        val id = pending ?: return@LaunchedEffect
        if (session == null) return@LaunchedEffect
        when (val current = screen) {
            AppScreen.Home,
            is AppScreen.Overview,
            AppScreen.Budgets,
            AppScreen.NetWorth,
            is AppScreen.Reconciliation,
            -> screen = AppScreen.Recognize(id)

            is AppScreen.Transactions -> {
                if (current.drilled) dropDrill = true
                screen = AppScreen.Recognize(id)
            }

            is AppScreen.Categories -> if (!current.back.isForm()) {
                if ((current.back as? AppScreen.Transactions)?.drilled == true) dropDrill = true
                listStale = true
                homeStale = true
                overviewStale = true
                budgetsStale = true
                screen = AppScreen.Recognize(id)
            }

            // Прежний импорт в просмотре или отказе вытесняется: модель уходит сразу, журнал удалит новая.
            is AppScreen.Recognize -> if (current.importId != id) {
                forms.viewModelStore.clear()
                screen = AppScreen.Recognize(id)
            }

            else -> Unit
        }
    }

    when (val current = screen) {
        AppScreen.Loading -> {
            var failure by remember { mutableStateOf<UiError?>(null) }
            var attempt by remember { mutableIntStateOf(0) }
            // На вход уводит только кончившийся токен. Обрыв связи оставляет сохранённую
            // сессию на месте: иначе старт без сети даёт пустую форму входа без объяснений.
            LaunchedEffect(attempt) {
                failure = null
                val error = graph.bootstrap()
                when {
                    error == null -> {
                        screen = graph.imports.pending.value?.let(AppScreen::Recognize)
                            ?: resumeTo.takeIf { it != AppScreen.Loading }
                            ?: AppScreen.Home
                        resumeTo = AppScreen.Loading
                    }

                    error is ApiFailure.Api && error.isUnauthorized -> screen = AppScreen.Login

                    else -> failure = error.toUiError()
                }
            }
            BootstrapScreen(
                error = failure,
                onRetry = { attempt++ },
                onSignOut = {
                    scope.launch {
                        graph.signOut()
                        screen = AppScreen.Login
                    }
                },
            )
        }

        AppScreen.Login -> {
            LaunchedEffect(Unit) {
                epoch++
                resumeTo = AppScreen.Loading
            }
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
            val model: HomeViewModel = viewModel(key = "home-${active.user.id}-$epoch") {
                HomeViewModel(graph.api, active.currency, active.zone, journals = graph.journals)
            }
            val home by model.state.collectAsStateWithLifecycle()
            val card by model.card.collectAsStateWithLifecycle()
            val importDraft by model.import.collectAsStateWithLifecycle()
            val waiting by graph.imports.waiting.collectAsStateWithLifecycle()
            // На возврате из фона, а не только при заходе: модель живёт всю сессию, а сводка
            // посчитана по «сегодня» и границам месяца, которые под свёрнутым экраном сменились.
            LifecycleResumeEffect(homeStale) {
                if (homeStale) {
                    model.refresh()
                    homeStale = false
                } else {
                    model.revalidate()
                }
                model.loadImport()
                onPauseOrDispose {}
            }
            WithNavBar(AppTab.HOME, onSelect = { screen = it.screen }) {
                HomeScreen(
                    state = home,
                    onRetry = model::refresh,
                    onAddTransaction = { screen = AppScreen.TransactionEdit(null) },
                    onSettings = { screen = AppScreen.Settings() },
                    card = card,
                    onReconciliation = { screen = AppScreen.Reconciliation(it) },
                    onAccounts = { screen = AppScreen.Settings(SettingsPage.Accounts()) },
                    // Пришедший импорт сейчас откроется сам и удалит этот журнал.
                    importDraft = importDraft.takeIf { pending == null && !waiting },
                    onResumeImport = { screen = AppScreen.Recognize(it) },
                    onDeleteImport = model::deleteImport,
                    onOverview = { screen = AppScreen.Overview() },
                    today = LocalDate.now(active.zone),
                )
            }
        }

        is AppScreen.Overview -> WithSession(session) { active ->
            val model: OverviewViewModel = viewModel(key = "overview-${active.user.id}-$epoch") {
                OverviewViewModel(graph.api, active.currency, current.period, active.zone)
            }
            val overview by model.state.collectAsStateWithLifecycle()
            // Период живёт в экране, чтобы пережить смерть процесса; модель его только догоняет.
            LaunchedEffect(current.period) { model.select(current.period) }
            LifecycleResumeEffect(overviewStale) {
                if (overviewStale) {
                    model.refresh()
                    overviewStale = false
                } else {
                    model.revalidate()
                }
                onPauseOrDispose {}
            }
            BackHandler { screen = AppScreen.Home }
            OverviewScreen(
                state = overview,
                onBack = { screen = AppScreen.Home },
                onSelect = { screen = AppScreen.Overview(it) },
                onRetryMonthly = model::retryMonthly,
                onRetrySummary = model::retrySummary,
                onOpenCategory = { type, category ->
                    val range = overview.range
                    screen = AppScreen.Transactions(
                        filters = TransactionFilters.overview(type, category, range.from, range.to),
                        overview = overview.period,
                    )
                },
            )
        }

        is AppScreen.Transactions -> WithSession(session) { active ->
            val model: TransactionsViewModel = viewModel(key = "transactions-${active.user.id}-$epoch") {
                TransactionsViewModel(graph.api, active)
            }
            val transactions by model.state.collectAsStateWithLifecycle()
            val filters by model.filters.collectAsStateWithLifecycle()
            val categories by model.categories.collectAsStateWithLifecycle()
            val accounts by model.accounts.collectAsStateWithLifecycle()
            // На возврате из фона, а не только при заходе: модель живёт всю сессию, и месяц
            // под фильтром «этот» мог смениться, пока экран лежал свёрнутым.
            LifecycleResumeEffect(listStale) {
                if (listStale) {
                    model.refresh()
                    listStale = false
                } else {
                    model.revalidate()
                }
                onPauseOrDispose {}
            }
            // Фильтр снаружи ставится один раз на ключ экрана: повторная композиция после поворота
            // иначе вернула бы его поверх того, что пользователь выбрал чипами.
            LaunchedEffect(current.filters) {
                if (dropDrill) {
                    dropDrill = false
                    if (current.filters == null) model.onFiltersChange(TransactionFilters())
                }
                current.filters?.let(model::applyFilters)
            }
            // Вкладка «Операции» после сверки или «Обзора» открывалась бы на их расшифровке.
            val leave = { next: AppScreen ->
                if (current.drilled) model.onFiltersChange(TransactionFilters())
                screen = next
            }
            BackHandler {
                leave(
                    current.reconciliation?.let(AppScreen::Reconciliation)
                        ?: current.overview?.let(AppScreen::Overview)
                        ?: AppScreen.Home,
                )
            }
            WithNavBar(
                AppTab.TRANSACTIONS,
                onSelect = { leave(it.screen) },
                fab = {
                    AddFab({ screen = AppScreen.TransactionEdit(null, back = current) }, R.string.transactions_add)
                },
            ) {
                TransactionsScreen(
                    state = transactions,
                    filters = filters,
                    categories = categories,
                    accounts = accounts,
                    onRetry = model::refresh,
                    onFiltersChange = { next ->
                        model.onFiltersChange(next)
                        if (current.filters != null) screen = current.copy(filters = next)
                    },
                    onLoadMore = model::loadMore,
                    onCreate = { screen = AppScreen.TransactionEdit(null, back = current) },
                    onOpen = { screen = AppScreen.TransactionEdit(it, back = current) },
                    importLaunchers = importLaunchers,
                    onManageCategories = { screen = AppScreen.Categories(back = current) },
                )
            }
        }

        is AppScreen.Categories -> WithSession(session) { active ->
            val model: CategoriesViewModel = viewModel(key = "categories-${active.user.id}-$epoch") {
                CategoriesViewModel(graph.api, active.isAdmin)
            }
            val categories by model.state.collectAsStateWithLifecycle()
            val editor by model.editor.collectAsStateWithLifecycle()
            val form = editor
            // Список подписывает строки именами категорий и фильтрует по ним, главная и «Обзор» —
            // расходы в сводке, бюджеты — имя категории в строке, формы — выбор категории, поэтому
            // уход с этого экрана помечает устаревшими всех: правку модель наружу не отдаёт.
            val leave = {
                listStale = true
                homeStale = true
                overviewStale = true
                budgetsStale = true
                categoriesStale = true
                screen = current.back
            }
            BackHandler { if (form == null) leave() else model.onDismiss() }
            if (form == null) {
                CategoriesScreen(
                    state = categories,
                    onRetry = model::refresh,
                    onAdd = model::onAdd,
                    onOpen = model::onOpen,
                    onBack = leave,
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

        AppScreen.Budgets -> WithSession(session) { active ->
            val model: BudgetsViewModel = viewModel(key = "budgets-${active.user.id}-$epoch") {
                BudgetsViewModel(graph.api, active.zone)
            }
            val budgets by model.state.collectAsStateWithLifecycle()
            val budgetFilter by model.filter.collectAsStateWithLifecycle()
            // «Сегодня» под фильтром считает сервер: ответ, полученный вчера, к возврату из
            // фона показывает чужой день.
            LifecycleResumeEffect(budgetsStale) {
                if (budgetsStale) {
                    model.refresh()
                    budgetsStale = false
                } else {
                    model.revalidate()
                }
                onPauseOrDispose {}
            }
            BackHandler { screen = AppScreen.Home }
            WithNavBar(
                AppTab.BUDGETS,
                onSelect = { screen = it.screen },
                fab = { AddFab({ screen = AppScreen.BudgetEdit(null) }, R.string.budgets_add) },
            ) {
                BudgetsScreen(
                    state = budgets,
                    filter = budgetFilter,
                    currency = active.currency,
                    onRetry = model::refresh,
                    onFilterChange = model::onFilterChange,
                    onCreate = { screen = AppScreen.BudgetEdit(null) },
                    onOpen = { screen = AppScreen.BudgetEdit(it) },
                )
            }
        }

        AppScreen.NetWorth -> WithSession(session) { active ->
            val model: NetWorthViewModel = viewModel(key = "networth-${active.user.id}-$epoch") {
                NetWorthViewModel(graph.api, active.zone)
            }
            val netWorth by model.state.collectAsStateWithLifecycle()
            val editor by model.editor.collectAsStateWithLifecycle()
            // `current` и последняя корзина отсечены по «сегодня» семьи, а оно под свёрнутым экраном сменилось.
            LifecycleResumeEffect(netWorthStale) {
                if (netWorthStale) {
                    model.refresh()
                    netWorthStale = false
                } else {
                    model.revalidate()
                }
                onPauseOrDispose {}
            }
            BackHandler { screen = AppScreen.Home }
            WithNavBar(
                AppTab.NET_WORTH,
                onSelect = { screen = it.screen },
                fab = { AddFab({ screen = AppScreen.HoldingEdit(null) }, R.string.net_worth_add) },
            ) {
                NetWorthScreen(
                    state = netWorth,
                    currency = active.currency,
                    today = LocalDate.now(active.zone),
                    zone = active.zone,
                    onRetry = model::refresh,
                    onAdd = { screen = AppScreen.HoldingEdit(null, side = it) },
                    onOpenValue = model::onOpenValue,
                    onEdit = { screen = AppScreen.HoldingEdit(it) },
                )
            }
            editor?.let { value ->
                ValueSheet(
                    state = value,
                    currency = active.currency,
                    onAmountChange = model::onAmountChange,
                    onDateChange = model::onDateChange,
                    onSave = model::onSaveValue,
                    onDismiss = model::onDismissValue,
                    onHistory = {
                        model.onDismissValue()
                        screen = AppScreen.HoldingHistory(value.holdingId)
                    },
                )
            }
        }

        is AppScreen.HoldingHistory -> WithSession(session) { active ->
            val model: HoldingHistoryViewModel =
                viewModel(viewModelStoreOwner = forms, key = "holding-history-${current.id}") {
                    HoldingHistoryViewModel(graph.api, current.id, active.zone)
                }
            val history by model.state.collectAsStateWithLifecycle()
            val editor by model.editor.collectAsStateWithLifecycle()
            // Правка истории двигает `current` и ряд: капитал перечитывается на возврате.
            val leave = {
                netWorthStale = true
                screen = AppScreen.NetWorth
            }
            LaunchedEffect(history is HoldingHistoryUiState.Gone) {
                if (history is HoldingHistoryUiState.Gone) leave()
            }
            BackHandler { leave() }
            HoldingHistoryScreen(
                state = history,
                currency = active.currency,
                onRetry = model::refresh,
                onLoadMore = model::loadMore,
                onOpenValue = model::onOpenValue,
                onEdit = {
                    netWorthStale = true
                    screen = AppScreen.HoldingEdit(current.id)
                },
                onBack = leave,
            )
            editor?.let { value ->
                ValueSheet(
                    state = value,
                    currency = active.currency,
                    onAmountChange = model::onAmountChange,
                    onDateChange = model::onDateChange,
                    onSave = model::onSaveValue,
                    onDismiss = model::onDismissValue,
                    onDelete = model::onDeleteValue,
                )
            }
        }

        is AppScreen.HoldingEdit -> WithSession(session) { active ->
            val model: HoldingEditViewModel =
                viewModel(viewModelStoreOwner = forms, key = "holding-${current.draft}") {
                    HoldingEditViewModel(
                        graph.api,
                        current.id,
                        current.draft,
                        current.side,
                        { LocalDate.now(active.zone) },
                        active.isAdmin,
                    )
                }
            val edit by model.state.collectAsStateWithLifecycle()
            LaunchedEffect(edit.done) {
                if (edit.done) {
                    netWorthStale = true
                    screen = AppScreen.NetWorth
                }
            }
            // Как у формы операции: уход во время отправки убил бы корутину, которую сервер уже мог применить.
            val leave = {
                if (!edit.submitting) {
                    if (edit.changed) netWorthStale = true
                    screen = AppScreen.NetWorth
                }
            }
            BackHandler { leave() }
            HoldingEditScreen(
                state = edit,
                currency = active.currency,
                onSideChange = model::onSideChange,
                onNameChange = model::onNameChange,
                onKindChange = model::onKindChange,
                onIncomeChange = model::onIncomeChange,
                onExpenseChange = model::onExpenseChange,
                onSubmit = model::onSubmit,
                onToggleArchive = model::onToggleArchive,
                onDelete = model::onDelete,
                onRetry = model::load,
                onBack = leave,
            )
        }

        is AppScreen.Reconciliation -> WithSession(session) { active ->
            val model: ReconciliationViewModel = viewModel(key = "reconciliation-${active.user.id}-$epoch") {
                ReconciliationViewModel(graph.api)
            }
            val reconciliation by model.state.collectAsStateWithLifecycle()
            val editor by model.editor.collectAsStateWithLifecycle()
            val today = YearMonth.now(active.zone)
            val month = minOf(current.month, today)
            // Каждый заход и смена месяца: итог считается из операций, а их правят и в других экранах.
            LifecycleResumeEffect(month) {
                model.load(month)
                onPauseOrDispose {}
            }
            // Карточка главной считается по тем же остаткам.
            val leave = {
                homeStale = true
                screen = AppScreen.Home
            }
            BackHandler { leave() }
            val correction = stringResource(R.string.reconciliation_correction)
            ReconciliationScreen(
                month = month,
                today = today,
                state = reconciliation,
                editor = editor,
                currency = active.currency,
                onBack = leave,
                onMonthChange = { screen = AppScreen.Reconciliation(it) },
                onRetry = model::refresh,
                onOpenBalance = model::onOpenBalance,
                onOpenOpening = model::onOpenOpening,
                onAddAccount = { screen = AppScreen.Settings(SettingsPage.Accounts()) },
                onOpenTransactions = {
                    screen = AppScreen.Transactions(TransactionFilters.reconciliation(month), reconciliation = month)
                },
                onCloseGap = { gap ->
                    screen = AppScreen.TransactionEdit(
                        id = null,
                        back = current,
                        prefill = gapCorrection(month, gap, LocalDate.now(active.zone), correction),
                    )
                },
                onAmountChange = model::onAmountChange,
                onToggleSign = model::onToggleSign,
                onSave = model::onSave,
                onClear = model::onClear,
                onDismissBalance = model::onDismissBalance,
            )
        }

        is AppScreen.Settings -> WithSession(session) { active ->
            SettingsHost(
                graph = graph,
                session = active,
                models = settings,
                page = current.page,
                onPageChange = { next ->
                    // Store чистится на самом переходе: эффект после него убил бы модель,
                    // которую новая страница успела создать в композиции.
                    settings.viewModelStore.clear()
                    screen = AppScreen.Settings(next)
                },
                onLeave = { screen = AppScreen.Home },
                onOpenCategories = { screen = AppScreen.Categories(back = AppScreen.Settings()) },
                onSessionChanged = { before, after ->
                    if (before.zone != after.zone ||
                        before.currency != after.currency ||
                        before.user.role != after.user.role
                    ) {
                        // Модели вкладок посчитаны в прежней зоне и валюте, а роль решает,
                        // какие действия им показывать.
                        epoch++
                        listStale = true
                        homeStale = true
                        budgetsStale = true
                    } else if (before.user != after.user) {
                        // Список подписывает операции именем автора.
                        listStale = true
                    }
                },
                onUsersChanged = { listStale = true },
                onSignedOut = {
                    scope.launch {
                        graph.signOut()
                        screen = AppScreen.Login
                    }
                },
            )
        }

        is AppScreen.BudgetEdit -> WithSession(session) { active ->
            val model: BudgetEditViewModel =
                viewModel(viewModelStoreOwner = forms, key = "budget-${current.draft}") {
                    BudgetEditViewModel(
                        graph.api,
                        current.id,
                        current.draft,
                        LocalDate.now(active.zone),
                        active.currency,
                    )
                }
            val edit by model.state.collectAsStateWithLifecycle()
            LaunchedEffect(edit.done) {
                if (edit.done) {
                    budgetsStale = true
                    homeStale = true
                    screen = AppScreen.Budgets
                }
            }
            // Как у формы операции: уход во время отправки убил бы корутину, а повтор с новым
            // черновиком создал бы второй бюджет.
            val leave = {
                if (!edit.submitting) {
                    if (edit.saved) {
                        budgetsStale = true
                        homeStale = true
                    }
                    screen = AppScreen.Budgets
                }
            }
            LaunchedEffect(categoriesStale) {
                if (categoriesStale) {
                    categoriesStale = false
                    model.reloadCategories()
                }
            }
            BackHandler { leave() }
            BudgetEditScreen(
                state = edit,
                onNameChange = model::onNameChange,
                onAmountChange = model::onAmountChange,
                onPeriodChange = model::onPeriodChange,
                onCategoryChange = model::onCategoryChange,
                onStartChange = model::onStartChange,
                onEndChange = model::onEndChange,
                onRecurringChange = model::onRecurringChange,
                onSubmit = model::onSubmit,
                onDelete = model::onDelete,
                onRetry = model::load,
                onBack = leave,
                onManageCategories = { screen = AppScreen.Categories(back = current) },
            )
        }

        is AppScreen.Recognize -> WithSession(session) { active ->
            val context = LocalContext.current
            val model: RecognizeViewModel =
                viewModel(viewModelStoreOwner = forms, key = "recognize-${current.importId}") {
                    RecognizeViewModel(
                        context.applicationContext as Application,
                        graph.api,
                        graph.imports,
                        graph.journals,
                        current.importId,
                        active.currency,
                        graph.lastAccount,
                    )
                }
            val recognize by model.state.collectAsStateWithLifecycle()
            val waiting by graph.imports.waiting.collectAsStateWithLifecycle()
            LaunchedEffect(recognize.savedCount) {
                if (recognize.savedCount > 0) {
                    listStale = true
                    homeStale = true
                    overviewStale = true
                    budgetsStale = true
                }
            }
            val leave = {
                if (!recognize.saving) {
                    scope.launch { model.abandon() }
                    screen = AppScreen.Transactions()
                }
            }
            LaunchedEffect(categoriesStale) {
                if (categoriesStale) {
                    categoriesStale = false
                    model.reloadCategories()
                }
            }
            BackHandler { leave() }
            RecognizeScreen(
                state = recognize,
                currency = active.currency,
                waiting = waiting,
                today = LocalDate.now(active.zone),
                onIncludedChange = model::onIncludedChange,
                onDateChange = model::onDateChange,
                onCategoryChange = model::onCategoryChange,
                onDescriptionChange = model::onDescriptionChange,
                onAccountChange = model::onAccountChange,
                onSave = model::save,
                onRetry = model::retry,
                onRetryRow = model::retryRow,
                onBack = leave,
                onManageCategories = { screen = AppScreen.Categories(back = current) },
            )
        }

        is AppScreen.TransactionEdit -> WithSession(session) { active ->
            // Ключ по черновику: без него следующий заход на форму достался бы модели прошлого,
            // уже сохранённого, и экран сразу закрылся бы.
            val model: TransactionEditViewModel =
                viewModel(viewModelStoreOwner = forms, key = "edit-${current.draft}") {
                    TransactionEditViewModel(
                        graph.api,
                        graph.lastAccount,
                        current.id,
                        current.draft,
                        LocalDate.now(active.zone),
                        active.currency,
                        current.prefill,
                    )
                }
            val edit by model.state.collectAsStateWithLifecycle()
            LaunchedEffect(edit.done) {
                if (edit.done) {
                    listStale = true
                    homeStale = true
                    overviewStale = true
                    // Операция меняет `spent` бюджета своей категории.
                    budgetsStale = true
                    screen = current.back
                }
            }
            // Уход с формы во время отправки убил бы её корутину: запись сервер уже мог
            // принять, а список о ней не узнал бы — и повтор создал бы вторую с новым черновиком.
            val leave = { if (!edit.submitting) screen = current.back }
            LaunchedEffect(categoriesStale) {
                if (categoriesStale) {
                    categoriesStale = false
                    model.reloadCategories()
                }
            }
            BackHandler { leave() }
            TransactionEditScreen(
                state = edit,
                onAmountChange = model::onAmountChange,
                onTypeChange = model::onTypeChange,
                onCategoryChange = model::onCategoryChange,
                onAccountChange = model::onAccountChange,
                onDateChange = model::onDateChange,
                onDescriptionChange = model::onDescriptionChange,
                onSubmit = model::onSubmit,
                onDelete = model::onDelete,
                onRetry = model::load,
                onBack = leave,
                onManageCategories = { screen = AppScreen.Categories(back = current) },
            )
        }
    }
}

/**
 * Хозяин моделей области: переживает поворот вместе с активити, но чистится при уходе из неё.
 * Областей две — формы и подразделы настроек, — и каждая чистится отдельно, поэтому это два
 * экземпляра с разными ключами, а не один общий store.
 */
private class ScopedModels :
    ViewModel(),
    ViewModelStoreOwner {
    override val viewModelStore = ViewModelStore()

    override fun onCleared() {
        viewModelStore.clear()
    }
}

/** Старт с сохранённым токеном: загрузка, а при отказе — причина, повтор и выход. */
@Composable
private fun BootstrapScreen(
    error: UiError?,
    onRetry: () -> Unit,
    onSignOut: () -> Unit,
) {
    if (error == null) {
        Centered { Text(stringResource(R.string.loading)) }
        return
    }
    Centered {
        Text(
            text = error.message(LocalContext.current.resources),
            color = MaterialTheme.colorScheme.error,
        )
        Button(onClick = onRetry) { Text(stringResource(R.string.retry)) }
        TextButton(onClick = onSignOut) { Text(stringResource(R.string.sign_out)) }
    }
}

/** Корневые экраны делят одну панель вкладок; сами экраны о ней не знают. */
@Composable
private fun WithNavBar(
    selected: AppTab,
    onSelect: (AppTab) -> Unit,
    fab: @Composable () -> Unit = {},
    content: @Composable () -> Unit,
) {
    Column(modifier = Modifier.fillMaxSize()) {
        Box(modifier = Modifier.weight(1f)) {
            content()
            Box(modifier = Modifier.align(Alignment.BottomEnd).padding(Dimens.SPACE_4)) { fab() }
        }
        AppNavBar(selected = selected, onSelect = onSelect)
    }
}

/** Кнопка «добавить» вкладки: одна на все вкладки со списком, чтобы ход был одинаковым. */
@Composable
private fun AddFab(
    onClick: () -> Unit,
    @StringRes label: Int,
) {
    FloatingActionButton(onClick = onClick) {
        Icon(AppIcons.Plus, contentDescription = stringResource(label))
    }
}

private fun AppScreen.isForm(): Boolean = this is AppScreen.TransactionEdit ||
    this is AppScreen.BudgetEdit ||
    this is AppScreen.HoldingEdit ||
    this is AppScreen.HoldingHistory ||
    this is AppScreen.Recognize

private val AppTab.screen: AppScreen
    get() = when (this) {
        AppTab.HOME -> AppScreen.Home
        AppTab.TRANSACTIONS -> AppScreen.Transactions()
        AppTab.BUDGETS -> AppScreen.Budgets
        AppTab.NET_WORTH -> AppScreen.NetWorth
    }

/** Возвращаются только экраны без живой модели формы; распознавание — пока его журнал на диске. */
private suspend fun resumable(
    graph: AppGraph,
    screen: AppScreen,
): Boolean = when (screen) {
    is AppScreen.Recognize -> withContext(Dispatchers.IO) { graph.journals.read(screen.importId) != null }
    is AppScreen.Overview -> true
    else -> false
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

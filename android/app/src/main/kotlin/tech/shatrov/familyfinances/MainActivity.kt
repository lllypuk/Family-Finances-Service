package tech.shatrov.familyfinances

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
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
import tech.shatrov.familyfinances.ui.login.LoginScreen
import tech.shatrov.familyfinances.ui.login.LoginViewModel

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val graph = AppGraph.create(applicationContext, BuildConfig.API_BASE_URL)
        setContent {
            AppTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    AppRoot(graph)
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
    val session by graph.session.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()

    // Смерть процесса возвращает сохранённый экран, но не сессию: без роли и валюты главной
    // нечего показывать, поэтому бутстрап прогоняется заново.
    LaunchedEffect(screen, session) {
        if (screen == AppScreen.Home && session == null) {
            screen = AppScreen.Loading
        }
    }

    LaunchedEffect(Unit) {
        graph.api.sessionExpired.collect { screen = AppScreen.Login }
    }

    when (screen) {
        AppScreen.Loading -> {
            LaunchedEffect(Unit) {
                screen = if (graph.bootstrap()) AppScreen.Home else AppScreen.Login
            }
            Centered { Text(stringResource(R.string.loading)) }
        }

        AppScreen.Login -> {
            val model: LoginViewModel = viewModel { LoginViewModel(graph.api) }
            val state by model.state.collectAsStateWithLifecycle()
            // Токен уже записан хранилищем: дальше бутстрап, а не сразу главная.
            LaunchedEffect(state.signedIn) {
                if (state.signedIn) screen = AppScreen.Loading
            }
            LoginScreen(
                state = state,
                onEmailChange = model::onEmailChange,
                onPasswordChange = model::onPasswordChange,
                onSubmit = model::onSubmit,
            )
        }

        AppScreen.Home -> Centered {
            val current = session
            if (current != null) {
                Text(stringResource(R.string.home_greeting, current.user.firstName))
                Text(stringResource(R.string.home_currency, current.currency))
            }
            Button(onClick = {
                scope.launch {
                    graph.signOut()
                    screen = AppScreen.Login
                }
            }) {
                Text(stringResource(R.string.sign_out))
            }
        }
    }
}

// Заглушка главной до задачи 7: экран есть, чтобы навигации было куда вести.
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

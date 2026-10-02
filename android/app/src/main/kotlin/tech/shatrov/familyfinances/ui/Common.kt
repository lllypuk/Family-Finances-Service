package tech.shatrov.familyfinances.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.format.currencySymbol

/** Заглушка на весь экран: загрузка, пустой список, отказ. */
@Composable
internal fun Centered(content: @Composable () -> Unit) {
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

/**
 * Шапка вкладки. Строка заголовка всегда высотой с кнопку, с действием или без: иначе
 * заголовок при переключении вкладок прыгает на 8dp.
 */
@Composable
internal fun ScreenHeader(
    title: String,
    subtitle: String? = null,
    action: (@Composable RowScope.() -> Unit)? = null,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(
                start = Dimens.SPACE_4,
                end = if (action == null) Dimens.SPACE_4 else Dimens.SPACE_2,
                top = Dimens.SPACE_2,
                bottom = Dimens.SPACE_2,
            ),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .heightIn(min = Dimens.TOUCH_MIN),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(text = title, style = MaterialTheme.typography.headlineSmall, modifier = Modifier.weight(1f))
            action?.invoke(this)
        }
        if (subtitle != null) {
            Text(
                text = subtitle,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

/** Сообщение под полем формы; `null` — поле без ошибки. */
@Composable
internal fun FieldError(text: String?) {
    if (text == null) return
    Text(text = text, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
}

/** Символ валюты семьи в поле суммы; пустой код — валюта ещё не пришла из сессии, суффикса нет. */
internal fun currencySuffix(currency: String): (@Composable () -> Unit)? {
    if (currency.isEmpty()) return null
    return { Text(currencySymbol(currency)) }
}

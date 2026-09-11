package tech.shatrov.familyfinances.ui

import androidx.annotation.StringRes
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.res.stringResource
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.LocalAppColors

/** Четыре корня приложения; формы вкладок не имеют и панель не показывают. */
enum class AppTab(
    @StringRes val label: Int,
    val icon: ImageVector,
) {
    HOME(R.string.home_title, AppIcons.Home),
    TRANSACTIONS(R.string.transactions_title, AppIcons.List),
    CATEGORIES(R.string.categories_title, AppIcons.Tag),
    BUDGETS(R.string.budgets_title, AppIcons.Target),
}

/** Нижняя панель вкладок. Отступ под системную панель уже дан корнем, поэтому свой — нулевой. */
@Composable
fun AppNavBar(
    selected: AppTab,
    onSelect: (AppTab) -> Unit,
) {
    val colors = LocalAppColors.current
    NavigationBar(
        containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        windowInsets = WindowInsets(0),
    ) {
        for (tab in AppTab.entries) {
            val label = stringResource(tab.label)
            NavigationBarItem(
                selected = tab == selected,
                onClick = { onSelect(tab) },
                icon = { Icon(tab.icon, contentDescription = null) },
                label = { Text(label) },
                colors = NavigationBarItemDefaults.colors(
                    selectedIconColor = colors.textInverse,
                    selectedTextColor = colors.textPrimary,
                    indicatorColor = colors.action,
                    unselectedIconColor = colors.textSecondary,
                    unselectedTextColor = colors.textSecondary,
                ),
            )
        }
    }
}

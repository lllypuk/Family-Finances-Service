package tech.shatrov.familyfinances.ui.categories

import androidx.annotation.StringRes
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.ui.message

/** Список категорий: доходные и расходные врозь, подкатегории — с отступом под родителем. */
@Composable
fun CategoriesScreen(
    state: CategoriesUiState,
    onBack: () -> Unit,
    onRetry: () -> Unit,
    onAdd: () -> Unit,
    onOpen: (Category) -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
            horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TextButton(onClick = onBack) { Text(stringResource(R.string.back)) }
            Text(
                text = stringResource(R.string.categories_title),
                style = MaterialTheme.typography.headlineSmall,
                modifier = Modifier.weight(1f),
            )
            TextButton(onClick = onAdd) { Text(stringResource(R.string.categories_add)) }
        }

        when (state) {
            CategoriesUiState.Loading -> Centered { CircularProgressIndicator() }

            is CategoriesUiState.Failure -> Centered {
                Text(
                    text = state.error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                Button(
                    onClick = onRetry,
                    modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN),
                ) {
                    Text(stringResource(R.string.retry))
                }
            }

            is CategoriesUiState.Ready ->
                if (state.isEmpty) {
                    Centered { Text(stringResource(R.string.categories_empty)) }
                } else {
                    Groups(state, onOpen)
                }
        }
    }
}

@Composable
private fun Groups(
    state: CategoriesUiState.Ready,
    onOpen: (Category) -> Unit,
) {
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(horizontal = Dimens.SPACE_4),
        verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
        contentPadding = PaddingValues(vertical = Dimens.SPACE_3),
    ) {
        section(R.string.categories_expense, state.expense, onOpen)
        section(R.string.categories_income, state.income, onOpen)
    }
}

private fun LazyListScope.section(
    @StringRes title: Int,
    nodes: List<CategoryNode>,
    onOpen: (Category) -> Unit,
) {
    if (nodes.isEmpty()) return
    item(key = title) {
        Text(
            text = stringResource(title),
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(top = Dimens.SPACE_2),
        )
    }
    for (node in nodes) {
        item(key = node.category.id) { CategoryItem(node.category, Dimens.SPACE_1, onOpen) }
        items(node.children, key = { it.id }) { child ->
            CategoryItem(child, Dimens.SPACE_6, onOpen)
        }
    }
}

@Composable
private fun CategoryItem(
    category: Category,
    indent: Dp,
    onOpen: (Category) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onOpen(category) }
            .heightIn(min = Dimens.TOUCH_MIN)
            .padding(start = indent),
        horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Surface(
            color = parseCategoryColor(category.color, MaterialTheme.colorScheme.outline),
            shape = CircleShape,
            modifier = Modifier.size(Dimens.SPACE_3),
        ) {}
        Text(
            text = category.name,
            style = MaterialTheme.typography.bodyMedium,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            modifier = Modifier.weight(1f),
        )
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

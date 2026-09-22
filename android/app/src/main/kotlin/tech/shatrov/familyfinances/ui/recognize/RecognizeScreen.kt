package tech.shatrov.familyfinances.ui.recognize

import android.graphics.BitmapFactory
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.ImageBitmap
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.Category
import tech.shatrov.familyfinances.core.api.TransactionType
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.AppIcons
import tech.shatrov.familyfinances.ui.Centered
import tech.shatrov.familyfinances.ui.DatePickerSheet
import tech.shatrov.familyfinances.ui.FieldError
import tech.shatrov.familyfinances.ui.RowPlace
import tech.shatrov.familyfinances.ui.categories.CategoryAvatar
import tech.shatrov.familyfinances.ui.format.formatDay
import tech.shatrov.familyfinances.ui.format.formatFullDay
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.groupedRow
import tech.shatrov.familyfinances.ui.message
import tech.shatrov.familyfinances.ui.rowPlace
import tech.shatrov.familyfinances.ui.transactions.AccountSheet
import tech.shatrov.familyfinances.ui.transactions.CategorySheet
import tech.shatrov.familyfinances.ui.transactions.TransactionField
import tech.shatrov.familyfinances.ui.transactions.asCategoryType
import tech.shatrov.familyfinances.ui.transactions.path
import java.io.File
import java.time.LocalDate
import java.util.UUID

private const val PREVIEW_MAX_SIDE = 512

/** Кандидаты распознавания под превью своих картинок; сохраняются только отмеченные и полные. */
@Composable
fun RecognizeScreen(
    state: RecognizeUiState,
    currency: String,
    waiting: Boolean,
    today: LocalDate,
    onIncludedChange: (UUID, Boolean) -> Unit,
    onDateChange: (UUID, LocalDate) -> Unit,
    onCategoryChange: (UUID, UUID) -> Unit,
    onDescriptionChange: (UUID, String) -> Unit,
    onAccountChange: (UUID?) -> Unit,
    onSave: () -> Unit,
    onRetry: () -> Unit,
    onRetryRow: (UUID) -> Unit,
    onBack: () -> Unit,
    onManageCategories: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    var dateFor by rememberSaveable { mutableStateOf<String?>(null) }
    var categoryFor by rememberSaveable { mutableStateOf<String?>(null) }
    var accountSheet by rememberSaveable { mutableStateOf(false) }
    val saving = state.saving

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier.padding(horizontal = Dimens.SPACE_2, vertical = Dimens.SPACE_2),
            horizontalArrangement = Arrangement.spacedBy(Dimens.SPACE_2),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // Уход убил бы корутину отправки: сервер запись уже мог принять, а строка осталась бы без статуса.
            IconButton(onClick = onBack, enabled = !saving) {
                Icon(AppIcons.ArrowLeft, contentDescription = stringResource(R.string.back))
            }
            Text(stringResource(R.string.recognize_title), style = MaterialTheme.typography.headlineSmall)
        }
        if (waiting) Note(stringResource(R.string.recognize_waiting))
        if (state.dropped > 0) Note(stringResource(R.string.recognize_dropped, state.dropped))

        when (val phase = state.phase) {
            RecognizePhase.Preparing, RecognizePhase.Recognizing -> Centered {
                CircularProgressIndicator()
                Text(stringResource(R.string.recognize_progress))
            }

            is RecognizePhase.Failure -> Centered {
                Text(
                    text = phase.error.message(LocalContext.current.resources),
                    color = MaterialTheme.colorScheme.error,
                )
                if (phase.retryable) {
                    Button(onClick = onRetry, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
                        Text(stringResource(R.string.retry))
                    }
                }
            }

            is RecognizePhase.Review ->
                if (phase.rows.isEmpty()) {
                    Centered {
                        Text(stringResource(R.string.recognize_empty))
                        Button(onClick = onBack, modifier = Modifier.heightIn(min = Dimens.TOUCH_MIN)) {
                            Text(stringResource(R.string.back))
                        }
                    }
                } else {
                    if (state.incomplete) Note(stringResource(R.string.recognize_incomplete))
                    val actions = RowActions(
                        currency = currency,
                        today = today,
                        saving = saving,
                        categories = state.categories,
                        onIncludedChange = onIncludedChange,
                        onPickDate = { dateFor = it.toString() },
                        onPickCategory = { categoryFor = it.toString() },
                        onDescriptionChange = onDescriptionChange,
                        onRetryRow = onRetryRow,
                    )
                    LazyColumn(
                        modifier = Modifier
                            .weight(1f)
                            .padding(horizontal = Dimens.SPACE_4),
                    ) {
                        groups(state.images, phase.rows, actions)
                    }
                    if (state.selectableAccounts.isNotEmpty()) {
                        val account = state.accounts.firstOrNull { it.id == state.accountId }
                        PickButton(
                            text = stringResource(
                                R.string.transaction_account_value,
                                account?.name ?: stringResource(R.string.transaction_no_account),
                            ),
                            attention = false,
                            enabled = !state.accountLocked,
                            onClick = { accountSheet = true },
                            modifier = Modifier.padding(horizontal = Dimens.SPACE_4),
                        )
                    }
                    Button(
                        onClick = onSave,
                        enabled = !saving && state.toSave > 0,
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(Dimens.SPACE_4)
                            .heightIn(min = Dimens.TOUCH_MIN),
                    ) {
                        if (saving) {
                            CircularProgressIndicator(modifier = Modifier.heightIn(max = Dimens.ICON_SIZE))
                        } else {
                            Text(stringResource(R.string.recognize_save, state.toSave))
                        }
                    }
                }
        }
    }

    val rows = (state.phase as? RecognizePhase.Review)?.rows.orEmpty()
    rows.firstOrNull { it.draft.toString() == dateFor }?.let { row ->
        DatePickerSheet(
            date = row.date ?: today,
            onPick = {
                onDateChange(row.draft, it)
                dateFor = null
            },
            onDismiss = { dateFor = null },
        )
    }
    if (accountSheet) {
        AccountSheet(
            accounts = state.selectableAccounts,
            selected = state.accountId,
            noneLabel = stringResource(R.string.transaction_no_account),
            onSelect = {
                onAccountChange(it)
                accountSheet = false
            },
            onDismiss = { accountSheet = false },
        )
    }
    rows.firstOrNull { it.draft.toString() == categoryFor }?.let { row ->
        CategorySheet(
            categories = state.categories.filter { it.type == row.type.asCategoryType() },
            selected = row.categoryId,
            onSelect = { picked ->
                onCategoryChange(row.draft, picked)
                categoryFor = null
            },
            onDismiss = { categoryFor = null },
            onManage = {
                categoryFor = null
                onManageCategories()
            },
        )
    }
}

private class RowActions(
    val currency: String,
    val today: LocalDate,
    val saving: Boolean,
    val categories: List<Category>,
    val onIncludedChange: (UUID, Boolean) -> Unit,
    val onPickDate: (UUID) -> Unit,
    val onPickCategory: (UUID) -> Unit,
    val onDescriptionChange: (UUID, String) -> Unit,
    val onRetryRow: (UUID) -> Unit,
)

private fun LazyListScope.groups(
    images: List<ImportImage>,
    rows: List<RecognizedRow>,
    actions: RowActions,
) {
    val byImage = rows.groupBy { it.image }
    images.forEachIndexed { index, image ->
        val own = byImage[index].orEmpty()
        item(key = "image-$index") { ImageHeader(index, image, own.isEmpty()) }
        rowItems(own, actions)
    }
    val unattached = byImage[null].orEmpty()
    if (unattached.isNotEmpty()) {
        item(key = "unattached") { GroupTitle(stringResource(R.string.recognize_unattached)) }
        rowItems(unattached, actions)
    }
}

private fun LazyListScope.rowItems(
    rows: List<RecognizedRow>,
    actions: RowActions,
) {
    itemsIndexed(rows, key = { _, row -> row.draft }) { index, row ->
        RowItem(row, rowPlace(index, rows.size), actions)
    }
}

@Composable
private fun ImageHeader(
    index: Int,
    image: ImportImage,
    empty: Boolean,
) {
    val number = index + 1
    when (image) {
        is ImportImage.Failed -> GroupTitle(stringResource(R.string.recognize_image_unreadable, number))

        is ImportImage.Ready -> Column(verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1)) {
            GroupTitle(stringResource(R.string.recognize_image, number))
            Preview(image.file)
            if (empty) {
                Text(
                    text = stringResource(R.string.recognize_image_empty),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun GroupTitle(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.titleMedium,
        modifier = Modifier.padding(top = Dimens.SPACE_4, bottom = Dimens.SPACE_2),
    )
}

@Composable
private fun Preview(file: File) {
    val bitmap by produceState<ImageBitmap?>(null, file) {
        value = withContext(Dispatchers.IO) { decodePreview(file) }
    }
    bitmap?.let {
        Image(
            bitmap = it,
            contentDescription = null,
            contentScale = ContentScale.Fit,
            modifier = Modifier
                .height(Dimens.PREVIEW_HEIGHT)
                .clip(RoundedCornerShape(Dimens.RADIUS_S)),
        )
    }
}

private fun decodePreview(file: File): ImageBitmap? {
    val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
    BitmapFactory.decodeFile(file.path, bounds)
    var sample = 1
    while (maxOf(bounds.outWidth, bounds.outHeight) / (sample * 2) >= PREVIEW_MAX_SIDE) sample *= 2
    return BitmapFactory.decodeFile(file.path, BitmapFactory.Options().apply { inSampleSize = sample })
        ?.asImageBitmap()
}

@Composable
private fun RowItem(
    row: RecognizedRow,
    place: RowPlace,
    actions: RowActions,
) {
    val colors = LocalAppColors.current
    val editable = !row.locked && !actions.saving
    val income = row.type == TransactionType.income
    val includeLabel = stringResource(R.string.recognize_include)
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .groupedRow(place, colors),
        verticalAlignment = Alignment.Top,
    ) {
        Checkbox(
            checked = row.included,
            onCheckedChange = { actions.onIncludedChange(row.draft, it) },
            enabled = editable,
            modifier = Modifier.semantics { contentDescription = includeLabel },
        )
        Column(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(Dimens.SPACE_1),
        ) {
            Text(
                text = formatMoney(
                    if (income) row.amountMinor else -row.amountMinor,
                    row.currency ?: actions.currency,
                    signed = true,
                ),
                style = MaterialTheme.typography.titleMedium,
                color = if (income) colors.income else colors.expense,
            )
            FieldError(row.fieldErrors[TransactionField.AMOUNT] ?: row.fieldErrors[TransactionField.TYPE])
            if (row.currencyMismatch) {
                Warning(stringResource(R.string.recognize_currency_mismatch, row.currency.orEmpty(), actions.currency))
            }

            OutlinedTextField(
                value = row.description,
                onValueChange = { actions.onDescriptionChange(row.draft, it) },
                label = { Text(stringResource(R.string.transaction_description)) },
                singleLine = true,
                enabled = editable,
                isError = TransactionField.DESCRIPTION in row.fieldErrors,
                supportingText = { FieldError(row.fieldErrors[TransactionField.DESCRIPTION]) },
                modifier = Modifier.fillMaxWidth(),
            )

            val date = row.date
            PickButton(
                text = when {
                    date == null -> stringResource(R.string.recognize_pick_date)
                    row.dateAssumed -> stringResource(R.string.recognize_date_assumed, formatFullDay(date))
                    else -> formatDay(date, actions.today)
                },
                attention = date == null || row.dateAssumed,
                enabled = editable,
            ) { actions.onPickDate(row.draft) }
            FieldError(row.fieldErrors[TransactionField.DATE])

            val category = actions.categories.firstOrNull { it.id == row.categoryId }
            PickButton(
                text = category?.path(actions.categories) ?: stringResource(R.string.recognize_pick_category),
                attention = category == null,
                enabled = editable,
                leading = category?.let { { CategoryAvatar(it.icon, it.color, it.name, Dimens.AVATAR_S) } },
            ) { actions.onPickCategory(row.draft) }
            FieldError(row.fieldErrors[TransactionField.CATEGORY])

            row.similarTo.firstOrNull()?.let { similar ->
                Warning(
                    stringResource(
                        R.string.recognize_similar,
                        similar.description,
                        formatDay(similar.date, actions.today),
                    ),
                )
            }

            Status(row, actions)
        }
    }
}

@Composable
private fun Status(
    row: RecognizedRow,
    actions: RowActions,
) {
    val colors = LocalAppColors.current
    when (val status = row.status) {
        RowStatus.Pending -> Unit

        RowStatus.Saving -> Text(stringResource(R.string.recognize_saving), style = MaterialTheme.typography.bodySmall)

        RowStatus.Saved -> Text(
            text = stringResource(R.string.recognize_saved),
            style = MaterialTheme.typography.bodySmall,
            color = colors.income,
        )

        is RowStatus.Failed -> Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = status.error.message(LocalContext.current.resources),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.error,
                modifier = Modifier.weight(1f),
            )
            TextButton(onClick = { actions.onRetryRow(row.draft) }, enabled = !actions.saving) {
                Text(stringResource(R.string.retry))
            }
        }
    }
}

@Composable
private fun PickButton(
    text: String,
    attention: Boolean,
    enabled: Boolean,
    modifier: Modifier = Modifier,
    leading: (@Composable () -> Unit)? = null,
    onClick: () -> Unit,
) {
    TextButton(onClick = onClick, enabled = enabled, modifier = modifier.heightIn(min = Dimens.TOUCH_MIN)) {
        leading?.let {
            it()
            Spacer(Modifier.width(Dimens.SPACE_2))
        }
        Text(text = text, color = if (attention && enabled) LocalAppColors.current.warning else Color.Unspecified)
    }
}

@Composable
private fun Warning(text: String) {
    Text(text = text, style = MaterialTheme.typography.bodySmall, color = LocalAppColors.current.warning)
}

@Composable
private fun Note(text: String) {
    Text(
        text = text,
        style = MaterialTheme.typography.bodySmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_1),
    )
}

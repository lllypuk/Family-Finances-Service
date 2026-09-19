package tech.shatrov.familyfinances.ui.overview

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.core.api.MonthTotals
import tech.shatrov.familyfinances.theme.Dimens
import tech.shatrov.familyfinances.theme.LocalAppColors
import tech.shatrov.familyfinances.ui.format.formatMoney
import tech.shatrov.familyfinances.ui.format.formatMonth
import tech.shatrov.familyfinances.ui.format.formatMonthName
import java.time.YearMonth

private val BARS_HEIGHT = 96.dp
private val BAR_GAP = 2.dp
private const val MUTED_ALPHA = 0.3f

/** Высоты столбиков в долях области, общий масштаб по максимуму обоих рядов. */
internal class BarGeometry(
    val income: List<Float>,
    val expenses: List<Float>,
)

internal fun barGeometry(months: List<MonthTotals>): BarGeometry {
    val high = months.maxOfOrNull { maxOf(it.incomeMinor, it.expensesMinor) }?.takeIf { it > 0 } ?: 1L
    return BarGeometry(
        income = months.map { it.incomeMinor.toFloat() / high },
        expenses = months.map { it.expensesMinor.toFloat() / high },
    )
}

/** Парные столбики дохода и расхода; месяцы вне [range] приглушены, тап выбирает месяц. */
@Composable
fun MonthlyBars(
    months: List<MonthTotals>,
    range: DateRange,
    currency: String,
    onSelectMonth: (YearMonth) -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    val geometry = barGeometry(months)
    Row(modifier = modifier.fillMaxWidth()) {
        months.forEachIndexed { index, totals ->
            val month = YearMonth.parse(totals.month)
            val inRange = !month.atEndOfMonth().isBefore(range.from) && !month.atDay(1).isAfter(range.to)
            val alpha = if (inRange) 1f else MUTED_ALPHA
            val description = stringResource(
                R.string.overview_bar_description,
                formatMonth(month.atDay(1)),
                formatMoney(totals.incomeMinor, currency),
                formatMoney(totals.expensesMinor, currency),
            )
            Column(
                modifier = Modifier
                    .weight(1f)
                    .clickable { onSelectMonth(month) }
                    .semantics { contentDescription = description },
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                BarPair(
                    income = geometry.income[index],
                    expenses = geometry.expenses[index],
                    incomeColor = colors.income.copy(alpha = alpha),
                    expenseColor = colors.expense.copy(alpha = alpha),
                )
                Text(
                    text = formatMonthName(month).take(1),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = alpha),
                    modifier = Modifier.padding(top = Dimens.SPACE_1),
                )
            }
        }
    }
}

@Composable
private fun BarPair(
    income: Float,
    expenses: Float,
    incomeColor: Color,
    expenseColor: Color,
) {
    Canvas(
        modifier = Modifier
            .fillMaxWidth()
            .height(BARS_HEIGHT)
            .padding(horizontal = BAR_GAP),
    ) {
        val gap = BAR_GAP.toPx()
        val width = (size.width - gap) / 2
        fun bar(
            share: Float,
            left: Float,
            color: Color,
        ) {
            val height = share * size.height
            if (height > 0f) drawRect(color, Offset(left, size.height - height), Size(width, height))
        }
        bar(income, 0f, incomeColor)
        bar(expenses, width + gap, expenseColor)
    }
}

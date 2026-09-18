package tech.shatrov.familyfinances.ui

import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDefaults
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.SelectableDates
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.ui.res.stringResource
import tech.shatrov.familyfinances.R
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset

/**
 * Календарь считает в UTC-полуночах, поэтому дата переводится через `ZoneOffset.UTC`.
 * [latest] — последний выбираемый день; `null` — без потолка.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun DatePickerSheet(
    date: LocalDate,
    onPick: (LocalDate) -> Unit,
    onDismiss: () -> Unit,
    latest: LocalDate? = null,
) {
    val picker = rememberDatePickerState(
        initialSelectedDateMillis = date.utcMillis(),
        yearRange = DatePickerDefaults.YearRange.let { if (latest == null) it else it.first..latest.year },
        selectableDates = if (latest == null) {
            DatePickerDefaults.AllDates
        } else {
            object : SelectableDates {
                override fun isSelectableDate(utcTimeMillis: Long): Boolean = utcTimeMillis <= latest.utcMillis()
            }
        },
    )
    DatePickerDialog(
        onDismissRequest = onDismiss,
        confirmButton = {
            TextButton(onClick = {
                val millis = picker.selectedDateMillis
                if (millis == null) {
                    onDismiss()
                } else {
                    onPick(Instant.ofEpochMilli(millis).atZone(ZoneOffset.UTC).toLocalDate())
                }
            }) {
                Text(stringResource(R.string.transaction_date_pick))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(stringResource(R.string.cancel)) }
        },
    ) {
        DatePicker(state = picker)
    }
}

private fun LocalDate.utcMillis(): Long = atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli()

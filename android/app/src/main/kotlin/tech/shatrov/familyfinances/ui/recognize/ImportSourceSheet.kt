package tech.shatrov.familyfinances.ui.recognize

import android.content.ActivityNotFoundException
import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import kotlinx.coroutines.launch
import tech.shatrov.familyfinances.R
import tech.shatrov.familyfinances.theme.Dimens

/** Запуск галереи и камеры; результат уходит в `onPicked` уже без листа. */
class ImportLaunchers(
    val gallery: () -> Unit,
    val camera: () -> Unit,
)

/**
 * Регистрируется в корне, а не в листе и не во вкладке: лист закрывается до ответа пикера, а после
 * смерти процесса за камерой корень восстанавливается первым — иначе снимок некому принять.
 */
@Composable
fun rememberImportLaunchers(onPicked: (List<Uri>) -> Unit): ImportLaunchers {
    val context = LocalContext.current
    var cameraUri by rememberSaveable { mutableStateOf<String?>(null) }
    val gallery = rememberLauncherForActivityResult(
        ActivityResultContracts.PickMultipleVisualMedia(ImportFiles.MAX_IMAGES),
    ) { uris -> if (uris.isNotEmpty()) onPicked(uris) }
    val camera = rememberLauncherForActivityResult(ActivityResultContracts.TakePicture()) { taken ->
        val uri = cameraUri
        cameraUri = null
        if (taken && uri != null) onPicked(listOf(Uri.parse(uri)))
    }
    return remember(gallery, camera) {
        ImportLaunchers(
            gallery = {
                gallery.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))
            },
            camera = {
                val uri = ImportFiles.cameraUri(context)
                cameraUri = uri.toString()
                try {
                    camera.launch(uri)
                } catch (_: ActivityNotFoundException) {
                    cameraUri = null
                }
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ImportSourceSheet(
    launchers: ImportLaunchers,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState()
    val scope = rememberCoroutineScope()
    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        ImportSourceSheetContent { launch ->
            scope.launch { sheetState.hide() }.invokeOnCompletion {
                if (!sheetState.isVisible) {
                    onDismiss()
                    launch(launchers)
                }
            }
        }
    }
}

// Как у листа категорий: `ModalBottomSheet` в Robolectric ненадёжен, проверяется тело.
@Composable
internal fun ImportSourceSheetContent(onChoose: ((ImportLaunchers) -> Unit) -> Unit) {
    Column(modifier = Modifier.fillMaxWidth()) {
        Text(
            text = stringResource(R.string.recognize_scan),
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(horizontal = Dimens.SPACE_4, vertical = Dimens.SPACE_2),
        )
        SourceRow(stringResource(R.string.recognize_from_gallery)) { onChoose { it.gallery() } }
        SourceRow(stringResource(R.string.recognize_take_photo)) { onChoose { it.camera() } }
    }
}

@Composable
private fun SourceRow(
    label: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .heightIn(min = Dimens.TOUCH_MIN)
            .padding(horizontal = Dimens.SPACE_4),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(text = label, style = MaterialTheme.typography.bodyLarge)
    }
}

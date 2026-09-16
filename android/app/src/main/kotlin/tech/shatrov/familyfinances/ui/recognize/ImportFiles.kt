package tech.shatrov.familyfinances.ui.recognize

import android.content.ContentResolver
import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.media.ExifInterface
import android.net.Uri
import androidx.core.content.FileProvider
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.ByteArrayOutputStream
import java.io.File
import java.io.IOException
import java.util.UUID

/** Картинка импорта: готовый JPEG или отказ по одному URI. */
sealed interface ImportImage {
    val uri: Uri

    data class Ready(
        override val uri: Uri,
        val file: File,
    ) : ImportImage

    data class Failed(
        override val uri: Uri,
        val reason: ImportFailure,
    ) : ImportImage
}

enum class ImportFailure { UNREADABLE, NOT_IMAGE, TOO_LARGE }

data class PreparedImport(
    val images: List<ImportImage>,
    val dropped: Int,
)

/** URI из share, галереи и камеры → JPEG под лимиты сервера в `cacheDir/import/<importId>/`. */
object ImportFiles {
    const val MAX_IMAGES = 5
    const val MAX_SIDE = 2048
    const val MAX_BYTES = 2 shl 20

    private const val START_QUALITY = 85
    private const val MIN_QUALITY = 60
    private const val QUALITY_STEP = 5
    private const val ROOT = "import"
    private const val CAMERA = "camera"
    private const val CAMERA_TTL_MS = 24L * 60 * 60 * 1000

    suspend fun prepare(
        context: Context,
        importId: UUID,
        uris: List<Uri>,
    ): PreparedImport = withContext(Dispatchers.IO) {
        val dir = importDir(context, importId).apply { mkdirs() }
        val images = uris.take(MAX_IMAGES).mapIndexed { i, uri ->
            convert(context.contentResolver, uri, File(dir, "$i.jpg"))
        }
        PreparedImport(images, dropped = (uris.size - MAX_IMAGES).coerceAtLeast(0))
    }

    fun discard(
        context: Context,
        importId: UUID,
    ) {
        importDir(context, importId).deleteRecursively()
    }

    /**
     * Снимок камеры переживает смерть процесса, пока камера открыта, поэтому его каталог
     * чистится по возрасту, а не целиком.
     */
    fun sweep(
        context: Context,
        now: Long = System.currentTimeMillis(),
    ) {
        root(context).listFiles().orEmpty().forEach { entry ->
            if (entry.name == CAMERA) {
                entry.listFiles().orEmpty().filter { now - it.lastModified() > CAMERA_TTL_MS }.forEach { it.delete() }
            } else {
                entry.deleteRecursively()
            }
        }
    }

    /** URI для `TakePicture`; каталог создаётся здесь — `FileProvider` без него файл не откроет. */
    fun cameraUri(context: Context): Uri {
        val dir = File(root(context), CAMERA).apply { mkdirs() }
        return FileProvider.getUriForFile(context, authority(context), File(dir, "${UUID.randomUUID()}.jpg"))
    }

    fun authority(context: Context): String = "${context.packageName}.files"

    private fun root(context: Context) = File(context.cacheDir, ROOT)

    private fun importDir(
        context: Context,
        importId: UUID,
    ) = File(root(context), importId.toString())

    private fun convert(
        resolver: ContentResolver,
        uri: Uri,
        target: File,
    ): ImportImage {
        val bitmap = try {
            decode(resolver, uri)
        } catch (_: IOException) {
            return ImportImage.Failed(uri, ImportFailure.UNREADABLE)
        } catch (_: SecurityException) {
            return ImportImage.Failed(uri, ImportFailure.UNREADABLE)
        } ?: return ImportImage.Failed(uri, ImportFailure.NOT_IMAGE)

        val out = ByteArrayOutputStream()
        var quality = START_QUALITY
        while (quality >= MIN_QUALITY) {
            out.reset()
            bitmap.compress(Bitmap.CompressFormat.JPEG, quality, out)
            if (out.size() <= MAX_BYTES) {
                bitmap.recycle()
                target.writeBytes(out.toByteArray())
                return ImportImage.Ready(uri, target)
            }
            quality -= QUALITY_STEP
        }
        bitmap.recycle()
        return ImportImage.Failed(uri, ImportFailure.TOO_LARGE)
    }

    private fun decode(
        resolver: ContentResolver,
        uri: Uri,
    ): Bitmap? {
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        open(resolver, uri).use { BitmapFactory.decodeStream(it, null, bounds) }
        if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return null

        val longest = maxOf(bounds.outWidth, bounds.outHeight)
        var sample = 1
        while (longest / (sample * 2) >= MAX_SIDE) sample *= 2
        val decoded = open(resolver, uri).use {
            BitmapFactory.decodeStream(it, null, BitmapFactory.Options().apply { inSampleSize = sample })
        } ?: return null

        val scale = MAX_SIDE.toFloat() / maxOf(decoded.width, decoded.height)
        val matrix = Matrix().apply { if (scale < 1f) setScale(scale, scale) }
        orient(matrix, orientation(resolver, uri))
        if (matrix.isIdentity) return decoded
        val transformed = Bitmap.createBitmap(decoded, 0, 0, decoded.width, decoded.height, matrix, true)
        if (transformed !== decoded) decoded.recycle()
        return transformed
    }

    private fun open(
        resolver: ContentResolver,
        uri: Uri,
    ) = resolver.openInputStream(uri) ?: throw IOException("no stream for $uri")

    // EXIF в PNG и прочем, что ExifInterface не читает, — не повод отказать картинке.
    private fun orientation(
        resolver: ContentResolver,
        uri: Uri,
    ): Int = try {
        open(resolver, uri).use {
            ExifInterface(it).getAttributeInt(ExifInterface.TAG_ORIENTATION, ExifInterface.ORIENTATION_NORMAL)
        }
    } catch (_: IOException) {
        ExifInterface.ORIENTATION_NORMAL
    }

    private fun orient(
        matrix: Matrix,
        orientation: Int,
    ) {
        when (orientation) {
            ExifInterface.ORIENTATION_FLIP_HORIZONTAL -> matrix.postScale(-1f, 1f)
            ExifInterface.ORIENTATION_ROTATE_180 -> matrix.postRotate(180f)
            ExifInterface.ORIENTATION_FLIP_VERTICAL -> matrix.postScale(1f, -1f)
            ExifInterface.ORIENTATION_TRANSPOSE -> matrix.apply { postRotate(90f) }.postScale(-1f, 1f)
            ExifInterface.ORIENTATION_ROTATE_90 -> matrix.postRotate(90f)
            ExifInterface.ORIENTATION_TRANSVERSE -> matrix.apply { postRotate(-90f) }.postScale(-1f, 1f)
            ExifInterface.ORIENTATION_ROTATE_270 -> matrix.postRotate(-90f)
        }
    }
}

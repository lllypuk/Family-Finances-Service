package tech.shatrov.familyfinances.ui.recognize

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.media.ExifInterface
import android.net.Uri
import androidx.test.core.app.ApplicationProvider
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import tech.shatrov.familyfinances.ROBOLECTRIC_SDK
import java.io.File
import java.util.UUID

// Без нативной графики Robolectric не кодирует и не декодирует настоящие байты.
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
@GraphicsMode(GraphicsMode.Mode.NATIVE)
class ImportFilesTest {
    private lateinit var context: Context
    private lateinit var src: File

    @Before
    fun setUp() {
        context = ApplicationProvider.getApplicationContext()
        src = File(context.filesDir, "src").apply { mkdirs() }
    }

    private fun image(
        name: String,
        width: Int,
        height: Int,
        format: Bitmap.CompressFormat,
    ): File {
        val bitmap = Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888)
        val canvas = Canvas(bitmap)
        canvas.drawColor(Color.WHITE)
        canvas.drawRect(0f, 0f, width / 2f, height.toFloat(), Paint().apply { color = Color.RED })
        return File(src, name).apply { outputStream().use { bitmap.compress(format, 90, it) } }
    }

    private fun bounds(file: File) = BitmapFactory.Options().apply {
        inJustDecodeBounds = true
        BitmapFactory.decodeFile(file.path, this)
    }

    @Test
    fun screenshotBecomesJpegWithinLimits() = runTest {
        val png = image("shot.png", 1080, 2400, Bitmap.CompressFormat.PNG)

        val result = ImportFiles.prepare(context, UUID.randomUUID(), listOf(Uri.fromFile(png)))

        val ready = result.images.single() as ImportImage.Ready
        val out = bounds(ready.file)
        assertEquals("image/jpeg", out.outMimeType)
        assertEquals(2048, maxOf(out.outWidth, out.outHeight))
        assertEquals(922, out.outWidth)
        assertTrue(ready.file.length() <= ImportFiles.MAX_BYTES)
    }

    // ROTATE_90 — по часовой: левая красная половина уходит наверх.
    @Test
    fun exifOrientationIsApplied() = runTest {
        val jpeg = image("rotated.jpg", 200, 100, Bitmap.CompressFormat.JPEG)
        ExifInterface(jpeg.path).apply {
            setAttribute(ExifInterface.TAG_ORIENTATION, ExifInterface.ORIENTATION_ROTATE_90.toString())
            saveAttributes()
        }

        val result = ImportFiles.prepare(context, UUID.randomUUID(), listOf(Uri.fromFile(jpeg)))

        val file = (result.images.single() as ImportImage.Ready).file
        val bitmap = BitmapFactory.decodeFile(file.path)
        assertEquals(100, bitmap.width)
        assertEquals(200, bitmap.height)
        assertTrue(Color.red(bitmap.getPixel(50, 20)) > 200 && Color.green(bitmap.getPixel(50, 20)) < 60)
        assertTrue(Color.green(bitmap.getPixel(50, 180)) > 200)
    }

    @Test
    fun sixthUriIsDropped() = runTest {
        val uris = (0..5).map { Uri.fromFile(image("$it.png", 40, 40, Bitmap.CompressFormat.PNG)) }

        val result = ImportFiles.prepare(context, UUID.randomUUID(), uris)

        assertEquals(uris.take(5), result.images.map { it.uri })
        assertTrue(result.images.all { it is ImportImage.Ready })
        assertEquals(1, result.dropped)
    }

    @Test
    fun badUriFailsOnlyItsRow() = runTest {
        val junk = File(src, "junk.bin").apply { writeText("not an image") }
        val missing = File(src, "missing.png")
        val good = image("good.png", 40, 40, Bitmap.CompressFormat.PNG)

        val result = ImportFiles.prepare(
            context,
            UUID.randomUUID(),
            listOf(Uri.fromFile(junk), Uri.fromFile(missing), Uri.fromFile(good)),
        )

        assertEquals(ImportImage.Failed(Uri.fromFile(junk), ImportFailure.NOT_IMAGE), result.images[0])
        assertEquals(ImportImage.Failed(Uri.fromFile(missing), ImportFailure.UNREADABLE), result.images[1])
        assertTrue(result.images[2] is ImportImage.Ready)
    }

    // Файл на месте каталога: запись падает так же, как после удаления каталога посреди пережатия.
    @Test
    fun unwritableTargetFailsRowWithoutThrowing() = runTest {
        val importId = UUID.randomUUID()
        File(context.filesDir, "import").mkdirs()
        File(context.filesDir, "import/$importId").writeText("x")
        val png = Uri.fromFile(image("a.png", 40, 40, Bitmap.CompressFormat.PNG))

        val result = ImportFiles.prepare(context, importId, listOf(png))

        assertEquals(ImportImage.Failed(png, ImportFailure.UNREADABLE), result.images.single())
    }

    @Test
    fun sweepKeepsOnlyFreshCameraShotsInCache() = runTest {
        val legacy = File(context.cacheDir, "import/${UUID.randomUUID()}").apply { mkdirs() }
        File(legacy, "0.jpg").writeText("x")
        val camera = File(context.cacheDir, "import/camera").apply { mkdirs() }
        val now = System.currentTimeMillis()
        val stale = File(camera, "stale.jpg").apply { writeText("x") }.apply { setLastModified(now - 2 * DAY_MS) }
        val fresh = File(camera, "fresh.jpg").apply { writeText("x") }

        ImportFiles.sweep(context, now)

        assertFalse(legacy.exists())
        assertFalse(stale.exists())
        assertTrue(fresh.exists())
    }

    @Test
    fun sweepKeepsOnlyImportsWithLiveJournal() = runTest {
        val now = System.currentTimeMillis()
        val journals = ImportJournalStore(ImportFiles.filesRoot(context)) { now }
        suspend fun prepared(): UUID {
            val id = UUID.randomUUID()
            ImportFiles.prepare(context, id, listOf(Uri.fromFile(image("$id.png", 40, 40, Bitmap.CompressFormat.PNG))))
            return id
        }
        fun journal(
            id: UUID,
            updatedAt: Long = now,
            version: Int = JOURNAL_VERSION,
        ) = journals.write(
            ImportJournal(version, id.toString(), updatedAt, emptyList(), 0, false, null, null, emptyList(), 0),
        )

        val live = prepared().also { journal(it) }
        val bare = prepared()
        val expired = prepared().also { journal(it, updatedAt = now - 2 * DAY_MS) }
        val foreign = prepared().also { journal(it, version = JOURNAL_VERSION + 1) }
        val garbage = prepared().also { dir(it).resolve("journal.json").writeText("{") }
        val stray = File(ImportFiles.filesRoot(context), "stray").apply { writeText("x") }

        ImportFiles.sweep(context, now)

        assertTrue(dir(live).resolve("0.jpg").exists())
        listOf(bare, expired, foreign, garbage).forEach { assertFalse(it.toString(), dir(it).exists()) }
        assertFalse(stray.exists())
    }

    private fun dir(id: UUID) = File(context.filesDir, "import/$id")

    @Test
    fun cameraUriGoesThroughFileProvider() {
        val uri = ImportFiles.cameraUri(context)

        assertEquals("content", uri.scheme)
        assertEquals(ImportFiles.authority(context), uri.authority)
        assertTrue(uri.path!!.startsWith("/camera/"))
        assertTrue(File(context.cacheDir, "import/camera").isDirectory)
    }

    private companion object {
        const val DAY_MS = 24L * 60 * 60 * 1000
    }
}

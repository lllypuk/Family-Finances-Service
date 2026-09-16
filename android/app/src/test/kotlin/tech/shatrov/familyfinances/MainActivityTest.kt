package tech.shatrov.familyfinances

import android.content.ClipData
import android.content.Intent
import android.net.Uri
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.Robolectric
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

// Robolectric 4.16 не знает SDK 37, поэтому sdk задан явно.
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class MainActivityTest {
    private val imports: ImportStore
        get() = ApplicationProvider.getApplicationContext<FamilyFinancesApp>().graph.imports

    @Test
    fun activityStartsWithComposeContent() {
        Robolectric.buildActivity(MainActivity::class.java).setup().use { controller ->
            assertTrue(controller.get().window.decorView.isAttachedToWindow)
        }
    }

    @Test
    fun recreationDoesNotOfferTheSameShareAgain() {
        Robolectric.buildActivity(MainActivity::class.java, send(shot("a"))).setup().use { controller ->
            val offered = imports.pending.value
            assertNotNull(offered)
            assertFalse(controller.get().intent.hasExtra(Intent.EXTRA_STREAM))

            controller.recreate()

            assertEquals(offered, imports.pending.value)
        }
    }

    @Test
    fun shareIntoOpenAppIsOffered() {
        Robolectric.buildActivity(MainActivity::class.java).setup().use { controller ->
            assertNull(imports.pending.value)

            controller.newIntent(send(shot("a")))
            val first = imports.pending.value
            controller.newIntent(send(shot("b")))

            assertNotNull(first)
            assertNotEquals(first, imports.pending.value)
        }
    }

    @Test
    fun launchFromRecentsIsNotAShare() {
        val stale = send(shot("a")).addFlags(Intent.FLAG_ACTIVITY_LAUNCHED_FROM_HISTORY)
        Robolectric.buildActivity(MainActivity::class.java, stale).setup().use {
            assertNull(imports.pending.value)
        }
    }

    @Test
    fun multipleShareMergesStreamAndClipData() {
        val intent = Intent(Intent.ACTION_SEND_MULTIPLE)
            .setType("image/png")
            .putParcelableArrayListExtra(Intent.EXTRA_STREAM, arrayListOf(shot("a"), shot("b")))
        intent.clipData = ClipData.newRawUri(null, shot("b")).apply { addItem(ClipData.Item(shot("c"))) }

        assertEquals(listOf(shot("a"), shot("b"), shot("c")), sharedImages(intent))
        assertEquals(emptyList<Uri>(), sharedImages(Intent(Intent.ACTION_VIEW, shot("a"))))
    }

    private fun shot(name: String): Uri = Uri.parse("content://gallery/$name")

    private fun send(uri: Uri): Intent = Intent(Intent.ACTION_SEND)
        .setType("image/png")
        .putExtra(Intent.EXTRA_STREAM, uri)
}

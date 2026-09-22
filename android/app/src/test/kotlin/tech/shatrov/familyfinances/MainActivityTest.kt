package tech.shatrov.familyfinances

import android.content.ClipData
import android.content.ComponentName
import android.content.Intent
import android.content.pm.ActivityInfo
import android.net.Uri
import androidx.compose.ui.graphics.toArgb
import androidx.core.view.WindowCompat
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
import tech.shatrov.familyfinances.theme.AppColors
import tech.shatrov.familyfinances.theme.DarkColors
import tech.shatrov.familyfinances.theme.LightColors
import tech.shatrov.familyfinances.theme.ThemeMode

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

    // Share из чужого приложения запускается в задаче отправителя: при singleTop там появилась бы
    // вторая активити, а не onNewIntent в открытой.
    @Test
    fun shareReusesTheAppTask() {
        val app = ApplicationProvider.getApplicationContext<FamilyFinancesApp>()
        val info = app.packageManager.getActivityInfo(ComponentName(app, MainActivity::class.java), 0)

        assertEquals(ActivityInfo.LAUNCH_SINGLE_TASK, info.launchMode)
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

    @Test
    @Config(qualifiers = "+notnight")
    fun systemModeInLightPhoneStartsLight() {
        assertWindow(expected = LightColors, mode = ThemeMode.System)
    }

    @Test
    @Config(qualifiers = "+night")
    fun systemModeInDarkPhoneStartsDark() {
        assertWindow(expected = DarkColors, mode = ThemeMode.System)
    }

    @Test
    @Config(qualifiers = "+notnight")
    fun darkOverrideInLightPhoneStartsDark() {
        assertWindow(expected = DarkColors, mode = ThemeMode.Dark)
    }

    @Test
    @Config(qualifiers = "+night")
    fun lightOverrideInDarkPhoneStartsLight() {
        assertWindow(expected = LightColors, mode = ThemeMode.Light)
    }

    // Фон окна сверяется с палитрой Compose: расхождение colors.xml вернуло бы вспышку молча.
    private fun assertWindow(
        expected: AppColors,
        mode: ThemeMode,
    ) {
        ApplicationProvider.getApplicationContext<FamilyFinancesApp>().graph.theme.write(mode)
        Robolectric.buildActivity(MainActivity::class.java).setup().use { controller ->
            val activity = controller.get()
            val attrs = activity.theme.obtainStyledAttributes(intArrayOf(android.R.attr.windowBackground))
            val background = attrs.getColor(0, 0)
            attrs.recycle()
            assertEquals(expected.canvas.toArgb(), background)

            val bars = WindowCompat.getInsetsController(activity.window, activity.window.decorView)
            assertEquals(expected == LightColors, bars.isAppearanceLightStatusBars)
            assertEquals(expected == LightColors, bars.isAppearanceLightNavigationBars)
        }
    }

    private fun shot(name: String): Uri = Uri.parse("content://gallery/$name")

    private fun send(uri: Uri): Intent = Intent(Intent.ACTION_SEND)
        .setType("image/png")
        .putExtra(Intent.EXTRA_STREAM, uri)
}

package tech.shatrov.familyfinances

import android.content.Context
import androidx.core.content.edit
import androidx.test.core.app.ApplicationProvider
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import tech.shatrov.familyfinances.theme.ThemeMode

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class ThemeStoreTest {
    private val context: Context = ApplicationProvider.getApplicationContext()

    @Test
    fun emptyPrefsStartWithSystem() {
        assertEquals(ThemeMode.System, PrefsThemeStore(context).mode.value)
    }

    @Test
    fun writtenModeIsReadByNewInstance() {
        PrefsThemeStore(context).write(ThemeMode.Dark)

        assertEquals(ThemeMode.Dark, PrefsThemeStore(context).mode.value)
    }

    @Test
    fun unknownStoredValueFallsBackToSystem() {
        context.getSharedPreferences("ui", Context.MODE_PRIVATE).edit { putString("theme_mode", "Amoled") }

        assertEquals(ThemeMode.System, PrefsThemeStore(context).mode.value)
    }

    @Test
    fun writeUpdatesModeOfSameInstance() {
        val store = PrefsThemeStore(context)

        store.write(ThemeMode.Light)

        assertEquals(ThemeMode.Light, store.mode.value)
    }

    @Test
    fun memoryStoreKeepsInitialAndWrites() {
        val store = MemoryThemeStore(ThemeMode.Dark)
        assertEquals(ThemeMode.Dark, store.mode.value)

        store.write(ThemeMode.System)

        assertEquals(ThemeMode.System, store.mode.value)
    }
}

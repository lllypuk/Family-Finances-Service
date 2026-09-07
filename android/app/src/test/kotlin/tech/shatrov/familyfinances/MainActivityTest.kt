package tech.shatrov.familyfinances

import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.Robolectric
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config

// Дымовой тест каркаса: проверяет не экран, а то, что Robolectric поднимает activity с Compose.
// Robolectric 4.16 не знает SDK 37, поэтому sdk задан явно.
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [36])
class MainActivityTest {
    @Test
    fun activityStartsWithComposeContent() {
        Robolectric.buildActivity(MainActivity::class.java).setup().use { controller ->
            assertTrue(controller.get().window.decorView.isAttachedToWindow)
        }
    }
}

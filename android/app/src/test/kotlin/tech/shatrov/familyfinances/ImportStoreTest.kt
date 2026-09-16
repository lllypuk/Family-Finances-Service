package tech.shatrov.familyfinances

import android.net.Uri
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import java.util.UUID

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [ROBOLECTRIC_SDK])
class ImportStoreTest {
    private val store = ImportStore()
    private val shot = Uri.parse("content://media/external/images/1")

    @Test
    fun offerIsClaimedExactlyOnce() {
        val id = store.offer(listOf(shot))
        assertEquals(id, store.pending.value)

        assertEquals(listOf(shot), store.claim(id)?.uris)
        assertNull(store.claim(id))
        assertNull(store.pending.value)
    }

    @Test
    fun claimOfForeignIdLeavesOfferInPlace() {
        val id = store.offer(listOf(shot))

        assertNull(store.claim(UUID.randomUUID()))
        assertEquals(id, store.pending.value)
    }

    @Test
    fun laterOfferReplacesUnclaimedOne() {
        val first = store.offer(listOf(shot))
        val second = store.offer(emptyList())

        assertNull(store.claim(first))
        assertEquals(second, store.pending.value)
    }

    @Test
    fun secondShareWaitsWhileImportIsHeld() {
        val first = store.offer(listOf(shot))
        store.claim(first)
        store.hold(first)

        val second = store.offer(listOf(shot))
        assertNull(store.pending.value)
        assertTrue(store.waiting.value)

        store.release(first)
        assertEquals(second, store.pending.value)
        assertFalse(store.waiting.value)
    }
}

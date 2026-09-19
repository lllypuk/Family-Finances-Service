package tech.shatrov.familyfinances.ui.recognize

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import tech.shatrov.familyfinances.core.api.RecognizeResult
import tech.shatrov.familyfinances.core.api.RecognizedTransaction
import tech.shatrov.familyfinances.core.api.TransactionType
import java.io.File
import java.time.LocalDate
import java.util.UUID

class ImportJournalTest {
    @get:Rule
    val tmp = TemporaryFolder()

    private lateinit var root: File
    private var now = NOW
    private lateinit var store: ImportJournalStore

    @Before
    fun setUp() {
        root = tmp.newFolder("import")
        store = ImportJournalStore(root) { now }
    }

    private fun journal(
        id: UUID = UUID.randomUUID(),
        updatedAt: Long = NOW,
    ) = ImportJournal(
        importId = id.toString(),
        updatedAt = updatedAt,
        images = listOf(
            JournalImage("content://a/1", file = "0.jpg"),
            JournalImage("content://a/2", failure = ImportFailure.NOT_IMAGE),
        ),
        dropped = 1,
        recognizing = false,
        result = RecognizeResult(
            items = listOf(
                RecognizedTransaction(
                    source = 0,
                    amountMinor = 45_000,
                    currency = null,
                    type = TransactionType.expense,
                    date = LocalDate.of(2026, 9, 18),
                    dateAssumed = false,
                    description = "Шаверма",
                    categoryId = CATEGORY,
                    similar = emptyList(),
                ),
            ),
            incomplete = false,
            model = "m",
        ),
        accountId = null,
        rows = listOf(
            JournalRow(
                UUID.randomUUID().toString(),
                true,
                "2026-09-18",
                CATEGORY.toString(),
                "Шаверма",
                JournalRowStatus.SAVING,
            ),
        ),
        savedCount = 0,
    )

    private fun file(id: String) = File(root, "$id/journal.json")

    @Test
    fun writtenJournalReadsBack() {
        val written = journal()

        store.write(written)

        val read = store.read(UUID.fromString(written.importId))
        assertEquals(written, read)
        assertEquals(LocalDate.of(2026, 9, 18), read!!.result!!.items.single().date)
        assertEquals(CATEGORY, read.result!!.items.single().categoryId)
    }

    @Test
    fun interruptedWriteKeepsPreviousJournal() {
        val first = journal()
        store.write(first)
        File(root, "${first.importId}/journal.json.tmp").writeText("{\"version\":1,\"impo")

        assertEquals(first, store.read(UUID.fromString(first.importId)))
    }

    @Test
    fun garbageReadsAsNull() {
        val id = UUID.randomUUID()
        file(id.toString()).apply { parentFile!!.mkdirs() }.writeText("not json")

        assertNull(store.read(id))
        assertNull(store.latest())
    }

    @Test
    fun foreignVersionReadsAsNull() {
        val j = journal().copy(version = JOURNAL_VERSION + 1)
        store.write(j)

        assertNull(store.read(UUID.fromString(j.importId)))
    }

    @Test
    fun versionIsWrittenToDisk() {
        val j = journal()
        store.write(j)

        assertTrue(file(j.importId).readText().contains("\"version\":$JOURNAL_VERSION"))
    }

    @Test
    fun writeAfterRemovalDoesNotReviveJournal() {
        val removals = listOf<(UUID) -> Unit>(
            { store.delete(it) },
            { store.deleteOthers(keep = UUID.randomUUID()) },
            { store.deleteAll() },
            { store.close(it) },
        )
        removals.forEach { remove ->
            val j = journal()
            val id = UUID.fromString(j.importId)
            store.write(j)

            remove(id)
            store.write(j)

            assertNull(store.read(id))
        }
    }

    @Test
    fun closeKeepsImages() {
        val j = journal()
        store.write(j)
        val image = File(root, "${j.importId}/0.jpg").apply { writeText("x") }

        store.close(UUID.fromString(j.importId))

        assertNull(store.read(UUID.fromString(j.importId)))
        assertTrue(image.exists())
    }

    @Test
    fun missingJournalReadsAsNull() {
        assertNull(store.read(UUID.randomUUID()))
        assertNull(ImportJournalStore(File(root, "absent")).latest())
    }

    @Test
    fun latestSkipsExpiredAndPicksNewest() {
        val old = journal(updatedAt = NOW - 2 * JOURNAL_TTL_MS)
        val earlier = journal(updatedAt = NOW - 1_000)
        val newest = journal(updatedAt = NOW - 10)
        listOf(old, earlier, newest).forEach(store::write)

        assertEquals(newest, store.latest())
        assertNull(store.read(UUID.fromString(old.importId)))

        now = NOW + JOURNAL_TTL_MS
        assertNull(store.latest())
    }

    @Test
    fun deleteRemovesOnlyItsDirectory() {
        val a = journal()
        val b = journal()
        store.write(a)
        store.write(b)

        store.delete(UUID.fromString(a.importId))

        assertNull(store.read(UUID.fromString(a.importId)))
        assertEquals(b, store.read(UUID.fromString(b.importId)))
    }

    @Test
    fun deleteOthersKeepsOne() {
        val keep = journal()
        store.write(keep)
        store.write(journal())
        File(root, "stray").mkdirs()

        store.deleteOthers(UUID.fromString(keep.importId))

        assertEquals(listOf(keep.importId), root.list()!!.toList())
    }

    @Test
    fun deleteAllRemovesRoot() {
        store.write(journal())

        store.deleteAll()

        assertEquals(false, root.exists())
        assertNull(store.latest())
    }

    private companion object {
        const val NOW = 1_800_000_000_000L
        val CATEGORY: UUID = UUID.fromString("11111111-2222-3333-4444-555555555555")
    }
}

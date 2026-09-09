package com.lord1egypt.pocketclaw

import java.io.File
import java.nio.file.Files
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Before
import org.junit.Test

/**
 * The Core private-state migration, over a real temporary filesDir.
 *
 * The states this covers are the ones an install actually reaches on upgrade,
 * and each of them either preserves the user's provider keys and channel tokens
 * or is the reason they are lost. `resolve` takes a filesDir rather than a
 * Context precisely so they can be reached here rather than only on a device.
 */
class PocketClawCoreStateTest {
    private lateinit var filesDir: File

    private val legacy: File get() = File(filesDir, "picoclaw")
    private val canonical: File get() = File(filesDir, PocketClawCoreState.CANONICAL_DIR_NAME)

    @Before
    fun setUp() {
        filesDir = Files.createTempDirectory("pocketclaw-files").toFile()
    }

    @After
    fun tearDown() {
        filesDir.deleteRecursively()
    }

    private fun seed(directory: File, vararg entries: Pair<String, String>) {
        directory.mkdirs()
        for ((relative, content) in entries) {
            val file = File(directory, relative)
            file.parentFile?.mkdirs()
            file.writeText(content)
        }
    }

    @Test
    fun `legacy alone is migrated to the canonical directory`() {
        seed(legacy, "config.json" to "{\"a\":1}")

        val resolved = PocketClawCoreState.resolve(filesDir)

        assertEquals(canonical, resolved)
        assertTrue(canonical.isDirectory)
        assertFalse("the legacy directory must not survive a completed move", legacy.exists())
        assertEquals("{\"a\":1}", File(canonical, "config.json").readText())
    }

    @Test
    fun `the complete tree survives, not only the files this build knows`() {
        // Core's state root is not a manifest of two filenames. A migration that
        // enumerated config.json and .security.yml would silently drop whatever
        // Core writes next, and the loss would surface much later as an install
        // that had quietly forgotten something.
        seed(
            legacy,
            "config.json" to "{}",
            ".security.yml" to "api_key: secret",
            ".hidden-state" to "hidden",
            "unknown-future-file.dat" to "unknown",
            "runtime/metadata.json" to "{\"managed\":true}",
            "runtime/nested/deeper/leaf.txt" to "leaf",
        )

        PocketClawCoreState.resolve(filesDir)

        assertEquals("{}", File(canonical, "config.json").readText())
        assertEquals("api_key: secret", File(canonical, ".security.yml").readText())
        assertEquals("hidden", File(canonical, ".hidden-state").readText())
        assertEquals("unknown", File(canonical, "unknown-future-file.dat").readText())
        assertEquals("{\"managed\":true}", File(canonical, "runtime/metadata.json").readText())
        assertEquals("leaf", File(canonical, "runtime/nested/deeper/leaf.txt").readText())
    }

    @Test
    fun `canonical alone is used untouched`() {
        seed(canonical, "config.json" to "{\"canonical\":true}")

        val resolved = PocketClawCoreState.resolve(filesDir)

        assertEquals(canonical, resolved)
        assertEquals("{\"canonical\":true}", File(canonical, "config.json").readText())
        assertFalse("nothing may create the legacy directory", legacy.exists())
    }

    @Test
    fun `a fresh install starts canonical and never creates the legacy directory`() {
        val resolved = PocketClawCoreState.resolve(filesDir)

        assertEquals(canonical, resolved)
        assertTrue(canonical.isDirectory)
        assertFalse(legacy.exists())
    }

    @Test
    fun `both present fails closed and changes nothing`() {
        // An interrupted migration, or a downgrade that let an older build
        // onboard fresh against a migrated install. The two directories can
        // hold different provider keys, and nothing on disk says which the user
        // meant — so this is reported, not guessed at.
        seed(legacy, "config.json" to "{\"legacy\":true}", ".security.yml" to "legacy secret")
        seed(canonical, "config.json" to "{\"canonical\":true}")

        try {
            PocketClawCoreState.resolve(filesDir)
            fail("a both-present state must not resolve to a usable directory")
        } catch (e: PocketClawCoreState.MigrationException) {
            assertTrue(e.message!!.contains("both"))
        }

        assertEquals("{\"legacy\":true}", File(legacy, "config.json").readText())
        assertEquals("legacy secret", File(legacy, ".security.yml").readText())
        assertEquals("{\"canonical\":true}", File(canonical, "config.json").readText())
    }

    @Test
    fun `a failed migration leaves the legacy state intact`() {
        // A file where the canonical directory belongs: rename(2) cannot
        // replace it, which stands in for the filesystem faults that make a
        // move fail on a device. What matters is that the only copy of the
        // user's state is still on disk afterwards.
        seed(legacy, "config.json" to "{\"legacy\":true}", ".security.yml" to "legacy secret")
        File(filesDir, PocketClawCoreState.CANONICAL_DIR_NAME).writeText("not a directory")

        try {
            PocketClawCoreState.resolve(filesDir)
            fail("a failed migration must not return a directory")
        } catch (e: PocketClawCoreState.MigrationException) {
            assertTrue(e.message!!.contains("could not move") || e.message!!.contains("both"))
        }

        assertTrue(legacy.isDirectory)
        assertEquals("{\"legacy\":true}", File(legacy, "config.json").readText())
        assertEquals("legacy secret", File(legacy, ".security.yml").readText())
    }

    @Test
    fun `the state machine maps each on-disk state to one action`() {
        assertEquals(
            PocketClawCoreState.Action.MIGRATE,
            PocketClawCoreState.decide(legacyExists = true, canonicalExists = false),
        )
        assertEquals(
            PocketClawCoreState.Action.USE_CANONICAL,
            PocketClawCoreState.decide(legacyExists = false, canonicalExists = true),
        )
        assertEquals(
            PocketClawCoreState.Action.CREATE_CANONICAL,
            PocketClawCoreState.decide(legacyExists = false, canonicalExists = false),
        )
        assertEquals(
            PocketClawCoreState.Action.CONFLICT,
            PocketClawCoreState.decide(legacyExists = true, canonicalExists = true),
        )
    }

    @Test
    fun `malformed legacy state is moved rather than judged`() {
        // Whether config.json parses is a question for config validation, after
        // the directory is where it belongs. Deciding it here would mean
        // refusing to migrate — or worse, discarding — a directory that still
        // holds a perfectly good .security.yml.
        seed(legacy, "config.json" to "{ this is not json", ".security.yml" to "api_key: secret")

        PocketClawCoreState.resolve(filesDir)

        assertEquals("{ this is not json", File(canonical, "config.json").readText())
        assertEquals("api_key: secret", File(canonical, ".security.yml").readText())
        assertFalse(legacy.exists())
    }
}

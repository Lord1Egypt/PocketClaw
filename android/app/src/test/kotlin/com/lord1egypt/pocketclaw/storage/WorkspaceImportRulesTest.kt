package com.lord1egypt.pocketclaw.storage

import java.io.File
import java.nio.file.Files
import java.util.Date
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

// PC-DEF-077. The legacy import copies names chosen by a document provider into
// the owner's workspace, and its notice appears only for a real old workspace.
// These are the rules that keep it from false alarms, merging, overwriting or
// escaping.
class WorkspaceImportRulesTest {

    private lateinit var scratch: File

    @Before
    fun setUp() {
        scratch = Files.createTempDirectory("legacy-rules").toFile()
    }

    @After
    fun tearDown() {
        scratch.setReadable(true, false)
        scratch.walkBottomUp().forEach { it.setReadable(true); it.setExecutable(true) }
        scratch.deleteRecursively()
    }

    private fun home() = File(scratch, "pocketclaw")

    private fun write(relative: String, text: String = "x") {
        val file = File(home(), relative)
        file.parentFile!!.mkdirs()
        file.writeText(text)
    }

    // --- detection ---------------------------------------------------------

    @Test
    fun `an absent path is not a legacy workspace`() {
        assertFalse(WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
    }

    @Test
    fun `an empty directory is not a legacy workspace`() {
        // The physical case: a folder made by hand in Samsung My Files.
        home().mkdirs()
        assertFalse(WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
    }

    @Test
    fun `an empty workspace subdirectory is not a legacy workspace`() {
        File(home(), "workspace").mkdirs()
        assertFalse(WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
    }

    @Test
    fun `an unrelated notes file is not a legacy workspace`() {
        write("notes.txt")
        assertFalse(WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
    }

    @Test
    fun `unrelated arbitrary files are not a legacy workspace`() {
        write("photo.jpg")
        write("workspace/random.md")
        write("workspace/memory/notes.txt")
        // Markers at the home root were never a PocketClaw layout.
        write("AGENT.md")
        write("SOUL.md")
        // A marker name that is a directory, not a file.
        File(home(), "workspace/USER.md").mkdirs()
        assertFalse(WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
    }

    @Test
    fun `one historical marker is enough`() {
        for (marker in WorkspaceImportRules.LEGACY_WORKSPACE_MARKERS) {
            home().deleteRecursively()
            write("workspace/$marker")
            assertTrue(marker, WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
        }
    }

    @Test
    fun `a normal old workspace is recognised`() {
        for (file in listOf("AGENT.md", "SOUL.md", "USER.md", "HEARTBEAT.md", "memory/MEMORY.md")) {
            write("workspace/$file")
        }
        write(".pocketclaw.pid")
        write("logs/gateway.log")
        assertTrue(WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
    }

    @Test
    fun `a partly populated old workspace is recognised`() {
        write("workspace/memory/MEMORY.md", "")
        write("workspace/sessions/one.json")
        assertTrue(WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
    }

    @Test
    fun `an unreadable workspace counts as absent`() {
        write("workspace/AGENT.md")
        val workspace = File(home(), "workspace")
        workspace.setExecutable(false)
        workspace.setReadable(false)
        try {
            // Root can read through a mode of 000; the assertion only holds
            // where permissions are enforced, which is every CI and dev user.
            if (!File(workspace, "AGENT.md").isFile) {
                assertFalse(WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
            }
        } finally {
            workspace.setReadable(true)
            workspace.setExecutable(true)
        }
    }

    @Test
    fun `detection never creates the path`() {
        assertFalse(WorkspaceImportRules.looksLikeLegacyWorkspace(home()))
        assertFalse(home().exists())
        assertFalse(File(home(), "workspace").exists())
    }

    @Test
    fun `repeated detection has no side effects`() {
        write("workspace/AGENT.md", "agent")
        fun snapshot() = home().walkTopDown()
            .map { "${it.relativeTo(home())}|${it.isDirectory}|${if (it.isFile) it.readText() else ""}|${it.lastModified()}" }
            .toList()
        val before = snapshot()
        repeat(5) { assertTrue(WorkspaceImportRules.looksLikeLegacyWorkspace(home())) }
        assertEquals(before, snapshot())
    }

    // --- naming and containment -------------------------------------------

    @Test
    fun `a plain file or folder name is kept`() {
        assertEquals("AGENT.md", WorkspaceImportRules.safeChildName("AGENT.md"))
        assertEquals("memory", WorkspaceImportRules.safeChildName(" memory "))
        assertEquals("notes 2026.txt", WorkspaceImportRules.safeChildName("notes 2026.txt"))
    }

    @Test
    fun `a name that could leave the destination is refused`() {
        for (name in listOf(null, "", "   ", ".", "..", "../x", "a/b", "a\\b", "bad\u0000name")) {
            assertNull("\"$name\" must be refused", WorkspaceImportRules.safeChildName(name))
        }
    }

    @Test
    fun `each destination is a new folder, never an existing one`() {
        val workspace = File(scratch, "workspace")
        val now = Date(1_790_000_000_000L)
        val first = WorkspaceImportRules.createFreshDestination(workspace, now)
        val second = WorkspaceImportRules.createFreshDestination(workspace, now)
        assertNotNull(first)
        assertNotNull(second)
        assertNotEquals(first, second)
        assertTrue(first!!.name.startsWith(WorkspaceImportRules.FOLDER_PREFIX))
        assertEquals(workspace, first.parentFile)
        assertTrue(first.isDirectory && second!!.isDirectory)
    }

    @Test
    fun `an inaccessible workspace yields no destination`() {
        val notADirectory = File(scratch, "workspace")
        notADirectory.writeText("a file where the workspace should be")
        assertNull(WorkspaceImportRules.createFreshDestination(notADirectory, Date()))
    }

    @Test
    fun `containment follows the resolved path`() {
        val root = File(scratch, "dest").apply { mkdirs() }
        assertTrue(WorkspaceImportRules.isInside(root, File(root, "AGENT.md")))
        assertTrue(WorkspaceImportRules.isInside(root, File(root, "memory/MEMORY.md")))
        assertFalse(WorkspaceImportRules.isInside(root, File(root, "../outside")))
        assertFalse(WorkspaceImportRules.isInside(root, root))
    }
}

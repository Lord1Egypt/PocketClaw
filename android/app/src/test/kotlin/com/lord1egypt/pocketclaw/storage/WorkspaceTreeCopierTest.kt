package com.lord1egypt.pocketclaw.storage

import java.io.ByteArrayInputStream
import java.io.File
import java.io.IOException
import java.io.InputStream
import java.nio.file.Files
import java.util.Date
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

// PC-DEF-077. The legacy import is copy-only into a fresh folder. These drive
// the copier with a fake, read-only source tree.
class WorkspaceTreeCopierTest {

    /** An in-memory tree. It has no write path, like the real one. */
    private class FakeTree(
        private val nodes: Map<String, List<ImportTree.Entry>>,
        private val files: Map<String, ByteArray>,
        private val failingOpens: Set<String> = emptySet(),
        private val unlistable: Set<String> = emptySet(),
    ) : ImportTree {
        val opened = mutableListOf<String>()
        override fun rootId() = "root"
        override fun children(parentId: String) =
            if (parentId in unlistable) null else nodes[parentId].orEmpty()
        override fun open(id: String): InputStream? {
            opened += id
            if (id in failingOpens) return object : InputStream() {
                override fun read(): Int = throw IOException("provider failed mid-read")
            }
            return files[id]?.let(::ByteArrayInputStream)
        }
    }

    private fun file(id: String, name: String?) = ImportTree.Entry(id, name, isDirectory = false)
    private fun dir(id: String, name: String) = ImportTree.Entry(id, name, isDirectory = true)

    private lateinit var workspace: File
    private val now = Date(1_790_000_000_000L)

    @Before
    fun setUp() {
        workspace = Files.createTempDirectory("workspace").toFile()
        File(workspace, "AGENT.md").writeText("current agent")
        File(workspace, "memory").mkdirs()
        File(workspace, "memory/MEMORY.md").writeText("current memory")
    }

    @After
    fun tearDown() {
        workspace.deleteRecursively()
    }

    private fun legacyTree() = FakeTree(
        nodes = mapOf(
            "root" to listOf(dir("ws", "workspace"), file("pid", ".pocketclaw.pid")),
            "ws" to listOf(file("agent", "AGENT.md"), dir("mem", "memory")),
            "mem" to listOf(file("memory", "MEMORY.md")),
        ),
        files = mapOf(
            "pid" to "123".toByteArray(),
            "agent" to "old agent".toByteArray(),
            "memory" to "old memory".toByteArray(),
        ),
    )

    private fun existingWorkspaceIsUntouched() {
        assertEquals("current agent", File(workspace, "AGENT.md").readText())
        assertEquals("current memory", File(workspace, "memory/MEMORY.md").readText())
    }

    @Test
    fun `a legacy tree is copied into a fresh folder without overwriting anything`() {
        val outcome = WorkspaceTreeCopier.copy(legacyTree(), workspace, now)
        assertEquals(ImportOutcome.COPIED, outcome.status)
        assertEquals(3, outcome.files)
        assertEquals(0, outcome.failed)
        val destination = File(workspace, outcome.folder)
        assertTrue(outcome.folder.startsWith(WorkspaceImportRules.FOLDER_PREFIX))
        assertTrue(WorkspaceImportRules.isInside(workspace, destination))
        assertEquals("old agent", File(destination, "workspace/AGENT.md").readText())
        assertEquals("old memory", File(destination, "workspace/memory/MEMORY.md").readText())
        existingWorkspaceIsUntouched()
    }

    @Test
    fun `the source is only read`() {
        val tree = legacyTree()
        WorkspaceTreeCopier.copy(tree, workspace, now)
        // ImportTree has no write operation; this pins that every file was
        // read exactly once and nothing else was asked of the source.
        assertEquals(listOf("pid", "agent", "memory").sorted(), tree.opened.sorted())
    }

    @Test
    fun `a file that fails mid-read leaves a partial import, not a partial file`() {
        val tree = FakeTree(
            nodes = mapOf("root" to listOf(file("good", "good.md"), file("bad", "bad.md"))),
            files = mapOf("good" to "ok".toByteArray(), "bad" to "never".toByteArray()),
            failingOpens = setOf("bad"),
        )
        val outcome = WorkspaceTreeCopier.copy(tree, workspace, now)
        assertEquals(ImportOutcome.PARTIAL, outcome.status)
        assertEquals(1, outcome.files)
        assertEquals(1, outcome.failed)
        val destination = File(workspace, outcome.folder)
        assertTrue(File(destination, "good.md").isFile)
        assertFalse(File(destination, "bad.md").exists())
    }

    @Test
    fun `a file that cannot be opened is counted, not fatal`() {
        val tree = FakeTree(
            nodes = mapOf("root" to listOf(file("a", "a.md"), file("gone", "gone.md"))),
            files = mapOf("a" to "a".toByteArray()),
        )
        val outcome = WorkspaceTreeCopier.copy(tree, workspace, now)
        assertEquals(ImportOutcome.PARTIAL, outcome.status)
        assertEquals(1, outcome.failed)
    }

    @Test
    fun `a name collision keeps the first entry and counts the second`() {
        val tree = FakeTree(
            nodes = mapOf("root" to listOf(file("one", "same.md"), file("two", "same.md"))),
            files = mapOf("one" to "first".toByteArray(), "two" to "second".toByteArray()),
        )
        val outcome = WorkspaceTreeCopier.copy(tree, workspace, now)
        assertEquals(ImportOutcome.PARTIAL, outcome.status)
        assertEquals("first", File(workspace, "${outcome.folder}/same.md").readText())
    }

    @Test
    fun `hostile names never leave the destination`() {
        val tree = FakeTree(
            nodes = mapOf(
                "root" to listOf(file("up", "../escape.md"), file("abs", "/etc/x"), file("ok", "fine.md")),
            ),
            files = mapOf("up" to "x".toByteArray(), "abs" to "x".toByteArray(), "ok" to "x".toByteArray()),
        )
        val outcome = WorkspaceTreeCopier.copy(tree, workspace, now)
        assertEquals(1, outcome.files)
        assertEquals(2, outcome.failed)
        assertFalse(File(workspace, "escape.md").exists())
    }

    @Test
    fun `an empty source leaves no folder behind`() {
        val tree = FakeTree(nodes = mapOf("root" to emptyList()), files = emptyMap())
        val outcome = WorkspaceTreeCopier.copy(tree, workspace, now)
        assertEquals(ImportOutcome.EMPTY, outcome.status)
        assertTrue(workspace.listFiles()!!.none { it.name.startsWith(WorkspaceImportRules.FOLDER_PREFIX) })
    }

    @Test
    fun `an unreadable source fails and leaves no folder behind`() {
        val tree = FakeTree(nodes = emptyMap(), files = emptyMap(), unlistable = setOf("root"))
        val outcome = WorkspaceTreeCopier.copy(tree, workspace, now)
        assertEquals(ImportOutcome.FAILED, outcome.status)
        assertTrue(workspace.listFiles()!!.none { it.name.startsWith(WorkspaceImportRules.FOLDER_PREFIX) })
        existingWorkspaceIsUntouched()
    }

    @Test
    fun `an inaccessible destination fails without touching the source`() {
        val blocked = File(workspace, "blocked").apply { writeText("not a directory") }
        val tree = legacyTree()
        val outcome = WorkspaceTreeCopier.copy(tree, blocked, now)
        assertEquals(ImportOutcome.FAILED, outcome.status)
        assertTrue(tree.opened.isEmpty())
    }

    @Test
    fun `two imports in the same second get separate folders`() {
        val first = WorkspaceTreeCopier.copy(legacyTree(), workspace, now)
        val second = WorkspaceTreeCopier.copy(legacyTree(), workspace, now)
        assertTrue(first.folder != second.folder)
        assertEquals("old agent", File(workspace, "${first.folder}/workspace/AGENT.md").readText())
        assertEquals("old agent", File(workspace, "${second.folder}/workspace/AGENT.md").readText())
    }

    @Test
    fun `only one import runs at a time`() {
        val gate = ImportGate()
        assertTrue(gate.tryAcquire())
        assertFalse("a second import while one is open must be refused", gate.tryAcquire())
        gate.release()
        assertTrue("the gate reopens once the first import has answered", gate.tryAcquire())
    }
}

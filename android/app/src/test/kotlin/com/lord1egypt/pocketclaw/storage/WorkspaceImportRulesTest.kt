package com.lord1egypt.pocketclaw.storage

import java.io.File
import java.nio.file.Files
import java.util.Date
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

// PC-DEF-077. The legacy import copies names chosen by a document provider into
// the owner's workspace, so these are the rules that keep it from merging,
// overwriting or escaping.
class WorkspaceImportRulesTest {

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
    fun `the destination is a new folder and never an existing one`() {
        val workspace = Files.createTempDirectory("workspace").toFile()
        try {
            val now = Date(1_790_000_000_000L)
            val first = WorkspaceImportRules.freshDestination(workspace, now)
            assertFalse(first.exists())
            assertTrue(first.name.startsWith(WorkspaceImportRules.FOLDER_PREFIX))
            assertEquals(workspace, first.parentFile)

            assertTrue(first.mkdirs())
            val second = WorkspaceImportRules.freshDestination(workspace, now)
            assertNotEquals(first, second)
            assertFalse(second.exists())
        } finally {
            workspace.deleteRecursively()
        }
    }

    @Test
    fun `containment follows the resolved path`() {
        val root = Files.createTempDirectory("dest").toFile()
        try {
            assertTrue(WorkspaceImportRules.isInside(root, File(root, "AGENT.md")))
            assertTrue(WorkspaceImportRules.isInside(root, File(root, "memory/MEMORY.md")))
            assertFalse(WorkspaceImportRules.isInside(root, File(root, "../outside")))
            assertFalse(WorkspaceImportRules.isInside(root, root))
        } finally {
            root.deleteRecursively()
        }
    }
}

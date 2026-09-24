package com.lord1egypt.pocketclaw.storage

import java.io.File
import java.io.FileOutputStream
import java.io.InputStream
import java.util.Date

/**
 * A source tree the import reads. It has no write operation at all, so the
 * import cannot modify, move or delete anything in the owner's old folder.
 */
interface ImportTree {
    data class Entry(val id: String?, val name: String?, val isDirectory: Boolean)

    fun rootId(): String

    /** The children of [parentId], or null when they cannot be listed. */
    fun children(parentId: String): List<Entry>?

    /** A stream over the file [id], or null when it cannot be opened. */
    fun open(id: String): InputStream?
}

/** What one import did. [folder] is the new folder's name inside the workspace. */
data class ImportOutcome(
    val status: String,
    val folder: String = "",
    val files: Int = 0,
    val failed: Int = 0,
) {
    fun asMap(): Map<String, Any> = mapOf(
        "status" to status,
        "folder" to folder,
        "files" to files,
        "failed" to failed,
    )

    companion object {
        const val COPIED = "copied"
        const val PARTIAL = "partial"
        const val FAILED = "failed"
        const val EMPTY = "empty"
        const val CANCELLED = "cancelled"
        const val BUSY = "busy"
        const val UNAVAILABLE = "unavailable"
    }
}

/**
 * Copies an [ImportTree] into a fresh folder inside a workspace. PC-DEF-077.
 *
 * Copy only: nothing in the workspace is overwritten -- every file lands in a
 * folder this call has just created, and a second entry with a taken name is
 * skipped and counted -- and the source is only ever read. A folder that ends
 * up holding nothing is removed again, so a cancelled-in-effect import leaves
 * no trace.
 */
object WorkspaceTreeCopier {
    /** Deep enough for any real workspace, shallow enough to stop a loop. */
    const val MAX_DEPTH = 32

    fun copy(tree: ImportTree, workspace: File, now: Date = Date()): ImportOutcome {
        val destination = WorkspaceImportRules.createFreshDestination(workspace, now)
            ?: return ImportOutcome(ImportOutcome.FAILED)

        var copied = 0
        var failed = 0

        fun walk(parentId: String, dir: File, depth: Int): Boolean {
            if (depth > MAX_DEPTH) {
                failed++
                return true
            }
            val children = tree.children(parentId) ?: run {
                failed++
                return false
            }
            for (child in children) {
                val name = WorkspaceImportRules.safeChildName(child.name)
                val id = child.id
                if (id == null || name == null) {
                    failed++
                    continue
                }
                val target = File(dir, name)
                if (!WorkspaceImportRules.isInside(destination, target) || target.exists()) {
                    failed++
                    continue
                }
                if (child.isDirectory) {
                    if (target.mkdir()) walk(id, target, depth + 1) else failed++
                    continue
                }
                try {
                    val input = tree.open(id)
                    if (input == null) {
                        failed++
                        continue
                    }
                    input.use { stream ->
                        FileOutputStream(target).use { out -> stream.copyTo(out) }
                    }
                    copied++
                } catch (e: Exception) {
                    // Only ever our own partial file inside the new folder.
                    target.delete()
                    failed++
                }
            }
            return true
        }

        val rootListed = walk(tree.rootId(), destination, 0)

        if (copied == 0) {
            destination.deleteRecursively()
            return when {
                !rootListed || failed > 0 -> ImportOutcome(ImportOutcome.FAILED, failed = failed)
                else -> ImportOutcome(ImportOutcome.EMPTY)
            }
        }
        val status = if (failed == 0) ImportOutcome.COPIED else ImportOutcome.PARTIAL
        return ImportOutcome(status, destination.name, copied, failed)
    }
}

/** One import at a time, from the picker opening until the copy has answered. */
class ImportGate {
    private var busy = false

    @Synchronized
    fun tryAcquire(): Boolean {
        if (busy) return false
        busy = true
        return true
    }

    @Synchronized
    fun release() {
        busy = false
    }
}

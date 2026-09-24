package com.lord1egypt.pocketclaw.storage

import java.io.File
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

/**
 * The pure rules of the legacy workspace import, kept apart from Android so
 * they are checked on the JVM.
 *
 * The import never merges: everything lands in one new folder inside the
 * workspace, no existing file is written, and a name the source provider hands
 * back is only ever used as a single path segment inside that folder.
 */
object WorkspaceImportRules {
    const val FOLDER_PREFIX = "imported-from-downloads-"

    /**
     * A document name usable as one file name, or null.
     *
     * Document providers return display names, which are not guaranteed to be
     * path-safe. A separator, a NUL, "." or ".." could otherwise walk the copy
     * out of its destination.
     */
    fun safeChildName(displayName: String?): String? {
        val name = displayName?.trim() ?: return null
        if (name.isEmpty() || name == "." || name == "..") return null
        if (name.contains('/') || name.contains('\\') || name.contains('\u0000')) return null
        return name
    }

    /**
     * A folder inside [workspace] that does not exist yet, named for [now].
     * A second import in the same second gets a numbered suffix, never the
     * first import's folder.
     */
    fun freshDestination(workspace: File, now: Date): File {
        val stamp = SimpleDateFormat("yyyyMMdd-HHmmss", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(now)
        var candidate = File(workspace, FOLDER_PREFIX + stamp)
        var suffix = 2
        while (candidate.exists()) {
            candidate = File(workspace, "$FOLDER_PREFIX$stamp-$suffix")
            suffix++
        }
        return candidate
    }

    /** Whether [target] resolves inside [root], symlinks and ".." included. */
    fun isInside(root: File, target: File): Boolean {
        val canonicalRoot = root.canonicalFile
        return generateSequence(target.canonicalFile) { it.parentFile }
            .drop(1)
            .any { it == canonicalRoot }
    }
}

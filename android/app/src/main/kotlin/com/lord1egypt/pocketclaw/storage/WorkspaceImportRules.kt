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
     * Files whose presence under `<home>/workspace/` identifies a PocketClaw
     * home. Core has always placed the agent workspace at `$HOME/workspace`
     * (`pkg.WorkspaceName`) and seeds it with AGENT.md, SOUL.md, USER.md and
     * memory/MEMORY.md; the heartbeat service writes HEARTBEAT.md and the
     * bootstrap record lives in `.pocketclaw/bootstrap.json`. No build ever
     * kept these directly under the home root, so the root is not searched.
     */
    val LEGACY_WORKSPACE_MARKERS = listOf(
        "AGENT.md",
        "SOUL.md",
        "USER.md",
        "HEARTBEAT.md",
        "memory/MEMORY.md",
        ".pocketclaw/bootstrap.json",
    )

    /**
     * Whether [home] holds an old PocketClaw workspace. PC-DEF-077.
     *
     * Mere existence is not evidence: an empty `Download/pocketclaw`, made by
     * hand in a file manager, was shown as an earlier workspace. One marker is
     * enough, because a partly populated genuine workspace must still count,
     * and a file named `workspace/AGENT.md` does not appear by accident.
     *
     * Read-only by construction: it only asks whether paths are directories or
     * regular files, so it cannot create, write or delete anything. Anything
     * it may not read counts as absent.
     */
    fun looksLikeLegacyWorkspace(home: File): Boolean = try {
        val workspace = File(home, "workspace")
        home.isDirectory && workspace.isDirectory &&
            LEGACY_WORKSPACE_MARKERS.any { File(workspace, it).isFile }
    } catch (e: SecurityException) {
        false
    }

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
     * Creates and returns a new folder inside [workspace] named for [now], or
     * null when none can be created. `mkdir` is the existence check, so two
     * imports in the same second can never share a folder.
     */
    fun createFreshDestination(workspace: File, now: Date): File? {
        if (!workspace.isDirectory && !workspace.mkdirs()) return null
        val stamp = SimpleDateFormat("yyyyMMdd-HHmmss", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(now)
        for (suffix in 1..100) {
            val name = if (suffix == 1) FOLDER_PREFIX + stamp else "$FOLDER_PREFIX$stamp-$suffix"
            val candidate = File(workspace, name)
            if (candidate.mkdir()) return candidate
            if (!workspace.isDirectory) return null
        }
        return null
    }

    /** Whether [target] resolves inside [root], symlinks and ".." included. */
    fun isInside(root: File, target: File): Boolean {
        val canonicalRoot = root.canonicalFile
        return generateSequence(target.canonicalFile) { it.parentFile }
            .drop(1)
            .any { it == canonicalRoot }
    }
}

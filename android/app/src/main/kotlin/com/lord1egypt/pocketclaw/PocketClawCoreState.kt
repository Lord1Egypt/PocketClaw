package com.lord1egypt.pocketclaw

import android.content.Context
import android.util.Log
import java.io.File
import java.io.IOException

/**
 * The one place that names Core's private state directory.
 *
 * Core keeps `config.json` and `.security.yml` here, plus the Managed Runtime
 * metadata and whatever else it decides to write next to its config — Core
 * derives `.security.yml` from the config file's own directory, so the two
 * always travel together and the directory, not the file, is the unit that
 * moves.
 *
 * This is app-private state and is not the user's workspace. The workspace is
 * `Android/data/<package>/files/pocketclaw`, or `filesDir/pocketclaw/` when
 * external storage is unavailable, and holds AGENT.md, SOUL.md, USER.md and
 * memory/ (PocketClawService.getWorkspacePath).
 * Those are the user's documents; nothing here reads or writes them.
 *
 * Every caller asks [directory] or [configFile] rather than spelling the path,
 * so a caller cannot reach a canonical directory that has not been migrated
 * yet — which for the config would mean Core onboarding into an empty
 * directory and presenting a factory reset as a successful start.
 */
object PocketClawCoreState {
    private const val TAG = "PocketClawCoreState"

    /** Canonical Core private state. Every current read, write and spawn uses this. */
    const val CANONICAL_DIR_NAME = "pocketclaw-core"

    /**
     * LEGACY MOVE-ONLY MIGRATION — the pre-Zero-Pico directory.
     *
     * Named here to find an existing install's state and move it across, once.
     * It is never created, never written, and never written back to. There is
     * no dual-write, no copy back, and no alias: after migration this build
     * behaves as though the directory does not exist.
     */
    private const val LEGACY_DIR_NAME = "picoclaw"

    /** Core's config file, inside whichever directory [directory] resolves to. */
    const val CONFIG_FILE_NAME = "config.json"

    /** What the current on-disk state calls for. */
    enum class Action {
        /** Legacy state exists alone: move it to the canonical name. */
        MIGRATE,

        /** Canonical state exists alone: the normal case after migration. */
        USE_CANONICAL,

        /** Neither exists: a fresh install starts canonical. */
        CREATE_CANONICAL,

        /**
         * Both exist. An interrupted migration, or a downgrade that ran an
         * older build against a migrated install and let it onboard fresh.
         * Either way the two directories may hold different provider keys and
         * channel tokens, and nothing on disk says which the user meant.
         */
        CONFLICT,
    }

    /** Raised instead of falling back to a path that would lose or expose state. */
    class MigrationException(message: String, cause: Throwable? = null) :
        IOException(message, cause)

    @Volatile
    private var resolved: File? = null

    /**
     * Decide from what exists on disk. Pure, so the states that matter can be
     * tested without an install in each of them.
     */
    fun decide(legacyExists: Boolean, canonicalExists: Boolean): Action = when {
        legacyExists && canonicalExists -> Action.CONFLICT
        legacyExists -> Action.MIGRATE
        canonicalExists -> Action.USE_CANONICAL
        else -> Action.CREATE_CANONICAL
    }

    /**
     * Returns the canonical directory, migrating legacy state first.
     *
     * Serialized and resolved once per process: every consumer — the Gateway
     * and web spawns, onboarding, and the Dart config editor through the method
     * channel — comes through here, and two of them can arrive on different
     * threads.
     *
     * Throws rather than returning a usable-looking path. A caller that got a
     * fresh empty directory after a failed migration would let Core onboard
     * into it, and the user would see an install with no providers, no channels
     * and no memory: a factory reset presented as a successful start, with the
     * real state still on disk and nothing pointing at it.
     */
    @Synchronized
    fun directory(context: Context): File {
        resolved?.let { return it }
        val migrating = File(context.filesDir, LEGACY_DIR_NAME).isDirectory
        val directory = resolve(context.filesDir)
        if (migrating) {
            Log.i(TAG, "Moved Core's private state to ${directory.name}")
        }
        resolved = directory
        return directory
    }

    /** Core's config file in the canonical directory. */
    fun configFile(context: Context): File = File(directory(context), CONFIG_FILE_NAME)

    /**
     * The state machine, over a filesDir rather than a Context so each case can
     * be exercised directly.
     */
    fun resolve(filesDir: File): File {
        val legacy = File(filesDir, LEGACY_DIR_NAME)
        val canonical = File(filesDir, CANONICAL_DIR_NAME)

        when (decide(legacy.isDirectory, canonical.isDirectory)) {
            Action.USE_CANONICAL -> return canonical

            Action.CREATE_CANONICAL -> {
                if (!canonical.isDirectory && !canonical.mkdirs()) {
                    throw MigrationException(
                        "could not create Core's private directory at ${canonical.absolutePath}",
                    )
                }
                return canonical
            }

            Action.CONFLICT -> throw MigrationException(
                "Core state exists in both ${legacy.absolutePath} and " +
                    "${canonical.absolutePath}. This is an interrupted migration or a " +
                    "downgrade, and the two may hold different provider keys and channel " +
                    "tokens. Neither was changed, merged or deleted. PocketClaw will not " +
                    "start until one is chosen.",
            )

            Action.MIGRATE -> {
                migrate(legacy, canonical)
                return canonical
            }
        }
    }

    /**
     * Move the whole directory across, and verify the move before trusting it.
     *
     * A rename, not a copy. Both directories are direct children of filesDir,
     * so they are always on one filesystem and this is a single atomic
     * `rename(2)`: the tree arrives whole or not at all, including files this
     * build has never heard of, hidden files and nested directories. Nothing
     * enumerates the contents, so nothing can migrate a subset of them.
     *
     * There is deliberately no copy fallback. A recursive copy would have to
     * reproduce the permissions on `.security.yml` — which holds provider API
     * keys and channel bot tokens in plaintext, because Android onboarding
     * declines credential encryption — and Java's file APIs cannot express
     * those modes or fsync a directory, so the honest failure modes are a
     * second plaintext copy of the user's secrets left in a partially-written
     * tree, or a delete of the original after an unverifiable copy. Since these
     * two paths cannot be on different filesystems, a failed rename means a
     * real filesystem or permission fault, and continuing through one is how
     * state gets lost. It fails closed instead, having changed nothing.
     */
    private fun migrate(legacy: File, canonical: File) {
        if (!legacy.renameTo(canonical)) {
            throw MigrationException(
                "could not move Core's private state from ${legacy.absolutePath} to " +
                    "${canonical.absolutePath}. Nothing was copied or deleted and the " +
                    "existing state is untouched.",
            )
        }
        // The source is gone only because the rename moved it. Verifying both
        // sides is what separates that from a rename that reported success
        // while leaving the tree somewhere unusable.
        if (!canonical.isDirectory || legacy.exists()) {
            throw MigrationException(
                "moving Core's private state to ${canonical.absolutePath} did not take " +
                    "effect: canonical=${canonical.isDirectory}, legacy=${legacy.exists()}",
            )
        }
    }
}

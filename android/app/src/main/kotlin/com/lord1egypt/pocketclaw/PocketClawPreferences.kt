package com.lord1egypt.pocketclaw

import android.content.Context
import android.content.SharedPreferences
import android.util.Log
import java.io.File

/**
 * The one place that names PocketClaw's app-private preference store.
 *
 * Three call sites used to declare `picoclaw_prefs` independently, which is how
 * a store gets renamed in two of them. The name lives here now, and everything
 * goes through [open] so the migration below cannot be bypassed by a caller
 * that reaches for `getSharedPreferences` directly.
 *
 * The legacy name is LEGACY READ-ONLY MIGRATION: it is read once to carry an
 * existing install's settings across, and never written or created again.
 */
object PocketClawPreferences {
    private const val TAG = "PocketClawPreferences"

    /** Canonical store. Every current read and write uses this. */
    const val NAME = "pocketclaw_prefs"

    /**
     * The per-install marker recording that the notification dialog has been
     * shown. A file under `noBackupFilesDir`, not a preference.
     *
     * PC-DEF-058, third attempt. It was a SharedPreference, and this app has
     * `allowBackup="true"` with only three `file`-domain paths excluded — so
     * `shared_prefs/pocketclaw_prefs.xml` is backup-eligible and a reinstall can
     * restore `notification_permission_asked = true` from a *previous* install.
     * The state machine then resolves DENIED, whose action is "offer Settings,
     * do not ask", and a genuinely fresh install never sees the dialog. Which is
     * what the device kept showing.
     *
     * "Have we asked *this install*" is per-install state by definition, so it
     * belongs somewhere a restore cannot reach. The legacy preference key
     * `notification_permission_asked` is deliberately **not** read: migrating it
     * would carry the restored value straight back in. The cost is that an
     * upgrade from an older build may raise the dialog once more — and only for
     * someone who has not already granted it, since a granted permission
     * short-circuits before this is consulted.
     */
    private const val NOTIFICATION_ASKED_MARKER = "notification-permission-asked"

    /**
     * LEGACY READ-ONLY MIGRATION — the pre-Zero-Pico store.
     *
     * Referenced solely to discover and copy an existing install's values. This
     * is not a current identity and must never appear on a write path.
     */
    private const val LEGACY_NAME = "picoclaw_prefs"

    @Volatile
    private var migrated = false

    /**
     * Returns the canonical preferences, migrating a legacy store first.
     *
     * Callers must use this rather than getSharedPreferences(NAME, …) so an
     * upgrade cannot read an empty canonical store before the migration has
     * run — which for BootReceiver would mean silently reverting the user's
     * auto-start choice to its default on the first boot after upgrade.
     */
    fun open(context: Context): SharedPreferences {
        migrateIfNeeded(context)
        return context.getSharedPreferences(NAME, Context.MODE_PRIVATE)
    }

    private fun prefsFile(context: Context, name: String): File =
        File(File(context.dataDir, "shared_prefs"), "$name.xml")

    /**
     * Copy legacy values across exactly once, verify them, and only then drop
     * the legacy store.
     *
     * Ordering is the whole design: the legacy file is the only copy of the
     * user's settings until the canonical one is proven durable, so it is
     * deleted last and only after a synchronous commit has been read back.
     */
    @Synchronized
    fun migrateIfNeeded(context: Context) {
        if (migrated) return
        migrated = true

        val legacyFile = prefsFile(context, LEGACY_NAME)
        if (!legacyFile.exists()) return

        val legacy = context.getSharedPreferences(LEGACY_NAME, Context.MODE_PRIVATE)
        val legacyValues = legacy.all
        if (legacyValues.isEmpty()) {
            deleteLegacy(context, "it held no values")
            return
        }

        val canonical = context.getSharedPreferences(NAME, Context.MODE_PRIVATE)
        val canonicalExists = prefsFile(context, NAME).exists()

        if (canonicalExists) {
            // Canonical state wins. A legacy store that agrees with it entirely
            // carries nothing we would lose, so it can go; anything else is a
            // conflict we preserve rather than resolve by guessing.
            val disagreeing = legacyValues.filter { (key, value) ->
                !canonical.contains(key) || canonical.all[key] != value
            }
            if (disagreeing.isEmpty()) {
                deleteLegacy(context, "every value matched the canonical store")
            } else {
                Log.w(
                    TAG,
                    "Legacy preferences disagree with canonical PocketClaw values for " +
                        "${disagreeing.keys.sorted()}; keeping both. Canonical values are " +
                        "in use; the legacy store was left untouched rather than merged.",
                )
            }
            return
        }

        val editor = canonical.edit()
        for ((key, value) in legacyValues) {
            when (value) {
                is Boolean -> editor.putBoolean(key, value)
                is Int -> editor.putInt(key, value)
                is Long -> editor.putLong(key, value)
                is Float -> editor.putFloat(key, value)
                is String -> editor.putString(key, value)
                is Set<*> -> {
                    @Suppress("UNCHECKED_CAST")
                    editor.putStringSet(key, value as Set<String>)
                }
                else -> Log.w(TAG, "Skipping legacy preference $key of unsupported type")
            }
        }

        if (!editor.commit()) {
            Log.w(TAG, "Could not commit migrated preferences; keeping the legacy store")
            return
        }

        val readBack = context.getSharedPreferences(NAME, Context.MODE_PRIVATE).all
        val missing = legacyValues.filter { (key, value) -> readBack[key] != value }
        if (missing.isNotEmpty()) {
            Log.w(
                TAG,
                "Migrated preferences did not read back for ${missing.keys.sorted()}; " +
                    "keeping the legacy store",
            )
            return
        }

        deleteLegacy(context, "${legacyValues.size} value(s) migrated and verified")
    }

    private fun deleteLegacy(context: Context, why: String) {
        val deleted = context.deleteSharedPreferences(LEGACY_NAME)
        Log.i(TAG, "Removed the legacy preference store ($why); deleted=$deleted")
    }

    /**
     * Whether PocketClaw has already shown the system notification-permission
     * dialog.
     *
     * PC-DEF-058. Android's own `shouldShowRequestPermissionRationale` cannot
     * answer this: it returns false both before the first ask and after a
     * permanent denial, so it cannot tell "never asked" from "refused for good".
     * PocketClaw keeps its own record so it asks exactly once and never nags.
     */
    private fun notificationAskedMarker(context: Context): File =
        File(context.applicationContext.noBackupFilesDir, NOTIFICATION_ASKED_MARKER)

    fun notificationPermissionAsked(context: Context): Boolean =
        notificationAskedMarker(context).exists()

    fun setNotificationPermissionAsked(context: Context, asked: Boolean) {
        val marker = notificationAskedMarker(context)
        try {
            if (asked) {
                marker.parentFile?.mkdirs()
                marker.createNewFile()
            } else {
                marker.delete()
            }
        } catch (e: Exception) {
            // A marker that cannot be written means the dialog may be offered
            // again on the next launch. That is the safe direction to fail: the
            // alternative is never asking at all.
            Log.w(TAG, "notification asked marker unavailable: ${e.message}")
        }
    }
}

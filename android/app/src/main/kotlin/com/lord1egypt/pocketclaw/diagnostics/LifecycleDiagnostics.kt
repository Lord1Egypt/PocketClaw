package com.lord1egypt.pocketclaw.diagnostics

import android.app.ActivityManager
import android.app.ApplicationExitInfo
import android.app.ApplicationStartInfo
import android.content.Context
import android.os.Build
import android.os.Process
import android.util.Log
import java.io.File
import java.io.IOException

/**
 * Lifecycle breadcrumbs for the reboot "keeps stopping" report (PC-DEF-085).
 *
 * No source path starts PocketClaw without the owner: there is no boot,
 * package-replaced or direct-boot component, no alarm, job or WorkManager
 * work, and the service is never sticky. So instead of guessing, the next
 * occurrence is recorded where it happens. Each process start records why
 * Android started it (ApplicationStartInfo, Android 15+) and how earlier
 * processes ended (ApplicationExitInfo, Android 11+); each automatic entry
 * point records itself; an uncaught exception records its class before the
 * normal crash handling runs.
 *
 * Local only: `noBackupFilesDir/diagnostics/lifecycle.log`, bounded, appended
 * to logcat under [TAG], and included in a log export when the owner makes
 * one. Recording must never be the reason the app fails, so only I/O and
 * security failures are absorbed here.
 */
object LifecycleDiagnostics {
    const val TAG = "PocketClawLifecycle"
    private const val DIR = "diagnostics"
    private const val JOURNAL = "lifecycle.log"
    private const val LAST_EXIT = "last-exit"

    private fun dir(context: Context) = File(context.applicationContext.noBackupFilesDir, DIR)

    private fun journal(context: Context) = LifecycleJournal(File(dir(context), JOURNAL))

    fun record(context: Context, component: String, event: String, detail: String = "") {
        val line = LifecycleText.line(
            System.currentTimeMillis(), Process.myPid(), component, event, detail,
        )
        Log.i(TAG, line)
        try {
            journal(context).append(line)
        } catch (e: IOException) {
            Log.w(TAG, "lifecycle journal unavailable: ${e.javaClass.simpleName}")
        } catch (e: SecurityException) {
            Log.w(TAG, "lifecycle journal unavailable: ${e.javaClass.simpleName}")
        }
    }

    fun read(context: Context): String = try {
        journal(context).read()
    } catch (e: IOException) {
        ""
    }

    /** Called first thing in Application.onCreate. */
    fun onProcessStart(context: Context) {
        record(context, "process", "start", startDetail(context))
        recordEarlierExits(context)
        installCrashBreadcrumb(context.applicationContext)
    }

    private fun startDetail(context: Context): String {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.VANILLA_ICE_CREAM) return "start-info=unavailable"
        val manager = context.getSystemService(ActivityManager::class.java) ?: return "start-info=unavailable"
        val info = try {
            manager.getHistoricalProcessStartReasons(1).firstOrNull()
        } catch (e: RuntimeException) {
            null
        } ?: return "start-info=none"
        if (info.pid != Process.myPid()) return "start-info=stale"
        return buildString {
            append("reason=").append(startReason(info.reason))
            append(" type=").append(startType(info.startType))
            if (Build.VERSION.SDK_INT >= 36) append(" component=").append(startComponent(info.startComponent))
            info.intent?.let { intent ->
                append(" action=").append(LifecycleText.intentAction(true, intent.action))
                intent.component?.let { append(" target=").append(it.className.substringAfterLast('.')) }
            }
            if (info.wasForceStopped()) append(" after-force-stop")
        }
    }

    private fun recordEarlierExits(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.R) return
        val manager = context.getSystemService(ActivityManager::class.java) ?: return
        val marker = File(dir(context), LAST_EXIT)
        val lastSeen = try {
            marker.takeIf { it.isFile }?.readText()?.trim()?.toLongOrNull() ?: 0L
        } catch (e: IOException) {
            0L
        }
        val exits = try {
            manager.getHistoricalProcessExitReasons(context.packageName, 0, 5)
        } catch (e: RuntimeException) {
            return
        }
        val fresh = exits.filter { it.timestamp > lastSeen }.sortedBy { it.timestamp }
        for (exit in fresh) {
            val detail = buildString {
                append("reason=").append(exitReason(exit.reason))
                append(" status=").append(exit.status)
                append(" importance=").append(exit.importance)
                append(" at=").append(exit.timestamp)
                LifecycleText.classOnly(exit.description)?.let { append(" error=").append(it) }
            }
            record(context, "process", "earlier-exit", detail)
        }
        fresh.lastOrNull()?.let {
            try {
                marker.parentFile?.mkdirs()
                marker.writeText(it.timestamp.toString())
            } catch (e: IOException) {
                Log.w(TAG, "could not record the last exit timestamp")
            }
        }
    }

    private fun installCrashBreadcrumb(context: Context) {
        val previous = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { thread, error ->
            record(
                context,
                "process",
                "uncaught",
                "thread=${thread.name.take(40)} " + LifecycleText.crashSummary(error, context.packageName),
            )
            if (previous != null) previous.uncaughtException(thread, error)
            else {
                Process.killProcess(Process.myPid())
                System.exit(10)
            }
        }
    }

    private fun startReason(value: Int): String = when (value) {
        ApplicationStartInfo.START_REASON_ALARM -> "alarm"
        ApplicationStartInfo.START_REASON_BACKUP -> "backup"
        ApplicationStartInfo.START_REASON_BOOT_COMPLETE -> "boot-complete"
        ApplicationStartInfo.START_REASON_BROADCAST -> "broadcast"
        ApplicationStartInfo.START_REASON_CONTENT_PROVIDER -> "content-provider"
        ApplicationStartInfo.START_REASON_JOB -> "job"
        ApplicationStartInfo.START_REASON_LAUNCHER -> "launcher"
        ApplicationStartInfo.START_REASON_LAUNCHER_RECENTS -> "launcher-recents"
        ApplicationStartInfo.START_REASON_OTHER -> "other"
        ApplicationStartInfo.START_REASON_PUSH -> "push"
        ApplicationStartInfo.START_REASON_SERVICE -> "service"
        ApplicationStartInfo.START_REASON_START_ACTIVITY -> "start-activity"
        else -> "unknown-$value"
    }

    private fun startType(value: Int): String = when (value) {
        ApplicationStartInfo.START_TYPE_COLD -> "cold"
        ApplicationStartInfo.START_TYPE_WARM -> "warm"
        ApplicationStartInfo.START_TYPE_HOT -> "hot"
        else -> "unset-$value"
    }

    private fun startComponent(value: Int): String = when (value) {
        ApplicationStartInfo.START_COMPONENT_ACTIVITY -> "activity"
        ApplicationStartInfo.START_COMPONENT_BROADCAST -> "broadcast"
        ApplicationStartInfo.START_COMPONENT_CONTENT_PROVIDER -> "content-provider"
        ApplicationStartInfo.START_COMPONENT_SERVICE -> "service"
        ApplicationStartInfo.START_COMPONENT_OTHER -> "other"
        else -> "unknown-$value"
    }

    private fun exitReason(value: Int): String = when (value) {
        ApplicationExitInfo.REASON_ANR -> "anr"
        ApplicationExitInfo.REASON_CRASH -> "crash"
        ApplicationExitInfo.REASON_CRASH_NATIVE -> "crash-native"
        ApplicationExitInfo.REASON_DEPENDENCY_DIED -> "dependency-died"
        ApplicationExitInfo.REASON_EXCESSIVE_RESOURCE_USAGE -> "excessive-resource"
        ApplicationExitInfo.REASON_EXIT_SELF -> "exit-self"
        ApplicationExitInfo.REASON_INITIALIZATION_FAILURE -> "initialization-failure"
        ApplicationExitInfo.REASON_LOW_MEMORY -> "low-memory"
        ApplicationExitInfo.REASON_OTHER -> "other"
        ApplicationExitInfo.REASON_PERMISSION_CHANGE -> "permission-change"
        ApplicationExitInfo.REASON_SIGNALED -> "signaled"
        ApplicationExitInfo.REASON_USER_REQUESTED -> "user-requested"
        ApplicationExitInfo.REASON_USER_STOPPED -> "user-stopped"
        else -> "unknown-$value"
    }
}

package com.lord1egypt.pocketclaw.diagnostics

import java.io.File
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

/**
 * A bounded, append-only local journal of lifecycle breadcrumbs.
 *
 * It exists for one question the source cannot answer: what started a
 * PocketClaw process, and how did the last one end, when nobody opened the app
 * (the reboot "keeps stopping" report). It never leaves the device, holds no
 * message text and no secrets -- only component names, lifecycle events,
 * intent actions and exception class names -- and never grows past [maxBytes].
 */
class LifecycleJournal(
    private val file: File,
    private val maxBytes: Int = MAX_BYTES,
) {
    companion object {
        const val MAX_BYTES = 32 * 1024
        const val MAX_LINE = 240
    }

    fun append(line: String) {
        file.parentFile?.mkdirs()
        file.appendText(line.replace('\n', ' ').take(MAX_LINE) + "\n")
        if (file.length() > maxBytes) trim()
    }

    fun read(): String = if (file.isFile) file.readText() else ""

    /** Keeps the newest lines that fit in half the budget. */
    private fun trim() {
        val kept = ArrayDeque<String>()
        var size = 0
        for (line in file.readLines().asReversed()) {
            val cost = line.length + 1
            if (size + cost > maxBytes / 2) break
            kept.addFirst(line)
            size += cost
        }
        val temp = File(file.parentFile, file.name + ".tmp")
        temp.writeText(kept.joinToString("\n", postfix = if (kept.isEmpty()) "" else "\n"))
        if (!temp.renameTo(file)) {
            file.writeText(temp.readText())
            temp.delete()
        }
    }
}

/** Formatting that keeps only what is safe to record. */
object LifecycleText {
    private val CLASS_NAME = Regex("^[A-Za-z_$][A-Za-z0-9_$.]*$")

    fun line(nowMillis: Long, pid: Int, component: String, event: String, detail: String): String {
        val stamp = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'", Locale.US).apply {
            timeZone = TimeZone.getTimeZone("UTC")
        }.format(Date(nowMillis))
        return listOf(stamp, "pid=$pid", component, event, detail)
            .filter { it.isNotEmpty() }
            .joinToString(" | ")
    }

    /** "null-intent" for an OS re-creation, the action otherwise. */
    fun intentAction(present: Boolean, action: String?): String = when {
        !present -> "null-intent"
        action.isNullOrBlank() -> "no-action"
        else -> action.takeLast(80)
    }

    /**
     * The exception class named at the start of an exit description, or null.
     * Anything after the class name -- the message -- is dropped: it can carry
     * paths, tokens or user text.
     */
    fun classOnly(description: String?): String? {
        val head = description?.substringBefore(':')?.trim()?.takeIf { it.isNotEmpty() } ?: return null
        return head.takeIf { it.length <= 120 && CLASS_NAME.matches(it) }
    }

    /** The throwable's class and the first frame in PocketClaw's own code. */
    fun crashSummary(error: Throwable, ownPackage: String): String {
        val frame = error.stackTrace.firstOrNull { it.className.startsWith(ownPackage) }
        val where = frame?.let { "${it.className.substringAfterLast('.')}.${it.methodName}" } ?: "-"
        val cause = generateSequence(error.cause) { it.cause }.lastOrNull()?.javaClass?.name
        return buildString {
            append("error=").append(error.javaClass.name)
            append(" at=").append(where)
            if (cause != null) append(" root=").append(cause)
        }
    }
}

package com.lord1egypt.pocketclaw.whatsapp

import android.content.Context
import android.os.FileObserver
import android.util.Log
import java.io.File
import org.json.JSONObject

/**
 * Serves the requests Core writes when the agent calls `whatsapp_self_chat`.
 *
 * Core runs as a child process of this app and cannot start an Activity, so
 * opening WhatsApp is something it has to ask for. The two processes share the
 * app's private storage, so a request is a file: Core writes `req-<id>.json`
 * and reads back `res-<id>.json`. No port is opened and no token is needed —
 * the directory is reachable only by this UID.
 *
 * The watcher replies to every request it picks up, including the ones it
 * cannot serve, so Core reports a real reason instead of waiting out its
 * timeout.
 */
class HostRequestWatcher(private val context: Context, private val directory: File) {

    companion object {
        private const val TAG = "HostRequestWatcher"

        /** The environment variable Core reads to find this directory. */
        const val ENV_OUTBOX = "POCKETCLAW_ANDROID_HOST_OUTBOX"

        private const val ACTION_SELF_CHAT = "whatsapp_self_chat"

        /** Where the directory lives, created on demand. */
        fun outboxDir(context: Context): File =
            File(context.filesDir, "host-outbox").apply { mkdirs() }
    }

    private var observer: FileObserver? = null

    fun start() {
        if (observer != null) return
        directory.mkdirs()

        // CLOSE_WRITE alone would miss the requests Core stages and renames,
        // which is how it avoids ever exposing a half-written file.
        val mask = FileObserver.CLOSE_WRITE or FileObserver.MOVED_TO
        // The File constructor is API 29, and PocketClaw ships minSdk 24, where
        // it would take the whole foreground service down with a
        // NoSuchMethodError before Core ever starts. The path constructor is
        // deprecated on newer releases but present on every one of them.
        @Suppress("DEPRECATION")
        val watcher = object : FileObserver(directory.absolutePath, mask) {
            override fun onEvent(event: Int, path: String?) {
                if (path == null) return
                serve(path)
            }
        }
        watcher.startWatching()
        observer = watcher

        // A request written while nothing was watching would otherwise sit
        // there until the next one arrived.
        drainPending()
        Log.i(TAG, "Watching for Core host requests")
    }

    fun stop() {
        observer?.stopWatching()
        observer = null
    }

    private fun drainPending() {
        directory.listFiles()?.forEach { file -> serve(file.name) }
    }

    private fun serve(fileName: String) {
        if (!fileName.startsWith("req-") || !fileName.endsWith(".json")) return
        val request = File(directory, fileName)
        if (!request.isFile) return

        val body = try {
            request.readText()
        } catch (e: Exception) {
            Log.w(TAG, "Could not read a host request: ${e.javaClass.simpleName}")
            return
        } finally {
            // Served or not, a request is consumed once. Leaving it would open
            // WhatsApp again on the next event in this directory.
            request.delete()
        }

        val parsed = try {
            JSONObject(body)
        } catch (e: Exception) {
            Log.w(TAG, "Discarded an unreadable host request")
            return
        }

        val id = parsed.optString("id")
        if (id.isEmpty() || !id.all { it in '0'..'9' || it in 'a'..'f' }) {
            Log.w(TAG, "Discarded a host request with no usable id")
            return
        }

        if (parsed.optString("action") != ACTION_SELF_CHAT) {
            reply(id, "error", "unsupported_action")
            return
        }

        val selfNumber = parsed.optString("number")
        val message = parsed.optString("message")
        if (message.isEmpty()) {
            reply(id, "error", "empty_message")
            return
        }

        val failure = WhatsAppSelfChatLauncher.open(context, selfNumber, message)
        if (failure == null) {
            reply(id, "opened", null)
        } else {
            reply(id, "error", failure)
        }
        // Only the outcome is recorded. The number and the body are private.
        Log.i(TAG, "whatsapp_self_chat status=${failure ?: "opened"} message_chars=${message.length}")
    }

    private fun reply(id: String, status: String, reason: String?) {
        val payload = JSONObject()
            .put("id", id)
            .put("status", status)
        if (reason != null) {
            payload.put("error", reason)
        }
        try {
            File(directory, "res-$id.json").writeText(payload.toString())
        } catch (e: Exception) {
            Log.w(TAG, "Could not answer a host request: ${e.javaClass.simpleName}")
        }
    }
}

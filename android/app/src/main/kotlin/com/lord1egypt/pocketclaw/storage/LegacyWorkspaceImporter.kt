package com.lord1egypt.pocketclaw.storage

import android.app.Activity
import android.content.ContentResolver
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Environment
import android.os.Handler
import android.os.Looper
import android.provider.DocumentsContract
import android.util.Log
import java.io.File
import java.io.FileOutputStream
import java.util.Date

/**
 * Copies a workspace left in `Download/pocketclaw` by an older install into the
 * current one, when and only when the owner asks.
 *
 * PC-DEF-077. PocketClaw no longer holds all-files access, so the old shared
 * folder cannot be read directly. The owner picks it with the system document
 * tree picker, which grants read access to that one tree and nothing else, and
 * its contents are copied into a new folder inside the agent's workspace
 * directory, where the agent can reach them. The source is
 * never modified or deleted, nothing in the workspace is overwritten, and the
 * two workspaces are never merged: the copy is a folder the owner can inspect
 * and move from.
 */
class LegacyWorkspaceImporter(private val activity: Activity) {

    companion object {
        private const val TAG = "LegacyWorkspace"

        /** Distinct from ChatImagePicker and anything Flutter's plugins use. */
        const val REQUEST_CODE = 0x9102

        const val STATUS_COPIED = "copied"
        const val STATUS_PARTIAL = "partial"
        const val STATUS_FAILED = "failed"
        const val STATUS_CANCELLED = "cancelled"
        const val STATUS_BUSY = "busy"
        const val STATUS_UNAVAILABLE = "unavailable"

        /** Deep enough for any real workspace, shallow enough to stop a loop. */
        private const val MAX_DEPTH = 32

        fun legacyDirectory(): File = File(
            Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS),
            "pocketclaw",
        )

        /**
         * Whether the old shared workspace can be seen.
         *
         * A directory lookup needs no permission on scoped storage, so an
         * upgraded install can usually tell. Where it cannot -- Android 7-9
         * without a storage grant -- the answer is false and nothing is offered,
         * which is the safe direction.
         */
        fun legacyDirectoryVisible(): Boolean = try {
            legacyDirectory().isDirectory
        } catch (e: SecurityException) {
            false
        }

        private fun initialTreeUri(): Uri = DocumentsContract.buildDocumentUri(
            "com.android.externalstorage.documents",
            "primary:${Environment.DIRECTORY_DOWNLOADS}/pocketclaw",
        )
    }

    data class Outcome(
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
    }

    private val mainHandler = Handler(Looper.getMainLooper())
    private var pending: ((Outcome) -> Unit)? = null
    private var destinationWorkspace: File? = null

    /** Opens the picker. [onResult] is always called exactly once. */
    fun start(workspace: File, onResult: (Outcome) -> Unit) {
        if (pending != null) {
            onResult(Outcome(STATUS_BUSY))
            return
        }
        val intent = Intent(Intent.ACTION_OPEN_DOCUMENT_TREE).apply {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                putExtra(DocumentsContract.EXTRA_INITIAL_URI, initialTreeUri())
            }
        }
        pending = onResult
        destinationWorkspace = workspace
        try {
            @Suppress("DEPRECATION")
            activity.startActivityForResult(intent, REQUEST_CODE)
        } catch (e: Exception) {
            Log.w(TAG, "No document tree picker available: ${e.javaClass.simpleName}")
            pending = null
            onResult(Outcome(STATUS_UNAVAILABLE))
        }
    }

    /** Routes an Activity result. Returns true when it was this importer's. */
    fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?): Boolean {
        if (requestCode != REQUEST_CODE) return false
        val answer = pending ?: return true
        pending = null
        val tree = if (resultCode == Activity.RESULT_OK) data?.data else null
        val workspace = destinationWorkspace
        if (tree == null || workspace == null) {
            answer(Outcome(STATUS_CANCELLED))
            return true
        }
        val resolver = activity.applicationContext.contentResolver
        // A workspace can hold many files; copying must not block the UI.
        Thread {
            val outcome = try {
                copyTree(resolver, tree, workspace)
            } catch (e: Exception) {
                Log.w(TAG, "legacy workspace import failed: ${e.javaClass.simpleName}")
                Outcome(STATUS_FAILED)
            }
            Log.i(
                TAG,
                "legacy workspace import status=${outcome.status}" +
                    " files=${outcome.files} failed=${outcome.failed}",
            )
            mainHandler.post { answer(outcome) }
        }.start()
        return true
    }

    /** Answers an import that can no longer be delivered. */
    fun cancelPending() {
        val answer = pending ?: return
        pending = null
        answer(Outcome(STATUS_CANCELLED))
    }

    private fun copyTree(resolver: ContentResolver, tree: Uri, workspace: File): Outcome {
        workspace.mkdirs()
        val destination = WorkspaceImportRules.freshDestination(workspace, Date())
        if (!destination.mkdirs()) return Outcome(STATUS_FAILED)

        var copied = 0
        var failed = 0

        fun walk(parentId: String, dir: File, depth: Int) {
            if (depth > MAX_DEPTH) {
                failed++
                return
            }
            val children = DocumentsContract.buildChildDocumentsUriUsingTree(tree, parentId)
            val cursor = resolver.query(
                children,
                arrayOf(
                    DocumentsContract.Document.COLUMN_DOCUMENT_ID,
                    DocumentsContract.Document.COLUMN_DISPLAY_NAME,
                    DocumentsContract.Document.COLUMN_MIME_TYPE,
                ),
                null,
                null,
                null,
            )
            if (cursor == null) {
                failed++
                return
            }
            cursor.use { rows ->
                while (rows.moveToNext()) {
                    val id = rows.getString(0)
                    val name = WorkspaceImportRules.safeChildName(rows.getString(1))
                    val mime = rows.getString(2)
                    if (id == null || name == null) {
                        failed++
                        continue
                    }
                    val target = File(dir, name)
                    // Never overwrite, even inside the new folder: two source
                    // entries with one name keep the first and count the second.
                    if (!WorkspaceImportRules.isInside(destination, target) || target.exists()) {
                        failed++
                        continue
                    }
                    if (mime == DocumentsContract.Document.MIME_TYPE_DIR) {
                        if (target.mkdir()) walk(id, target, depth + 1) else failed++
                        continue
                    }
                    try {
                        val source = DocumentsContract.buildDocumentUriUsingTree(tree, id)
                        val input = resolver.openInputStream(source)
                        if (input == null) {
                            failed++
                            continue
                        }
                        input.use { stream ->
                            FileOutputStream(target).use { out -> stream.copyTo(out) }
                        }
                        copied++
                    } catch (e: Exception) {
                        target.delete()
                        failed++
                    }
                }
            }
        }

        walk(DocumentsContract.getTreeDocumentId(tree), destination, 0)

        val status = when {
            failed == 0 -> STATUS_COPIED
            copied > 0 -> STATUS_PARTIAL
            else -> STATUS_FAILED
        }
        return Outcome(status, destination.name, copied, failed)
    }
}

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
import java.io.InputStream

/**
 * Copies a workspace left in `Download/pocketclaw` by an older install into the
 * current one, when and only when the owner asks.
 *
 * PC-DEF-077. PocketClaw no longer holds all-files access, so the old shared
 * folder cannot be read directly. The owner picks it with the system document
 * tree picker, which grants read access to that one tree and nothing else, and
 * its contents are copied into a new folder inside the agent's workspace
 * directory, where the agent can reach them. The source is never modified or
 * deleted, nothing in the workspace is overwritten, and the two workspaces are
 * never merged. The copy itself is [WorkspaceTreeCopier]; this class is the
 * Android glue around it.
 */
class LegacyWorkspaceImporter(private val activity: Activity) {

    companion object {
        private const val TAG = "LegacyWorkspace"

        /** Distinct from ChatImagePicker and anything Flutter's plugins use. */
        const val REQUEST_CODE = 0x9102

        fun legacyDirectory(): File = File(
            Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DOWNLOADS),
            "pocketclaw",
        )

        /**
         * Whether an old PocketClaw workspace is visible at `Download/pocketclaw`.
         *
         * Evidence, not existence: see [WorkspaceImportRules.looksLikeLegacyWorkspace].
         * Without a storage permission the app sees only files it created
         * itself, which is exactly what an earlier install of this package left
         * there; anything else counts as absent, the safe direction.
         */
        fun legacyWorkspacePresent(): Boolean =
            WorkspaceImportRules.looksLikeLegacyWorkspace(legacyDirectory())

        private fun initialTreeUri(): Uri = DocumentsContract.buildDocumentUri(
            "com.android.externalstorage.documents",
            "primary:${Environment.DIRECTORY_DOWNLOADS}/pocketclaw",
        )
    }

    private val mainHandler = Handler(Looper.getMainLooper())
    private val gate = ImportGate()
    private var pending: ((ImportOutcome) -> Unit)? = null
    private var destinationWorkspace: File? = null

    /** Opens the picker. [onResult] is always called exactly once. */
    fun start(workspace: File, onResult: (ImportOutcome) -> Unit) {
        if (!gate.tryAcquire()) {
            onResult(ImportOutcome(ImportOutcome.BUSY))
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
            finish(ImportOutcome(ImportOutcome.UNAVAILABLE))
        }
    }

    /** Routes an Activity result. Returns true when it was this importer's. */
    fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?): Boolean {
        if (requestCode != REQUEST_CODE) return false
        if (pending == null) return true
        val tree = if (resultCode == Activity.RESULT_OK) data?.data else null
        val workspace = destinationWorkspace
        if (tree == null || workspace == null) {
            // Cancelling touches nothing: no folder is created, nothing is read.
            finish(ImportOutcome(ImportOutcome.CANCELLED))
            return true
        }
        val source = SafImportTree(activity.applicationContext.contentResolver, tree)
        // A workspace can hold many files; copying must not block the UI. The
        // gate stays held until the answer is delivered, so a second import
        // cannot start while this one is still writing.
        Thread {
            val outcome = try {
                WorkspaceTreeCopier.copy(source, workspace)
            } catch (e: Exception) {
                Log.w(TAG, "legacy workspace import failed: ${e.javaClass.simpleName}")
                ImportOutcome(ImportOutcome.FAILED)
            }
            Log.i(
                TAG,
                "legacy workspace import status=${outcome.status}" +
                    " files=${outcome.files} failed=${outcome.failed}",
            )
            mainHandler.post { finish(outcome) }
        }.start()
        return true
    }

    /** Answers an import whose picker can no longer return. */
    fun cancelPending() {
        if (pending != null) finish(ImportOutcome(ImportOutcome.CANCELLED))
    }

    private fun finish(outcome: ImportOutcome) {
        val answer = pending
        pending = null
        destinationWorkspace = null
        gate.release()
        answer?.invoke(outcome)
    }
}

/** The owner-picked document tree, read through the grant the picker gave. */
private class SafImportTree(
    private val resolver: ContentResolver,
    private val tree: Uri,
) : ImportTree {
    override fun rootId(): String = DocumentsContract.getTreeDocumentId(tree)

    override fun children(parentId: String): List<ImportTree.Entry>? {
        val uri = DocumentsContract.buildChildDocumentsUriUsingTree(tree, parentId)
        val cursor = try {
            resolver.query(
                uri,
                arrayOf(
                    DocumentsContract.Document.COLUMN_DOCUMENT_ID,
                    DocumentsContract.Document.COLUMN_DISPLAY_NAME,
                    DocumentsContract.Document.COLUMN_MIME_TYPE,
                ),
                null,
                null,
                null,
            )
        } catch (e: SecurityException) {
            null
        } ?: return null
        return cursor.use { rows ->
            buildList {
                while (rows.moveToNext()) {
                    add(
                        ImportTree.Entry(
                            id = rows.getString(0),
                            name = rows.getString(1),
                            isDirectory = rows.getString(2) == DocumentsContract.Document.MIME_TYPE_DIR,
                        )
                    )
                }
            }
        }
    }

    override fun open(id: String): InputStream? =
        resolver.openInputStream(DocumentsContract.buildDocumentUriUsingTree(tree, id))
}

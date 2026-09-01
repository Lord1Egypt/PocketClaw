package com.lord1egypt.pocketclaw.media

import android.app.Activity
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.provider.MediaStore
import android.util.Log

/**
 * Opens Android's own image picker for the console's Chat attachment button.
 *
 * The system photo picker is used where it exists, and `ACTION_OPEN_DOCUMENT`
 * below that. Both hand back a `content://` URI already readable by this
 * process, which is what the WebView needs — and neither requires a storage
 * permission, so PocketClaw asks for none.
 */
class ChatImagePicker(private val activity: Activity) {

    companion object {
        private const val TAG = "ChatImagePicker"

        /** Distinct from anything Flutter's plugins use. */
        const val REQUEST_CODE = 0x9101

        /** The picker returned a file this app cannot read. */
        const val ERROR_UNREADABLE = "unreadable"

        /** The picker returned something that is not an image. */
        const val ERROR_UNSUPPORTED_TYPE = "unsupported_type"
    }

    /** Answered with the chosen URI, or null when the user cancelled. */
    private var pending: ((Result<String?>) -> Unit)? = null

    /**
     * Starts the picker. [onResult] is always called exactly once.
     *
     * A second request while one is open is answered as a cancellation rather
     * than queued: the WebView would otherwise leave its file input waiting
     * forever on a pick the user never sees.
     */
    fun pick(acceptTypes: List<String>, onResult: (Result<String?>) -> Unit) {
        if (pending != null) {
            onResult(Result.success(null))
            return
        }

        val mimeTypes = ChatImageFilter.mimeTypes(acceptTypes)
        val intent = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            // The system photo picker: no permission, and the user only ever
            // exposes the one image they chose.
            Intent(MediaStore.ACTION_PICK_IMAGES).apply { type = "image/*" }
        } else {
            Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
                addCategory(Intent.CATEGORY_OPENABLE)
                type = mimeTypes.singleOrNull() ?: "image/*"
                if (mimeTypes.size > 1) {
                    putExtra(Intent.EXTRA_MIME_TYPES, mimeTypes.toTypedArray())
                }
            }
        }

        pending = onResult
        try {
            @Suppress("DEPRECATION")
            activity.startActivityForResult(intent, REQUEST_CODE)
        } catch (e: Exception) {
            Log.w(TAG, "No image picker available: ${e.javaClass.simpleName}")
            pending = null
            onResult(Result.success(null))
        }
    }

    /**
     * Routes an Activity result. Returns true when it was this picker's.
     */
    fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?): Boolean {
        if (requestCode != REQUEST_CODE) return false
        val answer = pending ?: return true
        pending = null

        val uri = if (resultCode == Activity.RESULT_OK) data?.data else null
        if (uri == null) {
            // Cancelling is an ordinary outcome, not an error: Chat comes back
            // with no attachment and nothing else changes.
            Log.i(TAG, "chat attachment status=cancelled")
            answer(Result.success(null))
            return true
        }

        val mimeType = activity.contentResolver.getType(uri)
        if (!ChatImageFilter.isImage(mimeType)) {
            Log.i(TAG, "chat attachment status=rejected reason=$ERROR_UNSUPPORTED_TYPE")
            answer(Result.failure(IllegalArgumentException(ERROR_UNSUPPORTED_TYPE)))
            return true
        }

        // Prove the grant is usable before handing the URI on. A provider that
        // revoked access, or a file that has gone away, would otherwise surface
        // as an empty attachment with no explanation.
        try {
            activity.contentResolver.openInputStream(uri).use { stream ->
                if (stream == null) throw IllegalStateException(ERROR_UNREADABLE)
            }
        } catch (e: Exception) {
            Log.i(TAG, "chat attachment status=rejected reason=$ERROR_UNREADABLE")
            answer(Result.failure(IllegalStateException(ERROR_UNREADABLE)))
            return true
        }

        // The URI itself is not logged: it names a file in the user's gallery.
        Log.i(TAG, "chat attachment media_type=image status=selected")
        answer(Result.success(uri.toString()))
        return true
    }

    /**
     * Answers a pick that can no longer complete, so the page's file input is
     * released instead of staying stuck after the Activity goes away.
     */
    fun cancelPending() {
        val answer = pending ?: return
        pending = null
        answer(Result.success(null))
    }
}

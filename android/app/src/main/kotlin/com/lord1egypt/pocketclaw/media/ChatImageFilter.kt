package com.lord1egypt.pocketclaw.media

/**
 * Turns a web page's `accept` attribute into the MIME filter the Android
 * picker is given.
 *
 * Kept free of Android types so the filter is checked on the JVM. The V1 scope
 * is one image at a time, so anything the page asks for that is not an image is
 * dropped rather than widening the picker into a general file browser.
 */
object ChatImageFilter {

    private const val ANY_IMAGE = "image/*"

    /**
     * Returns the MIME types to offer. Never empty, and never anything outside
     * `image/`.
     *
     * An empty request, or one that names no usable image type, becomes
     * "any image" rather than nothing at all — a picker with an empty filter
     * shows the user no files and reads as the button being broken.
     */
    fun mimeTypes(acceptTypes: List<String>): List<String> {
        val images = acceptTypes
            .map { it.trim().lowercase() }
            .filter { it.startsWith("image/") }
            .distinct()
        if (images.isEmpty() || images.contains(ANY_IMAGE)) {
            return listOf(ANY_IMAGE)
        }
        return images
    }

    /** Whether a MIME type reported by the content resolver is an image. */
    fun isImage(mimeType: String?): Boolean =
        mimeType != null && mimeType.trim().lowercase().startsWith("image/")
}

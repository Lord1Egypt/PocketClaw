package com.lord1egypt.pocketclaw.media

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * What the Chat attachment picker is allowed to show.
 *
 * The console's file input asks for a list of image types; anything wider than
 * that would turn an image attachment into a general file browser, which is not
 * what V1 supports.
 */
class ChatImageFilterTest {

    @Test
    fun `the console's own accept list passes through unchanged`() {
        assertEquals(
            listOf("image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp"),
            ChatImageFilter.mimeTypes(
                listOf("image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp"),
            ),
        )
    }

    @Test
    fun `nothing requested still shows images, not an empty picker`() {
        assertEquals(listOf("image/*"), ChatImageFilter.mimeTypes(emptyList()))
    }

    @Test
    fun `a wildcard collapses to any image`() {
        assertEquals(
            listOf("image/*"),
            ChatImageFilter.mimeTypes(listOf("image/png", "image/*")),
        )
    }

    @Test
    fun `non-image types are dropped rather than widening the picker`() {
        assertEquals(
            listOf("image/png"),
            ChatImageFilter.mimeTypes(listOf("image/png", "application/pdf", "video/mp4")),
        )
        assertEquals(
            listOf("image/*"),
            ChatImageFilter.mimeTypes(listOf("application/pdf", "video/mp4")),
        )
    }

    @Test
    fun `casing and stray spacing do not defeat the filter`() {
        assertEquals(
            listOf("image/jpeg"),
            ChatImageFilter.mimeTypes(listOf(" Image/JPEG ", "image/jpeg")),
        )
    }

    @Test
    fun `only image content types are accepted back from the picker`() {
        assertTrue(ChatImageFilter.isImage("image/jpeg"))
        assertTrue(ChatImageFilter.isImage("IMAGE/PNG"))
        assertFalse(ChatImageFilter.isImage("application/pdf"))
        assertFalse(ChatImageFilter.isImage("video/mp4"))
        assertFalse(ChatImageFilter.isImage(null))
        assertFalse(ChatImageFilter.isImage(""))
    }
}

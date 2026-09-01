package com.lord1egypt.pocketclaw.whatsapp

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Which WhatsApp gets opened, and with what link.
 *
 * These run on the JVM, without a device: the two decisions that can send a
 * message to the wrong place — the package and the number — are functions of
 * their inputs, not of the platform.
 */
class WhatsAppSelfChatLinkTest {

    @Test
    fun `regular WhatsApp wins when both are installed`() {
        assertEquals(
            WhatsAppSelfChatLink.PACKAGE_STANDARD,
            WhatsAppSelfChatLink.choosePackage(
                setOf(
                    WhatsAppSelfChatLink.PACKAGE_STANDARD,
                    WhatsAppSelfChatLink.PACKAGE_BUSINESS,
                ),
            ),
        )
    }

    @Test
    fun `Business is used when it is the only one installed`() {
        assertEquals(
            WhatsAppSelfChatLink.PACKAGE_BUSINESS,
            WhatsAppSelfChatLink.choosePackage(setOf(WhatsAppSelfChatLink.PACKAGE_BUSINESS)),
        )
    }

    @Test
    fun `regular WhatsApp is used when it is the only one installed`() {
        assertEquals(
            WhatsAppSelfChatLink.PACKAGE_STANDARD,
            WhatsAppSelfChatLink.choosePackage(setOf(WhatsAppSelfChatLink.PACKAGE_STANDARD)),
        )
    }

    @Test
    fun `neither installed means there is nothing to open`() {
        assertNull(WhatsAppSelfChatLink.choosePackage(emptySet()))
        assertNull(WhatsAppSelfChatLink.choosePackage(setOf("com.example.notwhatsapp")))
    }

    @Test
    fun `a canonical number keeps its digits`() {
        assertEquals("201012345678", WhatsAppSelfChatLink.digitsOf("+201012345678"))
        assertEquals("905321234567", WhatsAppSelfChatLink.digitsOf("  +905321234567  "))
        assertEquals("201012345678", WhatsAppSelfChatLink.digitsOf("201012345678"))
    }

    @Test
    fun `a number that is not international is refused`() {
        // Anything that could address somebody else's chat has to fail here
        // rather than become a link.
        assertNull(WhatsAppSelfChatLink.digitsOf(""))
        assertNull(WhatsAppSelfChatLink.digitsOf("01012345678"))
        assertNull(WhatsAppSelfChatLink.digitsOf("+20 101 234 5678"))
        assertNull(WhatsAppSelfChatLink.digitsOf("+201234"))
        assertNull(WhatsAppSelfChatLink.digitsOf("+2010123456789012"))
        assertNull(WhatsAppSelfChatLink.digitsOf("+2010CALLME"))
    }

    @Test
    fun `the link carries the number and the message`() {
        val links = WhatsAppSelfChatLink.linksFor("201012345678", "PocketClaw WhatsApp test")

        assertEquals(
            listOf(
                "whatsapp://send?phone=201012345678&text=PocketClaw%20WhatsApp%20test",
                "https://wa.me/201012345678?text=PocketClaw%20WhatsApp%20test",
            ),
            links,
        )
    }

    @Test
    fun `spaces never reach WhatsApp as a literal plus`() {
        val links = WhatsAppSelfChatLink.linksFor("201012345678", "a b")
        assertTrue(links.all { it.endsWith("a%20b") })
        assertTrue(links.none { it.contains("a+b") })
    }

    @Test
    fun `a literal plus in the message survives`() {
        val links = WhatsAppSelfChatLink.linksFor("201012345678", "1+1")
        assertTrue(links.all { it.endsWith("1%2B1") })
    }

    @Test
    fun `Arabic text and an emoji survive encoding`() {
        val links = WhatsAppSelfChatLink.linksFor("201012345678", "مرحبا من PocketClaw 🦞")

        // UTF-8 percent-encoding, and no raw bytes left in the URL.
        val encoded = links.first().substringAfter("text=")
        assertEquals(
            "%D9%85%D8%B1%D8%AD%D8%A8%D8%A7%20%D9%85%D9%86%20PocketClaw%20%F0%9F%A6%9E",
            encoded,
        )
    }

    @Test
    fun `a visible window may start an activity`() {
        // ActivityManager.RunningAppProcessInfo.IMPORTANCE_FOREGROUND
        assertTrue(WhatsAppSelfChatLink.canStartActivity(100))
        assertTrue(WhatsAppSelfChatLink.canStartActivity(50))
    }

    @Test
    fun `a foreground service alone may not`() {
        // A blocked background start returns normally and shows nothing, so
        // anything short of a visible window has to be refused rather than
        // reported as opened.
        assertFalse(WhatsAppSelfChatLink.canStartActivity(125)) // FOREGROUND_SERVICE
        assertFalse(WhatsAppSelfChatLink.canStartActivity(200)) // VISIBLE
        assertFalse(WhatsAppSelfChatLink.canStartActivity(400)) // CACHED
    }
}

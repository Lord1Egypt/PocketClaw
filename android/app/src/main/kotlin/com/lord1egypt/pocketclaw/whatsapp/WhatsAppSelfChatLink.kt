package com.lord1egypt.pocketclaw.whatsapp

import java.net.URLEncoder

/**
 * How PocketClaw addresses the user's own WhatsApp chat.
 *
 * Deliberately free of Android types so the decisions that matter — which app
 * to open, and what link to hand it — are checked on the JVM rather than only
 * on a device.
 *
 * Nothing here reads WhatsApp, holds a session, or sends anything. It builds a
 * link. The Send button stays inside WhatsApp, under the user.
 */
object WhatsAppSelfChatLink {

    const val PACKAGE_STANDARD = "com.whatsapp"
    const val PACKAGE_BUSINESS = "com.whatsapp.w4b"

    /** E.164 bounds; see pkg/whatsapp/selfchat for the same rule in Core. */
    private const val MIN_DIGITS = 8
    private const val MAX_DIGITS = 15

    /**
     * Picks the app to open.
     *
     * Deterministic and documented: regular WhatsApp wins whenever it is
     * installed, Business is used only when it is the only one present, and
     * neither means there is nothing to open. A chooser was rejected because
     * this runs from an agent tool as well as from a button, and a dialog no
     * one is looking at is not an answer.
     */
    fun choosePackage(installedPackages: Set<String>): String? = when {
        installedPackages.contains(PACKAGE_STANDARD) -> PACKAGE_STANDARD
        installedPackages.contains(PACKAGE_BUSINESS) -> PACKAGE_BUSINESS
        else -> null
    }

    /**
     * Returns the canonical digits of an international number, or null when the
     * input is not one.
     *
     * Core and the console both normalise before this point. This re-checks
     * anyway: the number arrives over a bridge, and a malformed one would
     * otherwise become a link to somebody else's chat.
     */
    fun digitsOf(selfNumber: String): String? {
        val trimmed = selfNumber.trim()
        val body = if (trimmed.startsWith("+")) trimmed.substring(1) else trimmed
        if (body.length < MIN_DIGITS || body.length > MAX_DIGITS) return null
        if (body.startsWith("0")) return null
        if (!body.all { it in '0'..'9' }) return null
        return body
    }

    /**
     * The links to try, in order, for a validated number.
     *
     * The `whatsapp://` scheme is first because both packages register it
     * unconditionally; the `wa.me` https link is the documented click-to-chat
     * form and covers a build that handles only that. The caller stops at the
     * first one the chosen package can actually open.
     */
    fun linksFor(digits: String, message: String): List<String> {
        val text = encodeQueryComponent(message)
        return listOf(
            "whatsapp://send?phone=$digits&text=$text",
            "https://wa.me/$digits?text=$text",
        )
    }

    /**
     * Percent-encodes a query value.
     *
     * URLEncoder writes a space as "+", which a receiver is free to read as a
     * literal plus. Rewriting it to %20 keeps a message with spaces — or with
     * Arabic text and an emoji — arriving exactly as it was written.
     */
    private fun encodeQueryComponent(value: String): String =
        URLEncoder.encode(value, "UTF-8").replace("+", "%20")
}

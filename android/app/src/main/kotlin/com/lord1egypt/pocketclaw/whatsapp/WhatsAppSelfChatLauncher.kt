package com.lord1egypt.pocketclaw.whatsapp

import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.util.Log

/**
 * Opens WhatsApp on the user's own chat with a message prepared.
 *
 * The Intent is where PocketClaw stops. WhatsApp shows the compose box with the
 * text in it and waits; nothing here presses Send, reads a chat, or touches a
 * WhatsApp session.
 */
object WhatsAppSelfChatLauncher {

    private const val TAG = "WhatsAppSelfChat"

    /** Why nothing opened. These strings are the bridge's error vocabulary. */
    object Failure {
        const val INVALID_NUMBER = "invalid_number"
        const val NOT_INSTALLED = "not_installed"
        const val START_FAILED = "start_failed"
    }

    /**
     * Returns null on success, or one of [Failure] when WhatsApp did not open.
     *
     * Neither the number nor the message is logged.
     */
    fun open(context: Context, selfNumber: String, message: String): String? {
        val digits = WhatsAppSelfChatLink.digitsOf(selfNumber)
            ?: return Failure.INVALID_NUMBER

        val target = WhatsAppSelfChatLink.choosePackage(installedWhatsAppPackages(context))
            ?: return Failure.NOT_INSTALLED

        for (link in WhatsAppSelfChatLink.linksFor(digits, message)) {
            val intent = Intent(Intent.ACTION_VIEW, Uri.parse(link)).apply {
                setPackage(target)
                // The request can arrive from the foreground service on behalf
                // of the agent tool, which has no Activity of its own to start
                // from.
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            try {
                context.startActivity(intent)
                return null
            } catch (e: Exception) {
                // A package that does not handle this particular link shape is
                // expected; the next candidate covers it. The link is not
                // logged because it carries the message body.
                Log.i(TAG, "WhatsApp did not accept a link shape: ${e.javaClass.simpleName}")
            }
        }
        return Failure.START_FAILED
    }

    /**
     * Which WhatsApp packages this device has.
     *
     * Android 11+ hides other packages unless they are declared in <queries>,
     * which the manifest does for exactly these two and nothing else.
     */
    private fun installedWhatsAppPackages(context: Context): Set<String> {
        val manager = context.packageManager
        return setOf(
            WhatsAppSelfChatLink.PACKAGE_STANDARD,
            WhatsAppSelfChatLink.PACKAGE_BUSINESS,
        ).filterTo(mutableSetOf()) { packageName ->
            try {
                manager.getPackageInfo(packageName, 0)
                true
            } catch (e: PackageManager.NameNotFoundException) {
                false
            }
        }
    }
}

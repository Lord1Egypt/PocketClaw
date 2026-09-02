package com.lord1egypt.pocketclaw.whatsapp

import java.io.File

/**
 * Where the experimental WhatsApp Agent Channel keeps its state, and the
 * environment variables that tell Core about it.
 *
 * Both directories are derived from a caller-supplied root so this is testable
 * without an Android [android.content.Context]. The caller passes
 * `context.noBackupFilesDir`, and that choice carries the security properties:
 *
 *  - app-private, so no other app on the device can read it;
 *  - outside the workspace, which on Android is public external storage and is
 *    also the root the agent's own file tools are restricted to;
 *  - excluded from cloud backup and device-to-device transfer, because Android
 *    never includes `no_backup` in either — without needing a rule in
 *    backup_rules.xml or data_extraction_rules.xml, and so without touching the
 *    GitHub credential rules that already live there.
 *
 * The session database holds the linked device's Signal identity keys: whoever
 * reads it can impersonate the account. Core treats these variables as
 * authoritative and ignores any `session_store_path` in the config file, which
 * the agent can write.
 */
object WhatsAppAgentStorage {

    /** Names the whatsmeow session database directory for Core. */
    const val ENV_SESSION_DIR = "POCKETCLAW_WHATSAPP_SESSION_DIR"

    /**
     * Names the directory the gateway and the console backend exchange pairing
     * state through, including a live QR. Both are this UID, so it needs no
     * port and no shared secret.
     */
    const val ENV_PAIRING_DIR = "POCKETCLAW_WHATSAPP_PAIR_DIR"

    private const val SESSION_DIR_NAME = "whatsapp"
    private const val PAIRING_DIR_NAME = "whatsapp-pairing"

    /** The session database directory under [noBackupRoot]. */
    fun sessionDir(noBackupRoot: File): File = File(noBackupRoot, SESSION_DIR_NAME)

    /** The pairing state directory under [noBackupRoot]. */
    fun pairingDir(noBackupRoot: File): File = File(noBackupRoot, PAIRING_DIR_NAME)

    /**
     * Creates both directories and returns the environment entries Core reads.
     */
    fun environment(noBackupRoot: File): Map<String, String> {
        val session = sessionDir(noBackupRoot).apply { mkdirs() }
        val pairing = pairingDir(noBackupRoot).apply { mkdirs() }
        return mapOf(
            ENV_SESSION_DIR to session.absolutePath,
            ENV_PAIRING_DIR to pairing.absolutePath,
        )
    }
}

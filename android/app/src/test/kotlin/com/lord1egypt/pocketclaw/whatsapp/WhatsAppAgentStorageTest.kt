package com.lord1egypt.pocketclaw.whatsapp

import java.io.File
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder

/**
 * The WhatsApp Agent Channel's storage contract.
 *
 * The property under test is not "the paths are these strings" but "the paths
 * are derived from the no-backup root the caller passes, and from nothing else".
 * That is what keeps the whatsmeow session database — which holds the linked
 * device's identity keys — out of the workspace the agent can read and out of
 * Android backup.
 */
class WhatsAppAgentStorageTest {

    @get:Rule
    val temporaryFolder = TemporaryFolder()

    @Test
    fun `session and pairing directories live under the supplied root`() {
        val root = temporaryFolder.newFolder("no_backup")

        val session = WhatsAppAgentStorage.sessionDir(root)
        val pairing = WhatsAppAgentStorage.pairingDir(root)

        assertEquals(root, session.parentFile)
        assertEquals(root, pairing.parentFile)
    }

    @Test
    fun `session and pairing directories are distinct`() {
        val root = temporaryFolder.newFolder("no_backup")

        assertFalse(
            "the pairing snapshot must not share a directory with the session database",
            WhatsAppAgentStorage.sessionDir(root) == WhatsAppAgentStorage.pairingDir(root),
        )
    }

    @Test
    fun `environment names both directories for Core`() {
        val root = temporaryFolder.newFolder("no_backup")

        val environment = WhatsAppAgentStorage.environment(root)

        assertEquals(
            WhatsAppAgentStorage.sessionDir(root).absolutePath,
            environment[WhatsAppAgentStorage.ENV_SESSION_DIR],
        )
        assertEquals(
            WhatsAppAgentStorage.pairingDir(root).absolutePath,
            environment[WhatsAppAgentStorage.ENV_PAIRING_DIR],
        )
    }

    @Test
    fun `environment creates both directories`() {
        val root = temporaryFolder.newFolder("no_backup")

        WhatsAppAgentStorage.environment(root)

        assertTrue(WhatsAppAgentStorage.sessionDir(root).isDirectory)
        assertTrue(WhatsAppAgentStorage.pairingDir(root).isDirectory)
    }

    @Test
    fun `neither directory escapes the supplied root`() {
        // A path that walked out of the root would defeat the whole point: the
        // workspace and the public Download directory are both reachable from
        // one level up on a real device.
        val root = temporaryFolder.newFolder("no_backup")
        val rootPath = root.canonicalPath + File.separator

        for (directory in listOf(
            WhatsAppAgentStorage.sessionDir(root),
            WhatsAppAgentStorage.pairingDir(root),
        )) {
            assertTrue(
                "${directory.canonicalPath} escaped $rootPath",
                directory.canonicalPath.startsWith(rootPath),
            )
        }
    }
}

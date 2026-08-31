package com.lord1egypt.pocketclaw.security

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.io.File
import java.security.GeneralSecurityException
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec
import org.json.JSONObject

/**
 * PocketClaw's GitHub credential, encrypted under a key that never leaves the
 * Android Keystore.
 *
 * The token is the one secret in the app that a user cannot rotate by looking
 * at a config file, so it is not kept in one. Only ciphertext reaches storage,
 * and the key that decrypts it is generated inside the Keystore and cannot be
 * exported from it — an attacker with a copy of the app's data directory has an
 * opaque blob and nothing else.
 *
 * Nothing here returns the token to Flutter or to the agent. The only reader is
 * [token], which the service calls when it starts Core, so the credential
 * travels from the Keystore into the Core process environment and stops there.
 */
object GitHubCredentialStore {

    private const val KEY_ALIAS = "pocketclaw.github.credential.v1"
    private const val KEYSTORE = "AndroidKeyStore"
    private const val TRANSFORMATION = "AES/GCM/NoPadding"
    private const val TAG_BITS = 128

    /** Guards against reading a blob written by a future, different format. */
    private const val FORMAT_VERSION = 1

    private const val DIRECTORY = "credentials"
    private const val CIPHERTEXT_FILE = "github.bin"
    private const val METADATA_FILE = "github.json"

    /** What the UI is allowed to know: whether there is a credential, and whose. */
    data class Status(val connected: Boolean, val login: String?) {
        fun asMap(): Map<String, Any?> = mapOf("connected" to connected, "login" to login)
    }

    @Synchronized
    fun connect(context: Context, token: String, login: String) {
        require(token.isNotBlank()) { "refusing to store an empty credential" }

        val cipher = Cipher.getInstance(TRANSFORMATION).apply {
            // No IV is supplied here on purpose. The key is generated with
            // randomized encryption required, so the provider draws the nonce
            // from the platform CSPRNG and rejects any caller-chosen one.
            init(Cipher.ENCRYPT_MODE, secretKey())
        }
        val nonce = cipher.iv
        val ciphertext = cipher.doFinal(token.toByteArray(Charsets.UTF_8))

        val blob = ByteArray(2 + nonce.size + ciphertext.size)
        blob[0] = FORMAT_VERSION.toByte()
        blob[1] = nonce.size.toByte()
        System.arraycopy(nonce, 0, blob, 2, nonce.size)
        System.arraycopy(ciphertext, 0, blob, 2 + nonce.size, ciphertext.size)

        writeAtomically(ciphertextFile(context), blob)
        writeAtomically(
            metadataFile(context),
            JSONObject()
                .put("login", login)
                .put("connected_at", System.currentTimeMillis())
                .toString()
                .toByteArray(Charsets.UTF_8),
        )
    }

    /**
     * The stored credential, or null when there is none or it cannot be
     * decrypted.
     *
     * Every failure path deletes the material and reports "not connected". A
     * credential that will not decrypt is not recoverable — the Keystore key is
     * gone, or the blob was tampered with, and GCM's tag is what tells the
     * difference from a lucky-looking corruption. Carrying on with a partially
     * trusted secret would be worse than asking the user to connect again.
     */
    @Synchronized
    fun token(context: Context): String? {
        val blob = ciphertextFile(context).takeIf { it.isFile }?.readBytes() ?: return null
        if (blob.size < 3 || blob[0].toInt() != FORMAT_VERSION) {
            forget(context)
            return null
        }
        val nonceSize = blob[1].toInt()
        if (nonceSize <= 0 || blob.size <= 2 + nonceSize) {
            forget(context)
            return null
        }

        return try {
            val nonce = blob.copyOfRange(2, 2 + nonceSize)
            val ciphertext = blob.copyOfRange(2 + nonceSize, blob.size)
            val cipher = Cipher.getInstance(TRANSFORMATION).apply {
                init(Cipher.DECRYPT_MODE, existingKey() ?: return null, GCMParameterSpec(TAG_BITS, nonce))
            }
            String(cipher.doFinal(ciphertext), Charsets.UTF_8).trim().ifEmpty { null }
        } catch (failure: GeneralSecurityException) {
            forget(context)
            null
        } catch (failure: IllegalStateException) {
            forget(context)
            null
        }
    }

    /**
     * Whether a usable credential is present, and the account it belongs to.
     *
     * This decrypts rather than trusting the metadata file: metadata is
     * plaintext and could outlive the key, and reporting "Connected" for a
     * credential that can no longer be decrypted would send the user looking for
     * a fault in GitHub instead of reconnecting.
     */
    @Synchronized
    fun status(context: Context): Status {
        if (token(context) == null) return Status(connected = false, login = null)
        val login = try {
            metadataFile(context).takeIf { it.isFile }
                ?.readText(Charsets.UTF_8)
                ?.let { JSONObject(it).optString("login").ifBlank { null } }
        } catch (failure: Exception) {
            null
        }
        return Status(connected = true, login = login)
    }

    /** Removes the credential and its metadata. Other secrets are untouched. */
    @Synchronized
    fun disconnect(context: Context) {
        forget(context)
        try {
            KeyStore.getInstance(KEYSTORE).apply { load(null) }.deleteEntry(KEY_ALIAS)
        } catch (failure: GeneralSecurityException) {
            // The key is unusable either way; the ciphertext it protected is gone.
        }
    }

    private fun forget(context: Context) {
        ciphertextFile(context).delete()
        metadataFile(context).delete()
    }

    private fun secretKey(): SecretKey = existingKey() ?: generateKey()

    private fun existingKey(): SecretKey? = try {
        val keystore = KeyStore.getInstance(KEYSTORE).apply { load(null) }
        keystore.getKey(KEY_ALIAS, null) as? SecretKey
    } catch (failure: GeneralSecurityException) {
        null
    }

    private fun generateKey(): SecretKey {
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE)
        generator.init(
            KeyGenParameterSpec.Builder(
                KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                // The provider must choose the nonce. A reused nonce under one
                // GCM key is a total loss of confidentiality, and this is the
                // setting that makes reuse impossible rather than unlikely.
                .setRandomizedEncryptionRequired(true)
                .build(),
        )
        return generator.generateKey()
    }

    private fun directory(context: Context): File =
        File(context.filesDir, DIRECTORY).apply { mkdirs() }

    private fun ciphertextFile(context: Context) = File(directory(context), CIPHERTEXT_FILE)

    private fun metadataFile(context: Context) = File(directory(context), METADATA_FILE)

    /**
     * A half-written credential would decrypt as corrupt and disconnect the user
     * silently, so the new bytes are only ever swapped in whole.
     */
    private fun writeAtomically(destination: File, bytes: ByteArray) {
        val staging = File(destination.parentFile, destination.name + ".new")
        staging.writeBytes(bytes)
        if (!staging.renameTo(destination)) {
            destination.delete()
            if (!staging.renameTo(destination)) {
                staging.delete()
                throw IllegalStateException("could not store the credential")
            }
        }
    }
}

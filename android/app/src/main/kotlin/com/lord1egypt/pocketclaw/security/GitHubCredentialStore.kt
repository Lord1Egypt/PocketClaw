package com.lord1egypt.pocketclaw.security

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.io.File
import java.security.GeneralSecurityException
import java.security.KeyStore
import javax.crypto.AEADBadTagException
import javax.crypto.BadPaddingException
import javax.crypto.Cipher
import javax.crypto.IllegalBlockSizeException
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
    data class Status(
        val connected: Boolean,
        val login: String?,
        val unavailableReason: String? = null,
    ) {
        fun asMap(): Map<String, Any?> = mapOf(
            "connected" to connected,
            "login" to login,
            "unavailableReason" to unavailableReason,
        )
    }

    /**
     * The three answers reading the credential can give.
     *
     * [Unavailable] is not [Absent]. A Keystore that is momentarily unreachable
     * has not lost the user's credential, and treating the two the same would
     * turn a transient platform fault into a silent disconnection that the user
     * only discovers when a push fails.
     */
    sealed interface CredentialState {
        data class Present(val token: String) : CredentialState
        object Absent : CredentialState
        data class Unavailable(val reason: String) : CredentialState
    }

    /** What to do with stored material after a failure to read it. */
    internal enum class Recovery { DESTROY, PRESERVE }

    /**
     * Whether a read failure is evidence that the stored material is unusable.
     *
     * Destroying a credential is irreversible, so it happens only on positive
     * evidence: an authentication tag that does not verify, ciphertext that
     * cannot be a valid block sequence, or a key the platform says is
     * permanently invalidated. Everything else — a busy keystore, a provider
     * that failed to load, an error this code has never seen — preserves the
     * ciphertext and reports the credential unavailable for now.
     *
     * The permanently-invalidated case is matched by class name rather than by
     * type. android.security.keystore.KeyPermanentlyInvalidatedException cannot
     * be constructed in a JVM unit test, and a rule that could not be tested is
     * a rule that quietly rots.
     */
    internal fun classify(failure: Throwable): Recovery {
        var cause: Throwable? = failure
        while (cause != null) {
            when {
                cause is AEADBadTagException -> return Recovery.DESTROY
                cause is BadPaddingException -> return Recovery.DESTROY
                cause is IllegalBlockSizeException -> return Recovery.DESTROY
                cause.javaClass.simpleName == "KeyPermanentlyInvalidatedException" ->
                    return Recovery.DESTROY
            }
            cause = cause.cause
        }
        return Recovery.PRESERVE
    }

    /**
     * @param login the account the credential was proved to belong to. It is
     * required, which is what makes storing an unvalidated token impossible:
     * the only source of a login is a successful `gh api user`.
     */
    @Synchronized
    fun connect(context: Context, token: String, login: String) {
        require(token.isNotBlank()) { "refusing to store an empty credential" }
        require(login.isNotBlank()) {
            "refusing to store a credential that GitHub has not accepted"
        }

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
     * The stored credential, or why it cannot be produced.
     *
     * A blob that fails its authentication tag is deleted: GCM's tag is what
     * distinguishes tampering and corruption from bad luck, and carrying on with
     * a partially trusted secret would be worse than asking the user to connect
     * again. A blob that could not be read because the platform was unwilling is
     * kept, because nothing about it is known to be wrong.
     */
    @Synchronized
    fun read(context: Context): CredentialState {
        val file = ciphertextFile(context)
        if (!file.isFile) return CredentialState.Absent

        val blob = try {
            file.readBytes()
        } catch (failure: java.io.IOException) {
            return CredentialState.Unavailable("the credential file could not be read")
        }
        if (blob.size < 3 || blob[0].toInt() != FORMAT_VERSION) {
            forget(context)
            return CredentialState.Absent
        }
        val nonceSize = blob[1].toInt()
        if (nonceSize <= 0 || blob.size <= 2 + nonceSize) {
            forget(context)
            return CredentialState.Absent
        }

        return try {
            val key = existingKey()
                ?: return CredentialState.Unavailable("the Keystore key is not available")
            val nonce = blob.copyOfRange(2, 2 + nonceSize)
            val ciphertext = blob.copyOfRange(2 + nonceSize, blob.size)
            val cipher = Cipher.getInstance(TRANSFORMATION).apply {
                init(Cipher.DECRYPT_MODE, key, GCMParameterSpec(TAG_BITS, nonce))
            }
            val token = String(cipher.doFinal(ciphertext), Charsets.UTF_8).trim()
            if (token.isEmpty()) {
                forget(context)
                CredentialState.Absent
            } else {
                CredentialState.Present(token)
            }
        } catch (failure: Exception) {
            if (classify(failure) == Recovery.DESTROY) {
                forget(context)
                return CredentialState.Absent
            }
            CredentialState.Unavailable("the Android Keystore could not decrypt the credential")
        }
    }

    /** The credential for the Core process, or null when there is none to give. */
    @Synchronized
    fun token(context: Context): String? =
        (read(context) as? CredentialState.Present)?.token

    /**
     * Whether a usable credential is present, and the account it belongs to.
     *
     * This decrypts rather than trusting the metadata file: metadata is
     * plaintext and could outlive the key, and reporting "Connected" for a
     * credential that can no longer be decrypted would send the user looking for
     * a fault in GitHub instead of reconnecting. A transient failure is reported
     * as such, so the user is not told to reconnect over something that will fix
     * itself.
     */
    @Synchronized
    fun status(context: Context): Status = when (val state = read(context)) {
        is CredentialState.Present -> Status(connected = true, login = storedLogin(context))
        is CredentialState.Absent -> Status(connected = false, login = null)
        is CredentialState.Unavailable ->
            Status(connected = false, login = storedLogin(context), unavailableReason = state.reason)
    }

    private fun storedLogin(context: Context): String? = try {
        metadataFile(context).takeIf { it.isFile }
            ?.readText(Charsets.UTF_8)
            ?.let { JSONObject(it).optString("login").ifBlank { null } }
    } catch (failure: Exception) {
        null
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

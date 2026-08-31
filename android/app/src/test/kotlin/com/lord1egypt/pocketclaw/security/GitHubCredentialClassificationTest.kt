package com.lord1egypt.pocketclaw.security

import java.security.GeneralSecurityException
import java.security.KeyStoreException
import java.security.ProviderException
import javax.crypto.AEADBadTagException
import javax.crypto.BadPaddingException
import javax.crypto.IllegalBlockSizeException
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * Destroying a credential is irreversible, so the rule that decides to do it is
 * worth testing on its own.
 *
 * These run on the JVM, without a Keystore. That is the point: the decision is a
 * function of the failure, not of the platform, which is what lets it be checked
 * at all.
 */
class GitHubCredentialClassificationTest {

    /**
     * Stands in for android.security.keystore.KeyPermanentlyInvalidatedException,
     * which cannot be constructed off a device. The production rule matches on
     * the class name for exactly this reason.
     */
    private class KeyPermanentlyInvalidatedException :
        GeneralSecurityException("the key is gone")

    @Test
    fun `a failed authentication tag destroys the credential`() {
        assertEquals(
            GitHubCredentialStore.Recovery.DESTROY,
            GitHubCredentialStore.classify(AEADBadTagException("tag mismatch")),
        )
    }

    @Test
    fun `corrupt ciphertext destroys the credential`() {
        for (failure in listOf(
            BadPaddingException("bad padding"),
            IllegalBlockSizeException("not a block multiple"),
        )) {
            assertEquals(
                "expected DESTROY for ${failure::class.simpleName}",
                GitHubCredentialStore.Recovery.DESTROY,
                GitHubCredentialStore.classify(failure),
            )
        }
    }

    @Test
    fun `a permanently invalidated key destroys the credential`() {
        assertEquals(
            GitHubCredentialStore.Recovery.DESTROY,
            GitHubCredentialStore.classify(KeyPermanentlyInvalidatedException()),
        )
    }

    @Test
    fun `a transient platform failure preserves the credential`() {
        for (failure in listOf<Throwable>(
            KeyStoreException("keystore is busy"),
            ProviderException("provider failed to initialise"),
            IllegalStateException("keystore service unavailable"),
            GeneralSecurityException("something new and unclassified"),
            java.io.IOException("could not read the blob"),
        )) {
            assertEquals(
                "expected PRESERVE for ${failure::class.simpleName}",
                GitHubCredentialStore.Recovery.PRESERVE,
                GitHubCredentialStore.classify(failure),
            )
        }
    }

    /**
     * Providers wrap. A tag failure buried under a provider exception is still a
     * tag failure, and an unknown failure wrapping nothing recognisable is still
     * unknown.
     */
    @Test
    fun `the whole cause chain is examined`() {
        assertEquals(
            GitHubCredentialStore.Recovery.DESTROY,
            GitHubCredentialStore.classify(
                ProviderException("decrypt failed", AEADBadTagException("tag mismatch")),
            ),
        )
        assertEquals(
            GitHubCredentialStore.Recovery.PRESERVE,
            GitHubCredentialStore.classify(
                ProviderException("decrypt failed", KeyStoreException("busy")),
            ),
        )
    }
}

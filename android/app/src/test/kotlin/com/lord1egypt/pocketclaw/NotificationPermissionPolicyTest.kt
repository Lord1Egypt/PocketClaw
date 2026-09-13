package com.lord1egypt.pocketclaw

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * PC-DEF-058. The rules around asking for notification permission: ask once, never
 * nag, stay correct where the permission does not exist, and leave a way back for
 * someone who refused.
 */
class NotificationPermissionPolicyTest {

    @Test
    fun `below api 33 the permission does not exist`() {
        for (sdk in intArrayOf(24, 29, 31, 32)) {
            assertEquals(
                "sdk $sdk",
                NotificationPermissionState.NOT_REQUIRED,
                NotificationPermissionPolicy.resolve(sdk, permissionGranted = false, alreadyAsked = false),
            )
        }
    }

    @Test
    fun `a fresh install on api 33 has never been asked`() {
        assertEquals(
            NotificationPermissionState.NOT_REQUESTED,
            NotificationPermissionPolicy.resolve(33, permissionGranted = false, alreadyAsked = false),
        )
    }

    @Test
    fun `granted is granted whether or not it was asked`() {
        assertEquals(
            NotificationPermissionState.GRANTED,
            NotificationPermissionPolicy.resolve(34, permissionGranted = true, alreadyAsked = false),
        )
        assertEquals(
            NotificationPermissionState.GRANTED,
            NotificationPermissionPolicy.resolve(34, permissionGranted = true, alreadyAsked = true),
        )
    }

    @Test
    fun `asked and still not granted is a denial`() {
        assertEquals(
            NotificationPermissionState.DENIED,
            NotificationPermissionPolicy.resolve(34, permissionGranted = false, alreadyAsked = true),
        )
    }

    // The dialog is shown exactly once. Android stops showing it after a refusal,
    // so a second request is a silent no-op that makes the app look broken.
    @Test
    fun `the system dialog is offered only when it has never been asked`() {
        assertEquals(
            NotificationPermissionAction.REQUEST_SYSTEM_DIALOG,
            NotificationPermissionPolicy.actionFor(NotificationPermissionState.NOT_REQUESTED),
        )
        assertEquals(
            NotificationPermissionAction.OFFER_SETTINGS,
            NotificationPermissionPolicy.actionFor(NotificationPermissionState.DENIED),
        )
        assertEquals(
            NotificationPermissionAction.NONE,
            NotificationPermissionPolicy.actionFor(NotificationPermissionState.GRANTED),
        )
        assertEquals(
            NotificationPermissionAction.NONE,
            NotificationPermissionPolicy.actionFor(NotificationPermissionState.NOT_REQUIRED),
        )
    }

    // "Permission granted" and "the notification will appear" are different facts:
    // on every version the user can switch notifications off in Settings, and below
    // API 33 that is the only control there is.
    @Test
    fun `visibility follows the notification switch not only the permission`() {
        assertTrue(
            NotificationPermissionPolicy.notificationsExpectedVisible(
                NotificationPermissionState.GRANTED,
                notificationsEnabled = true,
            ),
        )
        assertFalse(
            NotificationPermissionPolicy.notificationsExpectedVisible(
                NotificationPermissionState.GRANTED,
                notificationsEnabled = false,
            ),
        )
        assertTrue(
            NotificationPermissionPolicy.notificationsExpectedVisible(
                NotificationPermissionState.NOT_REQUIRED,
                notificationsEnabled = true,
            ),
        )
        assertFalse(
            NotificationPermissionPolicy.notificationsExpectedVisible(
                NotificationPermissionState.NOT_REQUIRED,
                notificationsEnabled = false,
            ),
        )
        assertFalse(
            NotificationPermissionPolicy.notificationsExpectedVisible(
                NotificationPermissionState.DENIED,
                notificationsEnabled = true,
            ),
        )
    }

    // The Dart side matches on these strings.
    @Test
    fun `wire names are stable`() {
        assertEquals("notRequired", NotificationPermissionPolicy.wireName(NotificationPermissionState.NOT_REQUIRED))
        assertEquals("granted", NotificationPermissionPolicy.wireName(NotificationPermissionState.GRANTED))
        assertEquals("notRequested", NotificationPermissionPolicy.wireName(NotificationPermissionState.NOT_REQUESTED))
        assertEquals("denied", NotificationPermissionPolicy.wireName(NotificationPermissionState.DENIED))
    }
}

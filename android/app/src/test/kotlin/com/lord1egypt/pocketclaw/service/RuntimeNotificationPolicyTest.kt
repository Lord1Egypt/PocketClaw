package com.lord1egypt.pocketclaw.service

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * PC-DEF-070. "PocketClaw Running" showed over a stopped service and a stopped
 * Gateway. Both only came up afterwards, when the app was opened and launch
 * auto-start started them — so the notification had been describing a runtime
 * that no longer existed.
 *
 * These pin the two rules that make that impossible: Running is derived from a
 * live process and can never be reached from intent, and a notification found on
 * screen in a process that cannot be hosting the service is stale.
 */
class RuntimeNotificationPolicyTest {

    @Test
    fun `a live core process is the only thing that can say running`() {
        assertEquals(
            RuntimeNotificationState.RUNNING,
            RuntimeNotificationPolicy.resolve(processAlive = true, starting = false),
        )
    }

    @Test
    fun `starting is intent and never produces running`() {
        assertEquals(
            RuntimeNotificationState.STARTING,
            RuntimeNotificationPolicy.resolve(processAlive = false, starting = true),
        )
    }

    @Test
    fun `no process and not starting is stopped`() {
        assertEquals(
            RuntimeNotificationState.STOPPED,
            RuntimeNotificationPolicy.resolve(processAlive = false, starting = false),
        )
    }

    // The defect itself, stated as a property: nothing about intent, a
    // preference, or a previous process can reach RUNNING without a live one.
    @Test
    fun `running is unreachable without a live process`() {
        for (starting in booleanArrayOf(true, false)) {
            val state = RuntimeNotificationPolicy.resolve(
                processAlive = false,
                starting = starting,
            )
            assertFalse(
                "starting=$starting must not claim running",
                state == RuntimeNotificationState.RUNNING,
            )
        }
    }

    @Test
    fun `a notification found in a process that hosts no service is stale`() {
        assertTrue(
            RuntimeNotificationPolicy.isStaleOnProcessStart(
                serviceHostedInThisProcess = false,
            ),
        )
    }

    @Test
    fun `a notification is not stale while this process hosts the service`() {
        assertFalse(
            RuntimeNotificationPolicy.isStaleOnProcessStart(
                serviceHostedInThisProcess = true,
            ),
        )
    }

    // PC-DEF-070, reopened. The other direction of the contract: a runtime that
    // is genuinely running must have a notification. On a fresh install the
    // service auto-starts before the POST_NOTIFICATIONS dialog is answered, so
    // Android suppresses the foreground post; granting afterwards shows nothing
    // retroactively and nothing retries. Physically: isForeground=true, Core and
    // Gateway alive, port 18800 listening, granted=true, and zero notification
    // records.

    @Test
    fun `a granted permission re-renders the notification for a hosted service`() {
        assertTrue(
            RuntimeNotificationPolicy.shouldRenderOnPermissionChange(
                serviceHostedInThisProcess = true,
                notificationsEnabled = true,
            ),
        )
    }

    @Test
    fun `no service in this process means there is nothing to render`() {
        assertFalse(
            "re-rendering for a runtime this process does not host is how a " +
                "false Running notification would come back",
            RuntimeNotificationPolicy.shouldRenderOnPermissionChange(
                serviceHostedInThisProcess = false,
                notificationsEnabled = true,
            ),
        )
    }

    @Test
    fun `notifications still disabled means posting would be a no-op`() {
        assertFalse(
            RuntimeNotificationPolicy.shouldRenderOnPermissionChange(
                serviceHostedInThisProcess = true,
                notificationsEnabled = false,
            ),
        )
        assertFalse(
            RuntimeNotificationPolicy.shouldRenderOnPermissionChange(
                serviceHostedInThisProcess = false,
                notificationsEnabled = false,
            ),
        )
    }

    // The re-render goes through resolve(), so it inherits the first direction
    // of the contract: it cannot manufacture a Running claim for a dead runtime.
    @Test
    fun `a re-render of a stopped runtime is still not running`() {
        assertEquals(
            RuntimeNotificationState.STOPPED,
            RuntimeNotificationPolicy.resolve(processAlive = false, starting = false),
        )
    }
}

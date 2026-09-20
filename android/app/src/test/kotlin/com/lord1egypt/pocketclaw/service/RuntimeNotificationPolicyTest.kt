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
}

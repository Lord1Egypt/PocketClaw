package com.lord1egypt.pocketclaw.service

import org.junit.Assert.assertEquals
import org.junit.Test

// PC-DEF-085. Android may re-create a service on its own. That delivery has no
// intent, and it must never start Core: PocketClaw runs because the owner
// started it, never because Android restored it after a reboot or a kill.
class ServiceCommandTest {
    @Test
    fun `an OS re-creation with no intent is ignored`() {
        assertEquals(ServiceCommand.IGNORE_OS_RESTART, ServiceCommand.of(false, null))
        // Even if some action were somehow attached, no intent means no start.
        assertEquals(ServiceCommand.IGNORE_OS_RESTART, ServiceCommand.of(false, PocketClawService.ACTION_START))
    }

    @Test
    fun `explicit requests map to their commands`() {
        assertEquals(ServiceCommand.STOP, ServiceCommand.of(true, PocketClawService.ACTION_STOP))
        assertEquals(ServiceCommand.RESTART, ServiceCommand.of(true, PocketClawService.ACTION_RESTART))
        assertEquals(ServiceCommand.START, ServiceCommand.of(true, PocketClawService.ACTION_START))
    }

    @Test
    fun `an explicit intent without an action is a start`() {
        assertEquals(ServiceCommand.START, ServiceCommand.of(true, null))
    }

    @Test
    fun `repeated deliveries decide the same way every time`() {
        repeat(3) {
            assertEquals(ServiceCommand.START, ServiceCommand.of(true, PocketClawService.ACTION_START))
            assertEquals(ServiceCommand.IGNORE_OS_RESTART, ServiceCommand.of(false, null))
        }
    }
}

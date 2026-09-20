package com.lord1egypt.pocketclaw.service

import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicInteger
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * PC-DEF-072. `stopService()` joined the worker for five seconds and then
 * cleared `serviceThread` whether or not it had exited, so `startService()`'s
 * duplicate guard saw no owner and started a second worker over a live one —
 * and clearing the single `stopped` flag for the new start also un-stopped the
 * abandoned worker, which carried on and spawned a second Core on port 18800.
 *
 * The invariant under test is one sentence: **at most one worker may hold the
 * current epoch, and only the holder of the current epoch may start or register
 * a Core process.**
 *
 * The scenarios are driven with real threads and latches rather than a clock, so
 * "the join timed out" is produced by a worker that genuinely has not returned,
 * and nothing here depends on timing luck to fail.
 */
class CoreRuntimeOwnershipTest {

    /**
     * A worker that starts, holds until released, and reports whether it was
     * still the live runtime when it tried to take the Core port.
     */
    private class FakeWorker(
        private val ownership: CoreRuntimeOwnership,
        private val port: CorePortModel,
    ) {
        val started = CountDownLatch(1)
        val release = CountDownLatch(1)
        val finished = CountDownLatch(1)
        @Volatile var boundPort = false
        @Volatile var queuedStartRan = false
        lateinit var thread: Thread

        fun launch(epoch: Long): Thread {
            thread = Thread {
                try {
                    started.countDown()
                    release.await(10, TimeUnit.SECONDS)
                    // The single place a Core becomes this service's runtime.
                    boundPort = port.tryBind(epoch) { ownership.isCurrent(epoch) }
                } finally {
                    if (ownership.release(epoch)) queuedStartRan = true
                    finished.countDown()
                }
            }
            thread.start()
            return thread
        }
    }

    /**
     * Stands in for port 18800. Binding is only permitted to a caller that is
     * still the live runtime, and the model refuses — loudly — to hold two
     * bindings at once, which is the failure the defect produced.
     */
    private class CorePortModel {
        private val lock = Object()
        private var boundEpoch: Long? = null
        val bindAttempts = AtomicInteger(0)
        val concurrentBindings = AtomicInteger(0)

        fun tryBind(epoch: Long, stillCurrent: () -> Boolean): Boolean = synchronized(lock) {
            bindAttempts.incrementAndGet()
            if (!stillCurrent()) return false
            if (boundEpoch != null) {
                concurrentBindings.incrementAndGet()
                return false
            }
            boundEpoch = epoch
            true
        }

        fun unbind(epoch: Long) = synchronized(lock) {
            if (boundEpoch == epoch) boundEpoch = null
        }

        fun boundTo(): Long? = synchronized(lock) { boundEpoch }
    }

    // --- the ordinary paths, so the fix is not proved only by its edge case ---

    @Test
    fun `a successful stop releases ownership and the next start runs`() {
        val ownership = CoreRuntimeOwnership()
        val port = CorePortModel()

        val first = FakeWorker(ownership, port)
        val started = ownership.start(first::launch)
        assertTrue(started is CoreRuntimeOwnership.StartOutcome.Started)
        first.started.await(5, TimeUnit.SECONDS)

        val target = ownership.requestStop()
        assertNotNull(target)
        first.release.countDown()
        first.finished.await(5, TimeUnit.SECONDS)
        target!!.thread.join(5_000)

        assertFalse("the worker exited, so nobody owns the runtime", ownership.hasOwner())

        val second = FakeWorker(ownership, port)
        val outcome = ownership.start(second::launch)
        assertTrue(
            "a start after a clean stop must actually start",
            outcome is CoreRuntimeOwnership.StartOutcome.Started,
        )
        assertEquals(
            "each start is a new epoch",
            2L,
            (outcome as CoreRuntimeOwnership.StartOutcome.Started).epoch,
        )
        second.release.countDown()
        second.finished.await(5, TimeUnit.SECONDS)
    }

    @Test
    fun `a duplicate start over a healthy runtime is a no-op`() {
        val ownership = CoreRuntimeOwnership()
        val port = CorePortModel()
        val worker = FakeWorker(ownership, port)
        ownership.start(worker::launch)
        worker.started.await(5, TimeUnit.SECONDS)

        var secondLaunched = false
        val outcome = ownership.start { epoch ->
            secondLaunched = true
            Thread { }.also { it.start() }
        }

        assertEquals(CoreRuntimeOwnership.StartOutcome.AlreadyRunning, outcome)
        assertFalse("no second worker may be created", secondLaunched)

        worker.release.countDown()
        worker.finished.await(5, TimeUnit.SECONDS)
    }

    // --- the defect ---

    @Test
    fun `a join timeout does not release ownership while the worker is alive`() {
        val ownership = CoreRuntimeOwnership()
        val port = CorePortModel()
        val worker = FakeWorker(ownership, port)
        ownership.start(worker::launch)
        worker.started.await(5, TimeUnit.SECONDS)

        val target = ownership.requestStop()
        assertNotNull(target)
        // The worker is deliberately still holding, so this join expires.
        target!!.thread.join(150)
        assertTrue("the worker really is still alive", target.thread.isAlive)

        assertTrue(
            "a join that expired is not evidence the worker is gone",
            ownership.hasOwner(),
        )
        assertTrue(ownership.isStopping())
        assertFalse(
            "a stopping epoch is not the live runtime",
            ownership.isCurrent(target.epoch),
        )

        worker.release.countDown()
        worker.finished.await(5, TimeUnit.SECONDS)
    }

    @Test
    fun `a restart during a join timeout is queued, never started over the live worker`() {
        val ownership = CoreRuntimeOwnership()
        val port = CorePortModel()
        val first = FakeWorker(ownership, port)
        ownership.start(first::launch)
        first.started.await(5, TimeUnit.SECONDS)

        val target = ownership.requestStop()!!
        target.thread.join(150)
        assertTrue(target.thread.isAlive)

        var launched = false
        val outcome = ownership.start { epoch ->
            launched = true
            Thread { }.also { it.start() }
        }

        assertEquals(CoreRuntimeOwnership.StartOutcome.Queued, outcome)
        assertFalse(
            "ACTION_RESTART must not start a second Core over a still-live one",
            launched,
        )
        assertTrue("and it must not be dropped either", ownership.hasQueuedStart())

        first.release.countDown()
        first.finished.await(5, TimeUnit.SECONDS)
    }

    @Test
    fun `the old worker exits later and hands the queued start over`() {
        val ownership = CoreRuntimeOwnership()
        val port = CorePortModel()
        val first = FakeWorker(ownership, port)
        ownership.start(first::launch)
        first.started.await(5, TimeUnit.SECONDS)

        val target = ownership.requestStop()!!
        target.thread.join(150)
        assertEquals(CoreRuntimeOwnership.StartOutcome.Queued, ownership.start { Thread { } })

        // The worker finally finishes. Its own finally is the only release.
        first.release.countDown()
        assertTrue(first.finished.await(5, TimeUnit.SECONDS))
        target.thread.join(5_000)

        assertTrue("the release must hand the queued start back", first.queuedStartRan)
        assertFalse("and ownership must now be free", ownership.hasOwner())
        assertFalse(ownership.hasQueuedStart())

        val second = FakeWorker(ownership, port)
        val outcome = ownership.start(second::launch)
        assertTrue(
            "the queued restart is now able to run -- it was never lost",
            outcome is CoreRuntimeOwnership.StartOutcome.Started,
        )
        second.release.countDown()
        second.finished.await(5, TimeUnit.SECONDS)
    }

    @Test
    fun `the abandoned worker cannot bind the Core port and the replacement can`() {
        val ownership = CoreRuntimeOwnership()
        val port = CorePortModel()

        val abandoned = FakeWorker(ownership, port)
        ownership.start(abandoned::launch)
        abandoned.started.await(5, TimeUnit.SECONDS)

        val target = ownership.requestStop()!!
        target.thread.join(150)
        assertTrue(target.thread.isAlive)

        // A restart arrives while the old worker is wedged, then the old worker
        // wakes up and reaches its bind -- the exact interleaving that used to
        // produce two Cores.
        assertEquals(CoreRuntimeOwnership.StartOutcome.Queued, ownership.start { Thread { } })
        abandoned.release.countDown()
        abandoned.finished.await(5, TimeUnit.SECONDS)

        assertFalse(
            "a worker that lost the current epoch must not take the Core port",
            abandoned.boundPort,
        )
        assertNull("so nothing is bound", port.boundTo())

        val replacement = FakeWorker(ownership, port)
        ownership.start(replacement::launch)
        replacement.started.await(5, TimeUnit.SECONDS)
        replacement.release.countDown()
        replacement.finished.await(5, TimeUnit.SECONDS)

        assertTrue("the replacement is the live runtime and binds", replacement.boundPort)
        assertEquals(
            "port 18800 was never held by two epochs at once",
            0,
            port.concurrentBindings.get(),
        )
    }

    @Test
    fun `exactly one Core owner survives concurrent starts and stops`() {
        val ownership = CoreRuntimeOwnership()
        val port = CorePortModel()
        val launched = AtomicInteger(0)
        val liveWorkers = AtomicInteger(0)
        val peak = AtomicInteger(0)
        val threads = mutableListOf<Thread>()

        repeat(40) {
            val t = Thread {
                val outcome = ownership.start { epoch ->
                    launched.incrementAndGet()
                    Thread {
                        val live = liveWorkers.incrementAndGet()
                        peak.getAndUpdate { p -> maxOf(p, live) }
                        port.tryBind(epoch) { ownership.isCurrent(epoch) }
                        Thread.sleep(2)
                        port.unbind(epoch)
                        liveWorkers.decrementAndGet()
                        ownership.release(epoch)
                    }.also { w -> w.start() }
                }
                if (outcome !is CoreRuntimeOwnership.StartOutcome.Started) {
                    ownership.requestStop()
                }
            }
            threads += t
            t.start()
        }
        threads.forEach { it.join(10_000) }
        // Let any worker still running finish.
        val deadline = System.currentTimeMillis() + 5_000
        while (ownership.hasOwner() && System.currentTimeMillis() < deadline) Thread.sleep(5)

        assertTrue("at least one runtime ran", launched.get() >= 1)
        assertEquals("two workers were never live at once", 1, peak.get())
        assertEquals(
            "the Core port was never bound twice",
            0,
            port.concurrentBindings.get(),
        )
    }

    // --- the smaller rules the above depend on ---

    @Test
    fun `a stop cancels a start queued before it`() {
        val ownership = CoreRuntimeOwnership()
        val port = CorePortModel()
        val worker = FakeWorker(ownership, port)
        ownership.start(worker::launch)
        worker.started.await(5, TimeUnit.SECONDS)

        ownership.requestStop()
        assertEquals(CoreRuntimeOwnership.StartOutcome.Queued, ownership.start { Thread { } })
        assertTrue(ownership.hasQueuedStart())

        // The owner asked again for the runtime to be down. A start queued
        // before that must not resurrect it.
        ownership.requestStop()
        assertFalse(ownership.hasQueuedStart())

        worker.release.countDown()
        worker.finished.await(5, TimeUnit.SECONDS)
        assertFalse(worker.queuedStartRan)
    }

    @Test
    fun `requestStop with no owner is nothing to stop`() {
        val ownership = CoreRuntimeOwnership()
        assertNull(ownership.requestStop())
        assertFalse(ownership.hasOwner())
    }

    @Test
    fun `a stale epoch releases nothing`() {
        val ownership = CoreRuntimeOwnership()
        val port = CorePortModel()
        val worker = FakeWorker(ownership, port)
        val started = ownership.start(worker::launch)
            as CoreRuntimeOwnership.StartOutcome.Started
        worker.started.await(5, TimeUnit.SECONDS)

        assertFalse(
            "an epoch that never owned anything cannot release the live one",
            ownership.release(started.epoch - 1),
        )
        assertTrue(ownership.hasOwner())
        assertTrue(ownership.isCurrent(started.epoch))

        worker.release.countDown()
        worker.finished.await(5, TimeUnit.SECONDS)
    }
}

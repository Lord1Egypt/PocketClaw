package com.lord1egypt.pocketclaw.service

/**
 * Who is allowed to run the service-owned Core runtime, and when.
 *
 * PC-DEF-072. `stopService()` used to join the worker thread with a five-second
 * bound and then clear `serviceThread` whether or not the thread had exited.
 * `startService()`'s duplicate guard reads that field, so after a timed-out stop
 * it saw no owner and started a second worker over a live one — and because
 * `startService()` also cleared the `stopped` flag, the abandoned worker was
 * un-stopped and carried on into `runWebService()`. Two Core processes, both
 * reaching for port 18800.
 *
 * The rule this class exists to keep is one sentence:
 *
 *   **at most one worker may hold the current epoch, and only the holder of the
 *   current epoch may start or register a Core process.**
 *
 * A bounded join that expires is not evidence that a thread is gone, so
 * ownership is never released by the stopper. It is released by the worker
 * itself, from its own `finally`, which is the only place that knows it has
 * finished. A start that arrives while a stopping worker still holds ownership
 * is therefore not dropped and not forced through: it is queued, and the
 * worker runs it as it releases. That is what keeps a restart from being
 * silently ignored forever without letting it start a second Core.
 *
 * Deliberately free of Android types. The state machine is the part that was
 * wrong, and it is the part that has to be testable without a device.
 */
class CoreRuntimeOwnership {

    /** What a caller should do about a start request. */
    sealed interface StartOutcome {
        /** The worker was created and now owns [epoch]. */
        data class Started(val epoch: Long) : StartOutcome

        /**
         * A healthy owner is already running and was not asked to stop. This is
         * an ordinary duplicate ACTION_START and is correctly a no-op.
         */
        data object AlreadyRunning : StartOutcome

        /**
         * A stopping owner has not released yet, so nothing was started. The
         * start is recorded and will run the moment that owner releases — it is
         * not lost, and it is not forced over a live runtime.
         */
        data object Queued : StartOutcome
    }

    /** The owner a stop is aimed at, so the caller can act on that exact one. */
    data class StopTarget(val thread: Thread, val epoch: Long)

    private val lock = Object()

    /** The epoch of the current owner. Monotonic; 0 before the first start. */
    private var currentEpoch: Long = 0

    /** The worker holding [currentEpoch], or null when nobody holds it. */
    private var owner: Thread? = null

    /** Whether the current owner has been asked to stop. */
    private var stopping: Boolean = false

    /** A start that arrived while an owner had not yet released. */
    private var queuedStart: Boolean = false

    /**
     * Asks for a runtime.
     *
     * [launch] is invoked under the lock, exactly once, and only when there is
     * no owner. It receives the new epoch and must return the started worker.
     * Holding the lock across it is what makes "decide, create, record" atomic:
     * the previous code decided and recorded in two steps, which is the window
     * a second start walked through.
     */
    fun start(launch: (epoch: Long) -> Thread): StartOutcome = synchronized(lock) {
        if (owner != null) {
            // Never start over a live owner, whether or not it is on its way
            // out. A stopping owner still holds the port and the pid file.
            return if (stopping) {
                queuedStart = true
                StartOutcome.Queued
            } else {
                StartOutcome.AlreadyRunning
            }
        }
        currentEpoch += 1
        stopping = false
        queuedStart = false
        val epoch = currentEpoch
        owner = launch(epoch)
        StartOutcome.Started(epoch)
    }

    /**
     * Marks the current owner as stopping and names it, or returns null when
     * there is nothing to stop.
     *
     * A stop also cancels a queued start: the caller asked for the runtime to be
     * down, and a start queued before that request must not resurrect it.
     */
    fun requestStop(): StopTarget? = synchronized(lock) {
        queuedStart = false
        val thread = owner ?: return null
        stopping = true
        StopTarget(thread, currentEpoch)
    }

    /**
     * Released by the worker itself, from its `finally`.
     *
     * Returns true when a start was queued behind this owner and the caller
     * should now perform it. A worker that is no longer the registered owner —
     * which cannot normally happen, but is exactly the state the defect
     * produced — releases nothing and triggers nothing.
     */
    fun release(epoch: Long): Boolean = synchronized(lock) {
        if (epoch != currentEpoch || owner == null) return false
        owner = null
        stopping = false
        val queued = queuedStart
        queuedStart = false
        queued
    }

    /**
     * Whether [epoch] is still the live runtime.
     *
     * This replaces the old `stopped` flag, which was one boolean shared by
     * every worker that had ever run: clearing it for a new start also cleared
     * it for an abandoned one. An epoch answers per worker, so an abandoned
     * worker can never be un-stopped by somebody else's start.
     */
    fun isCurrent(epoch: Long): Boolean = synchronized(lock) {
        epoch == currentEpoch && owner != null && !stopping
    }

    /** Whether any worker still holds ownership. */
    fun hasOwner(): Boolean = synchronized(lock) { owner != null }

    /** Whether the current owner has been asked to stop and has not released. */
    fun isStopping(): Boolean = synchronized(lock) { owner != null && stopping }

    /** Whether a start is waiting for the current owner to release. */
    fun hasQueuedStart(): Boolean = synchronized(lock) { queuedStart }

    /** The current epoch, for logging. Never an identity of any kind. */
    fun epoch(): Long = synchronized(lock) { currentEpoch }
}

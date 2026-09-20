package com.lord1egypt.pocketclaw.service

/**
 * What the persistent PocketClaw notification is allowed to claim.
 *
 * PC-DEF-070. The notification was a log of the last thing the service thread
 * did, not a rendering of what was true. "Running (PID: n)" was written once,
 * at the moment the Core process was spawned, and nothing ever revised it
 * against the process actually being alive. A notification that outlived its
 * process therefore went on saying Running while nothing ran: the owner opened
 * the app to find Services and Gateway both off, and it was app-launch
 * auto-start -- not the notification -- that started them.
 *
 * The subject of this notification is **the PocketClaw foreground service and
 * the Core runtime process that service owns**, which is the same subject as
 * its Stop action. It is deliberately not the Gateway: Gateway auto-start is a
 * preference the owner may legitimately turn off, so a service running without
 * a Gateway is a correct state and must not be reported as a failure. The
 * Gateway's own state is reported by the Status screen, which can name both
 * facts separately.
 *
 * The decision lives here, apart from the platform calls, because its rules are
 * worth testing: Running is derived from a live process and never asserted,
 * intent is never evidence, and a notification found on screen in a process
 * that cannot be hosting the service is stale by construction.
 */
enum class RuntimeNotificationState {
    /** The service is coming up. Nothing is claimed about the runtime yet. */
    STARTING,

    /** This service holds a live Core process, right now. */
    RUNNING,

    /** The runtime is not up. The reason rides alongside. */
    STOPPED,
}

object RuntimeNotificationPolicy {
    /**
     * The state to render, derived from the only fact that can support a
     * Running claim: a Core process this service still holds, and that is still
     * alive at the moment the notification is built.
     *
     * [starting] is not allowed to produce RUNNING. It is the service's
     * *intent*, and intent is exactly what the defect mistook for truth.
     */
    fun resolve(processAlive: Boolean, starting: Boolean): RuntimeNotificationState = when {
        processAlive -> RuntimeNotificationState.RUNNING
        starting -> RuntimeNotificationState.STARTING
        else -> RuntimeNotificationState.STOPPED
    }

    /**
     * Whether a PocketClaw service notification found on screen belongs to a
     * process that no longer exists.
     *
     * Asked once, from `Application.onCreate`, which the platform runs before
     * any component of a newly created process. [serviceHostedInThisProcess] is
     * therefore false by construction there, and a notification that survived
     * the previous process is claiming a runtime that died with it.
     */
    fun isStaleOnProcessStart(serviceHostedInThisProcess: Boolean): Boolean =
        !serviceHostedInThisProcess
}

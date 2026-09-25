package com.lord1egypt.pocketclaw.service

/**
 * What one onStartCommand delivery asks the service to do.
 *
 * Kept pure so the restart rules are checked on the JVM: a delivery with no
 * intent is Android re-creating the service by itself, and that must never
 * start Core. PocketClaw starts only because the owner started it.
 */
enum class ServiceCommand {
    IGNORE_OS_RESTART,
    STOP,
    RESTART,
    START;

    companion object {
        fun of(hasIntent: Boolean, action: String?): ServiceCommand = when {
            !hasIntent -> IGNORE_OS_RESTART
            action == PocketClawService.ACTION_STOP -> STOP
            action == PocketClawService.ACTION_RESTART -> RESTART
            else -> START
        }
    }
}

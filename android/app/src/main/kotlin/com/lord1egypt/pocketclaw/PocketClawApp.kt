package com.lord1egypt.pocketclaw

import com.lord1egypt.pocketclaw.service.PocketClawService
import io.flutter.app.FlutterApplication

class PocketClawApp : FlutterApplication() {

    companion object {
        /**
         * The channel the foreground service posts on.
         *
         * Owned by [PocketClawNotificationChannels]; kept here as the name the
         * service already reads so the notification builder has one source.
         */
        const val CHANNEL_ID = PocketClawNotificationChannels.SERVICE_CHANNEL_ID
        const val CHANNEL_NAME = PocketClawNotificationChannels.SERVICE_CHANNEL_NAME
    }

    override fun onCreate() {
        super.onCreate()
        // Before anything can post a foreground notification.
        PocketClawNotificationChannels.ensure(this)
        // PC-DEF-070. Android runs this before any component of a newly created
        // process, so nothing in this process can be hosting the foreground
        // service yet. A PocketClaw service notification still on screen
        // therefore belongs to a process that no longer exists and is claiming a
        // runtime that died with it -- which is exactly what the owner saw:
        // "PocketClaw Running" over a stopped service and a stopped Gateway,
        // both of which only came up afterwards because launch auto-start
        // started them. Removed here rather than waited out, and never rewritten
        // into a different claim, because this process knows nothing yet.
        PocketClawService.cancelStaleRuntimeNotification(this)
        AnalyticsReporter.preInit(this)
    }
}

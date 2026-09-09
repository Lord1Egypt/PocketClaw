package com.lord1egypt.pocketclaw

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
        AnalyticsReporter.preInit(this)
    }
}

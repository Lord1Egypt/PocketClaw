package com.lord1egypt.pocketclaw

import android.app.NotificationChannel
import android.app.NotificationManager
import io.flutter.app.FlutterApplication

class PicoClawApp : FlutterApplication() {

    companion object {
        const val CHANNEL_ID = "picoclaw_service"
        const val CHANNEL_NAME = "PocketClaw service"
    }

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
        AnalyticsReporter.preInit(this)
    }

    private fun createNotificationChannel() {
        val channel = NotificationChannel(
            CHANNEL_ID,
            CHANNEL_NAME,
            NotificationManager.IMPORTANCE_LOW
        ).apply {
            description = "PocketClaw background service powered by PicoClaw Core"
            setShowBadge(false)
        }

        val manager = getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(channel)
    }
}

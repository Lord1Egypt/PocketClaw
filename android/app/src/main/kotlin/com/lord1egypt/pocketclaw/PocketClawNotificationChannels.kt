package com.lord1egypt.pocketclaw

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.os.Build
import android.util.Log

/**
 * The one place that names PocketClaw's notification channels, and the only
 * place the pre-Zero-Pico ids are allowed to appear.
 *
 * Android channel ids are permanent: they cannot be renamed, only replaced. So
 * removing the Pico identity means creating a new channel, carrying across
 * whatever the platform lets us read off the old one, and deleting the old one
 * — in that order, because the old channel is the only record of the user's
 * settings until the new one exists.
 *
 * Called from [PocketClawApp.onCreate] before any foreground notification is
 * posted, so the service never posts against a channel that is about to be
 * replaced.
 */
object PocketClawNotificationChannels {
    private const val TAG = "PocketClawChannels"

    /** The channel the foreground service posts on. */
    const val SERVICE_CHANNEL_ID = "pocketclaw_service"
    const val SERVICE_CHANNEL_NAME = "PocketClaw service"
    private const val SERVICE_CHANNEL_DESCRIPTION = "PocketClaw background service"

    /**
     * LEGACY READ-ONLY MIGRATION — the pre-Zero-Pico service channel.
     *
     * Read to copy its settings, then deleted. Never created.
     */
    private const val LEGACY_SERVICE_CHANNEL_ID = "picoclaw_service"

    /**
     * LEGACY READ-ONLY MIGRATION — a dead channel, deleted without replacement.
     *
     * `initializeBackgroundService()` created it and handed it to
     * flutter_background_service, but that service is configured with
     * `autoStart: false` and `startService()` is never called anywhere in the
     * app, so nothing has ever posted a notification on it. Creating a
     * `pocketclaw_foreground` twin would add a second empty entry to the user's
     * notification settings for the sake of symmetry. The Flutter
     * configuration now points at [SERVICE_CHANNEL_ID] instead, so it stays
     * valid if that service is ever actually started.
     */
    private const val LEGACY_FOREGROUND_CHANNEL_ID = "picoclaw_foreground"

    /** What to do about the service channel, given what already exists. */
    enum class ServiceChannelAction {
        /** Neither exists: first run, or a user who cleared app data. */
        CREATE_FRESH,

        /** Only the legacy channel: copy its settings across, then drop it. */
        MIGRATE_FROM_LEGACY,

        /** Both exist: canonical wins untouched, legacy is dropped. */
        KEEP_CANONICAL_DELETE_LEGACY,

        /** Only the canonical one: the steady state after migration. */
        KEEP_CANONICAL,
    }

    /**
     * The migration decision, kept pure so it can be reasoned about and tested
     * without an Android runtime.
     */
    fun decideServiceChannelAction(
        legacyExists: Boolean,
        canonicalExists: Boolean,
    ): ServiceChannelAction = when {
        legacyExists && canonicalExists -> ServiceChannelAction.KEEP_CANONICAL_DELETE_LEGACY
        legacyExists -> ServiceChannelAction.MIGRATE_FROM_LEGACY
        canonicalExists -> ServiceChannelAction.KEEP_CANONICAL
        else -> ServiceChannelAction.CREATE_FRESH
    }

    /**
     * Creates the canonical channel if needed and retires the legacy ones.
     *
     * A no-op below API 26, where notification channels do not exist. The
     * previous implementation had no such guard even though minSdk is 24, so
     * constructing a NotificationChannel on API 24/25 would have thrown at
     * application start.
     */
    fun ensure(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return

        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        val legacy = manager.getNotificationChannel(LEGACY_SERVICE_CHANNEL_ID)
        val canonical = manager.getNotificationChannel(SERVICE_CHANNEL_ID)

        when (decideServiceChannelAction(legacy != null, canonical != null)) {
            ServiceChannelAction.CREATE_FRESH ->
                manager.createNotificationChannel(defaultServiceChannel())

            ServiceChannelAction.MIGRATE_FROM_LEGACY -> {
                manager.createNotificationChannel(serviceChannelFrom(legacy!!))
                // Verify before dropping the only copy of the user's settings.
                if (manager.getNotificationChannel(SERVICE_CHANNEL_ID) == null) {
                    Log.w(TAG, "Canonical channel was not created; keeping the legacy channel")
                    return
                }
                manager.deleteNotificationChannel(LEGACY_SERVICE_CHANNEL_ID)
                Log.i(TAG, "Migrated the service notification channel to $SERVICE_CHANNEL_ID")
            }

            ServiceChannelAction.KEEP_CANONICAL_DELETE_LEGACY -> {
                // Canonical settings are the user's current ones. Do not
                // overwrite them from a stale legacy channel.
                manager.deleteNotificationChannel(LEGACY_SERVICE_CHANNEL_ID)
                Log.i(TAG, "Removed the legacy service notification channel")
            }

            ServiceChannelAction.KEEP_CANONICAL -> Unit
        }

        // The dead channel: delete if present, never create.
        if (manager.getNotificationChannel(LEGACY_FOREGROUND_CHANNEL_ID) != null) {
            manager.deleteNotificationChannel(LEGACY_FOREGROUND_CHANNEL_ID)
            Log.i(TAG, "Removed the unused legacy foreground notification channel")
        }
    }

    private fun defaultServiceChannel(): NotificationChannel =
        NotificationChannel(
            SERVICE_CHANNEL_ID,
            SERVICE_CHANNEL_NAME,
            NotificationManager.IMPORTANCE_LOW,
        ).apply {
            description = SERVICE_CHANNEL_DESCRIPTION
            setShowBadge(false)
        }

    /**
     * The canonical channel, carrying across what Android lets us read from the
     * channel being replaced.
     *
     * Every property below is a public API-26 getter and a public setter that
     * applies when a channel id is created for the first time. What cannot come
     * across is documented in PROJECT_STATE.md rather than papered over: a
     * channel the user had blocked or muted through system UI, any Do Not
     * Disturb override we lack policy access to set, and the deletion history
     * Android keeps against the old id. Those are accepted losses of changing
     * an id that Android does not allow to be renamed.
     */
    private fun serviceChannelFrom(previous: NotificationChannel): NotificationChannel =
        NotificationChannel(
            SERVICE_CHANNEL_ID,
            SERVICE_CHANNEL_NAME,
            previous.importance,
        ).apply {
            description = previous.description ?: SERVICE_CHANNEL_DESCRIPTION
            group = previous.group
            setSound(previous.sound, previous.audioAttributes)
            enableVibration(previous.shouldVibrate())
            vibrationPattern = previous.vibrationPattern
            enableLights(previous.shouldShowLights())
            lightColor = previous.lightColor
            setShowBadge(previous.canShowBadge())
            lockscreenVisibility = previous.lockscreenVisibility
            // Best effort: silently ignored unless the app holds DND policy
            // access, which PocketClaw does not request.
            setBypassDnd(previous.canBypassDnd())
        }
}

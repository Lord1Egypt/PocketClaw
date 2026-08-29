package com.lord1egypt.pocketclaw.service

import android.content.Context

data class LaunchAutoStartSnapshot(
    val serviceEnabled: Boolean,
    val gatewayEnabled: Boolean,
    val initialized: Boolean,
    val source: String,
) {
    fun asMap(): Map<String, Any> = mapOf(
        "serviceEnabled" to serviceEnabled,
        "gatewayEnabled" to gatewayEnabled,
        "initialized" to initialized,
        "source" to source,
    )
}

/**
 * Canonical app-private preference record shared by the Flutter bridge and the
 * foreground Service. Synchronous commits make an acknowledged toggle durable
 * before a subsequent manual Service start can consume it.
 */
object LaunchAutoStartPreferences {
    private const val PREF_NAME = "picoclaw_prefs"
    private const val KEY_INITIALIZED = "launch_auto_start_initialized_v1"
    private const val KEY_SERVICE = "service_launch_auto_start"
    private const val KEY_GATEWAY = "gateway_launch_auto_start"

    @Synchronized
    fun read(context: Context): LaunchAutoStartSnapshot {
        val prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
        return LaunchAutoStartSnapshot(
            serviceEnabled = prefs.getBoolean(KEY_SERVICE, true),
            gatewayEnabled = prefs.getBoolean(KEY_GATEWAY, true),
            initialized = prefs.getBoolean(KEY_INITIALIZED, false),
            source = "android_native_canonical",
        )
    }

    @Synchronized
    fun update(
        context: Context,
        serviceEnabled: Boolean? = null,
        gatewayEnabled: Boolean? = null,
    ): LaunchAutoStartSnapshot {
        val current = read(context)
        val nextService = serviceEnabled ?: current.serviceEnabled
        val nextGateway = gatewayEnabled ?: current.gatewayEnabled
        val committed = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
            .edit()
            .putBoolean(KEY_SERVICE, nextService)
            .putBoolean(KEY_GATEWAY, nextGateway)
            .putBoolean(KEY_INITIALIZED, true)
            .commit()
        check(committed) { "Could not persist launch auto-start preferences" }
        // Never acknowledge the requested values from memory. Return a fresh
        // canonical readback after the synchronous filesystem commit.
        return read(context)
    }
}

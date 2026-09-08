package com.lord1egypt.pocketclaw.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import com.lord1egypt.pocketclaw.service.PocketClawService

/**
 * 设备启动后自动启动 PicoClaw 服务（如果已开启自动启动）。
 */
class BootReceiver : BroadcastReceiver() {

    companion object {
        private const val TAG = "BootReceiver"
        private const val PREF_NAME = "picoclaw_prefs"
        private const val KEY_AUTO_START = "auto_start"
    }

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED) return

        val prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
        val autoStart = prefs.getBoolean(KEY_AUTO_START, false)

        if (autoStart) {
            Log.i(TAG, "Boot completed, auto-starting PicoClaw service")
            PocketClawService.start(context)
        } else {
            Log.i(TAG, "Boot completed, auto-start is disabled")
        }
    }
}

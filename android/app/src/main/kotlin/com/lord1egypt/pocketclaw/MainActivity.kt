package com.lord1egypt.pocketclaw

import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Environment
import android.provider.Settings
import com.lord1egypt.pocketclaw.media.ChatImagePicker
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine

class MainActivity : FlutterActivity() {
    private var methodChannel: PocketClawMethodChannel? = null

    /**
     * Owns the Chat attachment picker, because only an Activity receives an
     * Activity result. FlutterActivity is a plain android.app.Activity, so the
     * androidx result contracts are not available here.
     */
    private val chatImagePicker = ChatImagePicker(this)

    /** Guards the all-files-access prompt to one appearance per launch. */
    private var storageAccessPromptShown = false

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        methodChannel = PocketClawMethodChannel(this, flutterEngine, chatImagePicker)
    }

    @Suppress("DEPRECATION")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        if (chatImagePicker.onActivityResult(requestCode, resultCode, data)) return
        super.onActivityResult(requestCode, resultCode, data)
    }

    /**
     * Android delivers a permission answer to the Activity, so it is routed to the
     * method channel that asked. PC-DEF-058.
     */
    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray,
    ) {
        if (methodChannel?.onRequestPermissionsResult(requestCode) == true) return
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        // setIntent stays: FlutterActivity and plugins read getIntent(). What
        // was removed beside it is the analytics-only branch that logged the
        // incoming URI — see PC-DEF-024.
        setIntent(intent)
    }

    override fun onResume() {
        super.onResume()
        val storagePromptJustLaunched = requestAllFilesAccessIfNeeded()
        // PC-DEF-058, second attempt. Asking used to happen in the Settings
        // page's initState, and a fresh install never opens Settings -- it lands
        // on the Dashboard -- so the system dialog was never shown and the owner
        // had to enable notifications by hand. Every launch passes through here.
        //
        // Ordered after the storage prompt on purpose: when this same resume has
        // just sent the user to the all-files-access screen, the ask waits for
        // the resume that comes back, rather than being stacked behind it.
        methodChannel?.requestNotificationPermissionOnResume(storagePromptJustLaunched)
    }

    /**
     * Sends the user to the all-files-access screen if PocketClaw still needs it.
     *
     * Android 11+ 需要 MANAGE_EXTERNAL_STORAGE 才能写 Downloads 目录。
     * 若未授予，跳转系统设置页引导用户开启（只弹一次，直到用户授予或主动拒绝）。
     *
     * The "only once" the comment describes was never enforced, so every resume
     * jumped to Settings — including the resume that comes back from the Chat
     * attachment picker, which made choosing an image look like it had thrown
     * the user out of the app.
     *
     * @return whether this call launched the Settings screen.
     */
    private fun requestAllFilesAccessIfNeeded(): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.R ||
            storageAccessPromptShown ||
            Environment.isExternalStorageManager()
        ) {
            return false
        }
        storageAccessPromptShown = true
        try {
            startActivity(
                Intent(
                    Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION,
                    Uri.parse("package:$packageName")
                )
            )
        } catch (e: Exception) {
            startActivity(Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION))
        }
        return true
    }

    override fun cleanUpFlutterEngine(flutterEngine: FlutterEngine) {
        // A pick still open here can never be answered, and an unanswered pick
        // leaves the console's file input waiting forever.
        chatImagePicker.cancelPending()
        methodChannel?.dispose()
        methodChannel = null
        super.cleanUpFlutterEngine(flutterEngine)
    }
}

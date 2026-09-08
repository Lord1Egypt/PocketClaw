package com.lord1egypt.pocketclaw

import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.Settings
import android.util.Log
import com.lord1egypt.pocketclaw.media.ChatImagePicker
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine

class MainActivity : FlutterActivity() {
    companion object {
        private const val TAG = "MainActivity"
    }

    private var methodChannel: PocketClawMethodChannel? = null

    /**
     * Owns the Chat attachment picker, because only an Activity receives an
     * Activity result. FlutterActivity is a plain android.app.Activity, so the
     * androidx result contracts are not available here.
     */
    private val chatImagePicker = ChatImagePicker(this)

    /** Guards the all-files-access prompt to one appearance per launch. */
    private var storageAccessPromptShown = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        logIncomingIntent(intent)
    }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        methodChannel = PocketClawMethodChannel(this, flutterEngine, chatImagePicker)
    }

    @Suppress("DEPRECATION")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        if (chatImagePicker.onActivityResult(requestCode, resultCode, data)) return
        super.onActivityResult(requestCode, resultCode, data)
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        logIncomingIntent(intent)
    }

    override fun onResume() {
        super.onResume()
        // Android 11+ 需要 MANAGE_EXTERNAL_STORAGE 才能写 Downloads 目录。
        // 若未授予，跳转系统设置页引导用户开启（只弹一次，直到用户授予或主动拒绝）。
        //
        // The "only once" the comment describes was never enforced, so every
        // resume jumped to Settings — including the resume that comes back from
        // the Chat attachment picker, which made choosing an image look like it
        // had thrown the user out of the app.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R &&
            !storageAccessPromptShown &&
            !Environment.isExternalStorageManager()
        ) {
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
        }
    }

    override fun cleanUpFlutterEngine(flutterEngine: FlutterEngine) {
        // A pick still open here can never be answered, and an unanswered pick
        // leaves the console's file input waiting forever.
        chatImagePicker.cancelPending()
        methodChannel?.dispose()
        methodChannel = null
        super.cleanUpFlutterEngine(flutterEngine)
    }

    private fun logIncomingIntent(intent: Intent?) {
        val data = intent?.data ?: return
        if (data.scheme == BuildConfig.POCKETCLAW_UMENG_LINK_SCHEME) {
            Log.i(TAG, "Received Umeng link: $data")
        }
    }
}

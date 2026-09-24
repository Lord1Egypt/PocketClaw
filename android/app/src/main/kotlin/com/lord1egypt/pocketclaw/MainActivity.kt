package com.lord1egypt.pocketclaw

import android.content.Intent
import com.lord1egypt.pocketclaw.media.ChatImagePicker
import com.lord1egypt.pocketclaw.storage.LegacyWorkspaceImporter
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

    /**
     * Owns the one-time import of the pre-PC-DEF-077 shared workspace, for the
     * same reason: the document-tree picker answers through an Activity result.
     */
    private val legacyWorkspaceImporter = LegacyWorkspaceImporter(this)

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        methodChannel = PocketClawMethodChannel(
            this, flutterEngine, chatImagePicker, legacyWorkspaceImporter,
        )
    }

    @Suppress("DEPRECATION")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        if (chatImagePicker.onActivityResult(requestCode, resultCode, data)) return
        if (legacyWorkspaceImporter.onActivityResult(requestCode, resultCode, data)) return
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
        // PC-DEF-058, second attempt. Asking used to happen in the Settings
        // page's initState, and a fresh install never opens Settings -- it lands
        // on the Dashboard -- so the system dialog was never shown and the owner
        // had to enable notifications by hand. Every launch passes through here.
        //
        // Nothing else is requested here. The all-files-access redirect that
        // used to run first on every cold launch is gone with the permission
        // (PC-DEF-077): the workspace is app-specific storage, which needs none.
        methodChannel?.requestNotificationPermissionOnResume()
    }

    override fun cleanUpFlutterEngine(flutterEngine: FlutterEngine) {
        // A pick still open here can never be answered, and an unanswered pick
        // leaves the console's file input waiting forever.
        chatImagePicker.cancelPending()
        legacyWorkspaceImporter.cancelPending()
        methodChannel?.dispose()
        methodChannel = null
        super.cleanUpFlutterEngine(flutterEngine)
    }
}

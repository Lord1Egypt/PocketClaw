package com.lord1egypt.pocketclaw

/**
 * What PocketClaw knows about its permission to post notifications.
 *
 * PC-DEF-058. `POST_NOTIFICATIONS` was declared in the manifest but never
 * requested, so on a fresh install the persistent "PocketClaw Running" notification
 * simply never appeared and the owner had to go and enable it in Android Settings
 * by hand. The storage flow next to it has no runtime dialog and must jump to
 * Settings; this one does have a dialog, and should use it.
 *
 * The decision of *what to do next* lives here, separate from the platform calls,
 * because it has rules worth testing: ask once, never nag, stay correct on Android
 * versions where the permission does not exist, and offer a way back for someone
 * who said no.
 */
enum class NotificationPermissionState {
    /** This Android version has no runtime notification permission. */
    NOT_REQUIRED,

    /** Granted — either by the dialog or because the platform grants it. */
    GRANTED,

    /** Never asked on this install. The system dialog is the right next step. */
    NOT_REQUESTED,

    /** Asked and refused. Only Settings can change it now. */
    DENIED,
}

/** What the app should do about the current state. */
enum class NotificationPermissionAction {
    /** Nothing. Either granted or not applicable. */
    NONE,

    /** Show the standard Android permission dialog. */
    REQUEST_SYSTEM_DIALOG,

    /** The dialog will not appear again; offer Settings as explicit recovery. */
    OFFER_SETTINGS,
}

object NotificationPermissionPolicy {
    /**
     * Android 13 (API 33) is where `POST_NOTIFICATIONS` became a runtime
     * permission. Below it, notifications are allowed unless the user turned them
     * off in Settings, and there is no dialog to show.
     */
    const val RUNTIME_PERMISSION_SDK = 33

    /**
     * Resolves the state from the three facts the platform can report.
     *
     * [alreadyAsked] is PocketClaw's own record, not the platform's: Android offers
     * `shouldShowRequestPermissionRationale`, but it answers false both before the
     * first ask and after a permanent denial, so it cannot distinguish "never
     * asked" from "refused for good" on its own.
     */
    fun resolve(
        sdkInt: Int,
        permissionGranted: Boolean,
        alreadyAsked: Boolean,
    ): NotificationPermissionState = when {
        sdkInt < RUNTIME_PERMISSION_SDK -> NotificationPermissionState.NOT_REQUIRED
        permissionGranted -> NotificationPermissionState.GRANTED
        alreadyAsked -> NotificationPermissionState.DENIED
        else -> NotificationPermissionState.NOT_REQUESTED
    }

    /**
     * What to do about a state.
     *
     * A denial is never re-asked automatically. Android stops showing the dialog
     * after a refusal anyway, so a second request is a silent no-op that would make
     * the app look broken; Settings is the honest route.
     */
    fun actionFor(state: NotificationPermissionState): NotificationPermissionAction =
        when (state) {
            NotificationPermissionState.NOT_REQUIRED,
            NotificationPermissionState.GRANTED,
            -> NotificationPermissionAction.NONE
            NotificationPermissionState.NOT_REQUESTED ->
                NotificationPermissionAction.REQUEST_SYSTEM_DIALOG
            NotificationPermissionState.DENIED ->
                NotificationPermissionAction.OFFER_SETTINGS
        }

    /**
     * Whether PocketClaw may expect its foreground notification to be visible.
     *
     * Deliberately not the same question as "is the permission granted": on every
     * Android version the user can switch notifications off in Settings, and below
     * API 33 that is the *only* control. [notificationsEnabled] comes from
     * `NotificationManagerCompat.areNotificationsEnabled()`, which answers it on
     * all versions.
     */
    fun notificationsExpectedVisible(
        state: NotificationPermissionState,
        notificationsEnabled: Boolean,
    ): Boolean = when (state) {
        NotificationPermissionState.DENIED -> false
        else -> notificationsEnabled
    }

    /**
     * Whether a resume should raise the notification dialog.
     *
     * PC-DEF-058, second attempt. The first one asked from the Settings page's
     * initState, and a fresh install never opens Settings -- it lands on the
     * Dashboard -- so the dialog was never shown at all. Asking belongs on the
     * resume path, which every launch takes, and this is the rule for it.
     *
     * [storagePromptJustLaunched] is the ordering the owner requires: on a first
     * run the all-files-access screen is opened from the same resume, and
     * stacking a permission dialog behind a Settings activity the user is being
     * sent to is how a dialog gets dismissed unseen. The ask waits for the
     * resume that comes back.
     *
     * [alreadyPromptedThisLaunch] keeps one dialog per launch, so a resume from
     * the image picker or a screen lock cannot re-raise it.
     */
    fun shouldRequestOnResume(
        state: NotificationPermissionState,
        storagePromptJustLaunched: Boolean,
        alreadyPromptedThisLaunch: Boolean,
    ): Boolean {
        if (storagePromptJustLaunched || alreadyPromptedThisLaunch) return false
        return actionFor(state) == NotificationPermissionAction.REQUEST_SYSTEM_DIALOG
    }

    /** The wire value handed to Flutter. Stable: the Dart side matches on it. */
    fun wireName(state: NotificationPermissionState): String = when (state) {
        NotificationPermissionState.NOT_REQUIRED -> "notRequired"
        NotificationPermissionState.GRANTED -> "granted"
        NotificationPermissionState.NOT_REQUESTED -> "notRequested"
        NotificationPermissionState.DENIED -> "denied"
    }
}

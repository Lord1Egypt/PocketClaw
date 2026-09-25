// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get stop => 'Stop';

  @override
  String get webAdmin => 'Web Admin';

  @override
  String get logs => 'Logs';

  @override
  String get statusRunning => 'Running';

  @override
  String get statusStopped => 'Stopped';

  @override
  String get settings => 'Settings';

  @override
  String get address => 'Address';

  @override
  String get port => 'Port';

  @override
  String get save => 'Save';

  @override
  String get notStarted => 'Service Not Started';

  @override
  String get startHint => 'Please start the service from the dashboard first.';

  @override
  String get goToDashboard => 'Go to Dashboard';

  @override
  String get back => 'Back';

  @override
  String get forward => 'Forward';

  @override
  String get refresh => 'Refresh';

  @override
  String get publicMode => 'Public Mode';

  @override
  String get publicModeHintDesc =>
      'Expose the password-protected Dashboard to devices on your LAN. The Core gateway stays private.';

  @override
  String get publicModeApplying => 'Applying network mode...';

  @override
  String get themeSelection => 'Theme';

  @override
  String get launchService => 'LAUNCH SERVICE';

  @override
  String get stopService => 'STOP SERVICE';

  @override
  String get endpoint => 'ENDPOINT';

  @override
  String get statusActive => 'ACTIVE';

  @override
  String get statusSyncing => 'SYNCING';

  @override
  String get statusIdle => 'IDLE';

  @override
  String get publicModeEnabled => 'Public Mode Enabled';

  @override
  String get localMode => 'Local Mode';

  @override
  String get unableToGetDeviceIp => 'No LAN address available';

  @override
  String get localModeHint =>
      '1. Go to Service Config\n2. Turn on Public Mode\n3. Scan QR code to access PocketClaw';

  @override
  String get publicModeHint =>
      '1. Start the service\n2. Scan QR code to access PocketClaw';

  @override
  String get noLogsToExport => 'No logs to export';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Logs saved to Downloads: $path';
  }

  @override
  String get shareLogsText => 'PocketClaw logs';

  @override
  String get workspaceDirectory => 'Workspace';

  @override
  String get legacyWorkspaceTitle => 'Earlier workspace found';

  @override
  String legacyWorkspaceBody(Object path) {
    return 'An older version kept your workspace in $path. PocketClaw now uses its own app storage and has left that folder untouched. You can copy it in: the copy goes into a folder of its own, and nothing is overwritten or deleted.';
  }

  @override
  String get legacyWorkspaceImport => 'Copy into workspace';

  @override
  String get legacyWorkspaceHide => 'Hide';

  @override
  String legacyWorkspaceCopied(Object count, Object folder) {
    return 'Copied $count files into $folder.';
  }

  @override
  String legacyWorkspacePartial(Object count, Object folder, Object failed) {
    return 'Copied $count files into $folder. $failed could not be copied.';
  }

  @override
  String get legacyWorkspaceFailed => 'The folder could not be copied.';

  @override
  String get legacyWorkspaceEmpty => 'The selected folder had nothing to copy.';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'Logs saved to Downloads (Android media library): $name';
  }

  @override
  String shareFailed(Object error) {
    return 'Failed to open share dialog: $error';
  }

  @override
  String get exportLogs => 'Export Logs';

  @override
  String logEventsCount(int count) {
    return '$count EVENTS';
  }

  @override
  String get cancel => 'Cancel';

  @override
  String get language => 'Language';

  @override
  String get selectLanguage => 'Select Language';

  @override
  String get about => 'About';

  @override
  String get aboutDescription =>
      'PocketClaw is your private AI assistant workspace.';

  @override
  String get aboutAppVersionLabel => 'PocketClaw version';

  @override
  String get aboutCoreVersionLabel => 'Runtime version';

  @override
  String get aboutVersionUnavailable => 'Unavailable';

  @override
  String get close => 'Close';

  @override
  String get contextMemoryTitle => 'Telegram Context Memory';

  @override
  String get contextMemoryDescription =>
      'Controls how many recent conversation messages are sent to the AI. Older conversation stays in Telegram and is represented by the rolling summary.';

  @override
  String get contextMemoryHelp =>
      'More messages provide more recent context but use more tokens. Older messages remain in Telegram and may be retained through the rolling summary.';

  @override
  String get contextMemoryRecommended => 'Recommended';

  @override
  String get contextMemoryCustom => 'Custom';

  @override
  String get contextMemoryCustomLabel => 'Messages';

  @override
  String contextMemoryRangeError(int min, int max) {
    return 'Enter a whole number between $min and $max.';
  }

  @override
  String get contextMemorySaveFailed => 'Could not save this setting.';

  @override
  String get settingsSave => 'Save';

  @override
  String get autoStartServiceTitle => 'Start PocketClaw service automatically';

  @override
  String get autoStartGatewayTitle => 'Start Gateway automatically';

  @override
  String get autoStartPreferenceOn => 'Auto-start preference: ON';

  @override
  String get autoStartPreferenceOff => 'Auto-start preference: OFF';

  @override
  String get runtimeRunning => 'Runtime: Running';

  @override
  String get runtimeStarting => 'Runtime: Starting';

  @override
  String get runtimeStopped => 'Runtime: Stopped';

  @override
  String get gatewayAutoStartHint =>
      'Applies the next time the PocketClaw service starts. Gateway runtime is managed in the Dashboard.';

  @override
  String get manageTelegramConnection => 'Manage Telegram connection';

  @override
  String get manageModelsTitle => 'Manage Models';

  @override
  String get manageModelsDescription =>
      'Add, edit, test, and choose AI models.';

  @override
  String get githubChecking => 'Checking…';

  @override
  String get githubConnected => 'Connected';

  @override
  String get githubNotConnected => 'Not connected';

  @override
  String githubConnectedAs(String login) {
    return 'Connected as $login';
  }

  @override
  String get githubDescription =>
      'Used by the bundled gh and by Git over HTTPS. The token is encrypted on this device and is never shown again.';

  @override
  String get githubTestConnection => 'Test connection';

  @override
  String get githubDisconnect => 'Disconnect';

  @override
  String get githubConnectAction => 'Connect GitHub';

  @override
  String get githubConnect => 'Connect';

  @override
  String get githubTokenLabel => 'Personal access token';

  @override
  String get githubTokenHint =>
      'Paste a GitHub personal access token with the scopes you need (repo for private repositories).';

  @override
  String get githubTokenRejected => 'GitHub did not accept this token.';

  @override
  String get githubAuthWorking => 'GitHub authentication is working.';

  @override
  String get githubAuthNotWorking => 'GitHub authentication is not working.';

  @override
  String githubAuthenticatedAs(String login) {
    return 'Authenticated as $login.';
  }

  @override
  String get githubCredentialRemoveFailed => 'Could not remove the credential.';

  @override
  String get githubYourAccount => 'your GitHub account';

  @override
  String githubConnectedReport(String who, String outcome) {
    return 'Connected as $who. $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return 'Disconnected. $outcome';
  }

  @override
  String get credentialAppliedNow => 'gh and git can use it now.';

  @override
  String get credentialAppliesNextStart =>
      'It will be used the next time PocketClaw starts.';

  @override
  String get credentialAppliesDeferred =>
      'Saved. PocketClaw is busy starting, so it will apply automatically as soon as that finishes.';

  @override
  String get whatsNewTitle => 'What\'s New';

  @override
  String get whatsNewDescription => 'The main changes in this release.';

  @override
  String get whatsNewBadge => 'NEW';

  @override
  String get whatsNewSectionNew => 'New';

  @override
  String get whatsNewSectionImprovements => 'Improvements';

  @override
  String get whatsNewSectionFixes => 'Fixes';

  @override
  String get whatsNew020New1 =>
      'Managed Runtime installs and keeps PocketClaw\'s bundled tools up to date for you.';

  @override
  String get whatsNew020New2 =>
      'Bundled developer tools: Git, GitHub CLI, curl, ripgrep, jq and SQLite.';

  @override
  String get whatsNew020New3 =>
      'Support for PocketClaw\'s bundled Python 3.14 runtime.';

  @override
  String get whatsNew020New4 =>
      'Secure GitHub sign-in, shared by the bundled Git and GitHub CLI.';

  @override
  String get whatsNew020New5 =>
      'One-tap Telegram setup: PocketClaw creates your own bot, in the app or from the Dashboard in a browser. There is no token to copy, and only your own account can talk to it.';

  @override
  String get whatsNew020New6 =>
      'A Status view on the Dashboard: active work, channels, the model in use and runtime resources.';

  @override
  String get whatsNew020Improvement1 =>
      'More resilient providers: a failing request no longer ends the turn.';

  @override
  String get whatsNew020Improvement2 =>
      'A more accurate Managed Runtime catalog.';

  @override
  String get whatsNew020Improvement3 =>
      'A cleaner PocketClaw identity across the web interface and the default workspace.';

  @override
  String get whatsNew020Improvement4 =>
      'A new app icon and a refreshed About screen, drawn from PocketClaw\'s Aperture design.';

  @override
  String get whatsNew020Improvement5 =>
      'Assistant replies read more naturally, without a fixed sign-off at the end.';

  @override
  String get whatsNew020Improvement6 =>
      'PocketClaw no longer asks for the Phone permission — nothing in the app used it.';

  @override
  String get whatsNew020Improvement7 =>
      'Diagnostic logs are now kept privately inside the app instead of in your Downloads folder. Your workspace stays where it was.';

  @override
  String get whatsNew020Improvement8 =>
      'A substantial upgrade to how PocketClaw keeps its state and runs its services, for steadier day-to-day reliability.';

  @override
  String get whatsNew020Improvement9 =>
      'After this update, the Dashboard asks you to sign in once more.';

  @override
  String get whatsNew020Improvement10 =>
      'After this update, the Web chat channel starts a new conversation.';

  @override
  String get whatsNew020Improvement11 =>
      'After this update, it is worth checking your notification preferences once.';

  @override
  String get whatsNew020Fix1 =>
      'The Gateway now recovers from a stale process record left behind by an earlier run.';

  @override
  String get whatsNew020Fix2 =>
      'Channel lists with more than one entry are preserved correctly when settings are saved.';

  @override
  String get whatsNew021Fix1 =>
      'The bundled Git no longer crashes while cloning a repository or updating a branch.';

  @override
  String get whatsNew022Improvement1 =>
      'After a long task, Telegram delivers the answer as a new message, so it notifies you and appears below anything you sent meanwhile.';

  @override
  String get whatsNew022Improvement2 =>
      'Telegram tells you when your message is queued and how many are ahead of it.';

  @override
  String get whatsNew022Improvement3 =>
      'The workspace now lives in PocketClaw\'s own storage, and the app asks for no storage permission. A workspace an older version left in Download/pocketclaw is untouched and can be copied in from Settings.';

  @override
  String get whatsNew022Fix1 =>
      'Very large tool output no longer overflows the conversation.';

  @override
  String get whatsNew022Fix2 =>
      'Removed Settings entries that had no effect on Android: Devices, Launch at Login and Service Port.';

  @override
  String get whatsNew022Improvement4 =>
      'PocketClaw now requires Android 8.0 or newer.';

  @override
  String get whatsNew022Fix3 =>
      'Fixed an Android startup crash that could appear after restarting the phone.';

  @override
  String get whatsNew023Improvement1 =>
      'Security update: PocketClaw\'s background core is now built with a current, supported Go release and updated libraries that fix known vulnerabilities.';

  @override
  String get whatsNew023Improvement2 =>
      'The bundled GitHub command-line tool is updated to version 2.101.';

  @override
  String get whatsNew023Improvement3 =>
      'Build hardening in preparation for F-Droid: the app and its bundled tools are built entirely from source in a pinned, repeatable environment.';

  @override
  String get settingsGroupConnection => 'Connection';

  @override
  String get settingsGroupAgent => 'Agent';

  @override
  String get settingsGroupIntegrations => 'Integrations';

  @override
  String get settingsGroupAppearance => 'Appearance';

  @override
  String get statusTitle => 'Status';

  @override
  String get statusSectionSystem => 'System';

  @override
  String get statusSectionAi => 'AI';

  @override
  String get statusSectionActivity => 'Activity';

  @override
  String get statusSectionChannels => 'Channels';

  @override
  String get statusSectionResources => 'Resources';

  @override
  String get statusSinceGatewayStart => 'Since Gateway start';

  @override
  String get statusGateway => 'Gateway';

  @override
  String get statusUptime => 'Uptime';

  @override
  String get statusAppVersion => 'App version';

  @override
  String get statusCoreVersion => 'Core version';

  @override
  String get statusActiveModel => 'Active model';

  @override
  String get statusConfiguredDefault => 'Configured default';

  @override
  String get statusProvider => 'Provider';

  @override
  String get statusFallbacks => 'Fallbacks';

  @override
  String get statusActiveTurns => 'Active turns';

  @override
  String get statusActiveSubagents => 'Active subagents';

  @override
  String get statusWaiting => 'Waiting';

  @override
  String get statusCompleted => 'Completed';

  @override
  String get statusFailed => 'Failed';

  @override
  String get statusCancelled => 'Cancelled';

  @override
  String get statusToolCalls => 'Tool calls';

  @override
  String get statusToolCallsFailed => 'Tool calls failed';

  @override
  String get statusLastActivity => 'Last activity';

  @override
  String get statusCoreMemory => 'Core memory';

  @override
  String get statusCoreCpuTime => 'Core CPU time';

  @override
  String get statusNoChannels => 'No channels configured';

  @override
  String get statusDetailUnavailable => 'Detailed status is unavailable';

  @override
  String get statusJustNow => 'Just now';

  @override
  String statusMinutesAgo(int minutes) {
    return '${minutes}m ago';
  }

  @override
  String statusHoursAgo(int hours) {
    return '${hours}h ago';
  }

  @override
  String statusDaysAgo(int days) {
    return '${days}d ago';
  }

  @override
  String statusDurationSeconds(int seconds) {
    return '${seconds}s';
  }

  @override
  String statusDurationMinutes(int minutes, int seconds) {
    return '${minutes}m ${seconds}s';
  }

  @override
  String statusDurationHours(int hours, int minutes) {
    return '${hours}h ${minutes}m';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '${days}d ${hours}h';
  }

  @override
  String get notificationPermissionTitle => 'Notifications';

  @override
  String get notificationPermissionGranted =>
      'PocketClaw can show its Running notification.';

  @override
  String get notificationPermissionBlocked =>
      'Notifications are off, so the PocketClaw Running notification will not appear.';

  @override
  String get notificationPermissionOpenSettings => 'Open notification settings';

  @override
  String get whatsNew020New7 =>
      'Telegram management from the Dashboard: connect a bot, replace it, or disconnect it.';

  @override
  String get whatsNew020New8 =>
      'Provider and model management in the Dashboard: add a provider, rotate an API key, or remove a model.';

  @override
  String get whatsNew020Improvement12 =>
      'Telegram settings apply as soon as you save them, with no manual restart.';

  @override
  String get whatsNew020Improvement13 =>
      'When no AI model is set up, PocketClaw says exactly what is missing and gives you a code to look up.';

  @override
  String get whatsNew020Improvement14 =>
      'Signing in to the Dashboard takes you to the screen you asked for.';

  @override
  String get whatsNew020Improvement15 =>
      'Diagnostic logs stay detailed without ever containing your keys, tokens or message text.';

  @override
  String get whatsNew020Improvement16 =>
      'A Dashboard nobody has claimed yet is never reachable from the network, even with Public Mode on.';
}

// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => 'Run';

  @override
  String get stop => 'Stop';

  @override
  String get config => 'Config';

  @override
  String get webAdmin => 'Web Admin';

  @override
  String get logs => 'Logs';

  @override
  String get viewLogs => 'View Logs';

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
  String get showWindow => 'Show Window';

  @override
  String get exit => 'Exit';

  @override
  String get binaryPath => 'Binary Path';

  @override
  String get browse => 'Browse';

  @override
  String get pathError => 'Invalid Path';

  @override
  String get arguments => 'Arguments';

  @override
  String get argumentsHint => 'e.g. config.json';

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
  String get coreBinaryMissing =>
      'Core binary not found. Place the platform binary into app/bin/ or set the path in Settings.';

  @override
  String get coreStartFailed => 'Failed to start core service.';

  @override
  String get coreStopFailed => 'Failed to stop core service.';

  @override
  String get coreInvalidBinary => 'Invalid core binary file.';

  @override
  String coreUnknownError(Object code) {
    return 'Unknown core error: $code';
  }

  @override
  String get coreValid => 'Core binary is valid.';

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
  String get check => 'Check';

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
  String get deviceReportingTitle => 'Device compatibility feedback';

  @override
  String get deviceReportingSubtitle =>
      'Used only for OS-version and app-version compatibility checks. No chat messages, account details, or personal content are involved';

  @override
  String get deviceReportingConsentTitle => 'Help improve device compatibility';

  @override
  String get deviceReportingConsentDescription =>
      'When enabled, only an anonymous installation ID, OS version, and app version are sent to understand compatibility. Language and region can be collected by Firebase Analytics separately. No chat messages, typed content, account details, files, or custom settings are uploaded';

  @override
  String get deviceReportingBannerDescription =>
      'Only an anonymous installation ID, OS version, and app version are synced to improve compatibility. Language and region may be collected separately by Firebase Analytics. No chat messages, account details, files, or personal content are sent';

  @override
  String get deviceReportingWhatWillBeSent =>
      'Only these device details are included';

  @override
  String get deviceReportingDeviceLabel => 'Device Model';

  @override
  String get deviceReportingPlatformLabel => 'Device Category';

  @override
  String get deviceReportingSystemLabel => 'OS Version';

  @override
  String get deviceReportingTimingNote =>
      'A sync runs once when enabled, and again only after a system update is detected';

  @override
  String get deviceReportingDeny => 'Not now';

  @override
  String get deviceReportingAllow => 'Turn on';

  @override
  String get deviceReportingUploadSucceeded =>
      'Device compatibility feedback is on';

  @override
  String get deviceReportingUploadFailed =>
      'Device compatibility feedback is on, but the current device-info sync did not complete';

  @override
  String get deviceReportingDisabled => 'Device compatibility feedback is off';

  @override
  String get localModeHint =>
      '1. Go to Service Config\n2. Turn on Public Mode\n3. Scan QR code to access PocketClaw';

  @override
  String get publicModeHint =>
      '1. Start the service\n2. Scan QR code to access PocketClaw';

  @override
  String get noLogsToExport => 'No logs to export';

  @override
  String get logsSavedToMediaLibrary =>
      'Logs saved to Downloads (Android media library)';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Logs saved to Downloads: $path';
  }

  @override
  String get shareLogsText => 'PocketClaw logs';

  @override
  String get workspaceDirectory => 'Workspace';

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
  String get unsavedChanges => 'Unsaved Changes';

  @override
  String get unsavedChangesHint =>
      'You have unsaved changes. Do you want to discard them?';

  @override
  String get cancel => 'Cancel';

  @override
  String get discard => 'Discard';

  @override
  String get saved => 'Saved';

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
  String get whatsNew020New5 => 'Telegram integration, set up from Settings.';

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
  String get whatsNew020Fix1 =>
      'The Gateway now recovers from a stale process record left behind by an earlier run.';

  @override
  String get whatsNew020Fix2 =>
      'Channel lists with more than one entry are preserved correctly when settings are saved.';

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
}

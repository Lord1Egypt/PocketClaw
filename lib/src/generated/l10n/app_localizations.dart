import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_ar.dart';
import 'app_localizations_de.dart';
import 'app_localizations_en.dart';
import 'app_localizations_es.dart';
import 'app_localizations_fr.dart';
import 'app_localizations_hi.dart';
import 'app_localizations_id.dart';
import 'app_localizations_ja.dart';
import 'app_localizations_ko.dart';
import 'app_localizations_pt.dart';
import 'app_localizations_ru.dart';
import 'app_localizations_zh.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'l10n/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations? of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations);
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[
    Locale('ar'),
    Locale('de'),
    Locale('en'),
    Locale('es'),
    Locale('fr'),
    Locale('hi'),
    Locale('id'),
    Locale('ja'),
    Locale('ko'),
    Locale('pt'),
    Locale('ru'),
    Locale('zh'),
  ];

  /// The title of the application
  ///
  /// In en, this message translates to:
  /// **'PocketClaw'**
  String get appTitle;

  /// No description provided for @run.
  ///
  /// In en, this message translates to:
  /// **'Run'**
  String get run;

  /// No description provided for @stop.
  ///
  /// In en, this message translates to:
  /// **'Stop'**
  String get stop;

  /// No description provided for @config.
  ///
  /// In en, this message translates to:
  /// **'Config'**
  String get config;

  /// No description provided for @webAdmin.
  ///
  /// In en, this message translates to:
  /// **'Web Admin'**
  String get webAdmin;

  /// No description provided for @logs.
  ///
  /// In en, this message translates to:
  /// **'Logs'**
  String get logs;

  /// No description provided for @viewLogs.
  ///
  /// In en, this message translates to:
  /// **'View Logs'**
  String get viewLogs;

  /// No description provided for @statusRunning.
  ///
  /// In en, this message translates to:
  /// **'Running'**
  String get statusRunning;

  /// No description provided for @statusStopped.
  ///
  /// In en, this message translates to:
  /// **'Stopped'**
  String get statusStopped;

  /// No description provided for @settings.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settings;

  /// No description provided for @address.
  ///
  /// In en, this message translates to:
  /// **'Address'**
  String get address;

  /// No description provided for @port.
  ///
  /// In en, this message translates to:
  /// **'Port'**
  String get port;

  /// No description provided for @save.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get save;

  /// No description provided for @showWindow.
  ///
  /// In en, this message translates to:
  /// **'Show Window'**
  String get showWindow;

  /// No description provided for @exit.
  ///
  /// In en, this message translates to:
  /// **'Exit'**
  String get exit;

  /// No description provided for @binaryPath.
  ///
  /// In en, this message translates to:
  /// **'Binary Path'**
  String get binaryPath;

  /// No description provided for @browse.
  ///
  /// In en, this message translates to:
  /// **'Browse'**
  String get browse;

  /// No description provided for @pathError.
  ///
  /// In en, this message translates to:
  /// **'Invalid Path'**
  String get pathError;

  /// No description provided for @arguments.
  ///
  /// In en, this message translates to:
  /// **'Arguments'**
  String get arguments;

  /// No description provided for @argumentsHint.
  ///
  /// In en, this message translates to:
  /// **'e.g. config.json'**
  String get argumentsHint;

  /// No description provided for @notStarted.
  ///
  /// In en, this message translates to:
  /// **'Service Not Started'**
  String get notStarted;

  /// No description provided for @startHint.
  ///
  /// In en, this message translates to:
  /// **'Please start the service from the dashboard first.'**
  String get startHint;

  /// No description provided for @goToDashboard.
  ///
  /// In en, this message translates to:
  /// **'Go to Dashboard'**
  String get goToDashboard;

  /// No description provided for @back.
  ///
  /// In en, this message translates to:
  /// **'Back'**
  String get back;

  /// No description provided for @forward.
  ///
  /// In en, this message translates to:
  /// **'Forward'**
  String get forward;

  /// No description provided for @refresh.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get refresh;

  /// No description provided for @coreBinaryMissing.
  ///
  /// In en, this message translates to:
  /// **'Core binary not found. Place the platform binary into app/bin/ or set the path in Settings.'**
  String get coreBinaryMissing;

  /// No description provided for @coreStartFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to start core service.'**
  String get coreStartFailed;

  /// No description provided for @coreStopFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to stop core service.'**
  String get coreStopFailed;

  /// No description provided for @coreInvalidBinary.
  ///
  /// In en, this message translates to:
  /// **'Invalid core binary file.'**
  String get coreInvalidBinary;

  /// No description provided for @coreUnknownError.
  ///
  /// In en, this message translates to:
  /// **'Unknown core error: {code}'**
  String coreUnknownError(Object code);

  /// No description provided for @coreValid.
  ///
  /// In en, this message translates to:
  /// **'Core binary is valid.'**
  String get coreValid;

  /// No description provided for @publicMode.
  ///
  /// In en, this message translates to:
  /// **'Public Mode'**
  String get publicMode;

  /// No description provided for @publicModeHintDesc.
  ///
  /// In en, this message translates to:
  /// **'Expose the password-protected Dashboard to devices on your LAN. The Core gateway stays private.'**
  String get publicModeHintDesc;

  /// No description provided for @publicModeApplying.
  ///
  /// In en, this message translates to:
  /// **'Applying network mode...'**
  String get publicModeApplying;

  /// No description provided for @themeSelection.
  ///
  /// In en, this message translates to:
  /// **'Theme'**
  String get themeSelection;

  /// No description provided for @check.
  ///
  /// In en, this message translates to:
  /// **'Check'**
  String get check;

  /// No description provided for @launchService.
  ///
  /// In en, this message translates to:
  /// **'LAUNCH SERVICE'**
  String get launchService;

  /// No description provided for @stopService.
  ///
  /// In en, this message translates to:
  /// **'STOP SERVICE'**
  String get stopService;

  /// No description provided for @endpoint.
  ///
  /// In en, this message translates to:
  /// **'ENDPOINT'**
  String get endpoint;

  /// No description provided for @statusActive.
  ///
  /// In en, this message translates to:
  /// **'ACTIVE'**
  String get statusActive;

  /// No description provided for @statusSyncing.
  ///
  /// In en, this message translates to:
  /// **'SYNCING'**
  String get statusSyncing;

  /// No description provided for @statusIdle.
  ///
  /// In en, this message translates to:
  /// **'IDLE'**
  String get statusIdle;

  /// No description provided for @publicModeEnabled.
  ///
  /// In en, this message translates to:
  /// **'Public Mode Enabled'**
  String get publicModeEnabled;

  /// No description provided for @localMode.
  ///
  /// In en, this message translates to:
  /// **'Local Mode'**
  String get localMode;

  /// No description provided for @unableToGetDeviceIp.
  ///
  /// In en, this message translates to:
  /// **'No LAN address available'**
  String get unableToGetDeviceIp;

  /// No description provided for @deviceReportingTitle.
  ///
  /// In en, this message translates to:
  /// **'Device compatibility feedback'**
  String get deviceReportingTitle;

  /// No description provided for @deviceReportingSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Used only for OS-version and app-version compatibility checks. No chat messages, account details, or personal content are involved'**
  String get deviceReportingSubtitle;

  /// No description provided for @deviceReportingConsentTitle.
  ///
  /// In en, this message translates to:
  /// **'Help improve device compatibility'**
  String get deviceReportingConsentTitle;

  /// No description provided for @deviceReportingConsentDescription.
  ///
  /// In en, this message translates to:
  /// **'When enabled, only an anonymous installation ID, OS version, and app version are sent to understand compatibility. Language and region can be collected by Firebase Analytics separately. No chat messages, typed content, account details, files, or custom settings are uploaded'**
  String get deviceReportingConsentDescription;

  /// No description provided for @deviceReportingBannerDescription.
  ///
  /// In en, this message translates to:
  /// **'Only an anonymous installation ID, OS version, and app version are synced to improve compatibility. Language and region may be collected separately by Firebase Analytics. No chat messages, account details, files, or personal content are sent'**
  String get deviceReportingBannerDescription;

  /// No description provided for @deviceReportingWhatWillBeSent.
  ///
  /// In en, this message translates to:
  /// **'Only these device details are included'**
  String get deviceReportingWhatWillBeSent;

  /// No description provided for @deviceReportingDeviceLabel.
  ///
  /// In en, this message translates to:
  /// **'Device Model'**
  String get deviceReportingDeviceLabel;

  /// No description provided for @deviceReportingPlatformLabel.
  ///
  /// In en, this message translates to:
  /// **'Device Category'**
  String get deviceReportingPlatformLabel;

  /// No description provided for @deviceReportingSystemLabel.
  ///
  /// In en, this message translates to:
  /// **'OS Version'**
  String get deviceReportingSystemLabel;

  /// No description provided for @deviceReportingTimingNote.
  ///
  /// In en, this message translates to:
  /// **'A sync runs once when enabled, and again only after a system update is detected'**
  String get deviceReportingTimingNote;

  /// No description provided for @deviceReportingDeny.
  ///
  /// In en, this message translates to:
  /// **'Not now'**
  String get deviceReportingDeny;

  /// No description provided for @deviceReportingAllow.
  ///
  /// In en, this message translates to:
  /// **'Turn on'**
  String get deviceReportingAllow;

  /// No description provided for @deviceReportingUploadSucceeded.
  ///
  /// In en, this message translates to:
  /// **'Device compatibility feedback is on'**
  String get deviceReportingUploadSucceeded;

  /// No description provided for @deviceReportingUploadFailed.
  ///
  /// In en, this message translates to:
  /// **'Device compatibility feedback is on, but the current device-info sync did not complete'**
  String get deviceReportingUploadFailed;

  /// No description provided for @deviceReportingDisabled.
  ///
  /// In en, this message translates to:
  /// **'Device compatibility feedback is off'**
  String get deviceReportingDisabled;

  /// No description provided for @localModeHint.
  ///
  /// In en, this message translates to:
  /// **'1. Go to Service Config\n2. Turn on Public Mode\n3. Scan QR code to access PocketClaw'**
  String get localModeHint;

  /// No description provided for @publicModeHint.
  ///
  /// In en, this message translates to:
  /// **'1. Start the service\n2. Scan QR code to access PocketClaw'**
  String get publicModeHint;

  /// No description provided for @noLogsToExport.
  ///
  /// In en, this message translates to:
  /// **'No logs to export'**
  String get noLogsToExport;

  /// No description provided for @logsSavedToMediaLibrary.
  ///
  /// In en, this message translates to:
  /// **'Logs saved to Downloads (Android media library)'**
  String get logsSavedToMediaLibrary;

  /// No description provided for @logsSavedToDownloads.
  ///
  /// In en, this message translates to:
  /// **'Logs saved to Downloads: {path}'**
  String logsSavedToDownloads(Object path);

  /// No description provided for @shareLogsText.
  ///
  /// In en, this message translates to:
  /// **'PocketClaw logs'**
  String get shareLogsText;

  /// No description provided for @workspaceDirectory.
  ///
  /// In en, this message translates to:
  /// **'Workspace'**
  String get workspaceDirectory;

  /// No description provided for @logsSavedToMediaLibraryWithName.
  ///
  /// In en, this message translates to:
  /// **'Logs saved to Downloads (Android media library): {name}'**
  String logsSavedToMediaLibraryWithName(Object name);

  /// No description provided for @shareFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed to open share dialog: {error}'**
  String shareFailed(Object error);

  /// No description provided for @exportLogs.
  ///
  /// In en, this message translates to:
  /// **'Export Logs'**
  String get exportLogs;

  /// No description provided for @logEventsCount.
  ///
  /// In en, this message translates to:
  /// **'{count} EVENTS'**
  String logEventsCount(int count);

  /// No description provided for @unsavedChanges.
  ///
  /// In en, this message translates to:
  /// **'Unsaved Changes'**
  String get unsavedChanges;

  /// No description provided for @unsavedChangesHint.
  ///
  /// In en, this message translates to:
  /// **'You have unsaved changes. Do you want to discard them?'**
  String get unsavedChangesHint;

  /// No description provided for @cancel.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get cancel;

  /// No description provided for @discard.
  ///
  /// In en, this message translates to:
  /// **'Discard'**
  String get discard;

  /// No description provided for @saved.
  ///
  /// In en, this message translates to:
  /// **'Saved'**
  String get saved;

  /// No description provided for @language.
  ///
  /// In en, this message translates to:
  /// **'Language'**
  String get language;

  /// No description provided for @selectLanguage.
  ///
  /// In en, this message translates to:
  /// **'Select Language'**
  String get selectLanguage;

  /// No description provided for @about.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get about;

  /// No description provided for @aboutDescription.
  ///
  /// In en, this message translates to:
  /// **'PocketClaw is your private AI assistant workspace.'**
  String get aboutDescription;

  /// No description provided for @aboutAppVersionLabel.
  ///
  /// In en, this message translates to:
  /// **'PocketClaw version'**
  String get aboutAppVersionLabel;

  /// No description provided for @aboutCoreVersionLabel.
  ///
  /// In en, this message translates to:
  /// **'Runtime version'**
  String get aboutCoreVersionLabel;

  /// No description provided for @aboutVersionUnavailable.
  ///
  /// In en, this message translates to:
  /// **'Unavailable'**
  String get aboutVersionUnavailable;

  /// No description provided for @close.
  ///
  /// In en, this message translates to:
  /// **'Close'**
  String get close;

  /// No description provided for @contextMemoryTitle.
  ///
  /// In en, this message translates to:
  /// **'Telegram Context Memory'**
  String get contextMemoryTitle;

  /// No description provided for @contextMemoryDescription.
  ///
  /// In en, this message translates to:
  /// **'Controls how many recent conversation messages are sent to the AI. Older conversation stays in Telegram and is represented by the rolling summary.'**
  String get contextMemoryDescription;

  /// No description provided for @contextMemoryHelp.
  ///
  /// In en, this message translates to:
  /// **'More messages provide more recent context but use more tokens. Older messages remain in Telegram and may be retained through the rolling summary.'**
  String get contextMemoryHelp;

  /// No description provided for @contextMemoryRecommended.
  ///
  /// In en, this message translates to:
  /// **'Recommended'**
  String get contextMemoryRecommended;

  /// No description provided for @contextMemoryCustom.
  ///
  /// In en, this message translates to:
  /// **'Custom'**
  String get contextMemoryCustom;

  /// No description provided for @contextMemoryCustomLabel.
  ///
  /// In en, this message translates to:
  /// **'Messages'**
  String get contextMemoryCustomLabel;

  /// No description provided for @contextMemoryRangeError.
  ///
  /// In en, this message translates to:
  /// **'Enter a whole number between {min} and {max}.'**
  String contextMemoryRangeError(int min, int max);

  /// No description provided for @contextMemorySaveFailed.
  ///
  /// In en, this message translates to:
  /// **'Could not save this setting.'**
  String get contextMemorySaveFailed;

  /// No description provided for @settingsSave.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get settingsSave;

  /// No description provided for @autoStartServiceTitle.
  ///
  /// In en, this message translates to:
  /// **'Start PocketClaw service automatically'**
  String get autoStartServiceTitle;

  /// No description provided for @autoStartGatewayTitle.
  ///
  /// In en, this message translates to:
  /// **'Start Gateway automatically'**
  String get autoStartGatewayTitle;

  /// No description provided for @autoStartPreferenceOn.
  ///
  /// In en, this message translates to:
  /// **'Auto-start preference: ON'**
  String get autoStartPreferenceOn;

  /// No description provided for @autoStartPreferenceOff.
  ///
  /// In en, this message translates to:
  /// **'Auto-start preference: OFF'**
  String get autoStartPreferenceOff;

  /// No description provided for @runtimeRunning.
  ///
  /// In en, this message translates to:
  /// **'Runtime: Running'**
  String get runtimeRunning;

  /// No description provided for @runtimeStarting.
  ///
  /// In en, this message translates to:
  /// **'Runtime: Starting'**
  String get runtimeStarting;

  /// No description provided for @runtimeStopped.
  ///
  /// In en, this message translates to:
  /// **'Runtime: Stopped'**
  String get runtimeStopped;

  /// No description provided for @gatewayAutoStartHint.
  ///
  /// In en, this message translates to:
  /// **'Applies the next time the PocketClaw service starts. Gateway runtime is managed in the Dashboard.'**
  String get gatewayAutoStartHint;

  /// No description provided for @manageTelegramConnection.
  ///
  /// In en, this message translates to:
  /// **'Manage Telegram connection'**
  String get manageTelegramConnection;

  /// No description provided for @manageModelsTitle.
  ///
  /// In en, this message translates to:
  /// **'Manage Models'**
  String get manageModelsTitle;

  /// No description provided for @manageModelsDescription.
  ///
  /// In en, this message translates to:
  /// **'Add, edit, test, and choose AI models.'**
  String get manageModelsDescription;

  /// No description provided for @githubChecking.
  ///
  /// In en, this message translates to:
  /// **'Checking…'**
  String get githubChecking;

  /// No description provided for @githubConnected.
  ///
  /// In en, this message translates to:
  /// **'Connected'**
  String get githubConnected;

  /// No description provided for @githubNotConnected.
  ///
  /// In en, this message translates to:
  /// **'Not connected'**
  String get githubNotConnected;

  /// No description provided for @githubConnectedAs.
  ///
  /// In en, this message translates to:
  /// **'Connected as {login}'**
  String githubConnectedAs(String login);

  /// No description provided for @githubDescription.
  ///
  /// In en, this message translates to:
  /// **'Used by the bundled gh and by Git over HTTPS. The token is encrypted on this device and is never shown again.'**
  String get githubDescription;

  /// No description provided for @githubTestConnection.
  ///
  /// In en, this message translates to:
  /// **'Test connection'**
  String get githubTestConnection;

  /// No description provided for @githubDisconnect.
  ///
  /// In en, this message translates to:
  /// **'Disconnect'**
  String get githubDisconnect;

  /// No description provided for @githubConnectAction.
  ///
  /// In en, this message translates to:
  /// **'Connect GitHub'**
  String get githubConnectAction;

  /// No description provided for @githubConnect.
  ///
  /// In en, this message translates to:
  /// **'Connect'**
  String get githubConnect;

  /// No description provided for @githubTokenLabel.
  ///
  /// In en, this message translates to:
  /// **'Personal access token'**
  String get githubTokenLabel;

  /// No description provided for @githubTokenHint.
  ///
  /// In en, this message translates to:
  /// **'Paste a GitHub personal access token with the scopes you need (repo for private repositories).'**
  String get githubTokenHint;

  /// No description provided for @githubTokenRejected.
  ///
  /// In en, this message translates to:
  /// **'GitHub did not accept this token.'**
  String get githubTokenRejected;

  /// No description provided for @githubAuthWorking.
  ///
  /// In en, this message translates to:
  /// **'GitHub authentication is working.'**
  String get githubAuthWorking;

  /// No description provided for @githubAuthNotWorking.
  ///
  /// In en, this message translates to:
  /// **'GitHub authentication is not working.'**
  String get githubAuthNotWorking;

  /// No description provided for @githubAuthenticatedAs.
  ///
  /// In en, this message translates to:
  /// **'Authenticated as {login}.'**
  String githubAuthenticatedAs(String login);

  /// No description provided for @githubCredentialRemoveFailed.
  ///
  /// In en, this message translates to:
  /// **'Could not remove the credential.'**
  String get githubCredentialRemoveFailed;

  /// No description provided for @githubYourAccount.
  ///
  /// In en, this message translates to:
  /// **'your GitHub account'**
  String get githubYourAccount;

  /// No description provided for @githubConnectedReport.
  ///
  /// In en, this message translates to:
  /// **'Connected as {who}. {outcome}'**
  String githubConnectedReport(String who, String outcome);

  /// No description provided for @githubDisconnectedReport.
  ///
  /// In en, this message translates to:
  /// **'Disconnected. {outcome}'**
  String githubDisconnectedReport(String outcome);

  /// No description provided for @credentialAppliedNow.
  ///
  /// In en, this message translates to:
  /// **'gh and git can use it now.'**
  String get credentialAppliedNow;

  /// No description provided for @credentialAppliesNextStart.
  ///
  /// In en, this message translates to:
  /// **'It will be used the next time PocketClaw starts.'**
  String get credentialAppliesNextStart;

  /// No description provided for @credentialAppliesDeferred.
  ///
  /// In en, this message translates to:
  /// **'Saved. PocketClaw is busy starting, so it will apply automatically as soon as that finishes.'**
  String get credentialAppliesDeferred;

  /// No description provided for @whatsNewTitle.
  ///
  /// In en, this message translates to:
  /// **'What\'s New'**
  String get whatsNewTitle;

  /// No description provided for @whatsNewDescription.
  ///
  /// In en, this message translates to:
  /// **'The main changes in this release.'**
  String get whatsNewDescription;

  /// No description provided for @whatsNewBadge.
  ///
  /// In en, this message translates to:
  /// **'NEW'**
  String get whatsNewBadge;

  /// No description provided for @whatsNewSectionNew.
  ///
  /// In en, this message translates to:
  /// **'New'**
  String get whatsNewSectionNew;

  /// No description provided for @whatsNewSectionImprovements.
  ///
  /// In en, this message translates to:
  /// **'Improvements'**
  String get whatsNewSectionImprovements;

  /// No description provided for @whatsNewSectionFixes.
  ///
  /// In en, this message translates to:
  /// **'Fixes'**
  String get whatsNewSectionFixes;

  /// No description provided for @whatsNew020New1.
  ///
  /// In en, this message translates to:
  /// **'Managed Runtime installs and keeps PocketClaw\'s bundled tools up to date for you.'**
  String get whatsNew020New1;

  /// No description provided for @whatsNew020New2.
  ///
  /// In en, this message translates to:
  /// **'Bundled developer tools: Git, GitHub CLI, curl, ripgrep, jq and SQLite.'**
  String get whatsNew020New2;

  /// No description provided for @whatsNew020New3.
  ///
  /// In en, this message translates to:
  /// **'Support for PocketClaw\'s bundled Python 3.14 runtime.'**
  String get whatsNew020New3;

  /// No description provided for @whatsNew020New4.
  ///
  /// In en, this message translates to:
  /// **'Secure GitHub sign-in, shared by the bundled Git and GitHub CLI.'**
  String get whatsNew020New4;

  /// No description provided for @whatsNew020New5.
  ///
  /// In en, this message translates to:
  /// **'Telegram integration, set up from Settings.'**
  String get whatsNew020New5;

  /// No description provided for @whatsNew020Improvement1.
  ///
  /// In en, this message translates to:
  /// **'More resilient providers: a failing request no longer ends the turn.'**
  String get whatsNew020Improvement1;

  /// No description provided for @whatsNew020Improvement2.
  ///
  /// In en, this message translates to:
  /// **'A more accurate Managed Runtime catalog.'**
  String get whatsNew020Improvement2;

  /// No description provided for @whatsNew020Improvement3.
  ///
  /// In en, this message translates to:
  /// **'A cleaner PocketClaw identity across the web interface and the default workspace.'**
  String get whatsNew020Improvement3;

  /// No description provided for @whatsNew020Fix1.
  ///
  /// In en, this message translates to:
  /// **'The Gateway now recovers from a stale process record left behind by an earlier run.'**
  String get whatsNew020Fix1;

  /// No description provided for @whatsNew020Fix2.
  ///
  /// In en, this message translates to:
  /// **'Channel lists with more than one entry are preserved correctly when settings are saved.'**
  String get whatsNew020Fix2;

  /// Settings section label above the address, port and public mode controls
  ///
  /// In en, this message translates to:
  /// **'Connection'**
  String get settingsGroupConnection;

  /// Settings section label above the models and context memory controls
  ///
  /// In en, this message translates to:
  /// **'Agent'**
  String get settingsGroupAgent;

  /// Settings section label above the Telegram and GitHub controls
  ///
  /// In en, this message translates to:
  /// **'Integrations'**
  String get settingsGroupIntegrations;

  /// Settings section label above the theme controls
  ///
  /// In en, this message translates to:
  /// **'Appearance'**
  String get settingsGroupAppearance;

  /// Title of the Status screen
  ///
  /// In en, this message translates to:
  /// **'Status'**
  String get statusTitle;

  /// No description provided for @statusSectionSystem.
  ///
  /// In en, this message translates to:
  /// **'System'**
  String get statusSectionSystem;

  /// No description provided for @statusSectionAi.
  ///
  /// In en, this message translates to:
  /// **'AI'**
  String get statusSectionAi;

  /// No description provided for @statusSectionActivity.
  ///
  /// In en, this message translates to:
  /// **'Activity'**
  String get statusSectionActivity;

  /// No description provided for @statusSectionChannels.
  ///
  /// In en, this message translates to:
  /// **'Channels'**
  String get statusSectionChannels;

  /// No description provided for @statusSectionResources.
  ///
  /// In en, this message translates to:
  /// **'Resources'**
  String get statusSectionResources;

  /// Caption clarifying that the activity counters reset when the Gateway restarts
  ///
  /// In en, this message translates to:
  /// **'Since Gateway start'**
  String get statusSinceGatewayStart;

  /// No description provided for @statusGateway.
  ///
  /// In en, this message translates to:
  /// **'Gateway'**
  String get statusGateway;

  /// No description provided for @statusUptime.
  ///
  /// In en, this message translates to:
  /// **'Uptime'**
  String get statusUptime;

  /// No description provided for @statusAppVersion.
  ///
  /// In en, this message translates to:
  /// **'App version'**
  String get statusAppVersion;

  /// No description provided for @statusCoreVersion.
  ///
  /// In en, this message translates to:
  /// **'Core version'**
  String get statusCoreVersion;

  /// No description provided for @statusActiveModel.
  ///
  /// In en, this message translates to:
  /// **'Active model'**
  String get statusActiveModel;

  /// No description provided for @statusConfiguredDefault.
  ///
  /// In en, this message translates to:
  /// **'Configured default'**
  String get statusConfiguredDefault;

  /// No description provided for @statusProvider.
  ///
  /// In en, this message translates to:
  /// **'Provider'**
  String get statusProvider;

  /// No description provided for @statusFallbacks.
  ///
  /// In en, this message translates to:
  /// **'Fallbacks'**
  String get statusFallbacks;

  /// No description provided for @statusActiveTurns.
  ///
  /// In en, this message translates to:
  /// **'Active turns'**
  String get statusActiveTurns;

  /// No description provided for @statusActiveSubagents.
  ///
  /// In en, this message translates to:
  /// **'Active subagents'**
  String get statusActiveSubagents;

  /// No description provided for @statusWaiting.
  ///
  /// In en, this message translates to:
  /// **'Waiting'**
  String get statusWaiting;

  /// No description provided for @statusCompleted.
  ///
  /// In en, this message translates to:
  /// **'Completed'**
  String get statusCompleted;

  /// No description provided for @statusFailed.
  ///
  /// In en, this message translates to:
  /// **'Failed'**
  String get statusFailed;

  /// No description provided for @statusCancelled.
  ///
  /// In en, this message translates to:
  /// **'Cancelled'**
  String get statusCancelled;

  /// No description provided for @statusToolCalls.
  ///
  /// In en, this message translates to:
  /// **'Tool calls'**
  String get statusToolCalls;

  /// No description provided for @statusToolCallsFailed.
  ///
  /// In en, this message translates to:
  /// **'Tool calls failed'**
  String get statusToolCallsFailed;

  /// No description provided for @statusLastActivity.
  ///
  /// In en, this message translates to:
  /// **'Last activity'**
  String get statusLastActivity;

  /// No description provided for @statusCoreMemory.
  ///
  /// In en, this message translates to:
  /// **'Core memory'**
  String get statusCoreMemory;

  /// No description provided for @statusCoreCpuTime.
  ///
  /// In en, this message translates to:
  /// **'Core CPU time'**
  String get statusCoreCpuTime;

  /// Channel state shown when Start returned an error
  ///
  /// In en, this message translates to:
  /// **'Failed to start'**
  String get statusChannelFailedToStart;

  /// No description provided for @statusNoChannels.
  ///
  /// In en, this message translates to:
  /// **'No channels configured'**
  String get statusNoChannels;

  /// Shown when the detailed Status payload could not be obtained
  ///
  /// In en, this message translates to:
  /// **'Detailed status is unavailable'**
  String get statusDetailUnavailable;

  /// No description provided for @statusJustNow.
  ///
  /// In en, this message translates to:
  /// **'Just now'**
  String get statusJustNow;

  /// Relative time since the last activity
  ///
  /// In en, this message translates to:
  /// **'{minutes}m ago'**
  String statusMinutesAgo(int minutes);

  /// Relative time since the last activity
  ///
  /// In en, this message translates to:
  /// **'{hours}h ago'**
  String statusHoursAgo(int hours);

  /// Relative time since the last activity
  ///
  /// In en, this message translates to:
  /// **'{days}d ago'**
  String statusDaysAgo(int days);
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) => <String>[
    'ar',
    'de',
    'en',
    'es',
    'fr',
    'hi',
    'id',
    'ja',
    'ko',
    'pt',
    'ru',
    'zh',
  ].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'ar':
      return AppLocalizationsAr();
    case 'de':
      return AppLocalizationsDe();
    case 'en':
      return AppLocalizationsEn();
    case 'es':
      return AppLocalizationsEs();
    case 'fr':
      return AppLocalizationsFr();
    case 'hi':
      return AppLocalizationsHi();
    case 'id':
      return AppLocalizationsId();
    case 'ja':
      return AppLocalizationsJa();
    case 'ko':
      return AppLocalizationsKo();
    case 'pt':
      return AppLocalizationsPt();
    case 'ru':
      return AppLocalizationsRu();
    case 'zh':
      return AppLocalizationsZh();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}

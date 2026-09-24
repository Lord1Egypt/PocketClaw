import 'dart:async';
import 'dart:io';
import 'package:flutter/widgets.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../generated/l10n/app_localizations.dart';
import 'app_theme.dart';
import 'pocketclaw_channel.dart';
import 'public_mode_reconciliation.dart';
import 'plain_text_log_sanitizer.dart';
import 'status_snapshot.dart';
import '../native/core_service_adapter_factory.dart';
import '../native/core_service_adapter.dart';

enum ServiceStatus { stopped, running, starting }

/// What happened to a credential change that Core only reads when it launches.
enum CredentialApplyOutcome {
  /// Core was restarted, so the change is live.
  applied,

  /// Core was not running. The change is stored and will be read at next start.
  notRunning,

  /// Core was mid-transition. The restart is queued and runs as soon as the
  /// service settles, without interrupting whatever it is doing.
  deferred,
}

/// Why an app-launch auto-start evaluation did or did not start the service.
enum LaunchAutoStartDecision {
  start,
  alreadyEvaluated,
  preferenceOff,
  alreadyActive,
}

@immutable
class LanAddressCandidate {
  const LanAddressCandidate({
    required this.interfaceName,
    required this.address,
  });

  final String interfaceName;
  final String address;
}

class ServiceManager extends ChangeNotifier with WidgetsBindingObserver {
  static final ServiceManager _instance = ServiceManager._internal();
  factory ServiceManager() => _instance;
  ServiceManager._internal();

  final CoreServiceAdapter _adapter = CoreServiceAdapterFactory.create();
  String? _lastErrorCode;
  String _cachedAppVersion = 'unknown';
  String _cachedCoreVersion = '';

  String? get lastErrorCode => _lastErrorCode ?? _adapter.getLastErrorCode();

  ServiceStatus _status = ServiceStatus.stopped;
  bool _pendingCredentialRestart = false;
  final List<String> _logs = [];

  ServiceStatus get status => _status;
  List<String> get logs => List.unmodifiable(_logs);

  /// The launcher's loopback port. Not a setting: the Android host always
  /// starts the launcher with an explicit `-port 18800`, and the host's own
  /// bridge calls to the launcher are pinned to it.
  static const int dashboardPort = 18800;
  bool _publicMode = false;
  bool _isApplyingPublicMode = false;
  String? _publicModeApplyError;
  String? _lanAddress;
  String _workspacePath = '';

  int _nativePid = -1;
  String _healthStatus = '';
  String _healthUptime = '';

  // Status detail is opt-in. The base health poll below is shared by Start/Stop
  // and must keep running whatever screen is visible, so what is gated is the
  // extra work inside it, not the timer itself. When Status is not on screen
  // this stays false and the poll costs exactly what it always did.
  bool _statusDetailWanted = false;
  StatusSnapshot? _statusSnapshot;
  LaunchAutoStartPreferences _launchAutoStart =
      LaunchAutoStartPreferences.defaults;
  bool _launchAutoStartEvaluated = false;

  int get nativePid => _nativePid;
  String get healthStatus => _healthStatus;
  String get healthUptime => _healthUptime;

  /// The latest Status snapshot, or null when detail has not been requested,
  /// the gateway is stopped, or the host could not obtain it. Null means
  /// unavailable and the screen says so rather than showing zeroes.
  StatusSnapshot? get statusSnapshot => _statusSnapshot;

  /// Whether the Status screen is asking for the detailed payload.
  bool get statusDetailWanted => _statusDetailWanted;

  /// Turns the detailed Status payload on or off.
  ///
  /// Called when the Status tab is shown and hidden. Turning it off drops the
  /// retained snapshot so a stale reading can never be shown as current on the
  /// next visit.
  void setStatusDetailWanted(bool wanted) {
    if (_statusDetailWanted == wanted) return;
    _statusDetailWanted = wanted;
    if (!wanted) {
      _statusSnapshot = null;
    }
    notifyListeners();
  }

  /// Cached mirror of the Android host's canonical launch auto-start record.
  /// The host remains the only source of truth; this is refreshed from every
  /// read and every committed write.
  bool get serviceLaunchAutoStart => _launchAutoStart.serviceEnabled;
  bool get gatewayLaunchAutoStart => _launchAutoStart.gatewayEnabled;

  Timer? _nativePollingTimer;
  Timer? _lanAddressPollingTimer;
  int _lanAddressRefreshGeneration = 0;

  AppThemeMode _currentThemeMode = AppThemeMode.carbon;
  AppThemeMode get currentThemeMode => _currentThemeMode;
  Locale _currentLocale = const Locale('en');
  Locale get currentLocale => _currentLocale;
  String get _host => _publicMode ? '0.0.0.0' : '127.0.0.1';
  String get webUrl => 'http://$_host:$dashboardPort';
  String get localDashboardUrl => 'http://127.0.0.1:$dashboardPort';
  String? get lanAddress => _lanAddress;
  String? get publicDashboardUrl =>
      _lanAddress == null ? null : 'http://${_lanAddress!}:$dashboardPort';
  String? get connectableDashboardUrl =>
      _publicMode ? publicDashboardUrl : webUrl;
  bool get publicMode => _publicMode;
  bool get isApplyingPublicMode => _isApplyingPublicMode;
  String? get publicModeApplyError => _publicModeApplyError;
  String get workspacePath => _workspacePath;

  Future<String?> getDeviceIpAddress() async {
    try {
      if (Platform.isAndroid) {
        return await PocketClawChannel.getLanIpv4Address();
      }
      final interfaces = await NetworkInterface.list(
        type: InternetAddressType.IPv4,
        includeLinkLocal: false,
      );

      return selectUsableLanIpv4(
        interfaces.expand(
          (interface) => interface.addresses.map(
            (address) => LanAddressCandidate(
              interfaceName: interface.name,
              address: address.address,
            ),
          ),
        ),
      );
    } catch (e) {
      debugPrint('Failed to get device IP: $e');
      return null;
    }
  }

  @visibleForTesting
  static String? selectUsableLanIpv4(Iterable<LanAddressCandidate> candidates) {
    final usable = <({LanAddressCandidate candidate, int score})>[];
    for (final candidate in candidates) {
      final parsed = InternetAddress.tryParse(candidate.address.trim());
      if (parsed == null ||
          parsed.type != InternetAddressType.IPv4 ||
          parsed.isLoopback ||
          parsed.isLinkLocal ||
          parsed.isMulticast ||
          parsed.address == '0.0.0.0' ||
          parsed.address == '255.255.255.255') {
        continue;
      }

      final name = candidate.interfaceName.toLowerCase();
      if (name == 'lo' || name.startsWith('loopback')) continue;
      final isWifi =
          name.contains('wifi') ||
          name.contains('wlan') ||
          name.startsWith('wl');
      final isEthernet =
          name.contains('ethernet') ||
          name.startsWith('eth') ||
          name.startsWith('en');
      final isCellular =
          name.startsWith('rmnet') ||
          name.startsWith('wwan') ||
          name.startsWith('pdp_ip') ||
          name.startsWith('ccmni');
      if (isCellular) continue;
      final networkScore = (isWifi || isEthernet) ? 0 : 1;
      final addressScore = _isPrivateLanIpv4(parsed.address) ? 0 : 1;
      usable.add((
        candidate: candidate,
        score: addressScore * 2 + networkScore,
      ));
    }
    usable.sort((a, b) => a.score.compareTo(b.score));
    return usable.isEmpty ? null : usable.first.candidate.address.trim();
  }

  static bool _isPrivateLanIpv4(String address) {
    final parts = address.split('.').map(int.tryParse).toList();
    if (parts.length != 4 || parts.any((part) => part == null)) return false;
    final first = parts[0]!;
    final second = parts[1]!;
    return first == 10 ||
        (first == 172 && second >= 16 && second <= 31) ||
        (first == 192 && second == 168);
  }

  Future<void> refreshLanAddress() async {
    final generation = ++_lanAddressRefreshGeneration;
    final nextAddress = _publicMode ? await getDeviceIpAddress() : null;
    if (generation != _lanAddressRefreshGeneration) return;
    if (_lanAddress == nextAddress) return;
    _lanAddress = nextAddress;
    notifyListeners();
  }

  void _syncLanAddressPolling() {
    _lanAddressPollingTimer?.cancel();
    _lanAddressPollingTimer = null;
    if (!_publicMode) {
      _lanAddressRefreshGeneration++;
      if (_lanAddress != null) {
        _lanAddress = null;
      }
      return;
    }
    unawaited(refreshLanAddress());
    _lanAddressPollingTimer = Timer.periodic(
      const Duration(seconds: 3),
      (_) => refreshLanAddress(),
    );
  }

  @visibleForTesting
  void setLanAddressForTest(String? address) {
    _lanAddressRefreshGeneration++;
    _lanAddress = address;
    notifyListeners();
  }

  Future<void> init() async {
    WidgetsBinding.instance.addObserver(this);
    final prefs = await SharedPreferences.getInstance();
    _publicMode = prefs.getBool('publicMode') ?? false;

    final themeIndex = prefs.getInt('theme_mode') ?? 0;
    _currentThemeMode = AppThemeMode.values[themeIndex];

    final savedLocale = prefs.getString('locale');
    if (savedLocale != null) {
      _currentLocale = Locale(savedLocale);
    } else {
      // No saved preference, detect system language
      final systemLocale = Platform.localeName; // e.g. "zh_CN", "en_US"
      final systemLangCode = systemLocale.split('_').first.split('-').first;
      // Only use system language if it's supported
      final supportedCodes = AppLocalizations.supportedLocales
          .map((l) => l.languageCode)
          .toSet();
      _currentLocale = supportedCodes.contains(systemLangCode)
          ? Locale(systemLangCode)
          : const Locale('en');
    }

    if (Platform.isAndroid) {
      try {
        _workspacePath = await _adapter.getWorkspacePath();
        await _syncNativeServiceStatus();
        // Read last: the launch auto-start decision needs an accurate runtime
        // status more than it needs the preference, and this call must not be
        // able to skip the status sync above.
        _launchAutoStart =
            await PocketClawChannel.getLaunchAutoStartPreferences();
      } catch (_) {}
      _startNativePolling();
    }
    _syncLanAddressPolling();

    _cachedAppVersion = await _readAppVersion();

    notifyListeners();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    switch (state) {
      case AppLifecycleState.resumed:
        if (_publicMode) unawaited(refreshLanAddress());
        break;
      case AppLifecycleState.inactive:
      case AppLifecycleState.hidden:
      case AppLifecycleState.paused:
      case AppLifecycleState.detached:
        break;
    }
  }

  Future<String> getAppVersion() async {
    if (_cachedAppVersion == 'unknown') {
      _cachedAppVersion = await _readAppVersion();
    }
    return _cachedAppVersion;
  }

  /// Reads the Core runtime version, or null when it could not be read.
  ///
  /// PC-DEF-063. A failed probe is never cached. It used to be: the adapter
  /// answered 'unknown' for a failure, that string passed the non-empty test,
  /// and it became the displayed Core version until something re-probed
  /// successfully. Leaving the cache empty instead means the next read retries.
  Future<String?> getCoreVersion() async {
    final version = await _adapter.getCoreVersion();
    if (version == null || version.isEmpty) return null;
    if (version != _cachedCoreVersion) {
      _cachedCoreVersion = version;
      notifyListeners();
    }
    return version;
  }

  /// App version for display, or an empty string until it has been read.
  String get appVersion =>
      _cachedAppVersion == 'unknown' ? '' : _cachedAppVersion;

  /// Core version for display, read once and cached.
  ///
  /// Reading it means invoking the Core binary, so it is fetched on demand
  /// rather than on every poll: a version does not change while the process
  /// runs.
  /// The cached Core version, or an empty string while it is still unread.
  ///
  /// Empty means "not known yet", never "failed": a failed probe leaves the
  /// cache empty so the next read of this getter tries again.
  String get coreVersionLabel {
    if (_cachedCoreVersion.isEmpty) {
      unawaited(getCoreVersion());
    }
    return _cachedCoreVersion;
  }

  Future<String> _readAppVersion() async {
    try {
      final info = await PackageInfo.fromPlatform();
      return info.version;
    } catch (_) {
      return 'unknown';
    }
  }

  /// Runs one native status/log poll.
  ///
  /// Exposed because the polling timer only starts on Android, so a host test
  /// can never reach this path through [_startNativePolling] — and this path
  /// is exactly where a single backend line was being re-appended forever.
  @visibleForTesting
  Future<void> pollNativeServiceStatusForTest() => _syncNativeServiceStatus();

  Future<void> _syncNativeServiceStatus() async {
    try {
      final status = await PocketClawChannel.getServiceStatus();
      final isRunning = status['isRunning'] as bool? ?? false;
      _nativePid = status['pid'] as int? ?? -1;

      final oldStatus = _status;
      _status = isRunning ? ServiceStatus.running : ServiceStatus.stopped;

      // A credential change that arrived mid-transition is applied here, once
      // the service has settled. This is the existing poll, not a new watchdog:
      // the restart waits for a state the app already tracks rather than for a
      // timer, so it can never land on top of a start that is still in flight.
      if (_pendingCredentialRestart && _status != ServiceStatus.starting) {
        _pendingCredentialRestart = false;
        unawaited(applyCredentialChange());
      }

      // Drain the lines emitted since the last poll. `status['lastLog']` is a
      // sticky snapshot of the most recent line, so appending it here re-added
      // the same entry every three seconds until it filled the Logs screen and
      // evicted the real history.
      for (final line in await PocketClawChannel.takeNewLogs()) {
        _addLog(line);
      }

      final hadSnapshot = _statusSnapshot != null;
      if (isRunning) {
        try {
          final health = await PocketClawChannel.checkHealth(
            detail: _statusDetailWanted,
          );
          final isHealthy = health['isHealthy'] as bool? ?? false;
          _healthStatus = isHealthy ? 'Healthy' : 'Starting...';
          _healthUptime = health['uptime'] as String? ?? '';
          if (health['pid'] != null && (health['pid'] as int) > 0) {
            _nativePid = health['pid'] as int;
          }
          _statusSnapshot = _statusDetailWanted
              ? StatusSnapshot.tryParse(health['detail'] as String?)
              : null;
        } catch (_) {
          _healthStatus = 'Starting...';
          _statusSnapshot = null;
        }
      } else {
        _healthStatus = '';
        _healthUptime = '';
        _statusSnapshot = null;
      }

      // The Status screen reads a fresh snapshot every poll, so it has to be
      // told about a new one even when the service status itself is unchanged.
      if (oldStatus != _status ||
          (_statusDetailWanted && (_statusSnapshot != null || hadSnapshot))) {
        notifyListeners();
      }
    } catch (e) {
      debugPrint('Failed to sync Android service status: $e');
    }
  }

  void _startNativePolling() {
    _nativePollingTimer?.cancel();
    _nativePollingTimer = Timer.periodic(
      const Duration(seconds: 3),
      (_) => _syncNativeServiceStatus(),
    );
  }

  Future<void> setServiceLaunchAutoStart(bool enabled) =>
      _commitLaunchAutoStart(serviceEnabled: enabled);

  /// True once this process has already reconciled, so a repeat is a no-op.
  bool _publicModeReconciled = false;

  /// Re-applies Public Mode once the Dashboard has an owner. PC-DEF-040.
  ///
  /// Called at one host lifecycle transition -- leaving the setup page -- and
  /// never polled. PC-DEF-039 keeps an unclaimed dashboard on loopback, so
  /// without this a user with Public Mode on finishes setup and stays private
  /// with nothing in the UI explaining why.
  ///
  /// The privileged rebind stays native: this asks the host, and the Android
  /// bridge token never enters Dart or the WebView.
  Future<PublicModeReconciliation> reconcilePublicModeAfterSetup() async {
    if (!Platform.isAndroid) return PublicModeReconciliation.notRequested;
    try {
      final decision = resolvePublicModeReconciliation(
        dashboardInitialized:
            await PocketClawChannel.dashboardAuthInitialized(),
        desiredPublic: _publicMode,
        alreadyPublic: _publicModeReconciled,
      );
      if (decision == PublicModeReconciliation.reapply) {
        _publicModeReconciled = true;
        await PocketClawChannel.applyPublicMode(true);
        _addLog('Public Mode applied now that the Dashboard has a password');
        notifyListeners();
      }
      return decision;
    } catch (e) {
      // Never fatal. The preference is intact and the next service start
      // resolves exposure from it.
      debugPrint('Public Mode reconciliation failed: $e');
      return PublicModeReconciliation.dashboardNotInitialized;
    }
  }

  Future<void> setGatewayLaunchAutoStart(bool enabled) async {
    final wasEnabled = _launchAutoStart.gatewayEnabled;
    await _commitLaunchAutoStart(gatewayEnabled: enabled);

    // PC-DEF-034. Enabling this while the service is already running has to
    // start the Gateway now. Persisting and waiting for the next service start
    // is what made the preference look broken: the user turned it on, nothing
    // happened, and the only way forward was an undocumented manual start.
    //
    // Only on a real OFF -> ON transition, only when the host actually
    // persisted the change, and only when there is a running service to ask.
    if (!enabled || wasEnabled || !_launchAutoStart.gatewayEnabled) return;
    if (_status != ServiceStatus.running) return;

    try {
      final result = await PocketClawChannel.startGatewayNow();
      if (result == 'already_running') {
        _addLog('Gateway is already running');
      } else {
        _addLog('Gateway started');
      }
    } catch (e) {
      // The preference stays on: it was persisted before this ran, and the
      // next service start still honours it. Report the real failure rather
      // than reverting a choice the user made.
      _addLog(
        'Could not start the Gateway now; it will start with the service',
      );
      debugPrint('Immediate gateway start failed: $e');
    }
    notifyListeners();
  }

  Future<void> _commitLaunchAutoStart({
    bool? serviceEnabled,
    bool? gatewayEnabled,
  }) async {
    if (!Platform.isAndroid) return;
    try {
      // Adopt the host's post-commit readback, never the requested value, so
      // the switch can only settle on state that actually reached the disk.
      _launchAutoStart = await PocketClawChannel.setLaunchAutoStartPreferences(
        serviceEnabled: serviceEnabled,
        gatewayEnabled: gatewayEnabled,
      );
    } catch (e) {
      // The host refused to persist. Fall back to what is actually stored so
      // the switch reports the truth rather than an unsaved change.
      _addLog('Could not save the auto-start preference');
      debugPrint('Failed to persist launch auto-start preferences: $e');
      try {
        _launchAutoStart =
            await PocketClawChannel.getLaunchAutoStartPreferences();
      } catch (_) {}
    }
    notifyListeners();
  }

  /// Decides whether a true app-process launch should start the service.
  ///
  /// Pure so that the auto-start contract can be asserted without a device: it
  /// fires at most once per process, only when the user asked for it, and only
  /// when nothing is running.
  @visibleForTesting
  static LaunchAutoStartDecision decideLaunchAutoStart({
    required bool alreadyEvaluated,
    required bool preferenceEnabled,
    required ServiceStatus status,
  }) {
    if (alreadyEvaluated) return LaunchAutoStartDecision.alreadyEvaluated;
    if (!preferenceEnabled) return LaunchAutoStartDecision.preferenceOff;
    if (status != ServiceStatus.stopped) {
      return LaunchAutoStartDecision.alreadyActive;
    }
    return LaunchAutoStartDecision.start;
  }

  /// Starts the service once, at a true app-process launch, if the user asked
  /// for it and nothing is running yet.
  ///
  /// This deliberately has no resume, watchdog, or retry behaviour: auto-start
  /// means "start on the next legitimate app launch", so a service the user
  /// stopped by hand stays stopped until they start it again or relaunch the
  /// app. Gateway auto-start is not handled here at all — the preference
  /// travels to Core in the service environment and Core decides.
  Future<void> evaluateLaunchAutoStart() async {
    if (!Platform.isAndroid) return;
    final decision = decideLaunchAutoStart(
      alreadyEvaluated: _launchAutoStartEvaluated,
      preferenceEnabled: _launchAutoStart.serviceEnabled,
      status: _status,
    );
    _launchAutoStartEvaluated = true;
    switch (decision) {
      case LaunchAutoStartDecision.alreadyEvaluated:
        return;
      case LaunchAutoStartDecision.preferenceOff:
        _addLog('Auto-start skipped: service auto-start is off');
        return;
      case LaunchAutoStartDecision.alreadyActive:
        _addLog('Auto-start skipped: service is already ${_status.name}');
        return;
      case LaunchAutoStartDecision.start:
        _addLog('Auto-starting PocketClaw service on app launch');
        await start();
    }
  }

  Future<void> setTheme(AppThemeMode mode) async {
    _currentThemeMode = mode;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt('theme_mode', mode.index);
    notifyListeners();
  }

  Future<void> setLocale(Locale locale) async {
    _currentLocale = locale;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('locale', locale.languageCode);
    notifyListeners();
  }

  /// Records the network mode Core is started with. The host and port follow
  /// from it; neither is a setting of its own on Android.
  Future<void> setPublicModeConfig(bool publicMode) async {
    _publicMode = publicMode;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool('publicMode', _publicMode);
    _syncLanAddressPolling();
    notifyListeners();
  }

  /// Applies Public Mode transactionally. On a running Android service this
  /// asks the launcher to replace only port 18800's listeners and persists the
  /// setting only after the new bind succeeds. A failed bind reports the
  /// listener mode restored by the launcher.
  Future<bool> applyPublicMode(bool value) async {
    if (_isApplyingPublicMode) return false;
    _publicModeApplyError = null;

    if (!Platform.isAndroid || _status != ServiceStatus.running) {
      await setPublicModeConfig(value);
      return true;
    }

    _isApplyingPublicMode = true;
    notifyListeners();
    try {
      final result = await PocketClawChannel.applyPublicMode(value);
      await setPublicModeConfig(result.publicMode);
      if (!result.success || result.publicMode != value) {
        _publicModeApplyError = result.message.isNotEmpty
            ? result.message
            : (value
                  ? 'Could not enable LAN access. PocketClaw remains available locally.'
                  : 'Could not disable LAN access. PocketClaw remains in its previous network mode.');
        return false;
      }
      await refreshLanAddress();
      return true;
    } catch (_) {
      _publicModeApplyError = value
          ? 'Could not enable LAN access. PocketClaw remains available locally.'
          : 'Could not disable LAN access. PocketClaw remains in its previous network mode.';
      return false;
    } finally {
      _isApplyingPublicMode = false;
      notifyListeners();
    }
  }

  Timer? _notifyTimer;
  final List<String> _pendingLogs = [];

  void _addLog(String log) {
    final sanitized = PlainTextLogSanitizer.sanitize(log);
    if (sanitized.isEmpty) return;

    final lines = sanitized.split('\n').where((line) => line.isNotEmpty);
    _pendingLogs.addAll(lines);

    if (_notifyTimer == null || !_notifyTimer!.isActive) {
      _notifyTimer = Timer(const Duration(milliseconds: 100), () {
        if (_pendingLogs.isNotEmpty) {
          _logs.addAll(_pendingLogs);
          _pendingLogs.clear();

          if (_logs.length > 500) {
            _logs.removeRange(0, _logs.length - 500);
          }
          notifyListeners();
        }
      });
    }
  }

  /// True while a credential change is waiting for a safe moment to restart.
  bool get hasPendingCredentialRestart => _pendingCredentialRestart;

  /// Restarts Core so a credential it reads only at launch takes effect.
  ///
  /// The GitHub token is decrypted by the Android service when it builds Core's
  /// environment, so a change to it is inert until the process restarts. This
  /// reuses the same stop/start the config screen already performs when settings
  /// change; it adds no lifecycle machinery of its own.
  ///
  /// A service that is mid-transition is never interrupted. The change is queued
  /// and applied from the status poll once the service settles, which is why
  /// this can report [CredentialApplyOutcome.deferred] rather than blocking or
  /// forcing a stop.
  Future<CredentialApplyOutcome> applyCredentialChange() async {
    if (_status == ServiceStatus.starting) {
      _pendingCredentialRestart = true;
      notifyListeners();
      return CredentialApplyOutcome.deferred;
    }
    if (_status == ServiceStatus.stopped) {
      return CredentialApplyOutcome.notRunning;
    }
    // One intent, not stop-then-start. The outcome vocabulary is unchanged:
    // "applied" has always meant the restart was performed, and whether Core
    // came back up is reported by the status poll, not by this call.
    await restartCore();
    return CredentialApplyOutcome.applied;
  }

  /// The launch arguments Core is started with.
  ///
  /// Shared by start and restart so a restarted Core cannot come up with a
  /// different network mode than a started one.
  ///
  /// Derived from Public Mode alone. A free-form arguments field used to be
  /// appended here, and the Android host reads this string only to look for
  /// `-public` — so typing that token enabled LAN exposure without going
  /// through the Public Mode toggle that PC-DEF-020 made the one authority.
  String _launchArguments() =>
      _publicMode ? '-public -no-browser' : '-no-browser';

  /// Restarts Core so configuration it reads only at launch takes effect.
  ///
  /// PC-DEF-030. Callers used to write `await stop(); await start();`, which on
  /// Android is two service intents and an unconditional stopSelf() between
  /// them: the start is honoured and then destroyed, leaving PocketClaw stopped
  /// with no indication that anything went wrong. Telegram onboarding is the
  /// flow that made it visible — the bot stayed silent until the owner started
  /// the Service and the Gateway by hand.
  ///
  /// The platform is asked to restart instead. A service that is mid-transition
  /// is still never interrupted: `starting` defers, exactly as
  /// [applyCredentialChange] already does, and `stopped` is not a restart.
  Future<bool> restartCore() async {
    if (_status != ServiceStatus.running) return false;

    _status = ServiceStatus.starting;
    notifyListeners();

    try {
      final ok = await _adapter.restartService(
        port: dashboardPort,
        args: _launchArguments(),
      );
      if (!ok) {
        _status = ServiceStatus.stopped;
        final code = _adapter.getLastErrorCode();
        _addLog('Failed to restart service: ${code ?? 'unknown'}');
        notifyListeners();
        return false;
      }

      if (Platform.isAndroid) {
        _addLog('Restarting PocketClaw service...');
        // Same deferral the start path uses: the host reports the settled
        // state, this side does not guess at it.
        Future.delayed(const Duration(seconds: 2), () {
          _syncNativeServiceStatus();
        });
      } else {
        _status = ServiceStatus.running;
        _addLog('Service restarted on $webUrl');
      }
      notifyListeners();
      return true;
    } catch (e) {
      _status = ServiceStatus.stopped;
      _addLog('Failed to restart service: $e');
      notifyListeners();
      return false;
    }
  }

  Future<void> start() async {
    if (_status != ServiceStatus.stopped) return;

    _status = ServiceStatus.starting;
    notifyListeners();

    final String launchArgs = _launchArguments();
    try {
      final ok = await _adapter.startService(
        port: dashboardPort,
        args: launchArgs,
      );

      if (ok) {
        // The native service reports readiness; its status is read back
        // rather than assumed.
        _addLog('Starting PocketClaw service...');
        Future.delayed(const Duration(seconds: 2), () {
          _syncNativeServiceStatus();
        });
      } else {
        _status = ServiceStatus.stopped;
        final code = _adapter.getLastErrorCode();
        _addLog('Failed to start service: ${code ?? 'unknown'}');
      }
      notifyListeners();
    } catch (e) {
      _status = ServiceStatus.stopped;
      _addLog('Failed to start service: $e');
      notifyListeners();
    }
  }

  Future<void> stop() async {
    try {
      await _adapter.stopService();
      _status = ServiceStatus.stopped;
      _addLog('Stopping PocketClaw service...');
      notifyListeners();
    } catch (e) {
      _addLog('Failed to stop native service: $e');
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _notifyTimer?.cancel();
    _nativePollingTimer?.cancel();
    _lanAddressPollingTimer?.cancel();
    super.dispose();
  }
}

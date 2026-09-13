import 'dart:async';
import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../generated/l10n/app_localizations.dart';
import 'app_theme.dart';
import 'device_feedback_models.dart';
import 'pocketclaw_channel.dart';
import 'public_mode_reconciliation.dart';
import 'plain_text_log_sanitizer.dart';
import 'status_snapshot.dart';
import 'umeng_device_reporter.dart';
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
  static const List<Duration> _deviceFeedbackRetryDelays = [
    Duration(seconds: 15),
    Duration(minutes: 1),
    Duration(minutes: 5),
  ];
  static const DeviceTelemetryThresholds _telemetryThresholds =
      DeviceTelemetryThresholds();
  static const String _prefsTelemetryCreatedAt = 'telemetry_created_at';
  static const String _prefsTelemetryLastSeenAt = 'telemetry_last_seen_at';
  static const String _prefsTelemetryLastLaunchAt = 'telemetry_last_launch_at';
  static const String _prefsTelemetryLastForegroundAt =
      'telemetry_last_foreground_at';
  static const String _prefsTelemetryLastBackgroundAt =
      'telemetry_last_background_at';
  static const String _prefsTelemetryLastActiveAt = 'telemetry_last_active_at';
  static const String _prefsTelemetryLastUploadAttemptAt =
      'telemetry_last_upload_attempt_at';
  static const String _prefsTelemetryLastUploadedAt =
      'telemetry_last_uploaded_at';
  static const String _prefsTelemetryLastSyncFailureAt =
      'telemetry_last_sync_failure_at';
  static const String _prefsTelemetryLastReachabilityLossAt =
      'telemetry_last_reachability_loss_at';
  static const String _prefsTelemetryLastReactivatedAt =
      'telemetry_last_reactivated_at';
  static const String _prefsTelemetryLastStateChangedAt =
      'telemetry_last_state_changed_at';
  static const String _prefsTelemetryLastUploadedSignature =
      'telemetry_last_uploaded_signature';
  static const String _prefsTelemetryLastFailureMessage =
      'telemetry_last_failure_message';
  static const String _prefsTelemetryLastStateReason =
      'telemetry_last_state_reason';
  static const String _prefsTelemetryLastDerivedState =
      'telemetry_last_derived_state';
  static const String _prefsTelemetryLaunchCount = 'telemetry_launch_count';
  static const String _prefsTelemetryForegroundCount =
      'telemetry_foreground_count';
  static const String _prefsTelemetryUploadFailureCount =
      'telemetry_upload_failure_count';
  static const String _prefsTelemetryConsecutiveUploadFailures =
      'telemetry_consecutive_upload_failures';
  static const String _prefsTelemetryReachabilityLost =
      'telemetry_reachability_lost';
  static const String _rawAnalyticsProvider = String.fromEnvironment(
    'POCKETCLAW_ANALYTICS_PROVIDER',
    defaultValue: 'none',
  );
  static final DeviceFeedbackProvider _requestedDeviceFeedbackProvider =
      DeviceFeedbackProvider.fromEnvironmentValue(_rawAnalyticsProvider);
  static final DeviceFeedbackProvider _deviceFeedbackProvider =
      resolveDeviceFeedbackProvider(
        requested: _requestedDeviceFeedbackProvider,
        umengAppKey: _umengAppKey,
      );
  static const String _umengAppKey = String.fromEnvironment(
    'POCKETCLAW_UMENG_APP_KEY',
  );
  static const String _umengChannel = String.fromEnvironment(
    'POCKETCLAW_UMENG_CHANNEL',
    defaultValue: 'official',
  );
  static const String _distributionChannel = String.fromEnvironment(
    'POCKETCLAW_DISTRIBUTION_CHANNEL',
    defaultValue: _umengChannel,
  );
  static final bool _isTestEnvironment =
      Platform.environment.containsKey('FLUTTER_TEST') ||
      Platform.executable.contains('flutter_tester');

  static final ServiceManager _instance = ServiceManager._internal();
  factory ServiceManager() => _instance;
  ServiceManager._internal() {
    if (!kIsWeb && !_isTestEnvironment) {
      try {
        _signalSubscriptions.add(
          ProcessSignal.sigint.watch().listen((_) => stop()),
        );
      } catch (_) {}
      try {
        if (!Platform.isWindows) {
          _signalSubscriptions.add(
            ProcessSignal.sigterm.watch().listen((_) => stop()),
          );
        }
      } catch (_) {}
    }
  }

  final CoreServiceAdapter _adapter = CoreServiceAdapterFactory.create();
  final UmengDeviceReporter _umengReporter = UmengDeviceReporter();
  String? _lastErrorCode;
  String? _lastDeviceFeedbackSyncMessage;
  Future<DeviceFeedbackUploadResult>? _deviceFeedbackUploadTask;
  Timer? _deviceFeedbackRetryTimer;
  int _deviceFeedbackRetryAttempt = 0;
  bool _deviceFeedbackConfigurationNoticeEmitted = false;
  String _cachedAppVersion = 'unknown';
  String _cachedCoreVersion = '';
  DeviceTelemetrySnapshot? _lastTelemetrySnapshot;
  final List<StreamSubscription<ProcessSignal>> _signalSubscriptions = [];

  void _syncAdapterConfiguration() {
    final configuredPath = _binaryPath.trim().isEmpty ? null : _binaryPath;
    _adapter.setConfiguredPath(configuredPath);
  }

  String? get lastErrorCode => _lastErrorCode ?? _adapter.getLastErrorCode();
  String? get lastDeviceFeedbackSyncMessage => _lastDeviceFeedbackSyncMessage;
  bool get isDeviceFeedbackUploadInProgress =>
      _deviceFeedbackUploadTask != null;
  DeviceTelemetrySnapshot? get lastTelemetrySnapshot => _lastTelemetrySnapshot;

  ServiceStatus _status = ServiceStatus.stopped;
  bool _pendingCredentialRestart = false;
  final List<String> _logs = [];

  ServiceStatus get status => _status;
  List<String> get logs => List.unmodifiable(_logs);

  String _host = '127.0.0.1';
  int _port = 18800;
  String _binaryPath = '';
  String _arguments = '';
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
  bool _autoStart = false;
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
  bool get autoStart => _autoStart;

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
  DeviceFeedbackProvider get deviceFeedbackProvider => _deviceFeedbackProvider;
  bool get isDeviceFeedbackEnabled => switch (_deviceFeedbackProvider) {
    DeviceFeedbackProvider.none => false,
    DeviceFeedbackProvider.umeng => Platform.isAndroid,
  };

  @visibleForTesting
  static DeviceFeedbackProvider resolveDeviceFeedbackProvider({
    required DeviceFeedbackProvider requested,
    required String umengAppKey,
  }) {
    switch (requested) {
      case DeviceFeedbackProvider.umeng:
        return umengAppKey.trim().isNotEmpty
            ? DeviceFeedbackProvider.umeng
            : DeviceFeedbackProvider.none;
      case DeviceFeedbackProvider.none:
        return DeviceFeedbackProvider.none;
    }
  }

  String get webUrl => 'http://$_host:$_port';
  String get localDashboardUrl => 'http://127.0.0.1:$_port';
  String? get lanAddress => _lanAddress;
  String? get publicDashboardUrl =>
      _lanAddress == null ? null : 'http://${_lanAddress!}:$_port';
  String? get connectableDashboardUrl =>
      _publicMode ? publicDashboardUrl : webUrl;
  String get host => _host;
  int get port => _port;
  String get binaryPath => _binaryPath;
  String get arguments => _arguments;
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
    _host = prefs.getString('host') ?? '127.0.0.1';
    _port = prefs.getInt('port') ?? 18800;
    _binaryPath = prefs.getString('binaryPath') ?? '';
    _arguments = prefs.getString('arguments') ?? '';
    _publicMode = prefs.getBool('publicMode') ?? false;
    _host = _publicMode ? '0.0.0.0' : _host;
    _syncAdapterConfiguration();

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
      _port = 18800;
      _host = _publicMode ? '0.0.0.0' : '127.0.0.1';
      try {
        _autoStart = await PocketClawChannel.getAutoStart();
        _workspacePath = await _adapter.getWorkspacePath();
        await _syncNativeServiceStatus();
        // Read last: the launch auto-start decision needs an accurate runtime
        // status more than it needs the preference, and this call must not be
        // able to skip the status sync above.
        _launchAutoStart = await PocketClawChannel.getLaunchAutoStartPreferences();
      } catch (_) {}
      _startNativePolling();
    }
    _syncLanAddressPolling();

    try {
      _adapter.setLogHandler(_addLog);
    } catch (_) {}

    _reportMissingOptionalDeviceFeedbackConfigurationOnce();

    if (_deviceFeedbackProvider == DeviceFeedbackProvider.umeng) {
      try {
        await _umengReporter.ensureDefaultConsentApplied();
      } catch (_) {}
    }

    _cachedAppVersion = await _readAppVersion();
    await recordTelemetryLaunch();
    await recordTelemetryForeground();

    // 自动上报设备反馈（如果用户已同意且满足条件）
    unawaited(_autoUploadDeviceFeedbackIfNeeded());

    notifyListeners();
  }

  void _reportMissingOptionalDeviceFeedbackConfigurationOnce() {
    if (_deviceFeedbackConfigurationNoticeEmitted ||
        _deviceFeedbackProvider != DeviceFeedbackProvider.none ||
        _requestedDeviceFeedbackProvider == DeviceFeedbackProvider.none) {
      return;
    }
    _deviceFeedbackConfigurationNoticeEmitted = true;
    final message = switch (_requestedDeviceFeedbackProvider) {
      DeviceFeedbackProvider.umeng =>
        'Device feedback disabled: Umeng build configuration not provided.',
      DeviceFeedbackProvider.none => '',
    };
    _lastDeviceFeedbackSyncMessage = message;
    _addLog(message);
    debugPrint(message);
  }

  Future<bool> setWorkspacePath(String value) async {
    final ok = await _adapter.setWorkspacePath(value);
    if (ok) {
      _workspacePath = value;
      notifyListeners();
    }
    return ok;
  }

  /// 刷新 workspace path（用于权限变化后重新获取）
  Future<void> refreshWorkspacePath() async {
    if (Platform.isAndroid) {
      try {
        _workspacePath = await _adapter.getWorkspacePath();
        notifyListeners();
      } catch (_) {}
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    switch (state) {
      case AppLifecycleState.resumed:
        if (_publicMode) unawaited(refreshLanAddress());
        unawaited(recordTelemetryForeground());
        unawaited(_autoUploadDeviceFeedbackIfNeeded());
        break;
      case AppLifecycleState.inactive:
      case AppLifecycleState.hidden:
      case AppLifecycleState.paused:
      case AppLifecycleState.detached:
        unawaited(recordTelemetryBackground());
        break;
    }
  }

  Future<void> recordTelemetryLaunch({DateTime? now}) async {
    final prefs = await SharedPreferences.getInstance();
    final signalAt = (now ?? DateTime.now()).toUtc();
    final store = _loadTelemetryStore(prefs);
    final shouldMarkReactivated =
        store.lastDerivedState == DeviceTelemetryState.unreachable ||
        store.lastDerivedState == DeviceTelemetryState.suspectedUninstalled;
    final updatedStore = store.copyWith(
      createdAt: store.createdAt ?? signalAt,
      lastSeenAt: signalAt,
      lastLaunchAt: signalAt,
      lastActiveAt: signalAt,
      lastReactivatedAt: shouldMarkReactivated
          ? signalAt
          : store.lastReactivatedAt,
      launchCount: store.launchCount + 1,
    );
    await _persistTelemetryStore(prefs, updatedStore);
    await _refreshTelemetrySnapshot(prefs: prefs, now: signalAt, notify: false);
  }

  Future<void> recordTelemetryForeground({DateTime? now}) async {
    final prefs = await SharedPreferences.getInstance();
    final signalAt = (now ?? DateTime.now()).toUtc();
    final store = _loadTelemetryStore(prefs);
    final shouldMarkReactivated =
        store.lastDerivedState == DeviceTelemetryState.unreachable ||
        store.lastDerivedState == DeviceTelemetryState.suspectedUninstalled;
    final updatedStore = store.copyWith(
      createdAt: store.createdAt ?? signalAt,
      lastSeenAt: signalAt,
      lastForegroundAt: signalAt,
      lastActiveAt: signalAt,
      lastReactivatedAt: shouldMarkReactivated
          ? signalAt
          : store.lastReactivatedAt,
      foregroundCount: store.foregroundCount + 1,
    );
    await _persistTelemetryStore(prefs, updatedStore);
    await _refreshTelemetrySnapshot(prefs: prefs, now: signalAt, notify: false);
  }

  Future<void> recordTelemetryBackground({DateTime? now}) async {
    final prefs = await SharedPreferences.getInstance();
    final signalAt = (now ?? DateTime.now()).toUtc();
    final store = _loadTelemetryStore(prefs);
    final updatedStore = store.copyWith(
      createdAt: store.createdAt ?? signalAt,
      lastSeenAt: signalAt,
      lastBackgroundAt: signalAt,
    );
    await _persistTelemetryStore(prefs, updatedStore);
    await _refreshTelemetrySnapshot(prefs: prefs, now: signalAt, notify: false);
  }

  Future<DeviceTelemetrySnapshot> getDeviceTelemetrySnapshot({
    DateTime? now,
  }) async {
    return _refreshTelemetrySnapshot(now: now, notify: false);
  }

  Future<void> recordTelemetryUploadAttempt({DateTime? now}) async {
    final prefs = await SharedPreferences.getInstance();
    final signalAt = (now ?? DateTime.now()).toUtc();
    final store = _loadTelemetryStore(
      prefs,
    ).copyWith(lastUploadAttemptAt: signalAt);
    await _persistTelemetryStore(prefs, store);
  }

  Future<void> recordTelemetryUploadSuccess(
    DeviceTelemetrySnapshot snapshot, {
    DateTime? now,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    final signalAt = (now ?? DateTime.now()).toUtc();
    final currentStore = _loadTelemetryStore(prefs);
    final store = currentStore.copyWith(
      lastUploadedAt: signalAt,
      lastUploadedSignature: snapshot.buildUploadSignature(),
      lastFailureMessage: '',
      uploadFailureCount: 0,
      consecutiveUploadFailures: 0,
      reachabilityLost: false,
      lastReactivatedAt: snapshot.state == DeviceTelemetryState.reinstalled
          ? signalAt
          : currentStore.lastReactivatedAt,
    );
    await _persistTelemetryStore(prefs, store);
    final refreshedSnapshot = await _refreshTelemetrySnapshot(
      prefs: prefs,
      now: signalAt,
      notify: false,
    );
    final normalizedStore = _loadTelemetryStore(
      prefs,
    ).copyWith(lastUploadedSignature: refreshedSnapshot.buildUploadSignature());
    await _persistTelemetryStore(prefs, normalizedStore);
    _lastTelemetrySnapshot = await _refreshTelemetrySnapshot(
      prefs: prefs,
      now: signalAt,
      notify: false,
    );
  }

  Future<void> recordTelemetryUploadFailure(
    String message, {
    DateTime? now,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    final signalAt = (now ?? DateTime.now()).toUtc();
    final store = _loadTelemetryStore(prefs);
    final nextConsecutiveFailures = store.consecutiveUploadFailures + 1;
    final reachabilityLost =
        nextConsecutiveFailures >=
            _telemetryThresholds.reachabilityFailureThreshold ||
        store.reachabilityLost;
    final updatedStore = store.copyWith(
      lastSyncFailureAt: signalAt,
      lastFailureMessage: message,
      uploadFailureCount: store.uploadFailureCount + 1,
      consecutiveUploadFailures: nextConsecutiveFailures,
      reachabilityLost: reachabilityLost,
      lastReachabilityLossAt: reachabilityLost
          ? (store.lastReachabilityLossAt ?? signalAt)
          : store.lastReachabilityLossAt,
    );
    await _persistTelemetryStore(prefs, updatedStore);
    await _refreshTelemetrySnapshot(prefs: prefs, now: signalAt, notify: false);
  }

  Future<void> clearTelemetryState() async {
    final prefs = await SharedPreferences.getInstance();
    for (final key in const [
      _prefsTelemetryCreatedAt,
      _prefsTelemetryLastSeenAt,
      _prefsTelemetryLastLaunchAt,
      _prefsTelemetryLastForegroundAt,
      _prefsTelemetryLastBackgroundAt,
      _prefsTelemetryLastActiveAt,
      _prefsTelemetryLastUploadAttemptAt,
      _prefsTelemetryLastUploadedAt,
      _prefsTelemetryLastSyncFailureAt,
      _prefsTelemetryLastReachabilityLossAt,
      _prefsTelemetryLastReactivatedAt,
      _prefsTelemetryLastStateChangedAt,
      _prefsTelemetryLastUploadedSignature,
      _prefsTelemetryLastFailureMessage,
      _prefsTelemetryLastStateReason,
      _prefsTelemetryLastDerivedState,
      _prefsTelemetryLaunchCount,
      _prefsTelemetryForegroundCount,
      _prefsTelemetryUploadFailureCount,
      _prefsTelemetryConsecutiveUploadFailures,
      _prefsTelemetryReachabilityLost,
    ]) {
      await prefs.remove(key);
    }
    _lastTelemetrySnapshot = null;
  }

  Future<DeviceTelemetryRuntimeContext> _buildTelemetryRuntimeContext() async {
    if (_cachedAppVersion == 'unknown') {
      _cachedAppVersion = await _readAppVersion();
    }
    return DeviceTelemetryRuntimeContext(
      platform: Platform.operatingSystem,
      appVersion: _cachedAppVersion,
      channel: _distributionChannel.trim().isEmpty
          ? 'official'
          : _distributionChannel,
      region: _resolveTelemetryRegion(),
      provider: _deviceFeedbackProvider.name,
    );
  }

  String _resolveTelemetryRegion() {
    final countryCode = _currentLocale.countryCode;
    if (countryCode != null && countryCode.isNotEmpty) {
      return countryCode.toLowerCase();
    }
    return _currentLocale.languageCode.toLowerCase();
  }

  Future<String> getAppVersion() async {
    if (_cachedAppVersion == 'unknown') {
      _cachedAppVersion = await _readAppVersion();
    }
    return _cachedAppVersion;
  }

  Future<String> getCoreVersion() async {
    _syncAdapterConfiguration();
    final version = await _adapter.getCoreVersion();
    if (version.isNotEmpty && version != _cachedCoreVersion) {
      _cachedCoreVersion = version;
      notifyListeners();
    }
    return version;
  }

  /// App version for display, or an empty string until it has been read.
  String get appVersion => _cachedAppVersion == 'unknown' ? '' : _cachedAppVersion;

  /// Core version for display, read once and cached.
  ///
  /// Reading it means invoking the Core binary, so it is fetched on demand
  /// rather than on every poll: a version does not change while the process
  /// runs.
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

  DeviceTelemetryStore _loadTelemetryStore(SharedPreferences prefs) {
    return DeviceTelemetryStore(
      createdAt: _readTimestamp(prefs, _prefsTelemetryCreatedAt),
      lastSeenAt: _readTimestamp(prefs, _prefsTelemetryLastSeenAt),
      lastLaunchAt: _readTimestamp(prefs, _prefsTelemetryLastLaunchAt),
      lastForegroundAt: _readTimestamp(prefs, _prefsTelemetryLastForegroundAt),
      lastBackgroundAt: _readTimestamp(prefs, _prefsTelemetryLastBackgroundAt),
      lastActiveAt: _readTimestamp(prefs, _prefsTelemetryLastActiveAt),
      lastUploadAttemptAt: _readTimestamp(
        prefs,
        _prefsTelemetryLastUploadAttemptAt,
      ),
      lastUploadedAt: _readTimestamp(prefs, _prefsTelemetryLastUploadedAt),
      lastSyncFailureAt: _readTimestamp(
        prefs,
        _prefsTelemetryLastSyncFailureAt,
      ),
      lastReachabilityLossAt: _readTimestamp(
        prefs,
        _prefsTelemetryLastReachabilityLossAt,
      ),
      lastReactivatedAt: _readTimestamp(
        prefs,
        _prefsTelemetryLastReactivatedAt,
      ),
      lastStateChangedAt: _readTimestamp(
        prefs,
        _prefsTelemetryLastStateChangedAt,
      ),
      lastUploadedSignature: prefs.getString(
        _prefsTelemetryLastUploadedSignature,
      ),
      lastFailureMessage: prefs.getString(_prefsTelemetryLastFailureMessage),
      lastStateReason: prefs.getString(_prefsTelemetryLastStateReason),
      lastDerivedState: DeviceTelemetryState.fromWireValue(
        prefs.getString(_prefsTelemetryLastDerivedState),
      ),
      launchCount: prefs.getInt(_prefsTelemetryLaunchCount) ?? 0,
      foregroundCount: prefs.getInt(_prefsTelemetryForegroundCount) ?? 0,
      uploadFailureCount: prefs.getInt(_prefsTelemetryUploadFailureCount) ?? 0,
      consecutiveUploadFailures:
          prefs.getInt(_prefsTelemetryConsecutiveUploadFailures) ?? 0,
      reachabilityLost: prefs.getBool(_prefsTelemetryReachabilityLost) ?? false,
    );
  }

  Future<void> _persistTelemetryStore(
    SharedPreferences prefs,
    DeviceTelemetryStore store,
  ) async {
    await _writeTimestamp(prefs, _prefsTelemetryCreatedAt, store.createdAt);
    await _writeTimestamp(prefs, _prefsTelemetryLastSeenAt, store.lastSeenAt);
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastLaunchAt,
      store.lastLaunchAt,
    );
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastForegroundAt,
      store.lastForegroundAt,
    );
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastBackgroundAt,
      store.lastBackgroundAt,
    );
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastActiveAt,
      store.lastActiveAt,
    );
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastUploadAttemptAt,
      store.lastUploadAttemptAt,
    );
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastUploadedAt,
      store.lastUploadedAt,
    );
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastSyncFailureAt,
      store.lastSyncFailureAt,
    );
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastReachabilityLossAt,
      store.lastReachabilityLossAt,
    );
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastReactivatedAt,
      store.lastReactivatedAt,
    );
    await _writeTimestamp(
      prefs,
      _prefsTelemetryLastStateChangedAt,
      store.lastStateChangedAt,
    );
    await _writeString(
      prefs,
      _prefsTelemetryLastUploadedSignature,
      store.lastUploadedSignature,
    );
    await _writeString(
      prefs,
      _prefsTelemetryLastFailureMessage,
      store.lastFailureMessage,
    );
    await _writeString(
      prefs,
      _prefsTelemetryLastStateReason,
      store.lastStateReason,
    );
    await _writeString(
      prefs,
      _prefsTelemetryLastDerivedState,
      store.lastDerivedState.wireValue,
    );
    await prefs.setInt(_prefsTelemetryLaunchCount, store.launchCount);
    await prefs.setInt(_prefsTelemetryForegroundCount, store.foregroundCount);
    await prefs.setInt(
      _prefsTelemetryUploadFailureCount,
      store.uploadFailureCount,
    );
    await prefs.setInt(
      _prefsTelemetryConsecutiveUploadFailures,
      store.consecutiveUploadFailures,
    );
    await prefs.setBool(
      _prefsTelemetryReachabilityLost,
      store.reachabilityLost,
    );
  }

  Future<DeviceTelemetrySnapshot> _refreshTelemetrySnapshot({
    SharedPreferences? prefs,
    DateTime? now,
    bool notify = true,
  }) async {
    final sharedPrefs = prefs ?? await SharedPreferences.getInstance();
    final derivedAt = (now ?? DateTime.now()).toUtc();
    final context = await _buildTelemetryRuntimeContext();
    var store = _loadTelemetryStore(sharedPrefs);
    final initialSnapshot = DeviceTelemetryDeriver.derive(
      store: store,
      context: context,
      thresholds: _telemetryThresholds,
      now: derivedAt,
    );
    final stateChanged =
        initialSnapshot.state != store.lastDerivedState ||
        initialSnapshot.stateReason != store.lastStateReason;
    store = store.copyWith(
      lastDerivedState: initialSnapshot.state,
      lastStateReason: initialSnapshot.stateReason,
      lastStateChangedAt: stateChanged
          ? derivedAt
          : (store.lastStateChangedAt ?? derivedAt),
      lastActiveAt: initialSnapshot.lastActiveAt,
    );
    await _persistTelemetryStore(sharedPrefs, store);
    final finalSnapshot = DeviceTelemetryDeriver.derive(
      store: store,
      context: context,
      thresholds: _telemetryThresholds,
      now: derivedAt,
    );
    _lastTelemetrySnapshot = finalSnapshot;
    if (notify) {
      notifyListeners();
    }
    return finalSnapshot;
  }

  DateTime? _readTimestamp(SharedPreferences prefs, String key) {
    final raw = prefs.getString(key);
    if (raw == null || raw.isEmpty) {
      return null;
    }
    return DateTime.tryParse(raw)?.toUtc();
  }

  Future<void> _writeTimestamp(
    SharedPreferences prefs,
    String key,
    DateTime? value,
  ) async {
    if (value == null) {
      await prefs.remove(key);
      return;
    }
    await prefs.setString(key, value.toUtc().toIso8601String());
  }

  Future<void> _writeString(
    SharedPreferences prefs,
    String key,
    String? value,
  ) async {
    if (value == null || value.isEmpty) {
      await prefs.remove(key);
      return;
    }
    await prefs.setString(key, value);
  }

  Future<void> _autoUploadDeviceFeedbackIfNeeded() async {
    try {
      final isAllowed = await isDeviceFeedbackAllowed();
      final shouldUpload = await shouldAutoUploadDeviceFeedbackReport();

      if (isAllowed && shouldUpload) {
        triggerDeviceFeedbackUploadInBackground();
      }
    } catch (e) {
      // Silent error handling
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

  Future<void> setAutoStart(bool enabled) async {
    if (Platform.isAndroid) {
      await PocketClawChannel.setAutoStart(enabled);
      _autoStart = enabled;
      notifyListeners();
    }
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
        dashboardInitialized: await PocketClawChannel.dashboardAuthInitialized(),
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
      _addLog('Could not start the Gateway now; it will start with the service');
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
        _launchAutoStart = await PocketClawChannel.getLaunchAutoStartPreferences();
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

  Future<Map<String, String>> getDeviceFeedbackDeviceInfo() async {
    switch (_deviceFeedbackProvider) {
      case DeviceFeedbackProvider.umeng:
        return _umengReporter.collectSafeDeviceInfo();
      case DeviceFeedbackProvider.none:
        return const {};
    }
  }

  Future<bool> isDeviceFeedbackAllowed() async {
    switch (_deviceFeedbackProvider) {
      case DeviceFeedbackProvider.umeng:
        return _umengReporter.isUploadAllowed();
      case DeviceFeedbackProvider.none:
        return false;
    }
  }

  Future<bool> shouldAskForDeviceFeedbackUpload() async {
    return false;
  }

  Future<bool> shouldAutoUploadDeviceFeedbackReport() async {
    if (!await isDeviceFeedbackAllowed()) {
      return false;
    }

    final snapshot = await getDeviceTelemetrySnapshot();
    final signatureChanged =
        snapshot.lastUploadedSignature == null ||
        snapshot.lastUploadedSignature != snapshot.buildUploadSignature();

    final providerRequestedUpload = switch (_deviceFeedbackProvider) {
      DeviceFeedbackProvider.umeng => _umengReporter.shouldUpload(),
      DeviceFeedbackProvider.none => Future<bool>.value(false),
    };

    return signatureChanged ||
        snapshot.isStale ||
        await providerRequestedUpload;
  }

  Future<void> setDeviceFeedbackUploadAllowed(bool allowed) async {
    if (!allowed) {
      _resetDeviceFeedbackRetryState();
    }
    switch (_deviceFeedbackProvider) {
      case DeviceFeedbackProvider.umeng:
        await _umengReporter.setUploadAllowed(allowed);
        return;
      case DeviceFeedbackProvider.none:
        return;
    }
  }

  Future<DeviceFeedbackUploadResult> uploadDeviceFeedbackReport() async {
    final ongoingTask = _deviceFeedbackUploadTask;
    if (ongoingTask != null) {
      return ongoingTask;
    }

    _lastDeviceFeedbackSyncMessage = 'Syncing device feedback in background...';
    notifyListeners();

    final task = _uploadDeviceFeedbackReportInternal();
    _deviceFeedbackUploadTask = task;

    try {
      return await task;
    } finally {
      if (identical(_deviceFeedbackUploadTask, task)) {
        _deviceFeedbackUploadTask = null;
        notifyListeners();
      }
    }
  }

  void triggerDeviceFeedbackUploadInBackground() {
    if (!isDeviceFeedbackEnabled || isDeviceFeedbackUploadInProgress) {
      return;
    }
    _deviceFeedbackRetryTimer?.cancel();
    _deviceFeedbackRetryTimer = null;
    unawaited(uploadDeviceFeedbackReport());
  }

  Future<DeviceFeedbackUploadResult>
  _uploadDeviceFeedbackReportInternal() async {
    final attemptAt = DateTime.now().toUtc();
    await recordTelemetryUploadAttempt(now: attemptAt);
    final telemetrySnapshot = await getDeviceTelemetrySnapshot(now: attemptAt);
    late final DeviceFeedbackUploadResult result;
    switch (_deviceFeedbackProvider) {
      case DeviceFeedbackProvider.umeng:
        if (_umengAppKey.trim().isEmpty) {
          result = const DeviceFeedbackUploadResult(
            success: false,
            message: 'Missing POCKETCLAW_UMENG_APP_KEY build configuration.',
          );
          break;
        }
        result = await _umengReporter.uploadDeviceReport(
          appKey: _umengAppKey,
          channel: _umengChannel,
          telemetrySnapshot: telemetrySnapshot,
        );
        break;
      case DeviceFeedbackProvider.none:
        result = const DeviceFeedbackUploadResult(
          success: false,
          message: 'Device feedback provider is disabled.',
        );
        break;
    }
    _lastDeviceFeedbackSyncMessage = result.message;
    if (result.success) {
      await recordTelemetryUploadSuccess(telemetrySnapshot, now: attemptAt);
      _resetDeviceFeedbackRetryState(notify: false);
    } else {
      await recordTelemetryUploadFailure(result.message, now: attemptAt);
      await _scheduleDeviceFeedbackRetryIfNeeded(result);
    }
    _addLog(
      result.success
          ? 'Device feedback sync succeeded.'
          : 'Device feedback sync failed: ${result.message}',
    );
    if (!result.success) {
      debugPrint('Device feedback sync failed: ${result.message}');
    }
    return result;
  }

  Future<void> _scheduleDeviceFeedbackRetryIfNeeded(
    DeviceFeedbackUploadResult result,
  ) async {
    if (!_shouldRetryDeviceFeedback(result.message)) {
      _resetDeviceFeedbackRetryState(notify: false);
      return;
    }
    if (!await isDeviceFeedbackAllowed()) {
      _resetDeviceFeedbackRetryState(notify: false);
      return;
    }
    if (_deviceFeedbackRetryAttempt >= _deviceFeedbackRetryDelays.length) {
      _lastDeviceFeedbackSyncMessage =
          '${result.message} Auto retry stopped for now.';
      return;
    }

    final delay = _deviceFeedbackRetryDelays[_deviceFeedbackRetryAttempt];
    _deviceFeedbackRetryAttempt += 1;
    _deviceFeedbackRetryTimer?.cancel();
    _lastDeviceFeedbackSyncMessage =
        '${result.message} Retrying silently in ${delay.inSeconds}s.';
    _deviceFeedbackRetryTimer = Timer(delay, () {
      _deviceFeedbackRetryTimer = null;
      if (!isDeviceFeedbackEnabled || isDeviceFeedbackUploadInProgress) {
        return;
      }
      unawaited(uploadDeviceFeedbackReport());
    });
  }

  bool _shouldRetryDeviceFeedback(String message) {
    final normalized = message.toLowerCase();
    if (normalized.contains('missing picoclaw_') ||
        normalized.contains('provider is disabled') ||
        normalized.contains('projectid is empty') ||
        normalized.contains('apikey is empty') ||
        normalized.contains('appid is empty') ||
        normalized.contains('messagingsenderid is empty') ||
        normalized.contains('appkey is empty') ||
        normalized.contains('not supported on this platform') ||
        normalized.contains('only supported on android')) {
      return false;
    }
    return true;
  }

  void _resetDeviceFeedbackRetryState({bool notify = true}) {
    _deviceFeedbackRetryTimer?.cancel();
    _deviceFeedbackRetryTimer = null;
    _deviceFeedbackRetryAttempt = 0;
    if (notify) {
      notifyListeners();
    }
  }

  Future<void> updateConfig(
    String host,
    int port, {
    String? binaryPath,
    String? arguments,
    bool? publicMode,
  }) async {
    _host = host;
    _port = port;
    if (!(Platform.isWindows || Platform.isAndroid)) {
      if (binaryPath != null) _binaryPath = binaryPath;
    }
    if (arguments != null) _arguments = arguments;
    if (publicMode != null) {
      _publicMode = publicMode;
      _host = publicMode ? '0.0.0.0' : host;
    }
    _syncAdapterConfiguration();

    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('host', _host);
    await prefs.setInt('port', port);
    if (!(Platform.isWindows || Platform.isAndroid)) {
      if (binaryPath != null) await prefs.setString('binaryPath', binaryPath);
    }
    if (arguments != null) await prefs.setString('arguments', arguments);
    await prefs.setBool('publicMode', _publicMode);
    _syncLanAddressPolling();
    notifyListeners();
  }

  /// Applies Public Mode transactionally. On a running Android service this
  /// asks the launcher to replace only port 18800's listeners and persists the
  /// setting only after the new bind succeeds. A failed bind reports the
  /// listener mode restored by the launcher.
  Future<bool> applyPublicMode(
    bool value, {
    required int port,
    String? arguments,
  }) async {
    if (_isApplyingPublicMode) return false;
    _publicModeApplyError = null;

    if (!Platform.isAndroid || _status != ServiceStatus.running) {
      await updateConfig(
        value ? '0.0.0.0' : '127.0.0.1',
        port,
        arguments: arguments,
        publicMode: value,
      );
      return true;
    }

    _isApplyingPublicMode = true;
    notifyListeners();
    try {
      final result = await PocketClawChannel.applyPublicMode(value);
      await updateConfig(
        result.publicMode ? '0.0.0.0' : '127.0.0.1',
        port,
        arguments: arguments,
        publicMode: result.publicMode,
      );
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

  Future<bool> validateBinary([String? path]) async {
    _syncAdapterConfiguration();
    String? checkPath;
    if (path != null && path.isNotEmpty) {
      checkPath = path;
    } else if (_binaryPath.isNotEmpty) {
      checkPath = _binaryPath;
    }

    final ok = await _adapter.validateBinary(checkPath);
    _lastErrorCode = _adapter.getLastErrorCode();
    notifyListeners();
    return ok;
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
  String _launchArguments() {
    // Simple token logic (split by spaces and dedupe) instead of regex.
    // _arguments is initialized to '' and loaded with `?? ''` in init(), so
    // it's non-null.
    final tokens = _arguments.split(' ').where((t) => t.isNotEmpty).toList();

    if (_publicMode && !tokens.contains('-public')) {
      tokens.add('-public');
    }
    if (!tokens.contains('-no-browser')) {
      tokens.add('-no-browser');
    }
    return tokens.join(' ');
  }

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

    _syncAdapterConfiguration();
    _status = ServiceStatus.starting;
    notifyListeners();

    try {
      final ok = await _adapter.restartService(
        port: _port,
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
    _syncAdapterConfiguration();

    _status = ServiceStatus.starting;
    notifyListeners();

    final String launchArgs = _launchArguments();
    try {
      final ok = await _adapter.startService(port: _port, args: launchArgs);

      if (ok) {
        if (Platform.isAndroid) {
          // Android: keep original behavior — log and defer health check to native side
          _addLog('Starting PocketClaw service...');
          Future.delayed(const Duration(seconds: 2), () {
            _syncNativeServiceStatus();
          });
        } else {
          // Desktop: consider service running immediately
          _status = ServiceStatus.running;
          _addLog('Service started on $webUrl');
        }
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
    _deviceFeedbackRetryTimer?.cancel();
    _notifyTimer?.cancel();
    _nativePollingTimer?.cancel();
    _lanAddressPollingTimer?.cancel();
    for (final subscription in _signalSubscriptions) {
      subscription.cancel();
    }
    super.dispose();
  }
}

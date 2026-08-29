import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../generated/l10n/app_localizations.dart';
import 'app_theme.dart';
import 'device_feedback_models.dart';
import 'firebase_device_reporter.dart';
import 'launch_autostart_preferences.dart';
import 'picoclaw_channel.dart';
import 'plain_text_log_sanitizer.dart';
import 'umeng_device_reporter.dart';
import '../native/core_service_adapter_factory.dart';
import '../native/core_service_adapter.dart';
import 'autostart_coordinator.dart';

enum ServiceStatus { stopped, starting, running, stopping, failed }

@immutable
class LanAddressCandidate {
  const LanAddressCandidate({
    required this.interfaceName,
    required this.address,
  });

  final String interfaceName;
  final String address;
}

class ServiceManager extends ChangeNotifier
    with WidgetsBindingObserver
    implements AutoStartRuntime {
  static const Duration _serviceStartTimeout = Duration(seconds: 25);
  static const Duration _serviceStopTimeout = Duration(seconds: 15);
  static const Duration _gatewayStartTimeout = Duration(seconds: 20);
  static const Duration _readinessPollInterval = Duration(milliseconds: 250);
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
    'PICOCLAW_ANALYTICS_PROVIDER',
    defaultValue: 'none',
  );
  static final DeviceFeedbackProvider _requestedDeviceFeedbackProvider =
      DeviceFeedbackProvider.fromEnvironmentValue(_rawAnalyticsProvider);
  static final DeviceFeedbackProvider _deviceFeedbackProvider =
      resolveDeviceFeedbackProvider(
        requested: _requestedDeviceFeedbackProvider,
        firebaseProjectId: _firebaseProjectId,
        firebaseApiKey: _firebaseApiKey,
        firebaseAppId: _firebaseAppId,
        firebaseMessagingSenderId: _firebaseMessagingSenderId,
        umengAppKey: _umengAppKey,
      );
  static const String _firebaseProjectId = String.fromEnvironment(
    'PICOCLAW_FIREBASE_PROJECT_ID',
  );
  static const String _firebaseApiKey = String.fromEnvironment(
    'PICOCLAW_FIREBASE_API_KEY',
  );
  static const String _firebaseAppId = String.fromEnvironment(
    'PICOCLAW_FIREBASE_APP_ID',
  );
  static const String _firebaseMessagingSenderId = String.fromEnvironment(
    'PICOCLAW_FIREBASE_MESSAGING_SENDER_ID',
  );
  static const String _firebaseStorageBucket = String.fromEnvironment(
    'PICOCLAW_FIREBASE_STORAGE_BUCKET',
    defaultValue: '',
  );
  static const String _umengAppKey = String.fromEnvironment(
    'PICOCLAW_UMENG_APP_KEY',
  );
  static const String _umengChannel = String.fromEnvironment(
    'PICOCLAW_UMENG_CHANNEL',
    defaultValue: 'official',
  );
  static const String _distributionChannel = String.fromEnvironment(
    'PICOCLAW_DISTRIBUTION_CHANNEL',
    defaultValue: _umengChannel,
  );
  static final bool _isTestEnvironment =
      Platform.environment.containsKey('FLUTTER_TEST') ||
      Platform.executable.contains('flutter_tester');

  static final ServiceManager _instance = ServiceManager._internal();
  factory ServiceManager() => _instance;
  ServiceManager._internal() {
    _autoStartCoordinator = AutoStartCoordinator(
      runtime: this,
      logger: _addLifecycleLog,
    );
    if (!kIsWeb && !_isTestEnvironment) {
      try {
        _signalSubscriptions.add(
          ProcessSignal.sigint.watch().listen(
            (_) => stop(source: 'process_signal'),
          ),
        );
      } catch (_) {}
      try {
        if (!Platform.isWindows) {
          _signalSubscriptions.add(
            ProcessSignal.sigterm.watch().listen(
              (_) => stop(source: 'process_signal'),
            ),
          );
        }
      } catch (_) {}
    }
  }

  final CoreServiceAdapter _adapter = CoreServiceAdapterFactory.create();
  final LaunchAutoStartPreferenceStore _launchAutoStartPreferenceStore =
      LaunchAutoStartPreferenceStore(
        useNativeStore: Platform.isAndroid,
        readNative: PicoClawChannel.getLaunchAutoStartPreferences,
        writeNative: PicoClawChannel.setLaunchAutoStartPreferences,
      );
  late final AutoStartCoordinator _autoStartCoordinator;
  final FirebaseDeviceReporter _firebaseReporter = FirebaseDeviceReporter();
  final UmengDeviceReporter _umengReporter = UmengDeviceReporter();
  String? _lastErrorCode;
  String? _lastDeviceFeedbackSyncMessage;
  Future<DeviceFeedbackUploadResult>? _deviceFeedbackUploadTask;
  Timer? _deviceFeedbackRetryTimer;
  int _deviceFeedbackRetryAttempt = 0;
  bool _deviceFeedbackConfigurationNoticeEmitted = false;
  String _cachedAppVersion = 'unknown';
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
  AutoStartRuntimeState _gatewayStatus = AutoStartRuntimeState.stopped;
  final List<String> _logs = [];

  ServiceStatus get status => _status;
  AutoStartRuntimeState get gatewayStatus => _gatewayStatus;
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
  bool _autoStart = false;
  bool _serviceLaunchAutoStart = AutoStartPreferences.defaults.serviceEnabled;
  bool _gatewayLaunchAutoStart = AutoStartPreferences.defaults.gatewayEnabled;
  String _launchAutoStartPreferenceSource = 'defaults';
  String? _serviceStartError;
  String? _gatewayStartError;
  String? _gatewayRuntimeOperationId;
  String? _servicePreferenceError;
  String? _gatewayPreferenceError;
  Future<AutoStartTransitionResult>? _serviceStartTask;
  Future<AutoStartTransitionResult>? _gatewayStartTask;
  int _serviceTransitionGeneration = 0;
  int _gatewayTransitionGeneration = 0;
  bool _manualStopActive = false;
  bool _observeGatewayRuntime = false;
  Future<void>? _nativeStatusSyncTask;

  int get nativePid => _nativePid;
  String get healthStatus => _healthStatus;
  String get healthUptime => _healthUptime;
  bool get autoStart => _autoStart;
  bool get serviceLaunchAutoStart => _serviceLaunchAutoStart;
  bool get gatewayLaunchAutoStart => _gatewayLaunchAutoStart;
  String get launchAutoStartPreferenceSource =>
      _launchAutoStartPreferenceSource;
  String? get serviceStartError =>
      _servicePreferenceError ?? _serviceStartError;
  String? get gatewayStartError =>
      _gatewayPreferenceError ?? _gatewayStartError;
  bool get manualStopActive => _manualStopActive;

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
    DeviceFeedbackProvider.firebase => Platform.isAndroid || Platform.isIOS,
    DeviceFeedbackProvider.umeng => Platform.isAndroid,
  };

  @visibleForTesting
  static DeviceFeedbackProvider resolveDeviceFeedbackProvider({
    required DeviceFeedbackProvider requested,
    required String firebaseProjectId,
    required String firebaseApiKey,
    required String firebaseAppId,
    required String firebaseMessagingSenderId,
    required String umengAppKey,
  }) {
    switch (requested) {
      case DeviceFeedbackProvider.firebase:
        final configured = [
          firebaseProjectId,
          firebaseApiKey,
          firebaseAppId,
          firebaseMessagingSenderId,
        ].every((value) => value.trim().isNotEmpty);
        return configured
            ? DeviceFeedbackProvider.firebase
            : DeviceFeedbackProvider.none;
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
        return await PicoClawChannel.getLanIpv4Address();
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
    try {
      await _reloadLaunchAutoStartPreferences(notify: false);
    } catch (_) {
      _launchAutoStartPreferenceSource = 'native_canonical_unavailable';
      _servicePreferenceError = 'Auto-start preferences could not be loaded.';
      _gatewayPreferenceError = 'Auto-start preferences could not be loaded.';
    }
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
        _autoStart = await PicoClawChannel.getAutoStart();
        _workspacePath = await _adapter.getWorkspacePath();
        await _syncNativeServiceStatus();
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
      DeviceFeedbackProvider.firebase =>
        'Device feedback disabled: Firebase build configuration not provided.',
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
        unawaited(ensureAutoStart(source: 'app_resume_autostart'));
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
    return _adapter.getCoreVersion();
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
  Future<void> pollNativeServiceStatusForTest({bool includeGateway = false}) =>
      _syncNativeServiceStatus(includeGateway: includeGateway);

  @visibleForTesting
  void setRuntimeFailureForTest({String? serviceError, String? gatewayError}) {
    _status = ServiceStatus.failed;
    _gatewayStatus = AutoStartRuntimeState.failed;
    _serviceStartError = serviceError;
    _gatewayStartError = gatewayError;
  }

  @visibleForTesting
  static bool isCurrentOperationGenerationForTest({
    required int operationGeneration,
    required int currentGeneration,
  }) => operationGeneration == currentGeneration;

  Future<void> _syncNativeServiceStatus({bool? includeGateway}) async {
    final current = _nativeStatusSyncTask;
    if (current != null) {
      await current;
      return;
    }
    final task = _performNativeRuntimeSync(
      includeGateway ?? _observeGatewayRuntime,
    );
    _nativeStatusSyncTask = task;
    try {
      await task;
    } finally {
      if (identical(_nativeStatusSyncTask, task)) {
        _nativeStatusSyncTask = null;
      }
    }
  }

  Future<void> _performNativeRuntimeSync(bool includeGateway) async {
    try {
      final native = await PicoClawChannel.getServiceStatus();
      final isRunning = native['isRunning'] as bool? ?? false;
      final isStarting = native['isStarting'] as bool? ?? false;
      final isStopping = native['isStopping'] as bool? ?? false;
      final hasFailed = native['hasFailed'] as bool? ?? false;
      _manualStopActive =
          native['manualStopActive'] as bool? ?? _manualStopActive;
      _nativePid = native['pid'] as int? ?? -1;

      final oldServiceStatus = _status;
      final oldGatewayStatus = _gatewayStatus;
      final oldServiceError = _serviceStartError;
      final oldGatewayError = _gatewayStartError;
      var serviceHealthy = false;
      if (isStopping) {
        _status = ServiceStatus.stopping;
      } else if (isRunning) {
        try {
          final health = await PicoClawChannel.checkHealth();
          serviceHealthy = health['isHealthy'] as bool? ?? false;
          _healthStatus = serviceHealthy ? 'Healthy' : 'Starting...';
          _healthUptime = health['uptime'] as String? ?? '';
          if (health['pid'] != null && (health['pid'] as int) > 0) {
            _nativePid = health['pid'] as int;
          }
        } catch (_) {
          _healthStatus = 'Starting...';
        }
        _status = serviceHealthy
            ? ServiceStatus.running
            : ServiceStatus.starting;
      } else if (isStarting ||
          (_serviceStartTask != null && _status != ServiceStatus.stopping)) {
        _status = ServiceStatus.starting;
      } else if (hasFailed && !_manualStopActive) {
        _status = ServiceStatus.failed;
      } else if (_status != ServiceStatus.failed || _manualStopActive) {
        _status = ServiceStatus.stopped;
        if (_manualStopActive) _serviceStartError = null;
      }

      // Drain the lines emitted since the last poll. `status['lastLog']` is a
      // sticky snapshot of the most recent line, so appending it here re-added
      // the same entry every three seconds until it filled the Logs screen and
      // evicted the real history.
      for (final line in await PicoClawChannel.takeNewLogs()) {
        _addLog(line);
      }

      if (_status == ServiceStatus.running) {
        // Canonical readiness supersedes any timeout from an older operation.
        if (oldServiceStatus != ServiceStatus.running ||
            oldServiceError != null) {
          _serviceTransitionGeneration += 1;
        }
        _serviceStartError = null;
        if (includeGateway) {
          await _refreshGatewayRuntimeFromNative();
        }
      } else {
        _healthStatus = '';
        _healthUptime = '';
        if (_status == ServiceStatus.stopping) {
          _gatewayStatus = AutoStartRuntimeState.stopping;
        } else if (_gatewayStartTask == null) {
          _gatewayStatus = AutoStartRuntimeState.stopped;
        }
      }

      final operationId =
          native['lastStartOperationId'] as String? ??
          'runtime-${DateTime.now().toUtc().microsecondsSinceEpoch}';
      if (oldServiceStatus != _status) {
        _addLifecycleLog('service.runtime.state.changed', {
          'operation_id': operationId,
          'source': 'native_status_poll',
          'previous_state': oldServiceStatus.name,
          'target_state': _status.name,
          'result': 'observed',
        });
      }
      if (oldGatewayStatus != _gatewayStatus) {
        _addLifecycleLog('gateway.runtime.state.changed', {
          'operation_id': _gatewayRuntimeOperationId ?? operationId,
          'source': 'native_status_poll',
          'previous_state': oldGatewayStatus.name,
          'target_state': _gatewayStatus.name,
          'result': 'observed',
        });
      }
      if (oldServiceStatus != _status ||
          oldGatewayStatus != _gatewayStatus ||
          oldServiceError != _serviceStartError ||
          oldGatewayError != _gatewayStartError) {
        notifyListeners();
      }
    } catch (e) {
      debugPrint('Failed to sync Android service status: $e');
    }
  }

  Future<void> _refreshGatewayRuntimeFromNative() async {
    try {
      final previousStatus = _gatewayStatus;
      final previousError = _gatewayStartError;
      final native = await PicoClawChannel.getGatewayStatus();
      _gatewayRuntimeOperationId = native['operation_id'] as String?;
      _gatewayStatus = switch (native['gateway_status'] as String? ?? '') {
        'running' => AutoStartRuntimeState.running,
        'starting' || 'restarting' => AutoStartRuntimeState.starting,
        'stopping' => AutoStartRuntimeState.stopping,
        'error' => AutoStartRuntimeState.failed,
        _ => AutoStartRuntimeState.stopped,
      };
      if (_gatewayStatus == AutoStartRuntimeState.running ||
          _gatewayStatus == AutoStartRuntimeState.stopped) {
        if (_gatewayStatus == AutoStartRuntimeState.running &&
            (previousStatus != AutoStartRuntimeState.running ||
                previousError != null)) {
          _gatewayTransitionGeneration += 1;
        }
        _gatewayStartError = null;
      }
    } catch (_) {
      // Keep the last truthful Gateway state when the Dashboard bridge is
      // temporarily unavailable; the next bounded poll will try again.
    }
  }

  Future<void> setSettingsRuntimeObservation(bool visible) async {
    _observeGatewayRuntime = visible;
    if (!visible || !Platform.isAndroid) return;
    await _syncNativeServiceStatus(includeGateway: true);
    // If this call joined a service-only poll, run one explicit Gateway-aware
    // refresh immediately rather than waiting for the next three-second tick.
    await _syncNativeServiceStatus(includeGateway: true);
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
      await PicoClawChannel.setAutoStart(enabled);
      _autoStart = enabled;
      notifyListeners();
    }
  }

  Future<void> setServiceLaunchAutoStart(bool enabled) async {
    final operationId =
        'preference-service-${DateTime.now().toUtc().microsecondsSinceEpoch}';
    try {
      final snapshot = await _launchAutoStartPreferenceStore.update(
        serviceEnabled: enabled,
        operationId: operationId,
        logger: _addLifecycleLog,
      );
      _applyLaunchAutoStartSnapshot(snapshot);
      _servicePreferenceError = null;
      notifyListeners();
      if (enabled) {
        unawaited(ensureAutoStart(source: 'settings_autostart_enable'));
      }
    } catch (_) {
      _servicePreferenceError = 'Auto-start preference could not be saved.';
      notifyListeners();
    }
  }

  Future<void> setGatewayLaunchAutoStart(bool enabled) async {
    final operationId =
        'preference-gateway-${DateTime.now().toUtc().microsecondsSinceEpoch}';
    try {
      final snapshot = await _launchAutoStartPreferenceStore.update(
        gatewayEnabled: enabled,
        operationId: operationId,
        logger: _addLifecycleLog,
      );
      _applyLaunchAutoStartSnapshot(snapshot);
      _gatewayPreferenceError = null;
      notifyListeners();
      if (enabled) {
        unawaited(ensureAutoStart(source: 'settings_autostart_enable'));
      }
    } catch (_) {
      _gatewayPreferenceError = 'Auto-start preference could not be saved.';
      notifyListeners();
    }
  }

  Future<AutoStartEvaluation?> ensureAutoStart({required String source}) async {
    if (!Platform.isAndroid) return null;
    try {
      // Refresh the native manual-stop marker before evaluating a resume.
      // This closes the notification-stop -> app-resume race without polling
      // aggressively or relying on a stale Flutter snapshot.
      await _syncNativeServiceStatus(includeGateway: false);
      await _reloadLaunchAutoStartPreferences();
    } catch (_) {
      _addLifecycleLog('autostart.evaluate', {
        'operation_id':
            'preference-${DateTime.now().toUtc().microsecondsSinceEpoch}',
        'reason': 'preference_load_failed',
        'source': source,
        'service_autostart': _serviceLaunchAutoStart,
        'gateway_autostart': _gatewayLaunchAutoStart,
        'service_state': status.name,
        'gateway_state': _gatewayStatus.name,
        'preference_source': _launchAutoStartPreferenceSource,
        'result': 'failed',
      });
      return null;
    }
    final evaluation = await _autoStartCoordinator.ensureForAppOpen(
      preferences: AutoStartPreferences(
        serviceEnabled: _serviceLaunchAutoStart,
        gatewayEnabled: _gatewayLaunchAutoStart,
      ),
      source: source,
      preferenceSource: _launchAutoStartPreferenceSource,
      manualStopActive: _manualStopActive,
    );
    if (evaluation.error.isNotEmpty) {
      _serviceStartError = 'Automatic startup could not be evaluated.';
      notifyListeners();
    }
    return evaluation;
  }

  Future<void> _reloadLaunchAutoStartPreferences({bool notify = true}) async {
    final snapshot = await _launchAutoStartPreferenceStore.load();
    final changed =
        snapshot.preferences.serviceEnabled != _serviceLaunchAutoStart ||
        snapshot.preferences.gatewayEnabled != _gatewayLaunchAutoStart ||
        snapshot.source != _launchAutoStartPreferenceSource;
    _applyLaunchAutoStartSnapshot(snapshot);
    if (notify && changed) notifyListeners();
  }

  void _applyLaunchAutoStartSnapshot(
    LaunchAutoStartPreferenceSnapshot snapshot,
  ) {
    _serviceLaunchAutoStart = snapshot.preferences.serviceEnabled;
    _gatewayLaunchAutoStart = snapshot.preferences.gatewayEnabled;
    _launchAutoStartPreferenceSource = snapshot.source;
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
      case DeviceFeedbackProvider.firebase:
        return _firebaseReporter.collectSafeDeviceInfo();
      case DeviceFeedbackProvider.umeng:
        return _umengReporter.collectSafeDeviceInfo();
      case DeviceFeedbackProvider.none:
        return const {};
    }
  }

  Future<bool> isDeviceFeedbackAllowed() async {
    switch (_deviceFeedbackProvider) {
      case DeviceFeedbackProvider.firebase:
        return _firebaseReporter.isUploadAllowed();
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
      DeviceFeedbackProvider.firebase => _firebaseReporter.shouldUpload(),
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
      case DeviceFeedbackProvider.firebase:
        await _firebaseReporter.setUploadAllowed(allowed);
        return;
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
      case DeviceFeedbackProvider.firebase:
        if (_firebaseProjectId.trim().isEmpty) {
          result = const DeviceFeedbackUploadResult(
            success: false,
            message:
                'Missing PICOCLAW_FIREBASE_PROJECT_ID build configuration.',
          );
          break;
        }
        if (_firebaseApiKey.trim().isEmpty) {
          result = const DeviceFeedbackUploadResult(
            success: false,
            message: 'Missing PICOCLAW_FIREBASE_API_KEY build configuration.',
          );
          break;
        }
        if (_firebaseAppId.trim().isEmpty) {
          result = const DeviceFeedbackUploadResult(
            success: false,
            message: 'Missing PICOCLAW_FIREBASE_APP_ID build configuration.',
          );
          break;
        }
        if (_firebaseMessagingSenderId.trim().isEmpty) {
          result = const DeviceFeedbackUploadResult(
            success: false,
            message:
                'Missing PICOCLAW_FIREBASE_MESSAGING_SENDER_ID build configuration.',
          );
          break;
        }
        result = await _firebaseReporter.uploadDeviceReport(
          appId: _firebaseAppId,
          projectId: _firebaseProjectId,
          apiKey: _firebaseApiKey,
          messagingSenderId: _firebaseMessagingSenderId,
          storageBucket: _firebaseStorageBucket.isEmpty
              ? null
              : _firebaseStorageBucket,
          telemetrySnapshot: telemetrySnapshot,
        );
        break;
      case DeviceFeedbackProvider.umeng:
        if (_umengAppKey.trim().isEmpty) {
          result = const DeviceFeedbackUploadResult(
            success: false,
            message: 'Missing PICOCLAW_UMENG_APP_KEY build configuration.',
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

  Future<Map<String, String>> getFirebaseDeviceInfo() {
    return getDeviceFeedbackDeviceInfo();
  }

  Future<bool> isFirebaseUploadAllowed() {
    return isDeviceFeedbackAllowed();
  }

  Future<bool> shouldAskForFirebaseUpload() {
    return shouldAskForDeviceFeedbackUpload();
  }

  Future<bool> shouldAutoUploadFirebaseDeviceReport() {
    return shouldAutoUploadDeviceFeedbackReport();
  }

  Future<void> setFirebaseUploadAllowed(bool allowed) {
    return setDeviceFeedbackUploadAllowed(allowed);
  }

  Future<DeviceFeedbackUploadResult> uploadFirebaseDeviceReport() {
    return uploadDeviceFeedbackReport();
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
      final result = await PicoClawChannel.applyPublicMode(value);
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

  void _addLifecycleLog(String event, Map<String, Object?> metadata) {
    final safeMetadata = <String, Object?>{};
    for (final entry in metadata.entries) {
      final value = entry.value;
      safeMetadata[entry.key] = value is String
          ? PlainTextLogSanitizer.sanitize(value)
          : value;
    }
    _addLog(jsonEncode(<String, Object?>{'event': event, ...safeMetadata}));
  }

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

  String _buildLaunchArguments() {
    final tokens = _arguments
        .split(' ')
        .where((token) => token.isNotEmpty)
        .toList();
    if (_publicMode && !tokens.contains('-public')) tokens.add('-public');
    if (!tokens.contains('-no-browser')) tokens.add('-no-browser');
    if (!_gatewayLaunchAutoStart && !tokens.contains('-no-gateway-autostart')) {
      tokens.add('-no-gateway-autostart');
    }
    return tokens.join(' ');
  }

  @override
  Future<AutoStartRuntimeState> inspectServiceState() async {
    if (!Platform.isAndroid) {
      return _serviceRuntimeState;
    }
    try {
      final nativeStatus = await PicoClawChannel.getServiceStatus();
      final running = nativeStatus['isRunning'] as bool? ?? false;
      final starting = nativeStatus['isStarting'] as bool? ?? false;
      final stopping = nativeStatus['isStopping'] as bool? ?? false;
      final hasFailed = nativeStatus['hasFailed'] as bool? ?? false;
      _manualStopActive =
          nativeStatus['manualStopActive'] as bool? ?? _manualStopActive;
      if (stopping) {
        _status = ServiceStatus.stopping;
        return AutoStartRuntimeState.stopping;
      }
      if (running) {
        final health = await PicoClawChannel.checkHealth();
        if (health['isHealthy'] as bool? ?? false) {
          _status = ServiceStatus.running;
          _healthStatus = 'Healthy';
          _serviceStartError = null;
          return AutoStartRuntimeState.running;
        }
        _status = ServiceStatus.starting;
        return AutoStartRuntimeState.starting;
      }
      if (starting) {
        _status = ServiceStatus.starting;
        return AutoStartRuntimeState.starting;
      }
      if (hasFailed && !_manualStopActive) {
        _status = ServiceStatus.failed;
        return AutoStartRuntimeState.failed;
      }
      return _status == ServiceStatus.failed
          ? AutoStartRuntimeState.failed
          : AutoStartRuntimeState.stopped;
    } catch (_) {
      return _status == ServiceStatus.failed
          ? AutoStartRuntimeState.failed
          : AutoStartRuntimeState.stopped;
    }
  }

  AutoStartRuntimeState get _serviceRuntimeState => switch (_status) {
    ServiceStatus.running => AutoStartRuntimeState.running,
    ServiceStatus.starting => AutoStartRuntimeState.starting,
    ServiceStatus.stopping => AutoStartRuntimeState.stopping,
    ServiceStatus.failed => AutoStartRuntimeState.failed,
    ServiceStatus.stopped => AutoStartRuntimeState.stopped,
  };

  Future<bool> start({String source = 'manual'}) async {
    final operationId =
        '$source-${DateTime.now().toUtc().microsecondsSinceEpoch}';
    _manualStopActive = false;
    _addLifecycleLog('service.start.requested', {
      'operation_id': operationId,
      'source': source,
      'reason': 'explicit_request',
      'previous_state': _status.name,
      'target_state': ServiceStatus.running.name,
      'result': 'requested',
      'retry_count': 0,
    });
    final result = await ensureServiceReady(
      operationId: operationId,
      source: source,
    );
    return result.isReady;
  }

  @override
  Future<AutoStartTransitionResult> ensureServiceReady({
    required String operationId,
    required String source,
  }) {
    final current = _serviceStartTask;
    if (current != null) {
      _addLifecycleLog('service.start.skipped', {
        'operation_id': operationId,
        'source': source,
        'reason': 'transition_already_in_progress',
        'previous_state': _status.name,
        'target_state': ServiceStatus.running.name,
        'result': 'joined',
        'retry_count': 0,
      });
      return current;
    }
    final generation = ++_serviceTransitionGeneration;
    final task = _ensureServiceReadyInternal(
      operationId: operationId,
      source: source,
      generation: generation,
    );
    _serviceStartTask = task;
    task.whenComplete(() {
      if (identical(_serviceStartTask, task)) _serviceStartTask = null;
    });
    return task;
  }

  Future<AutoStartTransitionResult> _ensureServiceReadyInternal({
    required String operationId,
    required String source,
    required int generation,
  }) async {
    final stopwatch = Stopwatch()..start();
    var retries = 0;
    var actualStart = false;
    _serviceStartError = null;
    try {
      final previousState = await inspectServiceState();
      if (previousState == AutoStartRuntimeState.running) {
        return AutoStartTransitionResult(
          state: AutoStartRuntimeState.running,
          actualStart: false,
          reason: 'already_running',
          duration: stopwatch.elapsed,
        );
      }

      _status = ServiceStatus.starting;
      notifyListeners();
      if (previousState != AutoStartRuntimeState.starting) {
        _syncAdapterConfiguration();
        actualStart = true;
        final accepted = await _adapter.startService(
          port: _port,
          args: _buildLaunchArguments(),
          source: source,
          operationId: operationId,
        );
        if (!accepted) {
          final code = _adapter.getLastErrorCode() ?? 'core.start_failed';
          if (isCurrentOperationGenerationForTest(
            operationGeneration: generation,
            currentGeneration: _serviceTransitionGeneration,
          )) {
            _serviceStartError = 'PocketClaw service could not be started.';
            _lastErrorCode = code;
            _status = ServiceStatus.failed;
            notifyListeners();
          }
          return AutoStartTransitionResult(
            state: AutoStartRuntimeState.failed,
            actualStart: true,
            reason: 'start_rejected',
            error: code,
            duration: stopwatch.elapsed,
          );
        }
        if (!isCurrentOperationGenerationForTest(
          operationGeneration: generation,
          currentGeneration: _serviceTransitionGeneration,
        )) {
          return AutoStartTransitionResult(
            state: _serviceRuntimeState,
            actualStart: true,
            reason: _status == ServiceStatus.running
                ? 'superseded_by_runtime_success'
                : 'cancelled_by_stop',
            duration: stopwatch.elapsed,
          );
        }
        _addLog('Starting PocketClaw service...');
      }

      if (!Platform.isAndroid) {
        _status = ServiceStatus.running;
        _addLog('Service started on $webUrl');
        notifyListeners();
        return AutoStartTransitionResult(
          state: AutoStartRuntimeState.running,
          actualStart: actualStart,
          reason: 'ready',
          duration: stopwatch.elapsed,
        );
      }

      while (stopwatch.elapsed < _serviceStartTimeout) {
        if (!isCurrentOperationGenerationForTest(
          operationGeneration: generation,
          currentGeneration: _serviceTransitionGeneration,
        )) {
          return AutoStartTransitionResult(
            state: _serviceRuntimeState,
            actualStart: actualStart,
            reason: _status == ServiceStatus.running
                ? 'superseded_by_runtime_success'
                : 'cancelled_by_stop',
            duration: stopwatch.elapsed,
            retryCount: retries,
          );
        }
        final nativeStatus = await PicoClawChannel.getServiceStatus();
        if (nativeStatus['isRunning'] as bool? ?? false) {
          try {
            final health = await PicoClawChannel.checkHealth();
            if (health['isHealthy'] as bool? ?? false) {
              _status = ServiceStatus.running;
              _healthStatus = 'Healthy';
              _serviceStartError = null;
              _nativePid = health['pid'] as int? ?? _nativePid;
              notifyListeners();
              return AutoStartTransitionResult(
                state: AutoStartRuntimeState.running,
                actualStart: actualStart,
                reason: 'health_check_ready',
                duration: stopwatch.elapsed,
                retryCount: retries,
              );
            }
          } catch (_) {
            // The bounded readiness poll records one truthful timeout below.
          }
        }
        retries += 1;
        await Future<void>.delayed(_readinessPollInterval);
      }

      if (!isCurrentOperationGenerationForTest(
        operationGeneration: generation,
        currentGeneration: _serviceTransitionGeneration,
      )) {
        return AutoStartTransitionResult(
          state: _serviceRuntimeState,
          actualStart: actualStart,
          reason: 'superseded_by_newer_operation',
          duration: stopwatch.elapsed,
          retryCount: retries,
        );
      }
      final finalState = await inspectServiceState();
      if (finalState == AutoStartRuntimeState.running) {
        return AutoStartTransitionResult(
          state: finalState,
          actualStart: actualStart,
          reason: 'ready_at_timeout_boundary',
          duration: stopwatch.elapsed,
          retryCount: retries,
        );
      }
      if (isCurrentOperationGenerationForTest(
        operationGeneration: generation,
        currentGeneration: _serviceTransitionGeneration,
      )) {
        _serviceStartError = 'PocketClaw service startup timed out. Try again.';
        _status = ServiceStatus.failed;
        notifyListeners();
      }
      return AutoStartTransitionResult(
        state: AutoStartRuntimeState.failed,
        actualStart: actualStart,
        reason: 'health_check',
        error: 'service readiness timed out',
        timedOut: true,
        duration: stopwatch.elapsed,
        retryCount: retries,
        timeout: _serviceStartTimeout,
      );
    } catch (_) {
      if (isCurrentOperationGenerationForTest(
        operationGeneration: generation,
        currentGeneration: _serviceTransitionGeneration,
      )) {
        _serviceStartError = 'PocketClaw service could not be started.';
        _status = ServiceStatus.failed;
        notifyListeners();
      }
      return AutoStartTransitionResult(
        state: AutoStartRuntimeState.failed,
        actualStart: actualStart,
        reason: 'start_exception',
        error: 'service start failed',
        duration: stopwatch.elapsed,
        retryCount: retries,
      );
    }
  }

  @override
  Future<AutoStartRuntimeState> inspectGatewayState() async {
    if (!Platform.isAndroid) {
      return _gatewayStatus;
    }
    if (_status != ServiceStatus.running) {
      _gatewayStatus = _status == ServiceStatus.stopping
          ? AutoStartRuntimeState.stopping
          : AutoStartRuntimeState.stopped;
      return _gatewayStatus;
    }
    try {
      final status = await PicoClawChannel.getGatewayStatus();
      _gatewayStatus = switch (status['gateway_status'] as String? ?? '') {
        'running' => AutoStartRuntimeState.running,
        'starting' || 'restarting' => AutoStartRuntimeState.starting,
        'stopping' => AutoStartRuntimeState.stopping,
        'error' => AutoStartRuntimeState.failed,
        _ => AutoStartRuntimeState.stopped,
      };
      if (_gatewayStatus == AutoStartRuntimeState.running ||
          _gatewayStatus == AutoStartRuntimeState.stopped) {
        _gatewayStartError = null;
      }
      notifyListeners();
      return _gatewayStatus;
    } catch (_) {
      return _gatewayStatus;
    }
  }

  @override
  Future<AutoStartTransitionResult> ensureGatewayReady({
    required String operationId,
    required String source,
  }) {
    final current = _gatewayStartTask;
    if (current != null) return current;
    final generation = ++_gatewayTransitionGeneration;
    final task = _ensureGatewayReadyInternal(
      operationId: operationId,
      source: source,
      generation: generation,
    );
    _gatewayStartTask = task;
    task.whenComplete(() {
      if (identical(_gatewayStartTask, task)) _gatewayStartTask = null;
    });
    return task;
  }

  Future<AutoStartTransitionResult> _ensureGatewayReadyInternal({
    required String operationId,
    required String source,
    required int generation,
  }) async {
    final stopwatch = Stopwatch()..start();
    var retries = 0;
    var actualStart = false;
    _gatewayStartError = null;
    try {
      if (await inspectServiceState() != AutoStartRuntimeState.running) {
        if (isCurrentOperationGenerationForTest(
          operationGeneration: generation,
          currentGeneration: _gatewayTransitionGeneration,
        )) {
          _gatewayStartError = 'Start the PocketClaw service first.';
          _gatewayStatus = AutoStartRuntimeState.failed;
          notifyListeners();
        }
        return AutoStartTransitionResult(
          state: AutoStartRuntimeState.failed,
          actualStart: false,
          reason: 'service_not_ready',
          error: 'required service context is unavailable',
          duration: stopwatch.elapsed,
        );
      }

      final previousState = await inspectGatewayState();
      if (previousState == AutoStartRuntimeState.running) {
        return AutoStartTransitionResult(
          state: AutoStartRuntimeState.running,
          actualStart: false,
          reason: 'already_running',
          duration: stopwatch.elapsed,
        );
      }
      _gatewayStatus = AutoStartRuntimeState.starting;
      notifyListeners();
      if (previousState != AutoStartRuntimeState.starting) {
        actualStart = true;
        final response = await PicoClawChannel.startGateway();
        final responseStatus = response['status'] as String? ?? '';
        if (responseStatus == 'precondition_failed') {
          if (isCurrentOperationGenerationForTest(
            operationGeneration: generation,
            currentGeneration: _gatewayTransitionGeneration,
          )) {
            _gatewayStartError =
                response['message'] as String? ??
                'Gateway is not ready to start.';
            _gatewayStatus = AutoStartRuntimeState.failed;
            notifyListeners();
          }
          return AutoStartTransitionResult(
            state: AutoStartRuntimeState.failed,
            actualStart: true,
            reason: 'precondition_failed',
            error: _gatewayStartError!,
            duration: stopwatch.elapsed,
          );
        }
      }

      while (stopwatch.elapsed < _gatewayStartTimeout) {
        if (!isCurrentOperationGenerationForTest(
          operationGeneration: generation,
          currentGeneration: _gatewayTransitionGeneration,
        )) {
          return AutoStartTransitionResult(
            state: _gatewayStatus,
            actualStart: actualStart,
            reason: _gatewayStatus == AutoStartRuntimeState.running
                ? 'superseded_by_runtime_success'
                : 'cancelled_by_service_stop',
            duration: stopwatch.elapsed,
            retryCount: retries,
          );
        }
        final state = await inspectGatewayState();
        if (state == AutoStartRuntimeState.running) {
          _gatewayStartError = null;
          return AutoStartTransitionResult(
            state: state,
            actualStart: actualStart,
            reason: 'health_check_ready',
            duration: stopwatch.elapsed,
            retryCount: retries,
          );
        }
        if (state == AutoStartRuntimeState.failed) {
          if (isCurrentOperationGenerationForTest(
            operationGeneration: generation,
            currentGeneration: _gatewayTransitionGeneration,
          )) {
            _gatewayStartError = 'Gateway startup failed. Try again.';
            notifyListeners();
          }
          return AutoStartTransitionResult(
            state: state,
            actualStart: actualStart,
            reason: 'health_check_failed',
            error: 'gateway reported an error state',
            duration: stopwatch.elapsed,
            retryCount: retries,
          );
        }
        retries += 1;
        await Future<void>.delayed(_readinessPollInterval);
      }

      if (!isCurrentOperationGenerationForTest(
        operationGeneration: generation,
        currentGeneration: _gatewayTransitionGeneration,
      )) {
        return AutoStartTransitionResult(
          state: _gatewayStatus,
          actualStart: actualStart,
          reason: 'superseded_by_newer_operation',
          duration: stopwatch.elapsed,
          retryCount: retries,
        );
      }
      final finalState = await inspectGatewayState();
      if (finalState == AutoStartRuntimeState.running) {
        return AutoStartTransitionResult(
          state: finalState,
          actualStart: actualStart,
          reason: 'ready_at_timeout_boundary',
          duration: stopwatch.elapsed,
          retryCount: retries,
        );
      }
      if (isCurrentOperationGenerationForTest(
        operationGeneration: generation,
        currentGeneration: _gatewayTransitionGeneration,
      )) {
        _gatewayStatus = AutoStartRuntimeState.failed;
        _gatewayStartError = 'Gateway startup timed out. Try again.';
        notifyListeners();
      }
      return AutoStartTransitionResult(
        state: AutoStartRuntimeState.failed,
        actualStart: actualStart,
        reason: 'health_check',
        error: 'gateway readiness timed out',
        timedOut: true,
        duration: stopwatch.elapsed,
        retryCount: retries,
        timeout: _gatewayStartTimeout,
      );
    } catch (_) {
      if (isCurrentOperationGenerationForTest(
        operationGeneration: generation,
        currentGeneration: _gatewayTransitionGeneration,
      )) {
        _gatewayStatus = AutoStartRuntimeState.failed;
        _gatewayStartError = 'Gateway could not be started.';
        notifyListeners();
      }
      return AutoStartTransitionResult(
        state: AutoStartRuntimeState.failed,
        actualStart: actualStart,
        reason: 'start_exception',
        error: 'gateway start failed',
        duration: stopwatch.elapsed,
        retryCount: retries,
      );
    }
  }

  Future<void> stop({String source = 'manual'}) async {
    final operationId =
        'stop-$source-${DateTime.now().toUtc().microsecondsSinceEpoch}';
    final stopwatch = Stopwatch()..start();
    final previousState = _status;
    if (source == 'manual' || source == 'notification') {
      _manualStopActive = true;
    }
    _serviceTransitionGeneration += 1;
    _gatewayTransitionGeneration += 1;
    _status = ServiceStatus.stopping;
    _gatewayStatus = AutoStartRuntimeState.stopping;
    _serviceStartError = null;
    _gatewayStartError = null;
    _addLifecycleLog('service.stop.requested', {
      'operation_id': operationId,
      'source': source,
      'reason': source == 'manual'
          ? 'explicit_user_request'
          : 'explicit_runtime_request',
      'previous_state': previousState.name,
      'target_state': ServiceStatus.stopped.name,
      'result': 'requested',
      'retry_count': 0,
      'timeout_ms': _serviceStopTimeout.inMilliseconds,
    });
    notifyListeners();
    try {
      final accepted = await _adapter.stopService(
        source: source,
        operationId: operationId,
      );
      if (!accepted) throw StateError('Native Service rejected stop request');
      if (Platform.isAndroid) {
        var nativeStopped = false;
        while (stopwatch.elapsed < _serviceStopTimeout) {
          final remaining = _serviceStopTimeout - stopwatch.elapsed;
          if (remaining <= Duration.zero) break;
          final native = await PicoClawChannel.getServiceStatus().timeout(
            remaining,
          );
          nativeStopped =
              !(native['isRunning'] as bool? ?? false) &&
              !(native['isStarting'] as bool? ?? false) &&
              !(native['isStopping'] as bool? ?? false);
          if (nativeStopped) break;
          await Future<void>.delayed(_readinessPollInterval);
        }
        if (!nativeStopped) {
          throw TimeoutException('Native Service stop timed out');
        }
      }
      _status = ServiceStatus.stopped;
      _gatewayStatus = AutoStartRuntimeState.stopped;
      _addLog('Stopping PocketClaw service...');
      _addLifecycleLog('service.stop.completed', {
        'operation_id': operationId,
        'source': source,
        'reason': 'native_service_stopped',
        'duration_ms': stopwatch.elapsedMilliseconds,
        'previous_state': previousState.name,
        'target_state': ServiceStatus.stopped.name,
        'result': 'completed',
        'retry_count': 0,
        'timeout_ms': _serviceStopTimeout.inMilliseconds,
      });
      notifyListeners();
    } catch (e) {
      _status = ServiceStatus.failed;
      _serviceStartError = 'PocketClaw service could not be stopped.';
      _addLifecycleLog('service.stop.failed', {
        'operation_id': operationId,
        'source': source,
        'reason': e is TimeoutException ? 'timeout' : 'native_stop_failed',
        'duration_ms': stopwatch.elapsedMilliseconds,
        'previous_state': previousState.name,
        'target_state': ServiceStatus.stopped.name,
        'result': 'failed',
        'retry_count': 0,
        'timeout_ms': _serviceStopTimeout.inMilliseconds,
      });
      notifyListeners();
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

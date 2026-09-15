import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

/// The canonical launch auto-start record, as committed by the Android host.
///
/// Flutter never holds a second copy of this state: every read and write goes
/// through the native store, and the values below are always a post-commit
/// readback rather than the values that were requested.
class LaunchAutoStartPreferences {
  const LaunchAutoStartPreferences({
    required this.serviceEnabled,
    required this.gatewayEnabled,
    required this.initialized,
  });

  /// Fresh installations start with both components enabled.
  static const defaults = LaunchAutoStartPreferences(
    serviceEnabled: true,
    gatewayEnabled: true,
    initialized: false,
  );

  final bool serviceEnabled;
  final bool gatewayEnabled;

  /// False until the user has saved a choice at least once.
  final bool initialized;
}

class PublicModeApplyResult {
  const PublicModeApplyResult({
    required this.success,
    required this.publicMode,
    required this.message,
  });

  final bool success;
  final bool publicMode;
  final String message;
}

/// PocketClaw 原生 MethodChannel 客户端。
/// 仅在 Android 平台可用，用于与 Kotlin 原生服务层通信。
class PocketClawChannel {
  static const _channel = MethodChannel('com.lord1egypt.pocketclaw/pocketclaw');

  /// 启动 PocketClaw 前台服务
  static Future<bool> startService({int port = 18800, String args = ''}) async {
    final result = await _channel.invokeMethod<bool>('startService', {
      'port': port,
      'args': args,
    });
    return result ?? false;
  }

  /// 停止 PocketClaw 前台服务
  static Future<bool> stopService() async {
    final result = await _channel.invokeMethod<bool>('stopService');
    return result ?? false;
  }

  /// Rebinds only the authenticated Dashboard listener in the already-running
  /// Android launcher. The managed Core process is not restarted.
  static Future<PublicModeApplyResult> applyPublicMode(bool publicMode) async {
    final result = await _channel.invokeMethod<Map>('applyPublicMode', {
      'public': publicMode,
    });
    if (result == null) {
      return PublicModeApplyResult(
        success: false,
        publicMode: !publicMode,
        message: 'No response while applying network mode.',
      );
    }
    final mapped = Map<String, dynamic>.from(result);
    return PublicModeApplyResult(
      success: mapped['success'] as bool? ?? false,
      publicMode: mapped['public'] as bool? ?? !publicMode,
      message: mapped['message'] as String? ?? '',
    );
  }

  /// 获取服务状态
  static Future<Map<String, dynamic>> getServiceStatus() async {
    final result = await _channel.invokeMethod<Map>('getServiceStatus');
    if (result == null) return {'isRunning': false, 'pid': -1, 'lastLog': ''};
    return Map<String, dynamic>.from(result);
  }

  /// 检查 /health 端点
  static Future<Map<String, dynamic>> checkHealth({bool detail = false}) async {
    final result = await _channel.invokeMethod<Map>('checkHealth', {
      'detail': detail,
    });
    if (result == null) {
      return {'isHealthy': false, 'error': 'No response'};
    }
    return Map<String, dynamic>.from(result);
  }

  /// Reads the canonical launch auto-start record from the Android host.
  static Future<LaunchAutoStartPreferences>
  getLaunchAutoStartPreferences() async {
    final result = await _channel.invokeMethod<Map>(
      'getLaunchAutoStartPreferences',
    );
    return _mapLaunchAutoStart(result);
  }

  /// Commits a launch auto-start change and returns the native readback.
  ///
  /// Omitted fields are left untouched by the host, so a single toggle never
  /// rewrites the other component's preference.
  static Future<LaunchAutoStartPreferences> setLaunchAutoStartPreferences({
    bool? serviceEnabled,
    bool? gatewayEnabled,
  }) async {
    final result = await _channel.invokeMethod<Map>(
      'setLaunchAutoStartPreferences',
      <String, Object?>{
        'serviceEnabled': ?serviceEnabled,
        'gatewayEnabled': ?gatewayEnabled,
      },
    );
    return _mapLaunchAutoStart(result);
  }

  static LaunchAutoStartPreferences _mapLaunchAutoStart(
    Map<dynamic, dynamic>? r,
  ) {
    return LaunchAutoStartPreferences(
      serviceEnabled: r?['serviceEnabled'] as bool? ?? true,
      gatewayEnabled: r?['gatewayEnabled'] as bool? ?? true,
      initialized: r?['initialized'] as bool? ?? false,
    );
  }

  /// 读取 config.json 内容
  static Future<String> getConfig() async {
    final result = await _channel.invokeMethod<String>('getConfig');
    return result ?? '';
  }

  /// 保存 config.json 内容
  static Future<bool> saveConfig(String content) async {
    final result = await _channel.invokeMethod<bool>('saveConfig', {
      'content': content,
    });
    return result ?? false;
  }

  /// 获取完整日志
  /// Log lines emitted since the previous call, each delivered exactly once.
  ///
  /// Deliberately not `getServiceStatus`'s `lastLog`: that is a sticky
  /// snapshot, and polling it appended the same line on every tick.
  static Future<List<String>> takeNewLogs() async {
    final result = await _channel.invokeMethod<List<Object?>>('takeNewLogs');
    if (result == null) return const <String>[];
    return result.whereType<String>().toList(growable: false);
  }

  /// Persists Telegram credentials through Core so config.json and
  /// .security.yml are updated together by Core's normal SaveConfig path.
  /// Whether the Dashboard already has an owner. PC-DEF-040.
  static Future<bool> dashboardAuthInitialized() async =>
      await _channel.invokeMethod<bool>('dashboardAuthInitialized') ?? false;

  /// Asks the host to start the Gateway now, over the loopback Android
  /// bridge. Returns Core's status: "ok" or "already_running".
  ///
  /// The bridge token stays in the host process. Flutter requests the
  /// operation and never handles the credential. See PC-DEF-034.
  static Future<String> startGatewayNow() async {
    final status = await _channel.invokeMethod<String>('startGatewayNow');
    return status ?? 'ok';
  }

  static Future<bool> configureTelegram({
    required String token,
    required int ownerUserId,
  }) async {
    final result = await _channel.invokeMethod<bool>('configureTelegram', {
      'token': token,
      'ownerUserId': ownerUserId,
    });
    return result ?? false;
  }

  /// The Telegram context-memory setting, as Core will actually apply it.
  ///
  /// Core owns `config.json` and answers with the effective value, so Settings
  /// never parses or writes that file and there is only ever one writer.
  static Future<TelegramContextMemory> getTelegramContextMemory() async {
    final result = await _channel.invokeMapMethod<String, dynamic>(
      'getTelegramContextMemory',
    );
    return TelegramContextMemory.fromMap(result);
  }

  /// Stores a new Telegram context-memory limit through Core.
  ///
  /// Core validates the range and rejects anything outside it, and returns the
  /// value it stored. Nothing else in the configuration is touched, and no
  /// message, transcript or summary is removed.
  static Future<TelegramContextMemory> setTelegramContextMemory(
    int recentMessages,
  ) async {
    final result = await _channel.invokeMapMethod<String, dynamic>(
      'setTelegramContextMemory',
      {'recentMessages': recentMessages},
    );
    return TelegramContextMemory.fromMap(result);
  }

  /// Whether PocketClaw holds a GitHub credential, and for which account.
  ///
  /// The token itself is never returned across this channel. It is decrypted
  /// only by the Android service, when it starts Core, and reaches gh and git
  /// as an environment variable from there.
  static Future<GitHubConnection> getGitHubStatus() async {
    final result = await _channel.invokeMapMethod<String, dynamic>(
      'getGitHubStatus',
    );
    return GitHubConnection.fromMap(result);
  }

  /// Validates a candidate token through Core's bundled gh, then stores it
  /// encrypted under an Android Keystore key.
  ///
  /// Nothing is stored unless GitHub accepts the token. On failure the
  /// PlatformException message is Core's own, with the candidate scrubbed out.
  static Future<GitHubConnection> connectGitHub(String token) async {
    final result = await _channel.invokeMapMethod<String, dynamic>(
      'connectGitHub',
      {'token': token},
    );
    return GitHubConnection.fromMap(result);
  }

  /// Asks the running Core whether its GitHub credential authenticates, which
  /// is the same path an agent `gh` call takes.
  static Future<String> testGitHubConnection() async {
    final result = await _channel.invokeMapMethod<String, dynamic>(
      'testGitHubConnection',
    );
    return (result?['login'] as String?)?.trim() ?? '';
  }

  /// Removes the stored credential. Other secrets are untouched.
  static Future<bool> disconnectGitHub() async {
    final result = await _channel.invokeMethod<bool>('disconnectGitHub');
    return result ?? false;
  }

  /// Opens the Android photo picker and returns the chosen image as a URI
  /// string, or null when the user cancelled.
  ///
  /// [acceptTypes] are the MIME types the requesting page asked for; an empty
  /// list means "any image". The returned URI is readable by this process for
  /// as long as it lives, which is what the WebView needs to read the file.
  static Future<String?> pickChatImage({
    List<String> acceptTypes = const <String>[],
  }) async {
    return _channel.invokeMethod<String>('pickChatImage', {
      'acceptTypes': acceptTypes,
    });
  }

  static Future<String> getFullLog() async {
    final result = await _channel.invokeMethod<String>('getFullLog');
    return result ?? '';
  }

  /// 设置开机自启
  static Future<bool> setAutoStart(bool enabled) async {
    final result = await _channel.invokeMethod<bool>('setAutoStart', {
      'enabled': enabled,
    });
    return result ?? false;
  }

  /// 获取开机自启设置
  static Future<bool> getAutoStart() async {
    final result = await _channel.invokeMethod<bool>('getAutoStart');
    return result ?? false;
  }

  /// The Core runtime version, or null when the host could not read it.
  ///
  /// PC-DEF-063. This used to answer 'unknown' for both "the probe failed" and
  /// "there is no version", and the caller cached that string as the version --
  /// so one transient failure showed as the Core version until something
  /// re-probed. A failure is now an absence, which cannot be cached as a value.
  /// The literal is still mapped here because the host reads it out of a binary
  /// whose output this cannot assume.
  static Future<String?> getCoreVersion() async {
    try {
      final result = await _channel.invokeMethod<String>('getCoreVersion');
      final value = result?.trim() ?? '';
      if (value.isEmpty || value.toLowerCase() == 'unknown') return null;
      return value;
    } catch (_) {
      return null;
    }
  }

  /// 获取 config.json 文件路径
  static Future<String> getConfigPath() async {
    final result = await _channel.invokeMethod<String>('getConfigPath');
    return result ?? '';
  }

  /// Returns a usable IPv4 address from Android's active Wi-Fi or Ethernet
  /// network. A null result means there is currently no LAN destination that
  /// another device should be told to open.
  static Future<String?> getLanIpv4Address() async {
    final result = await _channel.invokeMethod<String>('getLanIpv4Address');
    final address = result?.trim() ?? '';
    return address.isEmpty ? null : address;
  }

  /// 获取 PocketClaw Channel token
  static Future<String> getPocketClawToken() async {
    final result = await _channel.invokeMethod<String>('getPocketClawToken');
    return result ?? '';
  }

  /// 获取安全的设备信息（避免敏感标识符）
  static Future<Map<String, String>> getSafeDeviceInfo() async {
    final result = await _channel.invokeMethod<Map>('getSafeDeviceInfo');
    if (result == null) return const {};
    return result.map(
      (key, value) => MapEntry(key.toString(), value?.toString() ?? ''),
    );
  }

  static Future<bool> setUmengAnalyticsConsent(bool enabled) async {
    final result = await _channel.invokeMethod<bool>(
      'setUmengAnalyticsConsent',
      {'enabled': enabled},
    );
    return result ?? false;
  }

  static Future<Map<String, dynamic>> uploadUmengDeviceReport(
    Map<String, Object?> payload,
  ) async {
    debugPrint('[PocketClawChannel] === uploadUmengDeviceReport START ===');
    debugPrint(
      '[PocketClawChannel] Calling native method with payload keys: ${payload.keys.toList()}',
    );

    try {
      final result = await _channel
          .invokeMethod<Map>('uploadUmengDeviceReport', payload)
          .timeout(const Duration(seconds: 8));

      debugPrint('[PocketClawChannel] Native method returned');

      if (result == null) {
        debugPrint('[PocketClawChannel] ERROR: Native returned null');
        return const {
          'success': false,
          'message': 'No response from native Umeng bridge.',
        };
      }

      final mappedResult = Map<String, dynamic>.from(result);
      debugPrint(
        '[PocketClawChannel] Result: success=${mappedResult['success']}, message=${mappedResult['message']}',
      );
      debugPrint('[PocketClawChannel] === uploadUmengDeviceReport END ===');
      return mappedResult;
    } catch (e) {
      debugPrint('[PocketClawChannel] ERROR: Exception caught: $e');
      debugPrint('[PocketClawChannel] === uploadUmengDeviceReport FAILED ===');
      return {'success': false, 'message': 'Exception: $e'};
    }
  }

  /// 检查存储权限是否已授予（Android 11+）
  static Future<bool> isStorageManagerGranted() async {
    final result = await _channel.invokeMethod<bool>('isStorageManagerGranted');
    return result ?? false;
  }

  /// 请求存储管理权限（Android 11+）
  static Future<bool> requestStorageManager() async {
    final result = await _channel.invokeMethod<bool>('requestStorageManager');
    return result ?? false;
  }

  /// Reads PocketClaw's notification-permission state without asking for anything.
  ///
  /// PC-DEF-058. Safe to call on every Settings build: it shows the dialog for
  /// nothing.
  static Future<NotificationPermissionStatus> getNotificationPermission() async {
    try {
      final result = await _channel.invokeMethod<Map<Object?, Object?>>(
        'getNotificationPermission',
      );
      return NotificationPermissionStatus.fromMap(result);
    } on PlatformException {
      return NotificationPermissionStatus.unknown();
    } on MissingPluginException {
      // A host without this method — a desktop build, or an older APK.
      return NotificationPermissionStatus.unknown();
    }
  }

  /// Shows the standard Android notification-permission dialog, once.
  ///
  /// The host declines to ask a second time, so calling this when the answer is
  /// already known returns the current state and shows nothing.
  static Future<NotificationPermissionStatus>
      requestNotificationPermission() async {
    try {
      final result = await _channel.invokeMethod<Map<Object?, Object?>>(
        'requestNotificationPermission',
      );
      return NotificationPermissionStatus.fromMap(result);
    } on PlatformException {
      return NotificationPermissionStatus.unknown();
    } on MissingPluginException {
      return NotificationPermissionStatus.unknown();
    }
  }

  /// Opens the per-app notification settings, for someone who said no earlier.
  static Future<bool> openNotificationSettings() async {
    try {
      final result = await _channel.invokeMethod<bool>(
        'openNotificationSettings',
      );
      return result ?? false;
    } on PlatformException {
      return false;
    } on MissingPluginException {
      return false;
    }
  }
}

/// Whether PocketClaw may post notifications, and what to do about it.
///
/// PC-DEF-058. `POST_NOTIFICATIONS` was declared and never requested, so on a
/// fresh install the persistent "PocketClaw Running" notification never appeared
/// and the owner enabled it in Android Settings by hand.
enum NotificationPermissionState {
  /// This Android version has no runtime notification permission.
  notRequired,
  granted,

  /// Never asked on this install; the system dialog is the right next step.
  notRequested,

  /// Asked and refused. Only Settings can change it now.
  denied,

  /// The host did not answer — an older APK, or a platform without the method.
  unknown,
}

class NotificationPermissionStatus {
  const NotificationPermissionStatus({
    required this.state,
    required this.notificationsEnabled,
    required this.expectedVisible,
  });

  factory NotificationPermissionStatus.unknown() =>
      const NotificationPermissionStatus(
        state: NotificationPermissionState.unknown,
        notificationsEnabled: false,
        expectedVisible: false,
      );

  factory NotificationPermissionStatus.fromMap(Map<Object?, Object?>? map) {
    if (map == null) return NotificationPermissionStatus.unknown();
    return NotificationPermissionStatus(
      state: _stateFromWire(map['state']),
      notificationsEnabled: map['notificationsEnabled'] == true,
      expectedVisible: map['expectedVisible'] == true,
    );
  }

  final NotificationPermissionState state;

  /// Whether notifications are switched on for PocketClaw at all. On Android
  /// below 13 this is the only control there is, so it is a separate fact from
  /// [state].
  final bool notificationsEnabled;

  /// Whether the foreground notification should be visible.
  final bool expectedVisible;

  /// Whether the system dialog is worth showing. False once the answer is known,
  /// so first-run setup never nags.
  bool get shouldRequest =>
      state == NotificationPermissionState.notRequested;

  /// Whether Settings should offer an explicit recovery action.
  bool get shouldOfferSettings =>
      state == NotificationPermissionState.denied ||
      (!notificationsEnabled && state != NotificationPermissionState.unknown);

  static NotificationPermissionState _stateFromWire(Object? raw) {
    switch (raw) {
      case 'notRequired':
        return NotificationPermissionState.notRequired;
      case 'granted':
        return NotificationPermissionState.granted;
      case 'notRequested':
        return NotificationPermissionState.notRequested;
      case 'denied':
        return NotificationPermissionState.denied;
      default:
        return NotificationPermissionState.unknown;
    }
  }
}

/// What Settings is allowed to know about the GitHub credential: that there is
/// one, and whose account it belongs to. There is deliberately no field for the
/// token, so no screen can render it and no log can capture it.
class GitHubConnection {
  const GitHubConnection({required this.connected, this.login});

  final bool connected;
  final String? login;

  static GitHubConnection fromMap(Map<String, dynamic>? map) {
    if (map == null) return const GitHubConnection(connected: false);
    final login = (map['login'] as String?)?.trim();
    return GitHubConnection(
      connected: map['connected'] == true,
      login: login == null || login.isEmpty ? null : login,
    );
  }
}

/// The Telegram context-memory setting and the bounds Core enforces.
class TelegramContextMemory {
  const TelegramContextMemory({
    required this.recentMessages,
    required this.min,
    required this.max,
    required this.defaultValue,
  });

  /// Shipped fallbacks, used when Core has not answered yet. They mirror the
  /// values Core enforces; Core remains the authority and rejects anything
  /// outside its own range regardless of what is shown here.
  static const int fallbackDefault = 15;
  static const int fallbackMin = 5;
  static const int fallbackMax = 50;

  static const TelegramContextMemory unknown = TelegramContextMemory(
    recentMessages: fallbackDefault,
    min: fallbackMin,
    max: fallbackMax,
    defaultValue: fallbackDefault,
  );

  final int recentMessages;
  final int min;
  final int max;
  final int defaultValue;

  factory TelegramContextMemory.fromMap(Map<String, dynamic>? map) {
    if (map == null) return unknown;
    int read(String key, int fallback) {
      final value = map[key];
      if (value is int && value > 0) return value;
      if (value is num && value > 0) return value.toInt();
      return fallback;
    }

    return TelegramContextMemory(
      recentMessages: read('recentMessages', fallbackDefault),
      min: read('min', fallbackMin),
      max: read('max', fallbackMax),
      defaultValue: read('default', fallbackDefault),
    );
  }

  TelegramContextMemory copyWith({int? recentMessages}) =>
      TelegramContextMemory(
        recentMessages: recentMessages ?? this.recentMessages,
        min: min,
        max: max,
        defaultValue: defaultValue,
      );

  /// Whether [value] is one Core will accept.
  bool accepts(int value) => value >= min && value <= max;
}

import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

import 'launch_autostart_preferences.dart';

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

/// PicoClaw 原生 MethodChannel 客户端。
/// 仅在 Android 平台可用，用于与 Kotlin 原生服务层通信。
class PicoClawChannel {
  static const _channel = MethodChannel('com.lord1egypt.pocketclaw/picoclaw');

  /// 启动 PicoClaw 前台服务
  static Future<bool> startService({
    int port = 18800,
    String args = '',
    String source = 'manual',
    String operationId = '',
  }) async {
    final result = await _channel.invokeMethod<bool>('startService', {
      'port': port,
      'args': args,
      'source': source,
      'operationId': operationId,
    });
    return result ?? false;
  }

  /// 停止 PicoClaw 前台服务
  static Future<bool> stopService({
    String source = 'manual',
    String operationId = '',
  }) async {
    final result = await _channel.invokeMethod<bool>('stopService', {
      'source': source,
      'operationId': operationId,
    });
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
  static Future<Map<String, dynamic>> checkHealth() async {
    final result = await _channel.invokeMethod<Map>('checkHealth');
    if (result == null) {
      return {'isHealthy': false, 'error': 'No response'};
    }
    return Map<String, dynamic>.from(result);
  }

  /// Reads the managed Core Gateway state through the loopback-only,
  /// per-process authenticated Android bridge.
  static Future<Map<String, dynamic>> getGatewayStatus() async {
    final result = await _channel.invokeMethod<Map>('getGatewayStatus');
    if (result == null) return {'gateway_status': 'stopped'};
    return Map<String, dynamic>.from(result);
  }

  /// Requests an idempotent managed Core Gateway start. Core serializes this
  /// with Dashboard/manual starts and returns the existing transition when one
  /// is already underway.
  static Future<Map<String, dynamic>> startGateway() async {
    final result = await _channel.invokeMethod<Map>('startGateway');
    if (result == null) return {'status': 'failed'};
    return Map<String, dynamic>.from(result);
  }

  static Future<NativeLaunchAutoStartPreferences>
  getLaunchAutoStartPreferences() async {
    final result = await _channel.invokeMethod<Map>(
      'getLaunchAutoStartPreferences',
    );
    return _mapLaunchAutoStartPreferences(result);
  }

  static Future<NativeLaunchAutoStartPreferences>
  setLaunchAutoStartPreferences({
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
    return _mapLaunchAutoStartPreferences(result);
  }

  static NativeLaunchAutoStartPreferences _mapLaunchAutoStartPreferences(
    Map<dynamic, dynamic>? result,
  ) {
    return NativeLaunchAutoStartPreferences(
      serviceEnabled: result?['serviceEnabled'] as bool? ?? true,
      gatewayEnabled: result?['gatewayEnabled'] as bool? ?? true,
      initialized: result?['initialized'] as bool? ?? false,
      source: result?['source'] as String? ?? 'native_canonical',
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

  static Future<String> getCoreVersion() async {
    try {
      final result = await _channel.invokeMethod<String>('getCoreVersion');
      final value = result?.trim() ?? '';
      return value.isEmpty ? 'unknown' : value;
    } catch (_) {
      return 'unknown';
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

  /// 获取 Pico Channel token
  static Future<String> getPicoToken() async {
    final result = await _channel.invokeMethod<String>('getPicoToken');
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
    debugPrint('[PicoClawChannel] === uploadUmengDeviceReport START ===');
    debugPrint(
      '[PicoClawChannel] Calling native method with payload keys: ${payload.keys.toList()}',
    );

    try {
      final result = await _channel
          .invokeMethod<Map>('uploadUmengDeviceReport', payload)
          .timeout(const Duration(seconds: 8));

      debugPrint('[PicoClawChannel] Native method returned');

      if (result == null) {
        debugPrint('[PicoClawChannel] ERROR: Native returned null');
        return const {
          'success': false,
          'message': 'No response from native Umeng bridge.',
        };
      }

      final mappedResult = Map<String, dynamic>.from(result);
      debugPrint(
        '[PicoClawChannel] Result: success=${mappedResult['success']}, message=${mappedResult['message']}',
      );
      debugPrint('[PicoClawChannel] === uploadUmengDeviceReport END ===');
      return mappedResult;
    } catch (e) {
      debugPrint('[PicoClawChannel] ERROR: Exception caught: $e');
      debugPrint('[PicoClawChannel] === uploadUmengDeviceReport FAILED ===');
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
}

import 'dart:async';
import 'package:flutter/services.dart';

import '../core/pocketclaw_channel.dart';
import 'core_service_adapter.dart';

class AndroidCoreServiceAdapter implements CoreServiceAdapter {
  static const MethodChannel _channel = MethodChannel(
    'com.lord1egypt.pocketclaw/pocketclaw',
  );
  String? _lastErrorCode;

  @override
  Future<bool> startService({int? port, String? args}) async {
    try {
      final Map<String, Object?> params = {
        'port': port ?? 18800,
        'args': args ?? '',
      };
      final result = await _channel.invokeMethod<bool>('startService', params);
      return result ?? false;
    } catch (_) {
      _lastErrorCode = 'core.start_failed';
      return false;
    }
  }

  @override
  Future<bool> stopService() async {
    try {
      final result = await _channel.invokeMethod<bool>('stopService');
      return result ?? false;
    } catch (_) {
      _lastErrorCode = 'core.stop_failed';
      return false;
    }
  }

  @override
  Future<bool> restartService({int? port, String? args}) async {
    try {
      final Map<String, Object?> params = {
        'port': port ?? 18800,
        'args': args ?? '',
      };
      final result = await _channel.invokeMethod<bool>(
        'restartService',
        params,
      );
      return result ?? false;
    } catch (_) {
      _lastErrorCode = 'core.restart_failed';
      return false;
    }
  }

  @override
  Future<Map<String, dynamic>> getServiceStatus() async {
    final res = await _channel.invokeMethod<dynamic>('getServiceStatus');
    return Map<String, dynamic>.from(res as Map);
  }

  @override
  Future<Map<String, dynamic>> checkHealth({bool detail = false}) async {
    final res = await _channel.invokeMethod<dynamic>('checkHealth', {
      'detail': detail,
    });
    return Map<String, dynamic>.from(res as Map);
  }

  @override
  Future<String?> getCoreVersion() async {
    return PocketClawChannel.getCoreVersion();
  }

  @override
  String? getLastErrorCode() => _lastErrorCode;

  @override
  Future<String> getWorkspacePath() async {
    try {
      final result = await _channel.invokeMethod<String>('getHomePath');
      return result ?? '';
    } catch (_) {
      return '';
    }
  }
}

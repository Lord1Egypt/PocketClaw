import 'dart:async';
import 'package:flutter_background_service/flutter_background_service.dart';

/// 初始化 Flutter 后台服务。
/// 在 Android 上，主要的 Go 二进制由原生 PocketClawService 前台服务管理，
/// 此处的 flutter_background_service 仅作为辅助保活机制。
Future<void> initializeBackgroundService() async {
  final service = FlutterBackgroundService();

  // No channel is created here. This service is configured with
  // `autoStart: false` and `startService()` is never called, so the channel it
  // used to create — `picoclaw_foreground` — was only ever an empty entry in
  // the user's notification settings. The native side owns the one real
  // channel; see PocketClawNotificationChannels, which also deletes the dead
  // one on upgrade. The configuration below points at that channel so it stays
  // valid if this service is ever actually started.

  await service.configure(
    androidConfiguration: AndroidConfiguration(
      onStart: onStart,
      autoStart: false, // 不自动启动，由原生服务管理
      isForegroundMode: true,
      notificationChannelId: 'pocketclaw_service',
      initialNotificationTitle: 'PocketClaw',
      initialNotificationContent: 'PocketClaw service is running',
      foregroundServiceNotificationId: 888,
    ),
    iosConfiguration: IosConfiguration(
      autoStart: true,
      onForeground: onStart,
      onBackground: onIosBackground,
    ),
  );
}

@pragma('vm:entry-point')
Future<bool> onIosBackground(ServiceInstance service) async {
  return true;
}

@pragma('vm:entry-point')
void onStart(ServiceInstance service) async {
  if (service is AndroidServiceInstance) {
    service.on('setAsForeground').listen((event) {
      service.setAsForegroundService();
    });
    service.on('setAsBackground').listen((event) {
      service.setAsBackgroundService();
    });
  }

  service.on('stopService').listen((event) {
    service.stopSelf();
  });
}

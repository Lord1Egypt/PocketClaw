import 'dart:async';
import 'package:flutter_background_service/flutter_background_service.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:flutter/foundation.dart';

/// 初始化 Flutter 后台服务。
/// 在 Android 上，主要的 Go 二进制由原生 PicoClawService 前台服务管理，
/// 此处的 flutter_background_service 仅作为辅助保活机制。
Future<void> initializeBackgroundService() async {
  final service = FlutterBackgroundService();

  // 创建通知渠道（Android 需要）
  try {
    if (defaultTargetPlatform == TargetPlatform.android) {
      const AndroidNotificationChannel channel = AndroidNotificationChannel(
        'picoclaw_foreground',
        'PicoClaw Service',
        description: 'Keep the PicoClaw server running in the background.',
        importance: Importance.low,
      );

      final FlutterLocalNotificationsPlugin flutterLocalNotificationsPlugin =
          FlutterLocalNotificationsPlugin();

      await flutterLocalNotificationsPlugin
          .resolvePlatformSpecificImplementation<
            AndroidFlutterLocalNotificationsPlugin
          >()
          ?.createNotificationChannel(channel);
    }
  } catch (e) {
    // 通知初始化失败不影响服务启动
  }

  await service.configure(
    androidConfiguration: AndroidConfiguration(
      onStart: onStart,
      autoStart: false, // 不自动启动，由原生服务管理
      isForegroundMode: true,
      notificationChannelId: 'picoclaw_foreground',
      initialNotificationTitle: 'PicoClaw',
      initialNotificationContent: 'PicoClaw service is running',
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

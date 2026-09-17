import 'dart:collection';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_controller.dart';

// PC-DEF-061. A ready answer must name the polling generation it authorized,
// and that same generation must still be active on a second read. Otherwise a
// handoff could open against a receiver a restart has already replaced -- the
// owner that acknowledged the first /start and then went away.

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.lord1egypt.pocketclaw/pocketclaw');

  tearDown(() async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  void respondWith(Queue<Object?> responses) {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method != 'telegramReadiness') return null;
          return responses.isEmpty ? null : responses.removeFirst();
        });
  }

  test('a stable generation across two reads is ready', () async {
    respondWith(Queue.of([
      <String, Object?>{'state': 'ready', 'ready': true, 'generation': 7},
      <String, Object?>{'state': 'ready', 'ready': true, 'generation': 7},
    ]));

    expect(await defaultTelegramRuntimeReady(), isTrue);
  });

  test('ready without a generation is not permission to open the handoff', () async {
    respondWith(Queue.of([
      <String, Object?>{'state': 'ready', 'ready': true},
      <String, Object?>{'state': 'ready', 'ready': true},
    ]));

    expect(await defaultTelegramRuntimeReady(), isFalse);
  });

  test('a generation that changed between reads is not ready', () async {
    respondWith(Queue.of([
      <String, Object?>{'state': 'ready', 'ready': true, 'generation': 7},
      <String, Object?>{'state': 'ready', 'ready': true, 'generation': 8},
    ]));

    expect(await defaultTelegramRuntimeReady(), isFalse);
  });

  test('a generation that went away is not ready', () async {
    respondWith(Queue.of([
      <String, Object?>{'state': 'ready', 'ready': true, 'generation': 7},
      <String, Object?>{'state': 'gateway_starting', 'ready': false},
    ]));

    expect(await defaultTelegramRuntimeReady(), isFalse);
  });

  test('an unavailable authority is silence, not permission', () async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          throw PlatformException(code: 'TELEGRAM_READINESS_FAILED');
        });

    expect(await defaultTelegramRuntimeReady(), isFalse);
  });
}
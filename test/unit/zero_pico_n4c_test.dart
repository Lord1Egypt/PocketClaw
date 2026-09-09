import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// N4C: Android notification channel identity.
///
/// Channel ids are permanent — Android will not rename one — so removing the
/// Pico identity means creating a replacement, carrying across what the
/// platform exposes, and deleting the original. These guard the ordering and
/// the scope, since the Kotlin has no JVM test harness in this repository.
void main() {
  const kotlin = 'android/app/src/main/kotlin/com/lord1egypt/pocketclaw';
  final channels = File('$kotlin/PocketClawNotificationChannels.kt').readAsStringSync();
  final app = File('$kotlin/PocketClawApp.kt').readAsStringSync();
  final service = File('$kotlin/service/PocketClawService.kt').readAsStringSync();
  final background = File('lib/src/core/background_service.dart').readAsStringSync();

  group('no current code creates a legacy channel', () {
    test('picoclaw_service is never created', () {
      expect(app, isNot(contains('picoclaw_service')));
      expect(service, isNot(contains('picoclaw_service')));
      expect(background, isNot(contains('picoclaw_service')));
      // In the migration owner it may only be read and deleted.
      expect(channels, isNot(contains('createNotificationChannel(NotificationChannel(\n            LEGACY')));
    });

    test('picoclaw_foreground is never created', () {
      expect(background, isNot(contains("'picoclaw_foreground'")));
      expect(app, isNot(contains('picoclaw_foreground')));
      // Dart no longer creates any channel at all.
      expect(background, isNot(contains('createNotificationChannel')));
      expect(background, isNot(contains('AndroidNotificationChannel(')));
    });

    test('legacy ids appear only in the migration owner, marked as such', () {
      for (final entity in Directory(kotlin).listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.kt')) continue;
        if (entity.path.endsWith('PocketClawNotificationChannels.kt')) continue;
        final source = entity.readAsStringSync();
        for (final legacy in ['picoclaw_service', 'picoclaw_foreground']) {
          expect(source, isNot(contains(legacy)), reason: entity.path);
        }
      }
      expect('LEGACY READ-ONLY MIGRATION'.allMatches(channels).length, 2,
          reason: 'both legacy ids must be explicitly marked');
    });
  });

  group('canonical identity', () {
    test('the active channel id is PocketClaw-named', () {
      expect(channels, contains('const val SERVICE_CHANNEL_ID = "pocketclaw_service"'));
    });

    test('the app and the service both use the owner\'s constant', () {
      expect(app, contains('PocketClawNotificationChannels.SERVICE_CHANNEL_ID'));
      expect(service, contains('PocketClawApp.CHANNEL_ID'),
          reason: 'the notification builder resolves through the same constant');
    });

    test('the Flutter configuration points at the one real channel', () {
      expect(background, contains("notificationChannelId: 'pocketclaw_service'"));
    });
  });

  group('migration state machine', () {
    test('all four cases are decided in one pure function', () {
      expect(channels, contains('fun decideServiceChannelAction('));
      for (final action in [
        'CREATE_FRESH',
        'MIGRATE_FROM_LEGACY',
        'KEEP_CANONICAL_DELETE_LEGACY',
        'KEEP_CANONICAL',
      ]) {
        expect(channels, contains(action));
      }
    });

    test('old-only creates the replacement before deleting the original', () {
      final createAt = channels.indexOf('manager.createNotificationChannel(serviceChannelFrom(legacy!!))');
      final verifyAt = channels.indexOf('manager.getNotificationChannel(SERVICE_CHANNEL_ID) == null');
      final deleteAt = channels.indexOf('manager.deleteNotificationChannel(LEGACY_SERVICE_CHANNEL_ID)');
      expect(createAt, greaterThan(-1));
      expect(verifyAt, greaterThan(createAt),
          reason: 'the replacement must be verified to exist');
      expect(deleteAt, greaterThan(verifyAt),
          reason: 'the legacy channel is the only record of the user settings '
              'until the replacement exists, so it is deleted last');
    });

    test('both-present keeps canonical without overwriting it', () {
      final branch = channels.substring(
        channels.indexOf('ServiceChannelAction.KEEP_CANONICAL_DELETE_LEGACY -> {'),
        channels.indexOf('ServiceChannelAction.KEEP_CANONICAL -> Unit'),
      );
      expect(branch, contains('deleteNotificationChannel(LEGACY_SERVICE_CHANNEL_ID)'));
      expect(branch, isNot(contains('createNotificationChannel')),
          reason: 'canonical settings are the user\'s current ones');
    });

    test('new-only does nothing', () {
      expect(channels, contains('ServiceChannelAction.KEEP_CANONICAL -> Unit'));
    });

    test('fresh state creates only the canonical channel', () {
      expect(channels, contains('ServiceChannelAction.CREATE_FRESH ->\n                manager.createNotificationChannel(defaultServiceChannel())'));
      expect(channels, contains('SERVICE_CHANNEL_ID,\n            SERVICE_CHANNEL_NAME,'));
    });
  });

  group('the dead channel', () {
    test('is deleted without a replacement', () {
      expect(channels, contains('deleteNotificationChannel(LEGACY_FOREGROUND_CHANNEL_ID)'));
      // The quoted literal, not the prose: the comment above deliberately
      // names the twin to explain why it is not created.
      expect(channels, isNot(contains('"pocketclaw_foreground"')),
          reason: 'a twin would add an empty entry to the user\'s settings for '
              'the sake of symmetry');
    });

    test('nothing anywhere creates a pocketclaw_foreground channel', () {
      for (final dir in [Directory(kotlin), Directory('lib')]) {
        for (final entity in dir.listSync(recursive: true)) {
          if (entity is! File) continue;
          if (!entity.path.endsWith('.kt') && !entity.path.endsWith('.dart')) continue;
          final source = entity.readAsStringSync();
          final quoted = entity.path.endsWith('.kt')
              ? '"pocketclaw_foreground"'
              : "'pocketclaw_foreground'";
          expect(source, isNot(contains(quoted)), reason: entity.path);
        }
      }
    });
  });

  test('migration is guarded for API < 26', () {
    expect(channels, contains('if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return'));
    // The previous implementation had no guard at all despite minSdk 24.
    expect(app, isNot(contains('NotificationChannel(')),
        reason: 'channel construction must live behind the guard');
  });

  test('no other Zero-Pico phase was widened', () {
    expect(service, contains('File(context.filesDir, "picoclaw")'));
    expect(service, contains('"PICOCLAW_HOME"'));
    expect(File('android/app/src/main/res/xml/backup_rules.xml').readAsStringSync(),
        contains('path="picoclaw/"'));
    expect(File('$kotlin/PocketClawPreferences.kt').readAsStringSync(),
        contains('const val NAME = "pocketclaw_prefs"'),
        reason: 'N4B stays as it was');
  });
}

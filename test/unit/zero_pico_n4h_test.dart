import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// N4H: the managed channel's identity.
///
/// The channel is `pocketclaw`, its client half is `pocketclaw_client` and its
/// owner is `pocketclaw-user`. The Core owns the migration and tests it where it
/// runs, in `pkg/config`; what is guarded here is the host side — the token
/// provider's name, the method-channel contract Dart and Kotlin have to agree
/// on, and the two surfaces this phase deliberately did not move.
void main() {
  const kotlin = 'android/app/src/main/kotlin/com/lord1egypt/pocketclaw';
  const core = 'core/src';

  String read(String path) => File(path).readAsStringSync();

  final service = read('$kotlin/service/PocketClawService.kt');
  final methodChannel = read('$kotlin/PocketClawMethodChannel.kt');
  final dartChannel = read('lib/src/core/pocketclaw_channel.dart');

  group('host identifiers carry the channel\'s name', () {
    test('the realtime token provider is PocketClaw-named', () {
      expect(service, contains('fun pocketClawTokenForHost(context: Context)'));
      expect(service, isNot(contains('picoTokenForHost')));
      expect(methodChannel, isNot(contains('picoTokenForHost')));
    });

    test('the method channel name is agreed on both sides', () {
      // A rename on one side alone leaves the chat unable to authenticate, and
      // nothing fails until a device runs it.
      expect(methodChannel, contains('"getPocketClawToken" ->'));
      expect(dartChannel, contains("invokeMethod<String>('getPocketClawToken')"));
      for (final source in [methodChannel, dartChannel]) {
        expect(source, isNot(contains('getPicoToken')));
      }
    });

    test('no Dart source names the channel credential after the old channel', () {
      for (final entity in Directory('lib').listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.dart')) continue;
        final body = entity.readAsStringSync();
        for (final identifier in const ['picoToken', '_picoToken', 'picoMsg']) {
          expect(body, isNot(contains(identifier)),
              reason: '${entity.path} still names $identifier');
        }
      }
    });
  });

  group('the canonical identity reaches the Core', () {
    test('the channel token env name is canonical on both halves', () {
      expect(service, contains('"POCKETCLAW_CHANNELS_POCKETCLAW_TOKEN" to pocketClawTokenForHost(context)'));
      for (final forbidden in const [
        'POCKETCLAW_CHANNELS_PICO_TOKEN',
        '"PICOCLAW_CHANNELS_PICO_TOKEN"',
      ]) {
        expect(service, isNot(contains(forbidden)));
      }
    });

    test('the Core declares the canonical identities once', () {
      final constants = read('$core/pkg/config/config_channel.go');
      expect(constants, contains('ChannelPocketClaw = "pocketclaw"'));
      expect(constants, contains('ChannelPocketClawClient  = "pocketclaw_client"'));
      expect(constants, contains('PocketClawOwnerPrincipal = "pocketclaw-user"'));
    });

    test('the legacy identities live in one file, marked as input', () {
      final legacy = read('$core/pkg/config/channel_legacy.go');
      expect(legacy, contains('LEGACY READ-ONLY MIGRATION'));
      expect(legacy, contains('LegacyChannelPocketClaw = "pico"'));
      expect(legacy, contains('LegacyChannelPocketClawClient = "pico_client"'));
      expect(legacy, contains('LegacyPocketClawOwnerPrincipal = "pico-user"'));
    });

    test('the channel package moved with the identity', () {
      expect(Directory('$core/pkg/channels/pocketclaw').existsSync(), isTrue);
      expect(Directory('$core/pkg/channels/pico').existsSync(), isFalse);
    });
  });

  group('deferred to the route and sanitizer phase', () {
    // Named here so the next phase's scope is a fact in the tests rather than a
    // note in a report, and so removing one of them fails loudly.
    test('the realtime HTTP routes are unchanged', () {
      expect(read('$core/pkg/channels/pocketclaw/pocketclaw.go'),
          contains('WebhookPath() string { return "/pico/" }'));
      expect(read('$core/web/backend/middleware/middleware.go'),
          contains('"/pico/ws"'));
    });

    test('the log sanitizer still recognises the internal route', () {
      // The route is legacy and the channel is canonical, so redaction has to
      // accept the pair. If either side moves without the other, the internal
      // path stops being redacted and nothing fails except this.
      final logger = read('$core/pkg/logger/logger.go');
      expect(logger, contains('internalRealtimeRoutePrefix = "/pico/"'));
      expect(logger, contains('realtimeChannelName = "pocketclaw"'));
      expect(logger, contains('legacyRealtimeChannelName = "pico"'));

      final frontend = read('$core/web/frontend/src/lib/plain-text-log.ts');
      expect(frontend, contains('hasExactLogToken(line, "channel=pocketclaw")'));
      expect(frontend, contains('hasExactLogToken(line, "channel=pico")'));
      expect(frontend, contains('(?:pico|pocketclaw)\\.go:'));
    });

    test('other Zero-Pico phases were not pulled forward', () {
      expect(read('$kotlin/PocketClawCoreState.kt'),
          contains('CANONICAL_DIR_NAME = "pocketclaw-core"'),
          reason: 'the private directory stays as N4F left it');
      expect(read('$core/pkg/pid/pidfile_migration.go'),
          contains('CanonicalPidFileName = ".pocketclaw.pid"'),
          reason: 'the PID record stays as N4G left it');
    });
  });
}

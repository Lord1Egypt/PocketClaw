import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/plain_text_log_sanitizer.dart';

/// N4I: the realtime route namespace, and the redaction keyed on it.
///
/// The Core proves route registration and server-side redaction where those run,
/// in `web/backend/api`. What is guarded here is the half that ships in the app:
/// the Flutter client's socket URL, the Dart sanitizer's behaviour on canonical
/// routes, and the absence of any client still asking for the old ones.
void main() {
  const core = 'core/src';
  const frontend = '$core/web/frontend/src';

  String read(String path) => File(path).readAsStringSync();

  group('the Dart sanitizer redacts the canonical route', () {
    // Behavioural, not textual. A pattern that was renamed but no longer
    // matches would pass a source check and silently stop redacting.
    test('a routine realtime socket poll is dropped', () {
      expect(
        PlainTextLogSanitizer.sanitize('DBG http server.go:88 > GET /pocketclaw/ws 101').trim(),
        isEmpty,
      );
    });

    test('a failed realtime request loses its path but keeps its status', () {
      final got = PlainTextLogSanitizer.sanitize(
        'WRN http server.go:88 > GET /pocketclaw/ws 403',
      );
      expect(got, contains('/internal realtime connection'));
      expect(got, contains('403'));
      expect(got, isNot(contains('/pocketclaw/ws')));
    });

    test('the realtime component and caller are not shown as themselves', () {
      final got = PlainTextLogSanitizer.sanitize(
        'INF pocketclaw pocketclaw.go:248 > Starting PocketClaw realtime channel',
      );
      expect(got, contains('realtime'));
      expect(got, isNot(contains('pocketclaw.go:248')));
    });

    test('a log written before the migration is still redacted', () {
      expect(
        PlainTextLogSanitizer.sanitize('DBG http server.go:88 > GET /pico/ws 101').trim(),
        isEmpty,
      );
      final got = PlainTextLogSanitizer.sanitize(
        'INF pico pico.go:248 > Starting PocketClaw realtime channel',
      );
      expect(got, isNot(contains('pico.go:248')));
    });

    test('unrelated lines are left exactly as they are', () {
      // A sanitizer broad enough to hide the route by hiding everything would
      // pass every test above and be worthless.
      for (final line in const [
        'INF tools loader.go:80 > Tools loaded count=17',
        'DBG http server.go:88 > GET /api/agents 200',
        'DBG http server.go:88 > GET /pocketclawish/ws 200',
        'INF picometer picophone.go:42 > unrelated component',
      ]) {
        expect(PlainTextLogSanitizer.sanitize(line), line);
      }
    });
  });

  group('no client asks for a legacy route', () {
    test('the Flutter chat socket is canonical', () {
      final chat = read('lib/src/ui/chat_page.dart');
      expect(chat, contains('/pocketclaw/ws?session_id='));
      expect(chat, isNot(contains('/pico/ws')));
    });

    test('no Dart source builds a legacy realtime URL', () {
      for (final entity in Directory('lib').listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.dart')) continue;
        final body = entity.readAsStringSync();
        // The sanitizer patterns legitimately mention the old prefix; they
        // match it, they do not request it.
        if (entity.path.endsWith('plain_text_log_sanitizer.dart')) continue;
        for (final legacy in const ['/pico/ws', '/pico/media/', '/api/pico/']) {
          expect(body, isNot(contains(legacy)),
              reason: '${entity.path} still refers to $legacy');
        }
      }
    });

    test('the console frontend is canonical', () {
      expect(read('$frontend/features/chat/controller.ts'),
          contains('/pocketclaw/ws'));
      final client = read('$frontend/api/pocketclaw.ts');
      for (final path in const [
        '/api/pocketclaw/info',
        '/api/pocketclaw/token',
        '/api/pocketclaw/setup',
      ]) {
        expect(client, contains(path));
      }
      expect(File('$frontend/api/pico.ts').existsSync(), isFalse);
      expect(File('$frontend/hooks/use-pico-chat.ts').existsSync(), isFalse);
      expect(File('$frontend/hooks/use-pocketclaw-chat.ts').existsSync(), isTrue);
    });

    test('no frontend source outside the sanitizer requests a legacy route', () {
      for (final entity in Directory(frontend).listSync(recursive: true)) {
        if (entity is! File) continue;
        if (!entity.path.endsWith('.ts') && !entity.path.endsWith('.tsx')) continue;
        if (entity.path.contains('plain-text-log')) continue;
        if (entity.path.contains('.test.')) continue;
        final body = entity.readAsStringSync();
        for (final legacy in const ['/pico/ws', '/pico/media/', '/api/pico/']) {
          expect(body, isNot(contains(legacy)),
              reason: '${entity.path} still refers to $legacy');
        }
      }
    });
  });

  group('the route derives from the channel identity', () {
    test('the Core declares the surface once', () {
      final routes = read('$core/pkg/config/realtime_routes.go');
      expect(routes, contains('RealtimeRoutePrefix = "/" + ChannelPocketClaw + "/"'));
      expect(routes, contains('RealtimeWebSocketPath = RealtimeRoutePrefix + "ws"'));
      expect(routes, contains('RealtimeMediaPrefix = RealtimeRoutePrefix + "media/"'));
      expect(routes, contains('RealtimeAPIPrefix = "/api/" + ChannelPocketClaw + "/"'));
    });

    test('the logger carries a pinned copy, because it cannot import config', () {
      final logger = read('$core/pkg/logger/logger.go');
      expect(logger, contains('internalRealtimeRoutePrefix = "/pocketclaw/"'));
      expect(logger, contains('legacyInternalRealtimeRoutePrefix = "/pico/"'));
    });
  });

  group('phase boundaries', () {
    test('conversation ids use the canonical identity', () {
      final channel = read('$core/pkg/channels/pocketclaw/pocketclaw.go');
      expect(channel, contains('chatIDPrefix = config.ChannelPocketClaw + ":"'));
      expect(channel, contains('legacyChatIDPrefix = config.LegacyChannelPocketClaw + ":"'),
          reason: 'an in-flight conversation id is still parsed, never minted');
      final client = read('$core/pkg/channels/pocketclaw/client.go');
      expect(client, contains('clientChatIDPrefix = config.ChannelPocketClawClient + ":"'));
    });

    test('deferred surfaces are untouched', () {
      // The cookie name belongs to its own phase, and so do the remaining
      // build-time and upstream defaults.
      expect(read('$core/web/backend/middleware/launcher_dashboard_auth.go'),
          contains('picoclaw_launcher_auth'));
      expect(read('lib/src/core/service_manager.dart'),
          contains('PICOCLAW_DISTRIBUTION_CHANNEL'));
    });

    test('earlier phases still hold', () {
      expect(read('$core/pkg/pid/pidfile_migration.go'),
          contains('CanonicalPidFileName = ".pocketclaw.pid"'));
      expect(read('android/app/src/main/kotlin/com/lord1egypt/pocketclaw/PocketClawCoreState.kt'),
          contains('CANONICAL_DIR_NAME = "pocketclaw-core"'));
    });
  });
}

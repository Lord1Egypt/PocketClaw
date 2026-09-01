import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/ui/webview/pocketclaw_host_bridge.dart';

void main() {
  group('bootstrapScript', () {
    test('publishes the host contract the console listens for', () {
      final script = PocketClawHostBridge.bootstrapScript(
        onboardingConfigured: true,
        telegramBotUsername: 'pocketclaw_ab12cd34_bot',
      );

      expect(script, contains('window.__pocketclawHost'));
      expect(script, contains('onboardingConfigured: true'));
      expect(script, contains('"pocketclaw_ab12cd34_bot"'));
      expect(script, contains('openTelegramOnboarding'));
      expect(script, contains('openExternal'));
      expect(script, contains("CustomEvent('pocketclaw:host-ready')"));
      expect(script, contains(PocketClawHostBridge.channelName));
    });

    test('reports an unconfigured build so the console offers manual setup',
        () {
      final script = PocketClawHostBridge.bootstrapScript(
        onboardingConfigured: false,
      );

      expect(script, contains('onboardingConfigured: false'));
      expect(script, contains('telegramBotUsername: null'));
    });

    test('encodes the username so it cannot break out of its string', () {
      const hostile = 'evil", x: (()=>1)(), y:"';
      final script = PocketClawHostBridge.bootstrapScript(
        onboardingConfigured: true,
        telegramBotUsername: hostile,
      );

      // The value is emitted as a JSON string literal, so the embedded quotes
      // are escaped and the payload stays data instead of becoming a second
      // property on the host object.
      expect(script, contains(jsonEncode(hostile)));
      expect(script, contains(r'\"'));
      expect(script, isNot(contains('telegramBotUsername: evil')));
    });

    test('carries no secret', () {
      final script = PocketClawHostBridge.bootstrapScript(
        onboardingConfigured: true,
        telegramBotUsername: 'pocketclaw_ab12cd34_bot',
      );

      expect(script.toLowerCase(), isNot(contains('token')));
      expect(script.toLowerCase(), isNot(contains('secret')));
    });
  });

  group('isSameOrigin', () {
    const console = 'http://127.0.0.1:18800/';

    test('accepts the console and its own routes', () {
      expect(
        PocketClawHostBridge.isSameOrigin('http://127.0.0.1:18800/', console),
        isTrue,
      );
      expect(
        PocketClawHostBridge.isSameOrigin(
          'http://127.0.0.1:18800/channels/telegram',
          console,
        ),
        isTrue,
      );
    });

    test('rejects any other origin, so a followed link gets no host', () {
      for (final url in <String>[
        'https://example.com/channels/telegram',
        'http://127.0.0.1:18801/channels/telegram',
        'https://127.0.0.1:18800/channels/telegram',
        'http://evil.test/',
        'file:///android_asset/x.html',
      ]) {
        expect(
          PocketClawHostBridge.isSameOrigin(url, console),
          isFalse,
          reason: '\$url must not receive the host bridge',
        );
      }
    });

    test('fails closed on missing or unparseable input', () {
      expect(PocketClawHostBridge.isSameOrigin(null, console), isFalse);
      expect(PocketClawHostBridge.isSameOrigin('not a url', console), isFalse);
      expect(PocketClawHostBridge.isSameOrigin(console, 'not a url'), isFalse);
    });
  });

  group('parseMessage', () {
    test('recognises the onboarding handoff', () {
      final request = PocketClawHostBridge.parseMessage(
        '{"type":"openTelegramOnboarding"}',
      );

      expect(request?.kind, HostRequestKind.openTelegramOnboarding);
    });

    test('recognises an external https link', () {
      final request = PocketClawHostBridge.parseMessage(
        '{"type":"openExternal","url":"https://t.me/pocketclaw_ab12cd34_bot"}',
      );

      expect(request?.kind, HostRequestKind.openExternal);
      expect(request?.url, 'https://t.me/pocketclaw_ab12cd34_bot');
    });

    test('drops schemes that would make the bridge a launch primitive', () {
      for (final url in <String>[
        'javascript:alert(1)',
        'file:///data/data/com.lord1egypt.pocketclaw/files/config.json',
        'intent://scan#Intent;scheme=zxing;end',
        'content://com.android.providers/1',
        '/relative/path',
      ]) {
        expect(
          PocketClawHostBridge.parseMessage(
            '{"type":"openExternal","url":"$url"}',
          ),
          isNull,
          reason: '$url must not be launchable',
        );
      }
    });

    test('drops malformed, unknown, and non-object messages', () {
      expect(PocketClawHostBridge.parseMessage('not json'), isNull);
      expect(PocketClawHostBridge.parseMessage('[]'), isNull);
      expect(PocketClawHostBridge.parseMessage('{"type":"wipeConfig"}'), isNull);
      expect(
        PocketClawHostBridge.parseMessage('{"type":"openExternal"}'),
        isNull,
      );
      expect(
        PocketClawHostBridge.parseMessage('{"type":"openExternal","url":42}'),
        isNull,
      );
    });
  });

  group('openWhatsAppSelfChat', () {
    test('the host contract exposes it to the console', () {
      final script = PocketClawHostBridge.bootstrapScript(
        onboardingConfigured: false,
      );

      expect(script, contains('openWhatsAppSelfChat: function'));
      expect(script, contains("type: 'openWhatsAppSelfChat'"));
      expect(script, contains('selfNumber: selfNumber'));
      expect(script, contains('message: message'));
    });

    test('a well-formed request carries the number and the message', () {
      final request = PocketClawHostBridge.parseMessage(
        jsonEncode({
          'type': 'openWhatsAppSelfChat',
          'selfNumber': ' +201012345678 ',
          'message': 'PocketClaw WhatsApp test',
        }),
      );

      expect(request, isNotNull);
      expect(request!.kind, HostRequestKind.openWhatsAppSelfChat);
      expect(request.selfNumber, '+201012345678');
      expect(request.message, 'PocketClaw WhatsApp test');
    });

    test('unicode reaches the host exactly as the console wrote it', () {
      const message = 'مرحبا من PocketClaw 🦞';

      final request = PocketClawHostBridge.parseMessage(
        jsonEncode({
          'type': 'openWhatsAppSelfChat',
          'selfNumber': '+201012345678',
          'message': message,
        }),
      );

      expect(request!.message, message);
    });

    test('an incomplete request is dropped rather than half-acted-on', () {
      for (final payload in <Map<String, Object?>>[
        {'type': 'openWhatsAppSelfChat'},
        {'type': 'openWhatsAppSelfChat', 'selfNumber': '+201012345678'},
        {'type': 'openWhatsAppSelfChat', 'message': 'hi'},
        {'type': 'openWhatsAppSelfChat', 'selfNumber': '', 'message': 'hi'},
        {'type': 'openWhatsAppSelfChat', 'selfNumber': '+2010', 'message': '  '},
        {'type': 'openWhatsAppSelfChat', 'selfNumber': 201012345678, 'message': 'hi'},
      ]) {
        expect(
          PocketClawHostBridge.parseMessage(jsonEncode(payload)),
          isNull,
          reason: 'accepted $payload',
        );
      }
    });
  });
}

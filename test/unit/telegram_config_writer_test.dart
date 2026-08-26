import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/telegram/telegram_config_writer.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_models.dart';

const _credentials = TelegramBotCredentials(
  token: '9001:CHILD-TOKEN',
  botUserId: 9001,
  botUsername: 'pocketclaw_abcd1234_bot',
  ownerUserId: 555,
);

Map<String, dynamic> applied(String raw) => jsonDecode(
      TelegramConfigWriter.applyToConfigJson(raw, _credentials),
    ) as Map<String, dynamic>;

void main() {
  group('applyToConfigJson', () {
    test('enables the telegram channel with the paired token', () {
      final config = applied('{}');
      final telegram = config['channel_list']['telegram'] as Map;
      expect(telegram['enabled'], isTrue);
      expect(telegram['type'], 'telegram');
      expect((telegram['settings'] as Map)['token'], '9001:CHILD-TOKEN');
    });

    test('locks the bot to the Telegram user who created it', () {
      final telegram = applied('{}')['channel_list']['telegram'] as Map;
      expect(telegram['allow_from'], <String>['555']);
    });

    test('preserves other telegram settings instead of replacing them', () {
      final existing = jsonEncode({
        'channel_list': {
          'telegram': {
            'enabled': false,
            'type': 'telegram',
            'reasoning_channel_id': 'keep-me',
            'settings': {
              'token': 'OLD-TOKEN',
              'use_markdown_v2': true,
              'proxy': 'socks5://example',
              'streaming': {'enabled': true},
            },
          },
        },
      });
      final telegram = applied(existing)['channel_list']['telegram'] as Map;
      final settings = telegram['settings'] as Map;

      expect(settings['token'], '9001:CHILD-TOKEN', reason: 'token replaced');
      expect(settings['use_markdown_v2'], isTrue);
      expect(settings['proxy'], 'socks5://example');
      expect((settings['streaming'] as Map)['enabled'], isTrue);
      expect(telegram['reasoning_channel_id'], 'keep-me');
    });

    test('leaves other channels untouched', () {
      final existing = jsonEncode({
        'channel_list': {
          'discord': {
            'enabled': true,
            'type': 'discord',
            'settings': {'token': 'DISCORD-TOKEN'},
          },
        },
      });
      final channels = applied(existing)['channel_list'] as Map;
      expect(channels['discord']['enabled'], isTrue);
      expect(channels['discord']['settings']['token'], 'DISCORD-TOKEN');
      expect(channels['telegram'], isNotNull);
    });

    test('leaves unrelated top-level config untouched', () {
      final existing = jsonEncode({
        'version': 3,
        'model_list': {'gpt': {'enabled': true}},
        'gateway': {'port': 18800},
      });
      final config = applied(existing);
      expect(config['version'], 3);
      expect(config['model_list']['gpt']['enabled'], isTrue);
      expect(config['gateway']['port'], 18800);
    });

    test('handles an empty configuration', () {
      final telegram = applied('   ')['channel_list']['telegram'] as Map;
      expect((telegram['settings'] as Map)['token'], '9001:CHILD-TOKEN');
    });

    test('rejects a configuration that is not a JSON object', () {
      expect(
        () => TelegramConfigWriter.applyToConfigJson('[1,2,3]', _credentials),
        throwsA(isA<FormatException>()),
      );
      expect(
        () => TelegramConfigWriter.applyToConfigJson('not json', _credentials),
        throwsA(isA<FormatException>()),
      );
    });
  });

  group('apply', () {
    test('writes the merged configuration exactly once', () async {
      var reads = 0;
      final writes = <String>[];
      final writer = TelegramConfigWriter(
        readConfig: () async {
          reads++;
          return '{}';
        },
        writeConfig: (content) async {
          writes.add(content);
          return true;
        },
      );

      await writer.apply(_credentials);

      expect(reads, 1);
      expect(writes, hasLength(1));
      final saved = jsonDecode(writes.single) as Map<String, dynamic>;
      expect(saved['channel_list']['telegram']['settings']['token'],
          '9001:CHILD-TOKEN');
    });

    test('reports a rejected save rather than claiming success', () async {
      final writer = TelegramConfigWriter(
        readConfig: () async => '{}',
        writeConfig: (_) async => false,
      );
      await expectLater(
        writer.apply(_credentials),
        throwsA(isA<TelegramOnboardingException>().having(
          (e) => e.kind,
          'kind',
          TelegramOnboardingErrorKind.configurationFailed,
        )),
      );
    });

    test('reports an unreadable configuration', () async {
      final writer = TelegramConfigWriter(
        readConfig: () async => throw Exception('channel down'),
        writeConfig: (_) async => true,
      );
      await expectLater(
        writer.apply(_credentials),
        throwsA(isA<TelegramOnboardingException>()),
      );
    });

    test('does not write anything when the existing config is malformed',
        () async {
      var wrote = false;
      final writer = TelegramConfigWriter(
        readConfig: () async => 'definitely not json',
        writeConfig: (_) async {
          wrote = true;
          return true;
        },
      );
      await expectLater(
        writer.apply(_credentials),
        throwsA(isA<TelegramOnboardingException>()),
      );
      expect(wrote, isFalse,
          reason: 'a malformed config must not be partially overwritten');
    });
  });

  group('TelegramBotCredentials', () {
    test('never prints the token', () {
      expect(_credentials.toString(), isNot(contains('CHILD-TOKEN')));
      expect(_credentials.toString(), contains('<redacted>'));
    });

    test('exposes the bot chat url', () {
      expect(_credentials.chatUrl, 'https://t.me/pocketclaw_abcd1234_bot');
    });
  });
}

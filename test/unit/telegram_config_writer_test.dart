import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/telegram/telegram_config_writer.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_models.dart';

const _credentials = TelegramBotCredentials(
  token: '9001:CHILD-TOKEN',
  botUserId: 9001,
  botUsername: 'pocketclaw_abcd1234_bot',
  ownerUserId: 555,
);

void main() {
  group('Core-authoritative credential write', () {
    test('writes the complete paired credential set exactly once', () async {
      final writes = <TelegramBotCredentials>[];
      final writer = TelegramConfigWriter(
        writeCredentials: (credentials) async {
          writes.add(credentials);
          return true;
        },
      );

      await writer.apply(_credentials);

      expect(writes, <TelegramBotCredentials>[_credentials]);
    });

    test('reports a rejected Core save rather than claiming success', () async {
      final writer = TelegramConfigWriter(writeCredentials: (_) async => false);
      await expectLater(
        writer.apply(_credentials),
        throwsA(
          isA<TelegramOnboardingException>().having(
            (error) => error.kind,
            'kind',
            TelegramOnboardingErrorKind.configurationFailed,
          ),
        ),
      );
    });

    test('reports an unavailable Core write boundary', () async {
      final writer = TelegramConfigWriter(
        writeCredentials: (_) async => throw Exception('Core unavailable'),
      );
      await expectLater(
        writer.apply(_credentials),
        throwsA(isA<TelegramOnboardingException>()),
      );
    });

    test('preserves the classified invalid-credential failure', () async {
      final writer = TelegramConfigWriter(
        writeCredentials: (_) async =>
            throw PlatformException(code: 'TELEGRAM_CREDENTIALS_INVALID'),
      );
      await expectLater(
        writer.apply(_credentials),
        throwsA(
          isA<TelegramOnboardingException>().having(
            (error) => error.kind,
            'kind',
            TelegramOnboardingErrorKind.invalidCredentials,
          ),
        ),
      );
    });

    test('rejects credentials without a verified numeric owner', () async {
      var writes = 0;
      final writer = TelegramConfigWriter(
        writeCredentials: (_) async {
          writes++;
          return true;
        },
      );
      await expectLater(
        writer.apply(
          const TelegramBotCredentials(
            token: '9001:CHILD-TOKEN',
            botUserId: 9001,
            botUsername: 'manual',
            ownerUserId: 0,
          ),
        ),
        throwsA(isA<TelegramOnboardingException>()),
      );
      expect(writes, 0);
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

import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/telegram/telegram_deep_link.dart';

/// PC-DEF-075. "Open chat" produced `https://t.me/@name` and Telegram answered
/// "Username not found", while the bot, token and channel were all fine.
/// Telegram's canonical bot link carries no `@` in the path.
void main() {
  group('canonical Telegram bot link', () {
    test('builds the canonical URL from a bare username', () {
      expect(
        telegramBotChatUrl('pocketclaw_ab12cd34_bot'),
        'https://t.me/pocketclaw_ab12cd34_bot',
      );
    });

    test('removes exactly one leading @ — the defect itself', () {
      expect(
        telegramBotChatUrl('@pocketclaw_ab12cd34_bot'),
        'https://t.me/pocketclaw_ab12cd34_bot',
      );
    });

    test('never produces a doubled @', () {
      for (final raw in [
        'pocketclaw_ab12cd34_bot',
        '@pocketclaw_ab12cd34_bot',
        '  @pocketclaw_ab12cd34_bot  ',
      ]) {
        expect(telegramBotChatUrl(raw), isNot(contains('@')), reason: raw);
      }
    });

    test('refuses @@name rather than quietly repairing it', () {
      expect(telegramBotChatUrl('@@pocketclaw_ab12cd34_bot'), isNull);
    });

    // RTL display moves the neutral '@' to the visual right, so what is on
    // screen and what belongs in the path are different strings.
    test('is unchanged by bidirectional display formatting', () {
      expect(telegramBotChatUrl('‏pocketclaw_ab12cd34_bot'), isNull);
      expect(telegramBotChatUrl('\u202Bpocketclaw_bot\u202C'), isNull);
      expect(
        telegramBotChatUrl('pocketclaw_ab12cd34_bot'),
        'https://t.me/pocketclaw_ab12cd34_bot',
      );
    });

    test('refuses anything that is not a username', () {
      for (final raw in [
        '',
        '@',
        'bot',
        'https://t.me/pocketclaw_bot',
        'pocketclaw/../evil',
        'pocketclaw bot',
        'pocketclaw.bot',
        'javascript:alert(1)',
        '//evil.example',
        null,
      ]) {
        expect(telegramBotChatUrl(raw), isNull, reason: '$raw');
      }
    });

    test('exposes the canonical username on its own, for display', () {
      expect(
        canonicalTelegramUsername('@pocketclaw_ab12cd34_bot'),
        'pocketclaw_ab12cd34_bot',
      );
      expect(canonicalTelegramUsername('@@x'), isNull);
    });
  });
}

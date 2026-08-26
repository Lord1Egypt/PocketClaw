import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_config.dart';

void main() {
  test('no onboarding endpoint is baked into the app by default', () {
    // A guessed or third-party endpoint must never ship. Until an operator
    // builds with --dart-define=POCKETCLAW_ONBOARDING_BASE_URL, there is none.
    expect(TelegramOnboardingConfig.baseUrl, isEmpty);
    expect(TelegramOnboardingConfig.isConfigured, isFalse);
  });
}

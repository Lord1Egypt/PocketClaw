import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/telegram/telegram_config_writer.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_client.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_config.dart';
import 'package:pocketclaw/src/telegram/telegram_onboarding_controller.dart';
import 'package:pocketclaw/src/ui/telegram_onboarding_page.dart';

/// Native implementation invoked only by explicit actions in the Core console.
abstract final class TelegramOnboardingLauncher {
  /// Where the paired bot's `@username` is remembered.
  ///
  /// Only the handle, never the token: the token belongs in Core's config and
  /// nowhere else. The Core console uses this public handle for Open Chat.
  static const String botUsernamePrefsKey = 'pocketclaw.telegram.bot_username';

  /// The paired bot's `@username`, or null if Telegram was set up manually or
  /// has never been paired on this device.
  static Future<String?> readConnectedBotUsername() async {
    final prefs = await SharedPreferences.getInstance();
    final stored = prefs.getString(botUsernamePrefsKey);
    if (stored == null || stored.trim().isEmpty) return null;
    return stored;
  }

  /// Runs the managed-bot flow and returns the paired bot's `@username`, or
  /// null if the user backed out without connecting.
  static Future<String?> startPairing(BuildContext context) async {
    final service = context.read<ServiceManager>();
    if (service.status == ServiceStatus.stopped) {
      await service.start();
      if (!context.mounted) return null;
    }
    final navigator = Navigator.of(context);
    const configWriter = TelegramConfigWriter();
    final controller = TelegramOnboardingController(
      client: TelegramOnboardingClient(
        baseUrl: TelegramOnboardingConfig.baseUrl,
      ),
      configWriter: configWriter,
      reloadCore: () => _reloadCore(service),
      openUrl: (url) =>
          launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication),
      serviceConfigured: TelegramOnboardingConfig.isConfigured,
    );

    try {
      await navigator.push(
        MaterialPageRoute<void>(
          builder: (_) => TelegramOnboardingPage(
            controller: controller,
            configWriter: configWriter,
            onManualConfigurationSaved: () => _reloadCore(service),
          ),
        ),
      );
      final username = controller.connectedBotUsername;
      if (username != null && username.isNotEmpty) {
        final prefs = await SharedPreferences.getInstance();
        await prefs.setString(botUsernamePrefsKey, username);
      }
      return username;
    } finally {
      controller.dispose();
    }
  }

  /// Makes a saved Telegram configuration live.
  ///
  /// PC-DEF-030. Core loads channel credentials at launch, and this used to be
  /// `stop()` followed by `start()` — two Android service intents with an
  /// unconditional stopSelf() between them, which left PocketClaw stopped often
  /// enough that the owner had to restart the Service and the Gateway by hand
  /// before a newly paired bot would answer. [ServiceManager.restartCore] is
  /// one intent the host executes in order.
  ///
  /// Throwing on failure is deliberate: the caller turns it into the failed
  /// stage, so a bot that is configured but not running is never presented as
  /// connected.
  static Future<void> _reloadCore(ServiceManager service) async {
    if (service.status != ServiceStatus.running) {
      // Nothing is running to reload. Core reads the saved configuration on its
      // next start, which is the correct outcome and not a failure.
      return;
    }
    if (!await service.restartCore()) {
      throw StateError('Core did not restart after the Telegram change');
    }
  }
}

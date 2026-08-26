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

/// The single way into managed-bot onboarding.
///
/// PocketClaw shows Telegram in two places: the native settings list, and
/// Channels → Telegram inside the embedded Core console. Milestone D shipped
/// the flow wired only to the first, so the second — the one users actually
/// reach — still opened the raw token form. Both now call [open], so there is
/// one pairing implementation and one user journey rather than one per surface.
abstract final class TelegramOnboardingLauncher {
  /// Where the paired bot's `@username` is remembered.
  ///
  /// Only the handle, never the token: the token belongs in Core's config and
  /// nowhere else. Core does not report the handle back, so without this the
  /// connected summary could not offer Open Chat after a restart.
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
  static Future<String?> open(BuildContext context) async {
    final service = context.read<ServiceManager>();
    final navigator = Navigator.of(context);
    const configWriter = TelegramConfigWriter();
    final controller = TelegramOnboardingController(
      client: TelegramOnboardingClient(
        baseUrl: TelegramOnboardingConfig.baseUrl,
      ),
      configWriter: configWriter,
      reloadCore: () async {
        // Core reads channel configuration at startup, so a newly written
        // Telegram token only takes effect after a restart.
        if (service.status == ServiceStatus.running) {
          await service.stop();
          await service.start();
        }
      },
      openUrl: (url) => launchUrl(
        Uri.parse(url),
        mode: LaunchMode.externalApplication,
      ),
      serviceConfigured: TelegramOnboardingConfig.isConfigured,
    );

    try {
      await navigator.push(
        MaterialPageRoute<void>(
          builder: (_) => TelegramOnboardingPage(
            controller: controller,
            configWriter: configWriter,
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
}

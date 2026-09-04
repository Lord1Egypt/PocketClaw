import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/config_page.dart';
import 'package:pocketclaw/src/ui/telegram_settings_card.dart';

/// Representative visible strings per locale. These assert what a user actually
/// reads, not merely that an ARB key exists — a key can exist and still hold
/// English.
const Map<String, Map<String, String>> expectedByLocale = {
  'en': {
    'autoStartService': 'Start PocketClaw service automatically',
    'autoStartGateway': 'Start Gateway automatically',
    'telegram': 'Manage Telegram connection',
  },
  'ar': {
    'autoStartService': 'تشغيل خدمة PocketClaw تلقائيًا',
    'autoStartGateway': 'تشغيل البوابة تلقائيًا',
    'telegram': 'إدارة اتصال تيليجرام',
  },
  'de': {
    'autoStartService': 'PocketClaw-Dienst automatisch starten',
    'autoStartGateway': 'Gateway automatisch starten',
    'telegram': 'Telegram-Verbindung verwalten',
  },
  'es': {
    'autoStartService': 'Iniciar el servicio de PocketClaw automáticamente',
    'autoStartGateway': 'Iniciar la puerta de enlace automáticamente',
    'telegram': 'Gestionar la conexión de Telegram',
  },
  'fr': {
    'autoStartService': 'Démarrer le service PocketClaw automatiquement',
    'autoStartGateway': 'Démarrer la passerelle automatiquement',
    'telegram': 'Gérer la connexion Telegram',
  },
  'hi': {
    'autoStartService': 'PocketClaw सेवा स्वतः प्रारंभ करें',
    'autoStartGateway': 'गेटवे स्वतः प्रारंभ करें',
    'telegram': 'Telegram कनेक्शन प्रबंधित करें',
  },
  'id': {
    'autoStartService': 'Jalankan layanan PocketClaw secara otomatis',
    'autoStartGateway': 'Jalankan Gateway secara otomatis',
    'telegram': 'Kelola koneksi Telegram',
  },
  'ja': {
    'autoStartService': 'PocketClaw サービスを自動的に起動',
    'autoStartGateway': 'ゲートウェイを自動的に起動',
    'telegram': 'Telegram 接続を管理',
  },
  'ko': {
    'autoStartService': 'PocketClaw 서비스 자동 시작',
    'autoStartGateway': '게이트웨이 자동 시작',
    'telegram': 'Telegram 연결 관리',
  },
  'pt': {
    'autoStartService': 'Iniciar o serviço do PocketClaw automaticamente',
    'autoStartGateway': 'Iniciar o gateway automaticamente',
    'telegram': 'Gerenciar a conexão do Telegram',
  },
  'ru': {
    'autoStartService': 'Запускать службу PocketClaw автоматически',
    'autoStartGateway': 'Запускать шлюз автоматически',
    'telegram': 'Управление подключением Telegram',
  },
  'zh': {
    'autoStartService': '自动启动 PocketClaw 服务',
    'autoStartGateway': '自动启动网关',
    'telegram': '管理 Telegram 连接',
  },
};

Future<void> pumpCards(WidgetTester tester, String locale) => tester.pumpWidget(
  MaterialApp(
    locale: Locale(locale),
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(
      body: Column(
        children: [
          AutoStartSettingsCard(
            serviceEnabled: true,
            gatewayEnabled: false,
            serviceStatus: ServiceStatus.running,
            onServiceChanged: (_) {},
            onGatewayChanged: (_) {},
          ),
          TelegramSettingsCard(onManage: (_) async {}),
        ],
      ),
    ),
  ),
);

void main() {
  testWidgets('Settings prose renders in the selected language', (
    tester,
  ) async {
    for (final entry in expectedByLocale.entries) {
      await pumpCards(tester, entry.key);
      await tester.pumpAndSettle();

      for (final expected in entry.value.entries) {
        expect(
          find.text(expected.value),
          findsOneWidget,
          reason:
              '${entry.key}: expected "${expected.value}" for ${expected.key}',
        );
      }
    }
  });

  // The failure this replaces: Settings in Arabic still showed English.
  testWidgets('no English Settings prose leaks into a translated locale', (
    tester,
  ) async {
    const englishOnly = <String>[
      'Start PocketClaw service automatically',
      'Start Gateway automatically',
      'Manage Telegram connection',
      'Runtime: Running',
      'Auto-start preference: ON',
    ];

    for (final locale in expectedByLocale.keys.where((l) => l != 'en')) {
      await pumpCards(tester, locale);
      await tester.pumpAndSettle();
      for (final english in englishOnly) {
        expect(
          find.text(english),
          findsNothing,
          reason: '$locale still shows the English "$english"',
        );
      }
    }
  });

  testWidgets('Arabic Settings is right-to-left', (tester) async {
    await pumpCards(tester, 'ar');
    await tester.pumpAndSettle();

    final direction = Directionality.of(
      tester.element(find.text('إدارة اتصال تيليجرام')),
    );
    expect(direction, TextDirection.rtl);
  });

  testWidgets('runtime and preference lines are localized too', (tester) async {
    await pumpCards(tester, 'ar');
    await tester.pumpAndSettle();
    expect(find.textContaining('حالة التشغيل: قيد التشغيل'), findsOneWidget);
    expect(
      find.textContaining('تفضيل التشغيل التلقائي: مفعّل'),
      findsOneWidget,
    );
  });
}

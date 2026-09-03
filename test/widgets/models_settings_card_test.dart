import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/models_settings_card.dart';
import 'package:pocketclaw/src/ui/telegram_settings_card.dart';

Future<List<String>> pumpModelsCard(
  WidgetTester tester, {
  Locale locale = const Locale('en'),
}) async {
  final opened = <String>[];
  await tester.pumpWidget(
    MaterialApp(
      locale: locale,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(
        body: ModelsSettingsCard(onManage: (path) async => opened.add(path)),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return opened;
}

void main() {
  testWidgets('the card is visible with its English label and helper', (
    tester,
  ) async {
    await pumpModelsCard(tester);

    expect(find.byKey(const Key('models-settings-card')), findsOneWidget);
    expect(find.text('Manage Models'), findsOneWidget);
    expect(find.text('Add, edit, test, and choose AI models.'), findsOneWidget);
  });

  // The card navigates. Every model, provider and key already lives in the
  // console page it opens; a second implementation here would be a second
  // writer for state Core owns.
  testWidgets('tapping opens the existing console Models route', (
    tester,
  ) async {
    final opened = await pumpModelsCard(tester);

    await tester.tap(find.byKey(const Key('models-settings-card')));
    await tester.pumpAndSettle();

    expect(opened, <String>['/models']);
    expect(canonicalModelsConsolePath, '/models');
  });

  testWidgets('it introduces no model or provider state of its own', (
    tester,
  ) async {
    await pumpModelsCard(tester);

    // A card that managed models would need inputs. This one has none: no
    // fields, no switches, no dropdowns — only a navigation row.
    expect(find.byType(TextField), findsNothing);
    expect(find.byType(Switch), findsNothing);
    expect(find.byType(DropdownButton<String>), findsNothing);
    expect(find.byType(ListTile), findsOneWidget);
  });

  testWidgets('Arabic renders right-to-left with translated strings', (
    tester,
  ) async {
    await pumpModelsCard(tester, locale: const Locale('ar'));

    final direction = Directionality.of(
      tester.element(find.byKey(const Key('models-settings-card'))),
    );
    expect(direction, TextDirection.rtl);
    expect(find.text('إدارة الموديلات'), findsOneWidget);
    expect(
      find.text(
        'إضافة الموديلات وتعديلها واختبارها واختيار الموديل الافتراضي.',
      ),
      findsOneWidget,
    );
  });

  testWidgets('the label is translated in every supported locale', (
    tester,
  ) async {
    const expected = <String, String>{
      'de': 'Modelle verwalten',
      'es': 'Gestionar modelos',
      'fr': 'Gérer les modèles',
      'hi': 'मॉडल प्रबंधित करें',
      'id': 'Kelola Model',
      'ja': 'モデルを管理',
      'ko': '모델 관리',
      'pt': 'Gerenciar modelos',
      'ru': 'Управление моделями',
      'zh': '管理模型',
    };
    for (final entry in expected.entries) {
      await pumpModelsCard(tester, locale: Locale(entry.key));
      expect(
        find.text(entry.value),
        findsOneWidget,
        reason: '${entry.key} should render "${entry.value}"',
      );
    }
  });

  // Adding the Models shortcut must not disturb the Telegram one it sits beside.
  testWidgets('the Telegram shortcut still opens its own route', (
    tester,
  ) async {
    final opened = <String>[];
    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: TelegramSettingsCard(
            onManage: (path) async => opened.add(path),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('telegram-settings-card')));
    await tester.pumpAndSettle();

    expect(opened, <String>['/channels/telegram']);
  });
}

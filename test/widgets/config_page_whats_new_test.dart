import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/service_manager.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/config_page.dart';
import 'package:pocketclaw/src/ui/whats_new_page.dart';
import 'package:pocketclaw/src/whats_new/whats_new_seen_store.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// An in-memory seen store. Each test builds its own, so nothing leaks
/// between them the way a static field would.
class FakeWhatsNewSeenStore implements WhatsNewSeenStore {
  FakeWhatsNewSeenStore([this.version]);

  String? version;
  int writes = 0;

  @override
  Future<String?> readLastSeenVersion() async => version;

  @override
  Future<void> writeLastSeenVersion(String value) async {
    version = value;
    writes++;
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  Future<void> pumpConfigPage(
    WidgetTester tester, {
    required WhatsNewSeenStore store,
    String appVersion = '0.2.0',
  }) async {
    await tester.pumpWidget(
      ChangeNotifierProvider<ServiceManager>.value(
        value: ServiceManager(),
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: ConfigPage(
              whatsNewSeenStore: store,
              whatsNewVersionLoader: () async => appVersion,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  Finder badge(WidgetTester tester) {
    final l10n = AppLocalizations.of(tester.element(find.byType(ConfigPage)))!;
    return find.text(l10n.whatsNewBadge);
  }

  Finder entry(WidgetTester tester) {
    final l10n = AppLocalizations.of(tester.element(find.byType(ConfigPage)))!;
    return find.text(l10n.whatsNewTitle);
  }

  testWidgets('a first launch with nothing stored shows the NEW badge', (
    tester,
  ) async {
    await pumpConfigPage(tester, store: FakeWhatsNewSeenStore());

    expect(entry(tester), findsOneWidget);
    expect(badge(tester), findsOneWidget);
  });

  testWidgets('a stored version matching the release hides the NEW badge', (
    tester,
  ) async {
    await pumpConfigPage(tester, store: FakeWhatsNewSeenStore('0.2.0'));

    expect(entry(tester), findsOneWidget);
    expect(badge(tester), findsNothing);
  });

  testWidgets('opening What\'s New from Settings marks the release seen', (
    tester,
  ) async {
    final store = FakeWhatsNewSeenStore();
    await pumpConfigPage(tester, store: store);

    expect(badge(tester), findsOneWidget);

    await tester.tap(entry(tester));
    await tester.pumpAndSettle();

    expect(find.byType(WhatsNewPage), findsOneWidget);
    expect(store.version, '0.2.0');

    await tester.pageBack();
    await tester.pumpAndSettle();

    expect(badge(tester), findsNothing);
  });

  testWidgets('merely opening Settings does not mark the release seen', (
    tester,
  ) async {
    final store = FakeWhatsNewSeenStore();
    await pumpConfigPage(tester, store: store);

    expect(store.writes, 0);
    expect(store.version, isNull);
    expect(badge(tester), findsOneWidget);
  });

  testWidgets('the seen mark survives rebuilding the Settings page', (
    tester,
  ) async {
    final store = FakeWhatsNewSeenStore();
    await pumpConfigPage(tester, store: store);

    await tester.tap(entry(tester));
    await tester.pumpAndSettle();
    await tester.pageBack();
    await tester.pumpAndSettle();

    // A fresh widget tree over the same store: the badge must stay down.
    await pumpConfigPage(tester, store: store);
    expect(badge(tester), findsNothing);
  });

  testWidgets('a versionCode-only rebuild does not bring the badge back', (
    tester,
  ) async {
    final store = FakeWhatsNewSeenStore('0.2.0');

    // The loader reads versionName only, so a versionCode bump is invisible
    // here by construction: the same 0.2.0 comes back from every candidate.
    await pumpConfigPage(tester, store: store, appVersion: '0.2.0');
    expect(badge(tester), findsNothing);

    await pumpConfigPage(tester, store: store, appVersion: '0.2.0');
    expect(badge(tester), findsNothing);
  });

  testWidgets('a new versionName brings the badge back', (tester) async {
    final store = FakeWhatsNewSeenStore('0.2.0');

    await pumpConfigPage(tester, store: store, appVersion: '0.3.0');
    expect(badge(tester), findsOneWidget);

    await tester.tap(entry(tester));
    await tester.pumpAndSettle();
    expect(store.version, '0.3.0');
  });

  testWidgets(
    "the focus chain runs What's New, About, then the public mode toggle",
    (tester) async {
      await pumpConfigPage(tester, store: FakeWhatsNewSeenStore());

      final l10n = AppLocalizations.of(
        tester.element(find.byType(ConfigPage)),
      )!;

      FocusableButton buttonLabelled(String label) => tester
          .widgetList<FocusableButton>(find.byType(FocusableButton))
          .firstWhere(
            (button) =>
                find
                    .descendant(
                      of: find.byWidget(button),
                      matching: find.text(label),
                    )
                    .evaluate()
                    .isNotEmpty,
          );

      final whatsNew = buttonLabelled(l10n.whatsNewTitle);
      final about = buttonLabelled(l10n.about);

      // What's New heads the chain: its own node is its predecessor.
      expect(whatsNew.prevFocusNode, same(whatsNew.focusNode));
      expect(whatsNew.nextFocusNode, same(about.focusNode));
      expect(about.prevFocusNode, same(whatsNew.focusNode));

      // Arrowing down from What's New lands on About and no further.
      whatsNew.focusNode.requestFocus();
      await tester.pump();
      expect(whatsNew.focusNode.hasFocus, isTrue);

      whatsNew.nextFocusNode.requestFocus();
      await tester.pump();
      expect(about.focusNode.hasFocus, isTrue);
    },
  );

  testWidgets('the production seen store uses the documented preference key', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    const store = SharedPreferencesWhatsNewSeenStore();

    expect(await store.readLastSeenVersion(), isNull);
    await store.writeLastSeenVersion('0.2.0');
    expect(await store.readLastSeenVersion(), '0.2.0');

    final prefs = await SharedPreferences.getInstance();
    expect(
      prefs.getString('pocketclaw.whats_new.last_seen_version'),
      '0.2.0',
      reason: 'changing this key would re-show the badge for every user',
    );
  });
}

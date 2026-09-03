import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/picoclaw_channel.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/context_memory_card.dart';

TelegramContextMemory setting(int recentMessages) => TelegramContextMemory(
  recentMessages: recentMessages,
  min: 5,
  max: 50,
  defaultValue: 15,
);

class Recorder {
  final List<int> saved = <int>[];
  TelegramContextMemory current;
  bool rejectEverything = false;

  Recorder(int initial) : current = setting(initial);

  Future<TelegramContextMemory> load() async => current;

  Future<TelegramContextMemory> save(int value) async {
    if (rejectEverything) {
      throw StateError('Core rejected the value');
    }
    saved.add(value);
    current = setting(value);
    return current;
  }
}

Future<Recorder> pumpCard(
  WidgetTester tester, {
  int initial = 15,
  Locale locale = const Locale('en'),
  bool rejectEverything = false,
}) async {
  final recorder = Recorder(initial)..rejectEverything = rejectEverything;
  await tester.pumpWidget(
    MaterialApp(
      locale: locale,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(
        // A fresh key forces a new State, so a loop that pumps again really
        // reloads instead of reusing the previous card.
        body: ContextMemoryCard(
          key: ValueKey<int>(initial + (rejectEverything ? 10000 : 0)),
          load: recorder.load,
          save: recorder.save,
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return recorder;
}

bool chipSelected(WidgetTester tester, String key) {
  final chip = tester.widget<ChoiceChip>(find.byKey(Key(key)));
  return chip.selected;
}

void main() {
  testWidgets('an unset value shows the recommended default', (tester) async {
    await pumpCard(tester, initial: 15);

    expect(chipSelected(tester, 'context-memory-preset-15'), isTrue);
    expect(find.textContaining('Recommended'), findsOneWidget);
    expect(find.byKey(const Key('context-memory-custom-field')), findsNothing);
  });

  testWidgets('each preset loads as its own selection', (tester) async {
    for (final preset in contextMemoryPresets) {
      await pumpCard(tester, initial: preset);
      expect(
        chipSelected(tester, 'context-memory-preset-$preset'),
        isTrue,
        reason: '$preset should load selected',
      );
      expect(chipSelected(tester, 'context-memory-preset-custom'), isFalse);
    }
  });

  testWidgets('a non-preset value loads as Custom with its number', (
    tester,
  ) async {
    await pumpCard(tester, initial: 17);

    expect(chipSelected(tester, 'context-memory-preset-custom'), isTrue);
    final field = tester.widget<TextField>(
      find.byKey(const Key('context-memory-custom-field')),
    );
    expect(field.controller!.text, '17');
  });

  testWidgets('choosing a preset stores exactly that value', (tester) async {
    final recorder = await pumpCard(tester, initial: 15);

    await tester.tap(find.byKey(const Key('context-memory-preset-25')));
    await tester.pumpAndSettle();

    expect(recorder.saved, <int>[25]);
    expect(recorder.current.recentMessages, 25);
    expect(chipSelected(tester, 'context-memory-preset-25'), isTrue);
  });

  testWidgets('a custom value inside the range is stored exactly', (
    tester,
  ) async {
    final recorder = await pumpCard(tester, initial: 15);

    await tester.tap(find.byKey(const Key('context-memory-preset-custom')));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('context-memory-custom-field')),
      '17',
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();

    expect(recorder.saved, <int>[17]);
    expect(recorder.current.recentMessages, 17);
  });

  testWidgets('values outside the range are rejected and nothing is stored', (
    tester,
  ) async {
    for (final rejected in <String>['0', '4', '51', '100000']) {
      final recorder = await pumpCard(tester, initial: 15);

      await tester.tap(find.byKey(const Key('context-memory-preset-custom')));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byKey(const Key('context-memory-custom-field')),
        rejected,
      );
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pumpAndSettle();

      expect(recorder.saved, isEmpty, reason: '$rejected must not be stored');
      expect(find.byKey(const Key('context-memory-error')), findsOneWidget);
    }
  });

  testWidgets('non-numeric input cannot reach the setting', (tester) async {
    final recorder = await pumpCard(tester, initial: 15);

    await tester.tap(find.byKey(const Key('context-memory-preset-custom')));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const Key('context-memory-custom-field')),
      'twenty',
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();

    expect(recorder.saved, isEmpty);
  });

  testWidgets('a rejected save restores the value Core is actually using', (
    tester,
  ) async {
    final recorder = await pumpCard(
      tester,
      initial: 15,
      rejectEverything: true,
    );

    await tester.tap(find.byKey(const Key('context-memory-preset-25')));
    await tester.pumpAndSettle();

    expect(recorder.current.recentMessages, 15);
    expect(chipSelected(tester, 'context-memory-preset-15'), isTrue);
    expect(find.byKey(const Key('context-memory-error')), findsOneWidget);
  });

  testWidgets('Arabic renders right-to-left with translated text', (
    tester,
  ) async {
    await pumpCard(tester, initial: 15, locale: const Locale('ar'));

    final direction = Directionality.of(
      tester.element(find.byKey(const Key('context-memory-card'))),
    );
    expect(direction, TextDirection.rtl);
    expect(find.text('ذاكرة سياق تيليجرام'), findsOneWidget);
    expect(find.textContaining('موصى به'), findsOneWidget);
  });

  testWidgets('the card exposes a focus node for D-pad navigation', (
    tester,
  ) async {
    final focusNode = FocusNode();
    addTearDown(focusNode.dispose);
    final recorder = Recorder(15);

    await tester.pumpWidget(
      MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: ContextMemoryCard(
            focusNode: focusNode,
            load: recorder.load,
            save: recorder.save,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    focusNode.requestFocus();
    await tester.pumpAndSettle();
    expect(focusNode.hasFocus, isTrue);
  });
}

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:pocketclaw/src/core/status_snapshot.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/ui/status_sections.dart';

/// The uptime transport contract, end to end.
///
/// The Status screen used to print Go's own `time.Duration.String()` verbatim —
/// "27.707765309s" — because nothing on the path from the gateway to the widget
/// ever parsed it. The payload now carries whole seconds as a number and the
/// formatting decision belongs here.
/// Whether a rendered duration carries fractional precision, as Go's
/// `Duration.String()` does ("27.707765309s"). A trailing abbreviation period
/// such as German "27 S." is not that.
bool _hasDecimalPrecision(String value) => RegExp(r'\d\.\d').hasMatch(value);

void main() {
  /// The exact JSON the Go gateway emits for a 27.707765309s uptime, captured
  /// from `json.Marshal(status.Snapshot{...})` rather than hand-written, so
  /// this test fails if the wire shape changes.
  const goPayloadFor27Seconds =
      '{"system":{"uptime_seconds":27},"activity":{"active_turns":1,'
      '"active_subagents":0,"waiting":0,"completed":12,"failed":0,'
      '"cancelled":0,"tool_calls":0,"tool_calls_failed":0,'
      '"last_activity_unix":0},"model":{"active_model":"mimo-v2.5",'
      '"configured_model":"","provider":"openai","fallback_count":3},'
      '"channels":[{"name":"telegram","configured":true,"started":true,'
      '"running":true}],"resources":{"memory_rss_bytes":88080384,'
      '"cpu_seconds":252.5}}';

  Future<AppLocalizations> localizationsFor(
    WidgetTester tester,
    Locale locale,
  ) async {
    late AppLocalizations l10n;
    await tester.pumpWidget(
      MaterialApp(
        locale: locale,
        localizationsDelegates: const [
          AppLocalizations.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) {
            l10n = AppLocalizations.of(context)!;
            return const SizedBox.shrink();
          },
        ),
      ),
    );
    return l10n;
  }

  test('the real Go payload parses to 27 whole seconds', () {
    final snapshot = StatusSnapshot.tryParse(goPayloadFor27Seconds)!;

    expect(snapshot.system.uptimeSeconds, 27);
    // Not the nanosecond count, and not a duration string.
    expect(snapshot.system.uptimeSeconds, isNot(27707765309));
    expect(snapshot.system.uptimeSeconds, isA<int>());
  });

  test('a missing system block reads as zero, not as garbage', () {
    final snapshot = StatusSnapshot.tryParse('{"activity":{}}')!;
    expect(snapshot.system.uptimeSeconds, 0);
  });

  testWidgets('the formatter drops one unit at a time', (tester) async {
    final l10n = await localizationsFor(tester, const Locale('en'));

    expect(StatusSections.formatUptime(l10n, 27), '27s');
    expect(StatusSections.formatUptime(l10n, 0), '0s');
    expect(StatusSections.formatUptime(l10n, 59), '59s');
    expect(StatusSections.formatUptime(l10n, 252), '4m 12s');
    expect(StatusSections.formatUptime(l10n, 60), '1m 0s');
    expect(StatusSections.formatUptime(l10n, 5040), '1h 24m');
    expect(StatusSections.formatUptime(l10n, 3600), '1h 0m');
    // >= 1 day
    expect(StatusSections.formatUptime(l10n, 183600), '2d 3h');
    expect(StatusSections.formatUptime(l10n, 86400), '1d 0h');
    // Never a negative duration.
    expect(StatusSections.formatUptime(l10n, -5), '0s');
  });

  testWidgets('the 27.7-second reading never renders as ~27 billion', (
    tester,
  ) async {
    final l10n = await localizationsFor(tester, const Locale('en'));
    final snapshot = StatusSnapshot.tryParse(goPayloadFor27Seconds)!;

    final rendered = StatusSections.formatUptime(
      l10n,
      snapshot.system.uptimeSeconds,
    );

    expect(rendered, '27s');
    expect(rendered.contains('27707765309'), isFalse);
    expect(_hasDecimalPrecision(rendered), isFalse);
  });

  testWidgets('every locale formats it without Go duration text', (
    tester,
  ) async {
    for (final locale in AppLocalizations.supportedLocales) {
      final l10n = await localizationsFor(tester, locale);

      for (final seconds in [27, 252, 5040, 183600]) {
        final rendered = StatusSections.formatUptime(l10n, seconds);
        expect(
          rendered.isNotEmpty,
          isTrue,
          reason: 'empty uptime for ${locale.languageCode}',
        );
        // Go's own formatting would carry fractional seconds or an h/m/s run
        // like "1h24m0s"; a localized rendering carries neither. The check is
        // for a dot *between digits* — German abbreviates seconds as "S." and
        // minutes as "Min.", and those periods are correct.
        expect(
          _hasDecimalPrecision(rendered),
          isFalse,
          reason: 'raw Go precision leaked for ${locale.languageCode}: $rendered',
        );
        expect(
          RegExp(r'^\d+h\d+m\d+s$').hasMatch(rendered),
          isFalse,
          reason: 'raw Go duration text for ${locale.languageCode}: $rendered',
        );
      }
    }
  });

  testWidgets('Arabic renders its own units, not English ones', (tester) async {
    final ar = await localizationsFor(tester, const Locale('ar'));
    final en = await localizationsFor(tester, const Locale('en'));

    final arabic = StatusSections.formatUptime(ar, 5040);
    expect(arabic, isNot(StatusSections.formatUptime(en, 5040)));
    expect(arabic.contains('ه'), isFalse); // not an English 'h' transliteration
    expect(arabic.contains('س'), isTrue); // Arabic hour marker
  });
}

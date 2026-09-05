import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/core/app_theme.dart';

/// Aperture emits both brightnesses from one system.
///
/// Light mode used to come from `PocketClawDesign` (Material 3 seeded from a
/// cyan) and dark mode from `AppTheme` (one of six FlexColorScheme themes).
/// They were unrelated, so toggling the theme did not adjust PocketClaw, it
/// swapped products. These assert the structure is now shared, and that
/// rebasing it did not disturb the six user-selectable modes.
void main() {
  group('the six modes survive the rebase', () {
    test('the enum keeps its members and its order', () {
      // The preference is persisted as an index into this list, so the order
      // is part of the storage contract: reordering it silently resolves every
      // saved choice to a different theme.
      expect(AppThemeMode.values.map((mode) => mode.name).toList(), [
        'carbon',
        'slate',
        'obsidian',
        'ebony',
        'nord',
        'sakura',
      ]);
    });

    test('every mode still produces both brightnesses', () {
      for (final mode in AppThemeMode.values) {
        expect(ApertureTheme.light(mode), isA<ThemeData>());
        expect(ApertureTheme.dark(mode), isA<ThemeData>());
      }
    });

    test('AppTheme.getTheme keeps working for existing callers', () {
      for (final mode in AppThemeMode.values) {
        expect(
          AppTheme.getTheme(mode).colorScheme.primary,
          ApertureTheme.dark(mode).colorScheme.primary,
        );
      }
    });

    test('the modes differ by accent, not by structure', () {
      final accents = <int>{};
      final radii = <double>{};
      for (final mode in AppThemeMode.values) {
        final theme = ApertureTheme.dark(mode);
        accents.add(theme.colorScheme.primary.toARGB32());
        final shape = theme.cardTheme.shape! as RoundedRectangleBorder;
        radii.add((shape.borderRadius as BorderRadius).topLeft.x);
      }
      expect(accents.length, greaterThan(3), reason: 'accents collapsed');
      expect(radii, {ApertureTheme.radiusMd}, reason: 'geometry drifted');
    });

    test('sakura stays a light theme in both slots, as it always has', () {
      expect(
        ApertureTheme.dark(AppThemeMode.sakura).brightness,
        Brightness.light,
      );
      expect(
        ApertureTheme.light(AppThemeMode.sakura).brightness,
        Brightness.light,
      );
    });
  });

  group('the neutrals carry the brand', () {
    for (final mode in AppThemeMode.values) {
      test('${mode.name} tints every neutral rather than leaving it grey', () {
        final theme = ApertureTheme.dark(mode);
        if (theme.brightness == Brightness.light) return;
        final tokens = _tokensOf(theme);

        final neutrals = {
          'surface1': tokens.surface1,
          'surface2': tokens.surface2,
          'surface3': tokens.surface3,
          'border': tokens.border,
          'text': tokens.text,
          'textMuted': tokens.textMuted,
        };
        for (final entry in neutrals.entries) {
          final chroma = _chromaOf(entry.value);
          // Chroma 0 is the stock framework default and the reason the old
          // palette read as generated rather than designed.
          expect(
            chroma,
            greaterThan(0.001),
            reason: '${mode.name} ${entry.key} is a chroma-0 grey',
          );
          expect(
            chroma,
            lessThan(0.04),
            reason: '${mode.name} ${entry.key} is not a neutral any more',
          );
        }
      });
    }

    test('obsidian is the one canvas allowed to be pure black', () {
      expect(
        _tokensOf(ApertureTheme.dark(AppThemeMode.obsidian)).canvas,
        const Color(0xFF000000),
      );

      for (final mode in AppThemeMode.values) {
        if (mode == AppThemeMode.obsidian) continue;
        final theme = ApertureTheme.dark(mode);
        if (theme.brightness == Brightness.light) continue;
        expect(
          _tokensOf(theme).canvas,
          isNot(const Color(0xFF000000)),
          reason: '${mode.name} paints the canvas pure black',
        );
      }
    });
  });

  group('the ink rule', () {
    test('dark mode puts dark ink on the bright accent', () {
      final tokens = _tokensOf(ApertureTheme.dark(AppThemeMode.carbon));
      expect(_lightnessOf(tokens.accent), closeTo(0.70, 0.02));
      expect(_lightnessOf(tokens.accentInk), lessThan(0.3));
    });

    test('light mode inverts it, because the accent is dark there', () {
      final tokens = _tokensOf(ApertureTheme.light(AppThemeMode.carbon));
      expect(_lightnessOf(tokens.accent), lessThan(0.6));
      expect(_lightnessOf(tokens.accentInk), greaterThan(0.9));
    });
  });

  test('live machine state is never the interactive accent', () {
    for (final mode in AppThemeMode.values) {
      final tokens = _tokensOf(ApertureTheme.dark(mode));
      expect(
        tokens.signal,
        isNot(tokens.accent),
        reason: '${mode.name} cannot tell live state from clickable',
      );
    }
  });

  group('depth is drawn, not shadowed', () {
    test('cards are flat with a hairline', () {
      final theme = ApertureTheme.dark(AppThemeMode.carbon);
      expect(theme.cardTheme.elevation, 0);
      final shape = theme.cardTheme.shape! as RoundedRectangleBorder;
      expect(shape.side.color, _tokensOf(theme).border);
    });

    test('each surface step is lighter than the one below it', () {
      final tokens = _tokensOf(ApertureTheme.dark(AppThemeMode.carbon));
      final steps = [
        tokens.canvas,
        tokens.surface1,
        tokens.surface2,
        tokens.surface3,
      ].map(_lightnessOf).toList();
      for (var i = 1; i < steps.length; i++) {
        expect(steps[i], greaterThan(steps[i - 1]));
      }
    });
  });

  test('the primary button meets the touch minimum without help', () {
    final style = ApertureTheme.dark(
      AppThemeMode.carbon,
    ).elevatedButtonTheme.style!;
    expect(
      style.minimumSize!.resolve({})!.height,
      greaterThanOrEqualTo(ApertureTheme.minTouchTarget),
    );
  });

  group('ApertureBracket', () {
    // The bracket cannot be a border side: Flutter asserts that a borderRadius
    // may only be given for a border whose sides share a colour. It is drawn
    // as a clipped strip on the inline-start edge instead, which keeps it a
    // logical edge and mirrors it with no locale branch.
    testWidgets('draws on the start edge and mirrors in Arabic', (
      tester,
    ) async {
      Future<Rect> markerIn(TextDirection direction) async {
        await tester.pumpWidget(
          Directionality(
            textDirection: direction,
            child: Center(
              child: SizedBox(
                width: 200,
                height: 60,
                child: ApertureBracket(
                  fill: const Color(0xFF101010),
                  bracket: const Color(0xFF00B4C8),
                  borderColor: const Color(0xFF303030),
                  borderWidth: 1,
                  child: const SizedBox.expand(),
                ),
              ),
            ),
          ),
        );
        return tester.getRect(find.byType(ColoredBox));
      }

      final ltr = await markerIn(TextDirection.ltr);
      final rtl = await markerIn(TextDirection.rtl);

      expect(ltr.width, 2);
      expect(rtl.width, 2);
      expect(ltr.left, lessThan(rtl.left));
    });

    testWidgets('draws nothing when there is no selection', (tester) async {
      await tester.pumpWidget(
        const Directionality(
          textDirection: TextDirection.ltr,
          child: Center(
            child: SizedBox(
              width: 200,
              height: 60,
              child: ApertureBracket(
                fill: Color(0xFF101010),
                borderColor: Color(0xFF303030),
                borderWidth: 1,
                child: SizedBox.expand(),
              ),
            ),
          ),
        ),
      );
      expect(find.byType(ColoredBox), findsNothing);
    });
  });
}

ApertureColors _tokensOf(ThemeData theme) =>
    theme.extensions.values.whereType<ApertureColors>().single;

/// The colour read back in OKLab, the space the palette is specified in.
/// Reading it out is what turns "every neutral carries the brand hue" into a
/// testable claim rather than an assertion of taste.
({double lightness, double chroma}) _oklab(Color color) {
  double linear(double channel) => channel <= 0.04045
      ? channel / 12.92
      : math.pow((channel + 0.055) / 1.055, 2.4).toDouble();

  final r = linear(color.r);
  final g = linear(color.g);
  final b = linear(color.b);

  double cbrt(double value) => math.pow(value, 1 / 3).toDouble();
  final l = cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
  final m = cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
  final s = cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);

  final lightness = 0.2104542553 * l + 0.7936177850 * m - 0.0040720468 * s;
  final a = 1.9779984951 * l - 2.4285922050 * m + 0.4505937099 * s;
  final bb = 0.0259040371 * l + 0.7827717662 * m - 0.8086757660 * s;
  return (lightness: lightness, chroma: math.sqrt(a * a + bb * bb));
}

double _lightnessOf(Color color) => _oklab(color).lightness;
double _chromaOf(Color color) => _oklab(color).chroma;

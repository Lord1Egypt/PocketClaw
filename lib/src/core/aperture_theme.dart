import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import 'package:pocketclaw/src/core/app_theme.dart';

/// APERTURE — the one structural design system both PocketClaw brightnesses
/// are drawn from.
///
/// Light mode used to come from `PocketClawDesign` (Material 3 seeded from a
/// cyan) and dark mode from `AppTheme` (one of six FlexColorScheme themes).
/// They were unrelated systems, so toggling the theme did not adjust
/// PocketClaw; it swapped products. Aperture emits both brightnesses from one
/// set of tokens.
///
/// The six user-selectable modes are kept, the enum is kept and the persisted
/// preference is kept. They stop being six products and become six *accents*
/// over one structure: the same surfaces, radii, typography, focus logic and
/// component geometry in every one of them.
///
/// Two rules carry the identity:
///
///  * Every neutral holds a small chroma at the accent's hue. Chroma-0 grey is
///    the default of every framework, which is why it reads as one.
///  * `obsidian` is the sanctioned exception to "never pure black" — AMOLED
///    panels genuinely benefit, so its canvas stays at zero.
abstract final class ApertureTheme {
  // Radii. These are the same numbers the embedded console uses, so a card in
  // the Flutter shell and a card in the WebView inside it share a corner.
  static const double radiusXs = 6;
  static const double radiusSm = 10;
  static const double radiusMd = 14;
  static const double radiusLg = 20;

  static const double spaceXs = 4;
  static const double spaceSm = 8;
  static const double spaceMd = 16;
  static const double spaceLg = 24;

  /// Minimum interactive hit area, in logical pixels.
  static const double minTouchTarget = 44;

  static ThemeData light(AppThemeMode mode) =>
      _build(_ApertureSpec.of(mode), Brightness.light);

  static ThemeData dark(AppThemeMode mode) {
    final spec = _ApertureSpec.of(mode);
    // `sakura` has always been a light-only theme, and rebasing it must not
    // silently turn a user's saved preference into something else.
    return _build(spec, spec.lightOnly ? Brightness.light : Brightness.dark);
  }

  static ThemeData _build(_ApertureSpec spec, Brightness brightness) {
    final tokens = _ApertureTokens.resolve(spec, brightness);
    final scheme = ColorScheme(
      brightness: brightness,
      primary: tokens.accent,
      onPrimary: tokens.accentInk,
      primaryContainer: tokens.accentSoft,
      onPrimaryContainer: tokens.text,
      secondary: tokens.accent,
      onSecondary: tokens.accentInk,
      secondaryContainer: tokens.accentSoft,
      onSecondaryContainer: tokens.text,
      tertiary: tokens.signal,
      onTertiary: tokens.accentInk,
      error: tokens.danger,
      onError: tokens.accentInk,
      errorContainer: tokens.dangerSoft,
      onErrorContainer: tokens.text,
      surface: tokens.surface1,
      onSurface: tokens.text,
      surfaceContainerLowest: tokens.canvas,
      surfaceContainerLow: tokens.surface1,
      surfaceContainer: tokens.surface2,
      surfaceContainerHigh: tokens.surface2,
      surfaceContainerHighest: tokens.surface3,
      onSurfaceVariant: tokens.textMuted,
      outline: tokens.border,
      outlineVariant: tokens.borderStrong,
      inverseSurface: tokens.text,
      onInverseSurface: tokens.canvas,
      shadow: const Color(0xFF000000),
      scrim: const Color(0xFF000000),
    );

    final base = ThemeData(
      useMaterial3: true,
      brightness: brightness,
      colorScheme: scheme,
      scaffoldBackgroundColor: tokens.canvas,
      canvasColor: tokens.canvas,
      dividerColor: tokens.border,
      textTheme: GoogleFonts.interTextTheme(
        ThemeData(brightness: brightness).textTheme,
      ).apply(bodyColor: tokens.text, displayColor: tokens.text),
      visualDensity: VisualDensity.adaptivePlatformDensity,
    );

    final shapeSm = RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(radiusSm),
    );

    return base.copyWith(
      extensions: [tokens],
      // Depth is drawn: a lightness step and a hairline, not a shadow. Dark
      // drop shadows are close to invisible, and leaning on them is why dark
      // interfaces so often look flat.
      cardTheme: CardThemeData(
        color: tokens.surface1,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusMd),
          side: BorderSide(color: tokens.border),
        ),
      ),
      appBarTheme: AppBarTheme(
        backgroundColor: tokens.canvas,
        foregroundColor: tokens.text,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
      ),
      dialogTheme: DialogThemeData(
        backgroundColor: tokens.surface2,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusLg),
          side: BorderSide(color: tokens.border),
        ),
      ),
      dividerTheme: DividerThemeData(color: tokens.border, thickness: 1),
      listTileTheme: ListTileThemeData(
        iconColor: tokens.textMuted,
        textColor: tokens.text,
        minVerticalPadding: 12,
        shape: shapeSm,
      ),
      switchTheme: SwitchThemeData(
        thumbColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.selected)
              ? tokens.accentInk
              : tokens.textFaint,
        ),
        trackColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.selected)
              ? tokens.accent
              : tokens.surface3,
        ),
        trackOutlineColor: WidgetStateProperty.resolveWith(
          (states) => states.contains(WidgetState.selected)
              ? tokens.accent
              : tokens.border,
        ),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: tokens.accent,
          foregroundColor: tokens.accentInk,
          elevation: 0,
          minimumSize: const Size(0, minTouchTarget),
          padding: const EdgeInsets.symmetric(horizontal: spaceMd),
          textStyle: const TextStyle(fontWeight: FontWeight.w500),
          shape: shapeSm,
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: tokens.accent,
          foregroundColor: tokens.accentInk,
          minimumSize: const Size(0, minTouchTarget),
          shape: shapeSm,
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: tokens.text,
          minimumSize: const Size(0, minTouchTarget),
          side: BorderSide(color: tokens.border),
          shape: shapeSm,
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: tokens.accent,
          minimumSize: const Size(0, minTouchTarget),
          shape: shapeSm,
        ),
      ),
      iconTheme: IconThemeData(color: tokens.textMuted),
      chipTheme: ChipThemeData(
        backgroundColor: tokens.surface2,
        side: BorderSide(color: tokens.border),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: tokens.field,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: spaceMd,
          vertical: 14,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusSm),
          borderSide: BorderSide(color: tokens.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusSm),
          borderSide: BorderSide(color: tokens.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusSm),
          borderSide: BorderSide(color: tokens.accent, width: 2),
        ),
        labelStyle: TextStyle(color: tokens.textMuted),
        hintStyle: TextStyle(color: tokens.textFaint),
      ),
      navigationRailTheme: NavigationRailThemeData(
        backgroundColor: tokens.surface1,
        indicatorColor: tokens.accentSoft,
        selectedIconTheme: IconThemeData(color: tokens.accent),
        unselectedIconTheme: IconThemeData(color: tokens.textFaint),
        selectedLabelTextStyle: TextStyle(
          color: tokens.accent,
          fontWeight: FontWeight.w600,
        ),
        unselectedLabelTextStyle: TextStyle(color: tokens.textFaint),
      ),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: tokens.surface3,
        contentTextStyle: TextStyle(color: tokens.text),
        behavior: SnackBarBehavior.floating,
        shape: shapeSm,
      ),
      progressIndicatorTheme: ProgressIndicatorThemeData(
        color: tokens.accent,
        linearTrackColor: tokens.surface3,
      ),
    );
  }
}

/// One mode's accent, and whether it darkens the canvas all the way.
class _ApertureSpec {
  const _ApertureSpec({
    required this.accentHue,
    required this.accentChroma,
    this.neutralHue = 245,
    this.trueBlack = false,
    this.lightOnly = false,
  });

  final double accentHue;
  final double accentChroma;

  /// The hue every neutral is tinted with. Only `ebony` moves it, because a
  /// warm structure is what made that mode recognisable.
  final double neutralHue;

  /// AMOLED: canvas at pure black. The one sanctioned exception.
  final bool trueBlack;

  /// `sakura` has always resolved to a light theme in both slots.
  final bool lightOnly;

  /// The structure is constant; only the accent and the canvas depth vary, so
  /// every mode is recognisably PocketClaw.
  static _ApertureSpec of(AppThemeMode mode) => switch (mode) {
    AppThemeMode.carbon => const _ApertureSpec(
      accentHue: 208,
      accentChroma: 0.130,
    ),
    AppThemeMode.slate => const _ApertureSpec(
      accentHue: 78,
      accentChroma: 0.140,
    ),
    AppThemeMode.obsidian => const _ApertureSpec(
      accentHue: 208,
      accentChroma: 0.130,
      trueBlack: true,
    ),
    AppThemeMode.ebony => const _ApertureSpec(
      accentHue: 92,
      accentChroma: 0.130,
      neutralHue: 60,
    ),
    AppThemeMode.nord => const _ApertureSpec(
      accentHue: 220,
      accentChroma: 0.100,
    ),
    AppThemeMode.sakura => const _ApertureSpec(
      accentHue: 350,
      accentChroma: 0.130,
      lightOnly: true,
    ),
  };
}

/// The resolved Aperture tokens, carried on the theme so a widget can reach a
/// surface or a status colour without re-deriving it.
@immutable
class _ApertureTokens extends ThemeExtension<_ApertureTokens>
    implements ApertureColors {
  const _ApertureTokens({
    required this.canvas,
    required this.surface1,
    required this.surface2,
    required this.surface3,
    required this.border,
    required this.borderStrong,
    required this.text,
    required this.textMuted,
    required this.textFaint,
    required this.accent,
    required this.accentInk,
    required this.accentSoft,
    required this.signal,
    required this.success,
    required this.warning,
    required this.danger,
    required this.dangerSoft,
    required this.field,
  });

  @override
  final Color canvas;
  @override
  final Color surface1;
  @override
  final Color surface2;
  @override
  final Color surface3;
  @override
  final Color border;
  @override
  final Color borderStrong;
  @override
  final Color text;
  @override
  final Color textMuted;
  @override
  final Color textFaint;
  @override
  final Color accent;
  @override
  final Color accentInk;
  @override
  final Color accentSoft;
  @override
  final Color signal;
  @override
  final Color success;
  @override
  final Color warning;
  @override
  final Color danger;
  @override
  final Color dangerSoft;
  @override
  final Color field;

  static _ApertureTokens resolve(_ApertureSpec spec, Brightness brightness) {
    final hue = spec.neutralHue;
    if (brightness == Brightness.dark) {
      final accent = _oklch(0.70, spec.accentChroma, spec.accentHue);
      return _ApertureTokens(
        canvas: spec.trueBlack
            ? const Color(0xFF000000)
            : _oklch(0.17, 0.012, hue),
        surface1: _oklch(spec.trueBlack ? 0.19 : 0.21, 0.014, hue),
        surface2: _oklch(spec.trueBlack ? 0.23 : 0.25, 0.016, hue),
        surface3: _oklch(spec.trueBlack ? 0.27 : 0.29, 0.018, hue),
        border: _oklch(0.33, 0.020, hue),
        borderStrong: _oklch(0.44, 0.026, hue),
        text: _oklch(0.97, 0.004, hue),
        textMuted: _oklch(0.74, 0.014, hue),
        textFaint: _oklch(0.60, 0.016, hue),
        accent: accent,
        // The accent sits at L 0.70, so white ink on it fails contrast. Dark
        // ink is also what makes a primary button read as illuminated.
        accentInk: _oklch(0.16, 0.020, hue),
        accentSoft: accent.withValues(alpha: 0.14),
        signal: _oklch(0.78, 0.130, 195),
        success: _oklch(0.72, 0.150, 158),
        warning: _oklch(0.80, 0.140, 78),
        danger: _oklch(0.64, 0.200, 25),
        dangerSoft: _oklch(0.64, 0.200, 25).withValues(alpha: 0.14),
        field: _oklch(spec.trueBlack ? 0.27 : 0.29, 0.018, hue),
      );
    }

    // Light mode inverts the ink rule: the accent is dark enough there that
    // light ink is the correct one.
    final accent = _oklch(0.55, spec.accentChroma, spec.accentHue);
    return _ApertureTokens(
      canvas: _oklch(0.985, 0.003, hue),
      surface1: const Color(0xFFFFFFFF),
      surface2: _oklch(0.975, 0.004, hue),
      surface3: _oklch(0.955, 0.006, hue),
      border: _oklch(0.905, 0.008, hue),
      borderStrong: _oklch(0.820, 0.012, hue),
      text: _oklch(0.22, 0.015, hue),
      textMuted: _oklch(0.46, 0.018, hue),
      textFaint: _oklch(0.58, 0.016, hue),
      accent: accent,
      accentInk: _oklch(0.99, 0.005, spec.accentHue),
      accentSoft: accent.withValues(alpha: 0.12),
      signal: _oklch(0.60, 0.130, 195),
      success: _oklch(0.55, 0.140, 158),
      warning: _oklch(0.62, 0.140, 68),
      danger: _oklch(0.55, 0.200, 25),
      dangerSoft: _oklch(0.55, 0.200, 25).withValues(alpha: 0.12),
      field: const Color(0xFFFFFFFF),
    );
  }

  @override
  _ApertureTokens copyWith() => this;

  @override
  _ApertureTokens lerp(ThemeExtension<_ApertureTokens>? other, double t) =>
      t < 0.5 ? this : (other as _ApertureTokens? ?? this);
}

/// What a widget reads off the theme. Named separately from the token class so
/// product code depends on the roles, never on the ramp.
abstract interface class ApertureColors {
  Color get canvas;
  Color get surface1;
  Color get surface2;
  Color get surface3;
  Color get border;
  Color get borderStrong;
  Color get text;
  Color get textMuted;
  Color get textFaint;
  Color get accent;
  Color get accentInk;
  Color get accentSoft;

  /// Live machine state only — gateway running, agent streaming. Never a
  /// button, never a link, never a heading.
  Color get signal;
  Color get success;
  Color get warning;
  Color get danger;
  Color get dangerSoft;
  Color get field;
}

extension ApertureThemeAccess on BuildContext {
  /// The Aperture tokens for the theme in force.
  ApertureColors get aperture =>
      Theme.of(this).extension<_ApertureTokens>() ??
      _ApertureTokens.resolve(
        _ApertureSpec.of(AppThemeMode.carbon),
        Theme.of(this).brightness,
      );
}

/// A rounded surface carrying the Aperture bracket on its inline-start edge.
///
/// The bracket cannot be one side of the box decoration: Flutter asserts that
/// a `borderRadius` may only be given for a border whose sides share a colour,
/// so a 2px accent edge against three hairline edges is not expressible there.
/// It is drawn as a clipped strip behind the content instead — which keeps it
/// a *logical* edge, so it still mirrors in Arabic with no locale branch.
///
/// [bracket] is what selection looks like. [borderColor] and [borderWidth] are
/// what focus looks like. They are deliberately separate, because a control can
/// be focused and not selected, selected and not focused, or both, and all four
/// combinations have to stay distinguishable.
class ApertureBracket extends StatelessWidget {
  const ApertureBracket({
    super.key,
    required this.child,
    required this.fill,
    required this.borderColor,
    required this.borderWidth,
    this.bracket,
    this.radius = ApertureTheme.radiusMd,
    this.padding = EdgeInsets.zero,
  });

  final Widget child;
  final Color fill;
  final Color borderColor;
  final double borderWidth;

  /// The selection marker. Null draws no bracket.
  final Color? bracket;
  final double radius;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      decoration: BoxDecoration(
        color: fill,
        borderRadius: BorderRadius.circular(radius),
        border: Border.all(color: borderColor, width: borderWidth),
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(radius - borderWidth),
        child: Stack(
          children: [
            if (bracket != null)
              PositionedDirectional(
                start: 0,
                top: 0,
                bottom: 0,
                width: 2,
                child: ColoredBox(color: bracket!),
              ),
            Padding(padding: padding, child: child),
          ],
        ),
      ),
    );
  }
}

/// OKLCH to sRGB.
///
/// The palette is specified in OKLCH because that is the space in which "one
/// step darker" and "the same chroma at a different hue" mean what they say.
/// Dart has no OKLCH literal, so the conversion lives here rather than being
/// pre-baked into hex — hex constants would hide the arithmetic that makes
/// this a system instead of a list of colours.
Color _oklch(double l, double c, double hueDegrees) {
  final h = hueDegrees * (math.pi / 180.0);
  final a = c * math.cos(h);
  final b = c * math.sin(h);

  final lc = l + 0.3963377774 * a + 0.2158037573 * b;
  final mc = l - 0.1055613458 * a - 0.0638541728 * b;
  final sc = l - 0.0894841775 * a - 1.2914855480 * b;

  final l3 = lc * lc * lc;
  final m3 = mc * mc * mc;
  final s3 = sc * sc * sc;

  final r = 4.0767416621 * l3 - 3.3077115913 * m3 + 0.2309699292 * s3;
  final g = -1.2684380046 * l3 + 2.6097574011 * m3 - 0.3413193965 * s3;
  final bl = -0.0041960863 * l3 - 0.7034186147 * m3 + 1.7076147010 * s3;

  return Color.fromARGB(255, _channel(r), _channel(g), _channel(bl));
}

int _channel(double linear) {
  final clamped = linear.clamp(0.0, 1.0);
  final encoded = clamped <= 0.0031308
      ? 12.92 * clamped
      : 1.055 * math.pow(clamped, 1 / 2.4) - 0.055;
  return (encoded * 255).round().clamp(0, 255);
}

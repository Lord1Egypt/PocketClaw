import 'package:flutter/material.dart';

import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/core/app_theme.dart';

/// Layout tokens for PocketClaw-owned UI.
///
/// This used to generate light mode from a Material 3 seed while `AppTheme`
/// generated dark mode from an unrelated system. Both brightnesses now come
/// from [ApertureTheme], so the spacing and radius constants here — which the
/// onboarding page and several cards already use — line up with the theme
/// rather than describing a second, conflicting one.
abstract final class PocketClawDesign {
  static const seed = Color(0xFF00B8D9);
  static const radiusSmall = ApertureTheme.radiusSm;
  static const radiusMedium = ApertureTheme.radiusMd;
  static const spaceSmall = ApertureTheme.spaceSm;
  static const spaceMedium = ApertureTheme.spaceMd;
  static const spaceLarge = ApertureTheme.spaceLg;

  static ThemeData lightTheme([AppThemeMode mode = AppThemeMode.carbon]) =>
      ApertureTheme.light(mode);
  static ThemeData darkTheme([AppThemeMode mode = AppThemeMode.carbon]) =>
      ApertureTheme.dark(mode);
}

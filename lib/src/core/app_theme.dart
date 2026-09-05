import 'package:flutter/material.dart';

import 'package:pocketclaw/src/core/aperture_theme.dart';

/// The six user-selectable themes.
///
/// The enum, its order and the persisted index are unchanged: the preference
/// is stored as `AppThemeMode.values[index]`, so renaming or reordering a
/// member would silently resolve every saved choice to a different theme.
///
/// What changed under Aperture is what each one *means*. They used to be six
/// unrelated products — neon cyan, amber, pure white, gold, frost blue, pink,
/// each with its own surfaces and geometry. They are now six accents over one
/// structure: same surfaces, radii, typography, focus logic and component
/// geometry, with only the accent hue and the canvas depth varying.
enum AppThemeMode {
  carbon, // Graphite structure, Claw accent (default)
  slate, // Graphite structure, Amber accent
  obsidian, // True-black canvas for AMOLED, Claw accent
  ebony, // Warm-graphite structure, Gold accent
  nord, // Graphite structure, Frost accent
  sakura, // Light mode, Rose accent
}

class AppTheme {
  /// Kept as the dark-brightness entry point so existing callers — including
  /// the Settings theme swatches — keep working unchanged.
  static ThemeData getTheme(AppThemeMode mode) => ApertureTheme.dark(mode);
}

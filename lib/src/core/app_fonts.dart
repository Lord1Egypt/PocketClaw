import 'package:flutter/material.dart';

/// PocketClaw's typography, from assets bundled in the APK.
///
/// This replaced `google_fonts`, which fetched Inter and Fira Code from
/// Google's servers on first use and cached them on device. That made ordinary
/// UI rendering depend on the network and on a Google endpoint — wrong for an
/// app whose whole claim is that it runs on your phone, and disqualifying for
/// official F-Droid, which does not accept runtime downloads of unpackaged
/// assets.
///
/// The families are declared in `pubspec.yaml` with one file per weight, so
/// Flutter resolves them locally and there is no fetch path left to disable.
class AppFonts {
  const AppFonts._();

  /// The UI family. Bundled at w400, w500, w600, w700, w800 and w900.
  static const String sans = 'Inter';

  /// The monospace family, for logs and anything column-aligned. Bundled at
  /// w400 and w600.
  static const String mono = 'FiraCode';

  /// Inter, with an optional weight and any other [TextStyle] field.
  ///
  /// A drop-in for what `AppFonts.inter(...)` returned, minus the download.
  static TextStyle inter({
    double? fontSize,
    FontWeight? fontWeight,
    Color? color,
    double? height,
    double? letterSpacing,
    FontStyle? fontStyle,
    TextDecoration? decoration,
  }) => TextStyle(
    fontFamily: sans,
    fontSize: fontSize,
    fontWeight: fontWeight,
    color: color,
    height: height,
    letterSpacing: letterSpacing,
    fontStyle: fontStyle,
    decoration: decoration,
  );

  /// Fira Code, for monospaced text.
  static TextStyle firaCode({
    double? fontSize,
    FontWeight? fontWeight,
    Color? color,
    double? height,
    double? letterSpacing,
  }) => TextStyle(
    fontFamily: mono,
    fontSize: fontSize,
    fontWeight: fontWeight,
    color: color,
    height: height,
    letterSpacing: letterSpacing,
  );

  /// Applies Inter across a whole [TextTheme], the way
  /// `GoogleFonts.interTextTheme` did.
  static TextTheme interTextTheme(TextTheme base) =>
      base.apply(fontFamily: sans);
}

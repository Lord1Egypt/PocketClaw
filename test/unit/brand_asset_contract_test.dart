import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter_test/flutter_test.dart';

/// Branding assets are derived, not drawn.
///
/// One geometry — the APERTURE mark in Core's
/// `web/frontend/scripts/generate-brand-assets.py` — produces the Android
/// launcher resources and, through `tool/generate_android_launcher_icons.py`,
/// the README icon `assets/branding/pocketclaw-icon.png`. The README used to
/// show a separate glossy 3D mark that nothing on the phone displays; the
/// generator's `--check` mode now holds the README picture to the same bytes as
/// the geometry, so a hand-edit or a stale commit fails here rather than
/// drifting from the launcher again (PC-DEF-081).
///
/// The desktop tray and Windows icons this generator also used to write
/// (`assets/app_icon.png`, `assets/icon.ico`) are gone with the desktop code:
/// they were bundled into every APK and read by nothing Android runs.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('every tracked branding asset is what the generator produces', () {
    final result = Process.runSync(
      'python3',
      ['tool/generate_android_launcher_icons.py', '--check'],
    );
    expect(
      result.exitCode,
      0,
      reason: 'branding assets are stale:\n${result.stdout}${result.stderr}',
    );
  });

  test('regeneration is deterministic', () {
    // --check writing nothing and passing twice in a row is the property that
    // matters: the generator is a pure function of the tracked geometry.
    for (var i = 0; i < 2; i++) {
      final result = Process.runSync(
        'python3',
        ['tool/generate_android_launcher_icons.py', '--check'],
      );
      expect(result.exitCode, 0, reason: 'run ${i + 1} disagreed');
    }
  });

  test('the historical and desktop-only artwork is gone', () {
    // The PicoClaw lobster lived at assets/app_icon.png and assets/icon.ico
    // through 0.2.0+59; later those paths held desktop-only tray icons. The
    // glossy mark was the README's picture. None of them is PocketClaw's
    // installed identity, and none of them may come back.
    for (final path in const [
      'assets/app_icon.png',
      'assets/icon.ico',
      'assets/branding/pocketclaw-mark.png',
    ]) {
      expect(File(path).existsSync(), isFalse, reason: '$path is back');
    }
  });

  test('the README shows the generated icon, not a separate mark', () {
    final readme = File('README.md').readAsStringSync();
    expect(readme, contains('assets/branding/pocketclaw-icon.png'));
    expect(readme, isNot(contains('pocketclaw-mark.png')));
  });

  test('no branding picture is bundled into the APK as a Flutter asset', () {
    // Nothing the app runs reads one, and Flutter bundles every declared asset
    // regardless of use. The launcher icon travels as Android resources.
    final pubspec = File('pubspec.yaml').readAsStringSync();
    expect(pubspec, isNot(contains('assets/branding/')));
    expect(pubspec, isNot(contains('.png')));
    expect(pubspec, isNot(contains('.ico')));
  });

  test('pocketclaw-icon.png is a 512px launcher tile with transparent corners', () async {
    final bytes = File('assets/branding/pocketclaw-icon.png').readAsBytesSync();
    final codec = await ui.instantiateImageCodec(bytes);
    final image = (await codec.getNextFrame()).image;
    addTearDown(image.dispose);

    expect(image.width, 512);
    expect(image.height, 512);

    final data = await image.toByteData(format: ui.ImageByteFormat.rawRgba);
    final pixels = data!.buffer.asUint8List();

    int alphaAt(int x, int y) => pixels[(y * image.width + x) * 4 + 3];

    // A rounded tile: the very corner is cut away, the centre is not.
    expect(alphaAt(0, 0), 0, reason: 'the tile corner should be transparent');
    expect(alphaAt(256, 256), 255, reason: 'the tile centre should be opaque');
  });

  test('the source-of-truth chain is documented where it is implemented', () {
    final generator =
        File('tool/generate_android_launcher_icons.py').readAsStringSync();
    expect(generator, contains('pocketclaw-icon.png'));
    expect(generator, isNot(contains('app_icon.png"')));
    expect(generator, isNot(contains('icon.ico"')));
    // The geometry must still come from Core, not be restated here.
    expect(
      generator,
      contains('core/src/web/frontend/scripts/generate-brand-assets.py'),
    );
    expect(generator, isNot(contains('STROKES = [')),
        reason: 'a second copy of the geometry would drift');
  });
}

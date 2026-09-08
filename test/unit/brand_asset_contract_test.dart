import 'dart:io';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter_test/flutter_test.dart';

/// The desktop identity assets are derived, not drawn.
///
/// `assets/app_icon.png` and `assets/icon.ico` were the last PicoClaw lobster
/// artwork in the tree — an orange crustacean shipping inside every APK,
/// unreferenced by anything Android runs, because they are declared as Flutter
/// assets and Flutter bundles what it is told to bundle regardless of platform.
///
/// They now come out of `tool/generate_android_launcher_icons.py`, from the
/// same canonical APERTURE geometry as the launcher and Core's own favicon.
/// These guards hold the derivation rather than the pixels: the generator's
/// `--check` mode regenerates every artifact and compares bytes, so a hand-edit
/// or a stale commit fails here rather than shipping.
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

  test('the historical lobster assets are gone', () {
    // Digests of the PicoClaw FUI artwork as tracked through 0.2.0+59. Pinned
    // so that restoring either file from history fails loudly instead of
    // quietly reintroducing another product's branding.
    const lobsterPng =
        '68a133e77515857a245da31ecdeba843a75ad2237790da4b2d3456501e1bd647';
    const lobsterIco =
        'dfff202743ae3222b273117cfbf80363b460dcdf2172eed2cb7d942cc0676547';

    expect(_sha256Of('assets/app_icon.png'), isNot(lobsterPng));
    expect(_sha256Of('assets/icon.ico'), isNot(lobsterIco));
  });

  test('app_icon.png is a 512px tile with transparent corners', () async {
    final bytes = File('assets/app_icon.png').readAsBytesSync();
    final codec = await ui.instantiateImageCodec(bytes);
    final image = (await codec.getNextFrame()).image;
    addTearDown(image.dispose);

    expect(image.width, 512);
    expect(image.height, 512);

    final data =
        await image.toByteData(format: ui.ImageByteFormat.rawRgba);
    final pixels = data!.buffer.asUint8List();

    int alphaAt(int x, int y) => pixels[(y * image.width + x) * 4 + 3];

    // A rounded tile: the very corner is cut away, the centre is not.
    expect(alphaAt(0, 0), 0, reason: 'the tile corner should be transparent');
    expect(alphaAt(256, 256), 255, reason: 'the tile centre should be opaque');
  });

  test('icon.ico carries the frames Windows actually asks for', () {
    // A single 256px frame — what the lobster file had — leaves a 16px tray
    // request to be downscaled at draw time by whatever is asking, and it
    // blurs. Parsed straight from the ICONDIR rather than through a decoder.
    final bytes = File('assets/icon.ico').readAsBytesSync();
    final header = ByteData.sublistView(bytes);

    expect(header.getUint16(0, Endian.little), 0, reason: 'ICO reserved field');
    expect(header.getUint16(2, Endian.little), 1, reason: 'ICO type');

    final count = header.getUint16(4, Endian.little);
    final widths = <int>{};
    for (var i = 0; i < count; i++) {
      final entry = 6 + i * 16;
      // 0 in the ICONDIRENTRY width byte means 256.
      final raw = bytes[entry];
      widths.add(raw == 0 ? 256 : raw);
    }

    expect(widths, {16, 24, 32, 48, 64, 128, 256});
  });

  test('the source-of-truth chain is documented where it is implemented', () {
    final generator =
        File('tool/generate_android_launcher_icons.py').readAsStringSync();
    expect(generator, contains('assets/app_icon.png'));
    expect(generator, contains('assets/icon.ico'));
    // The geometry must still come from Core, not be restated here.
    expect(
      generator,
      contains('core/src/web/frontend/scripts/generate-brand-assets.py'),
    );
    expect(generator, isNot(contains('STROKES = [')),
        reason: 'a second copy of the geometry would drift');
  });
}

String _sha256Of(String path) {
  final result = Process.runSync('sha256sum', [path]);
  return (result.stdout as String).split(' ').first;
}

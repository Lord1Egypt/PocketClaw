import 'dart:io';
import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter_test/flutter_test.dart';

/// The Android launcher icon is the flat APERTURE mark.
///
/// It used to be the glossy 3D mark — gradients, bevels and a specular orb —
/// scaled by `flutter_launcher_icons` from a single flattened PNG. Two things
/// were wrong with that beyond the artwork. The legacy pre-26 icons were
/// generated straight from a transparent source, so on API 24/25 the launcher
/// drew a floating mark with no tile behind it; and the generator's Android
/// source still pointed at that PNG, so any future run would have reverted the
/// launcher regardless of what was committed.
///
/// Every raster here is now derived from the one canonical geometry by
/// `tool/generate_android_launcher_icons.py`. These guards hold the derivation,
/// not the pixels: they check what the resources resolve to, that the legacy
/// tiles are opaque as a property of the file format, that nothing in the
/// launcher carries colour the Aperture ramp cannot produce, and that the mark
/// still lands inside Android's mask-safe circle once the adaptive inset is
/// applied.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const res = 'android/app/src/main/res';

  // The Aperture tokens the canonical raster script draws with.
  const canvas = (r: 11, g: 16, b: 20);
  const claw = (r: 0, g: 180, b: 200);

  // Adaptive layers are 108dp square, legacy icons 48dp.
  const densities = <String, double>{
    'mdpi': 1,
    'hdpi': 1.5,
    'xhdpi': 2,
    'xxhdpi': 3,
    'xxxhdpi': 4,
  };

  group('the manifest resolves to the adaptive launcher icon', () {
    final manifest =
        File('android/app/src/main/AndroidManifest.xml').readAsStringSync();

    test('names the launcher icon', () {
      expect(manifest, contains('android:icon="@mipmap/ic_launcher"'));
    });

    // A round-icon family would be a second set of artwork to keep in step for
    // the sake of API 25 launchers that the corrected opaque legacy icon
    // already serves. Its absence is the decision, so it is asserted.
    test('declares no roundIcon, and ships no round resource to point at', () {
      expect(manifest, isNot(contains('android:roundIcon')));
      final strays = Directory(res)
          .listSync(recursive: true)
          .whereType<File>()
          .where((file) => file.path.contains('ic_launcher_round'))
          .toList();
      expect(strays, isEmpty);
    });
  });

  group('the adaptive icon points at APERTURE-derived layers', () {
    final adaptive =
        File('$res/mipmap-anydpi-v26/ic_launcher.xml').readAsStringSync();

    test('layers the derived foreground over the canvas colour', () {
      expect(adaptive, contains('@color/ic_launcher_background'));
      expect(adaptive, contains('@drawable/ic_launcher_foreground'));
    });

    test('paints the background with the Aperture canvas', () {
      final colors = File('$res/values/colors.xml').readAsStringSync();
      expect(
        colors,
        contains('<color name="ic_launcher_background">#0b1014</color>'),
      );
      // The splash keeps its own colour; only launcher resources moved.
      expect(colors, contains('name="pocketclaw_splash_background"'));
    });

    test('carries a monochrome layer for Android themed icons', () {
      expect(adaptive, contains('<monochrome>'));
      expect(adaptive, contains('@drawable/ic_launcher_monochrome'));
    });

    test('every referenced drawable exists at every density', () {
      for (final bucket in densities.keys) {
        for (final name in const [
          'ic_launcher_foreground',
          'ic_launcher_monochrome',
        ]) {
          expect(
            File('$res/drawable-$bucket/$name.png').existsSync(),
            isTrue,
            reason: 'drawable-$bucket/$name.png is missing',
          );
        }
      }
    });
  });

  group('the legacy API 24/25 icons are opaque tiles', () {
    test('all five densities exist at the right pixel size', () {
      densities.forEach((bucket, factor) {
        final header = _readHeader('$res/mipmap-$bucket/ic_launcher.png');
        final expected = (48 * factor).round();
        expect(
          header.width,
          expected,
          reason: 'mipmap-$bucket/ic_launcher.png is the wrong size',
        );
        expect(header.height, expected);
      });
    });

    // Encoded without an alpha channel at all, so the transparent-background
    // defect cannot come back through a re-encode that merely happens to be
    // opaque today.
    test('carry no alpha channel', () {
      for (final bucket in densities.keys) {
        final header = _readHeader('$res/mipmap-$bucket/ic_launcher.png');
        expect(
          header.colourType,
          _pngTruecolour,
          reason: 'mipmap-$bucket/ic_launcher.png must be RGB, not RGBA',
        );
      }
    });

    test('are filled to the edge with the Aperture canvas', () async {
      for (final bucket in densities.keys) {
        final pixels = await _decode('$res/mipmap-$bucket/ic_launcher.png');
        for (final corner in [
          const (0, 0),
          (pixels.width - 1, 0),
          (0, pixels.height - 1),
          (pixels.width - 1, pixels.height - 1),
        ]) {
          final pixel = pixels.at(corner.$1, corner.$2);
          expect(pixel.a, 255, reason: 'mipmap-$bucket corner is translucent');
          expect(
            [pixel.r, pixel.g, pixel.b],
            [canvas.r, canvas.g, canvas.b],
            reason: 'mipmap-$bucket corner is not the Aperture canvas',
          );
        }
      }
    });
  });

  group('no launcher resource carries the old artwork', () {
    // The glossy 3D mark has white speculars and mid-blue bevels; the PicoClaw
    // lobster is orange. Neither can occur on a ramp between the Aperture
    // canvas and the claw, so bounding the ramp rejects both without pinning
    // the test to a digest that any legitimate refinement would break.
    test('every launcher pixel lies on the canvas-to-claw ramp', () async {
      final launcherPngs = [
        for (final bucket in densities.keys) ...[
          '$res/mipmap-$bucket/ic_launcher.png',
          '$res/drawable-$bucket/ic_launcher_foreground.png',
        ],
      ];

      for (final path in launcherPngs) {
        final pixels = await _decode(path);
        for (var i = 0; i < pixels.data.length; i += 4) {
          // Anything faint enough is edge antialiasing and says nothing about
          // the artwork. The threshold is deliberately well below opaque: the
          // 3D mark is mostly semi-transparent bevel, and skipping everything
          // under full opacity let it through this guard unchallenged.
          if (pixels.data[i + 3] < 128) continue;
          final r = pixels.data[i];
          final g = pixels.data[i + 1];
          final b = pixels.data[i + 2];
          // Red is 11 at the canvas end and 0 at the claw end, and stays there
          // whether the decoder hands back premultiplied channels or not; the
          // resampler's ringing is what the headroom is for.
          expect(r, lessThanOrEqualTo(24), reason: '$path has a warm pixel');
          expect(g, lessThanOrEqualTo(claw.g + 24), reason: '$path is too hot');
          expect(
            b,
            greaterThanOrEqualTo(g - 10),
            reason: '$path has a pixel off the ramp',
          );
        }
      }
    });

    test('the glossy 3D mark is no longer a launcher source', () {
      // It stays in the tree for the splash, which this milestone left alone.
      expect(File('assets/branding/pocketclaw-mark.png').existsSync(), isTrue);
      expect(
        File('$res/mipmap-anydpi-v26/ic_launcher.xml').readAsStringSync(),
        isNot(contains('pocketclaw_mark')),
      );
    });
  });

  group('the mark survives every adaptive mask', () {
    // Android guarantees only the central 66dp of the 108dp canvas; the
    // remainder is at the mask's discretion. The XML insets the foreground by
    // 16%, so the check has to be made on the composed result rather than on
    // the drawable.
    test('all ink lands inside the 33dp mask-safe radius', () async {
      const inset = 0.16;
      final pixels =
          await _decode('$res/drawable-xxxhdpi/ic_launcher_foreground.png');
      final dpPerPixel = 108.0 / pixels.width;
      final centre = (pixels.width - 1) / 2.0;

      var maxRadius = 0.0;
      for (var y = 0; y < pixels.height; y++) {
        for (var x = 0; x < pixels.width; x++) {
          if (pixels.at(x, y).a <= 8) continue;
          final dx = x - centre;
          final dy = y - centre;
          final radius = math.sqrt(dx * dx + dy * dy);
          if (radius > maxRadius) maxRadius = radius;
        }
      }

      final composed = maxRadius * dpPerPixel * (1 - 2 * inset);
      expect(
        composed,
        lessThanOrEqualTo(33.0),
        reason: 'the mark reaches ${composed.toStringAsFixed(2)}dp from centre',
      );
      // Guard the other way too: a mark that shrank to nothing would also pass
      // the line above.
      expect(composed, greaterThan(24.0));
    });
  });

  group('the monochrome layer is a flat silhouette', () {
    test('is white wherever it is opaque, and is not empty', () async {
      final pixels =
          await _decode('$res/drawable-xxxhdpi/ic_launcher_monochrome.png');
      var opaque = 0;
      for (var i = 0; i < pixels.data.length; i += 4) {
        if (pixels.data[i + 3] != 255) continue;
        opaque++;
        expect([pixels.data[i], pixels.data[i + 1], pixels.data[i + 2]],
            [255, 255, 255]);
      }
      expect(opaque, greaterThan(0));
    });
  });

  group('the generator cannot revert the launcher', () {
    final pubspec = File('pubspec.yaml').readAsStringSync();

    test('flutter_launcher_icons no longer generates Android', () {
      final block = pubspec.substring(pubspec.indexOf('flutter_launcher_icons:',
          pubspec.indexOf('dev_dependencies:')));
      expect(block, contains('android: false'));
      expect(block, isNot(contains('android: true')));
    });

    test('holds no pointer back to the glossy 3D mark', () {
      expect(pubspec, isNot(contains('adaptive_icon_foreground')));
      expect(pubspec, isNot(contains('adaptive_icon_background')));
      expect(
        pubspec,
        isNot(contains('image_path: "assets/branding/pocketclaw-mark.png"')),
      );
    });

    test('the derived master exists and is labelled as derived', () {
      expect(
        File('assets/branding/android-launcher/ic_launcher_master.png')
            .existsSync(),
        isTrue,
      );
      expect(
        File('assets/branding/android-launcher/README.md').readAsStringSync(),
        contains('derived artifact'),
      );
    });
  });

  group('there is no fourth copy of the geometry', () {
    const canonical =
        'core/src/web/frontend/scripts/generate-brand-assets.py';
    final generator =
        File('tool/generate_android_launcher_icons.py').readAsStringSync();

    test('the Android generator imports the canonical mark', () {
      expect(generator, contains(canonical));
      expect(generator, contains('draw_mark'));
    });

    test('the Android generator defines no geometry of its own', () {
      expect(generator, isNot(contains('STROKES = ')));
      expect(generator, isNot(contains('INK_BOX = ')));
      expect(generator, isNot(contains('M7.5 12H4v8h16v-8h-3.5')));
    });

    test('the canonical source still holds the refined geometry', () {
      final script = File(canonical).readAsStringSync();
      expect(script, contains('(7.5, 12)'));
      expect(script, contains('(10, 13.5)'));
    });
  });
}

const _pngTruecolour = 2;

({int width, int height, int colourType}) _readHeader(String path) {
  final bytes = File(path).readAsBytesSync();
  final view = ByteData.sublistView(bytes);
  // 8-byte signature, then the IHDR chunk: length, type, width, height,
  // bit depth, colour type.
  expect(String.fromCharCodes(bytes.sublist(12, 16)), 'IHDR',
      reason: '$path is not a PNG');
  return (
    width: view.getUint32(16),
    height: view.getUint32(20),
    colourType: bytes[25],
  );
}

class _Pixels {
  _Pixels(this.data, this.width, this.height);

  final Uint8List data;
  final int width;
  final int height;

  ({int r, int g, int b, int a}) at(int x, int y) {
    final i = (y * width + x) * 4;
    return (r: data[i], g: data[i + 1], b: data[i + 2], a: data[i + 3]);
  }
}

Future<_Pixels> _decode(String path) async {
  final codec =
      await ui.instantiateImageCodec(File(path).readAsBytesSync());
  final frame = await codec.getNextFrame();
  final image = frame.image;
  final bytes = await image.toByteData(format: ui.ImageByteFormat.rawRgba);
  final pixels =
      _Pixels(bytes!.buffer.asUint8List(), image.width, image.height);
  image.dispose();
  return pixels;
}


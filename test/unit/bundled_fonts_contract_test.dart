import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/app_fonts.dart';
import 'package:yaml/yaml.dart';

/// Typography must resolve entirely from the APK.
///
/// google_fonts fetched Inter and Fira Code from Google's servers at first use.
/// The property under test is not "fonts look right" but that there is no fetch
/// path left at all — an app that renders its own UI over the network is not
/// offline-capable, whatever its feature list says.
void main() {
  final pubspec = loadYaml(File('pubspec.yaml').readAsStringSync()) as YamlMap;
  final flutterSection = pubspec['flutter'] as YamlMap;

  group('no runtime font fetching', () {
    test('google_fonts is not a dependency', () {
      expect((pubspec['dependencies'] as YamlMap).containsKey('google_fonts'),
          isFalse);
      expect(File('pubspec.lock').readAsStringSync(),
          isNot(contains('google_fonts')),
          reason: 'a transitive reintroduction would bring the fetch back');
    });

    test('nothing in the app references Google font endpoints', () {
      final offenders = <String>[];
      for (final entity in Directory('lib').listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.dart')) continue;
        final body = entity.readAsStringSync();
        for (final needle in [
          'fonts.googleapis.com',
          'fonts.gstatic.com',
          'GoogleFonts.',
          "package:google_fonts",
        ]) {
          // The helper's doc comment explains what it replaced; that is prose,
          // not a call site.
          if (body.contains(needle) &&
              !entity.path.endsWith('core/app_fonts.dart')) {
            offenders.add('${entity.path}: $needle');
          }
        }
      }
      expect(offenders, isEmpty);
    });

    test('the font helper makes no network call of its own', () {
      // Comments stripped first: the doc comment explains what it replaced and
      // says "fetched", which is prose about the past, not a call site.
      final code = File('lib/src/core/app_fonts.dart')
          .readAsLinesSync()
          .where((l) => !l.trimLeft().startsWith('//'))
          .join('\n');
      for (final needle in ['http', 'Uri.', 'HttpClient', 'fetch', 'Socket']) {
        expect(code, isNot(contains(needle)),
            reason: 'the bundled-font path must not reach the network');
      }
    });
  });

  group('bundled assets', () {
    test('both families are declared with per-weight files', () {
      final fonts = flutterSection['fonts'] as YamlList;
      final families = {
        for (final f in fonts)
          (f as YamlMap)['family'] as String: [
            for (final v in f['fonts'] as YamlList) (v as YamlMap)['weight'],
          ],
      };
      expect(families.keys, containsAll(<String>['Inter', 'FiraCode']));
      expect(families['Inter'], containsAll(<int>[400, 500, 600, 700, 800, 900]));
      expect(families['FiraCode'], containsAll(<int>[400, 600]));
    });

    test('every declared asset exists and is a real TrueType file', () {
      for (final family in flutterSection['fonts'] as YamlList) {
        for (final entry in (family as YamlMap)['fonts'] as YamlList) {
          final path = (entry as YamlMap)['asset'] as String;
          final file = File(path);
          expect(file.existsSync(), isTrue, reason: '$path is declared but absent');
          // TrueType outlines start with 0x00010000; OpenType/CFF with 'OTTO'.
          final head = file.readAsBytesSync().sublist(0, 4);
          expect(head, anyOf(equals([0, 1, 0, 0]), equals(utf8.encode('OTTO'))),
              reason: '$path is not a font file');
        }
      }
    });

    test('every weight the code asks for is a weight we bundle', () {
      // A weight with no file does not fail — Flutter silently synthesises the
      // nearest one, which is how a design drifts without anyone noticing.
      final declared = <String, Set<int>>{};
      for (final family in flutterSection['fonts'] as YamlList) {
        declared[(family as YamlMap)['family'] as String] = {
          for (final v in family['fonts'] as YamlList) (v as YamlMap)['weight'] as int,
        };
      }
      final asked = RegExp(r'AppFonts\.(inter|firaCode)\([^)]*FontWeight\.w(\d00)',
          dotAll: true);
      for (final entity in Directory('lib').listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.dart')) continue;
        for (final m in asked.allMatches(entity.readAsStringSync())) {
          final family = m.group(1) == 'inter' ? 'Inter' : 'FiraCode';
          final weight = int.parse('${m.group(2)}');
          expect(declared[family], contains(weight),
              reason: '${entity.path} asks for $family w$weight, which is not bundled');
        }
      }
    });

    test('the OFL licences travel with the fonts', () {
      for (final licence in [
        'assets/fonts/Inter-OFL.txt',
        'assets/fonts/FiraCode-OFL.txt',
      ]) {
        final body = File(licence).readAsStringSync();
        expect(body, contains('SIL OPEN FONT LICENSE Version 1.1'),
            reason: '$licence must be the OFL text redistribution requires');
      }
      expect(File('assets/fonts/README.md').existsSync(), isTrue,
          reason: 'provenance and digests must be recorded');
      // Committed is not enough: OFL-1.1 asks the licence to travel with the
      // font software, and the APK is a redistribution.
      final assets = (loadYaml(File('pubspec.yaml').readAsStringSync())
          as YamlMap)['flutter']['assets'] as YamlList;
      final declared = assets.map((a) => a.toString()).toList();
      expect(declared, containsAll(<String>[
        'assets/fonts/Inter-OFL.txt',
        'assets/fonts/FiraCode-OFL.txt',
      ]), reason: 'the OFL texts must be packaged, not only tracked');
    });
  });

  group('typography still resolves', () {
    test('the helper returns the bundled families, not a system fallback', () {
      expect(AppFonts.inter().fontFamily, 'Inter');
      expect(AppFonts.firaCode().fontFamily, 'FiraCode');
      expect(AppFonts.inter(fontWeight: FontWeight.w900).fontWeight,
          FontWeight.w900);
    });

    test('the text theme applies Inter across every style', () {
      final themed = AppFonts.interTextTheme(ThemeData.light().textTheme);
      for (final style in [
        themed.bodyMedium,
        themed.titleLarge,
        themed.labelSmall,
        themed.displayLarge,
      ]) {
        expect(style?.fontFamily, 'Inter');
      }
    });
  });
}

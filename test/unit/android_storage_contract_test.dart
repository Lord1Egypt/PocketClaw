import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:pocketclaw/src/core/legacy_workspace.dart';

/// PC-DEF-077. The workspace is app-specific storage, PocketClaw declares no
/// storage permission, and an older `Download/pocketclaw` workspace is only
/// ever copied in when the owner asks. Static source assertions: each names the
/// regression it exists to catch.
void main() {
  const kotlin = 'android/app/src/main/kotlin/com/lord1egypt/pocketclaw';
  String read(String path) => File(path).readAsStringSync();
  String declared(String path) =>
      read(path).replaceAll(RegExp(r'<!--.*?-->', dotAll: true), '');

  group('no storage permission', () {
    test('the manifest declares none, and no legacy storage opt-in', () {
      final manifest = declared('android/app/src/main/AndroidManifest.xml');
      for (final symbol in const [
        'MANAGE_EXTERNAL_STORAGE',
        'READ_EXTERNAL_STORAGE',
        'WRITE_EXTERNAL_STORAGE',
        'requestLegacyExternalStorage',
      ]) {
        expect(manifest, isNot(contains(symbol)), reason: symbol);
      }
    });

    test('no code sends the user to the all-files-access screen', () {
      for (final file
          in Directory(kotlin)
              .listSync(recursive: true)
              .whereType<File>()
              .where((f) => f.path.endsWith('.kt'))) {
        final source = file.readAsStringSync();
        expect(source, isNot(contains('ALL_FILES_ACCESS')), reason: file.path);
        expect(
          source,
          isNot(contains('isExternalStorageManager')),
          reason: file.path,
        );
      }
    });
  });

  group('workspace location', () {
    test('the workspace is app-specific storage and nothing else', () {
      final service = read('$kotlin/service/PocketClawService.kt');
      final start = service.indexOf('fun getWorkspacePath(');
      final body = service.substring(
        start,
        service.indexOf('\n        }', start),
      );
      expect(body, contains('getExternalFilesDir(null)'));
      expect(body, contains('File(context.filesDir, "pocketclaw")'));
      expect(
        body,
        isNot(contains('DIRECTORY_DOWNLOADS')),
        reason: 'the workspace must not silently move back to shared storage',
      );
    });
  });

  group('legacy workspace import', () {
    const importer = '$kotlin/storage/LegacyWorkspaceImporter.kt';

    const copier = '$kotlin/storage/WorkspaceTreeCopier.kt';
    const rules = '$kotlin/storage/WorkspaceImportRules.kt';

    test('it never deletes or rewrites the source', () {
      final importerSource = read(importer);
      for (final forbidden in const [
        'deleteDocument',
        'openOutputStream',
        'takePersistableUriPermission',
      ]) {
        expect(importerSource, isNot(contains(forbidden)), reason: forbidden);
      }
      // The source interface has no write operation to call.
      final tree = read(copier);
      final start = tree.indexOf('interface ImportTree {');
      final body = tree.substring(start, tree.indexOf('\n}', start));
      expect(body, isNot(contains('delete')));
      expect(body, isNot(contains('write')));
      expect(body, isNot(contains('OutputStream')));
    });

    test('it copies into a fresh folder and never overwrites', () {
      final source = read(copier);
      expect(source, contains('WorkspaceImportRules.createFreshDestination('));
      expect(source, contains('target.exists()'));
      expect(
        source,
        contains('WorkspaceImportRules.isInside(destination, target)'),
      );
    });

    test('the notice needs workspace evidence, not a bare directory', () {
      // PC-DEF-077: an empty Download/pocketclaw made by hand in a file manager
      // was shown as an earlier workspace because the check was isDirectory.
      final channel = read('$kotlin/PocketClawMethodChannel.kt');
      final start = channel.indexOf('"getLegacyWorkspaceStatus" -> {');
      final body = channel.substring(
        start,
        channel.indexOf('"importLegacyWorkspace"', start),
      );
      expect(
        body,
        contains('LegacyWorkspaceImporter.legacyWorkspacePresent()'),
      );
      expect(
        read(importer),
        contains(
          'WorkspaceImportRules.looksLikeLegacyWorkspace(legacyDirectory())',
        ),
      );
      final detector = read(rules);
      final d = detector.indexOf('fun looksLikeLegacyWorkspace(');
      final detectorBody = detector.substring(
        d,
        detector.indexOf('\n    }', d),
      );
      for (final forbidden in const [
        'mkdir',
        'createNewFile',
        'writeText',
        'delete',
        'renameTo',
      ]) {
        expect(detectorBody, isNot(contains(forbidden)), reason: forbidden);
      }
    });

    test('hiding the notice only changes UI state', () {
      final page = read('lib/src/ui/config_page.dart');
      final start = page.indexOf(
        'Future<void> _hideLegacyWorkspaceNotice() async {',
      );
      final body = page.substring(start, page.indexOf('\n  }', start));
      expect(body, contains('setBool(_legacyWorkspaceNoticeHiddenKey, true)'));
      expect(body, isNot(contains('PocketClawChannel')));
    });

    test('the host results parse, and unknown statuses are failures', () {
      final copied = LegacyWorkspaceImportResult.fromMap(const {
        'status': 'copied',
        'folder': 'imported-from-downloads-20260924-120000',
        'files': 12,
        'failed': 0,
      });
      expect(copied.status, LegacyWorkspaceImportStatus.copied);
      expect(copied.files, 12);
      expect(copied.copiedAnything, isTrue);

      expect(
        LegacyWorkspaceImportResult.fromMap(const {
          'status': 'cancelled',
        }).copiedAnything,
        isFalse,
      );
      expect(
        LegacyWorkspaceImportResult.fromMap(const {
          'status': 'surprise',
        }).status,
        LegacyWorkspaceImportStatus.failed,
      );
      expect(LegacyWorkspaceStatus.fromMap(null).visible, isFalse);
    });
  });
}

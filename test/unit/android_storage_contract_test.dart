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

    test('it never deletes or rewrites the source', () {
      final source = read(importer);
      expect(source, isNot(contains('deleteDocument')));
      expect(
        source,
        isNot(contains('openOutputStream')),
        reason: 'the source tree is read-only to the import',
      );
      expect(
        source,
        isNot(contains('takePersistableUriPermission')),
        reason: 'a one-time copy keeps no standing grant',
      );
    });

    test(
      'the copy lands inside the agent workspace, where the agent can read it',
      () {
        final channel = read('$kotlin/PocketClawMethodChannel.kt');
        final start = channel.indexOf('"importLegacyWorkspace" -> {');
        final body = channel.substring(
          start,
          channel.indexOf('importer.start(', start),
        );
        expect(
          body,
          contains('PocketClawService.getWorkspacePath(context), "workspace"'),
          reason:
              'restrict_to_workspace hides anything beside the workspace directory',
        );
      },
    );

    test('it copies into a fresh folder and never overwrites', () {
      final source = read(importer);
      expect(source, contains('WorkspaceImportRules.freshDestination('));
      expect(source, contains('target.exists()'));
      expect(
        source,
        contains('WorkspaceImportRules.isInside(destination, target)'),
      );
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

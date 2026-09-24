/// PC-DEF-077. PocketClaw's workspace moved from the shared `Download/pocketclaw`
/// folder into app-specific storage. An older install may have left a workspace
/// behind; it is only ever copied in when the owner asks, into a folder of its
/// own, and the source is never touched.
library;

/// Whether an older shared workspace is visible to this install.
class LegacyWorkspaceStatus {
  const LegacyWorkspaceStatus({required this.path, required this.visible});

  factory LegacyWorkspaceStatus.fromMap(Map<Object?, Object?>? map) =>
      LegacyWorkspaceStatus(
        path: map?['path'] as String? ?? '',
        visible: map?['visible'] == true,
      );

  static const none = LegacyWorkspaceStatus(path: '', visible: false);

  final String path;
  final bool visible;
}

enum LegacyWorkspaceImportStatus {
  copied,
  partial,
  failed,
  cancelled,
  busy,
  unavailable;

  static LegacyWorkspaceImportStatus parse(Object? raw) => values.firstWhere(
    (value) => value.name == raw,
    orElse: () => LegacyWorkspaceImportStatus.failed,
  );
}

/// What one import did. [folder] is the new folder inside the workspace.
class LegacyWorkspaceImportResult {
  const LegacyWorkspaceImportResult({
    required this.status,
    this.folder = '',
    this.files = 0,
    this.failed = 0,
  });

  factory LegacyWorkspaceImportResult.fromMap(Map<Object?, Object?>? map) =>
      LegacyWorkspaceImportResult(
        status: LegacyWorkspaceImportStatus.parse(map?['status']),
        folder: map?['folder'] as String? ?? '',
        files: (map?['files'] as num?)?.toInt() ?? 0,
        failed: (map?['failed'] as num?)?.toInt() ?? 0,
      );

  final LegacyWorkspaceImportStatus status;
  final String folder;
  final int files;
  final int failed;

  /// Whether anything reached the workspace, so the notice has done its job.
  bool get copiedAnything =>
      status == LegacyWorkspaceImportStatus.copied ||
      status == LegacyWorkspaceImportStatus.partial;
}

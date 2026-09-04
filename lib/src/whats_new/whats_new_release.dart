import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';

/// Resolves one line of release-notes copy for the active locale.
///
/// Release structure lives in Dart; every user-facing word lives in the ARB
/// bundles, so a release note can never ship untranslated English prose.
typedef WhatsNewText = String Function(AppLocalizations l10n);

enum WhatsNewSectionKind { added, improved, fixed }

class WhatsNewSection {
  const WhatsNewSection({required this.kind, required this.bullets});

  final WhatsNewSectionKind kind;
  final List<WhatsNewText> bullets;

  String title(AppLocalizations l10n) => switch (kind) {
    WhatsNewSectionKind.added => l10n.whatsNewSectionNew,
    WhatsNewSectionKind.improved => l10n.whatsNewSectionImprovements,
    WhatsNewSectionKind.fixed => l10n.whatsNewSectionFixes,
  };
}

class WhatsNewRelease {
  const WhatsNewRelease({required this.version, required this.sections});

  /// The public release name, matching the app's versionName. Never a
  /// versionCode: an internal build can be recompiled without the user having
  /// anything new to read.
  final String version;
  final List<WhatsNewSection> sections;
}

final WhatsNewRelease whatsNewRelease020 = WhatsNewRelease(
  version: '0.2.0',
  sections: [
    WhatsNewSection(
      kind: WhatsNewSectionKind.added,
      bullets: [
        (l10n) => l10n.whatsNew020New1,
        (l10n) => l10n.whatsNew020New2,
        (l10n) => l10n.whatsNew020New3,
        (l10n) => l10n.whatsNew020New4,
        (l10n) => l10n.whatsNew020New5,
      ],
    ),
    WhatsNewSection(
      kind: WhatsNewSectionKind.improved,
      bullets: [
        (l10n) => l10n.whatsNew020Improvement1,
        (l10n) => l10n.whatsNew020Improvement2,
        (l10n) => l10n.whatsNew020Improvement3,
      ],
    ),
    WhatsNewSection(
      kind: WhatsNewSectionKind.fixed,
      bullets: [
        (l10n) => l10n.whatsNew020Fix1,
        (l10n) => l10n.whatsNew020Fix2,
      ],
    ),
  ],
);

/// The release the What's New page renders.
final WhatsNewRelease currentWhatsNewRelease = whatsNewRelease020;

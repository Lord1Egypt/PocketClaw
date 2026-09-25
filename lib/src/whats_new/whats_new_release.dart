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
        // PC-DEF-064. New5 used to read "Telegram integration, set up from
        // Settings", which stopped being where setup happens: the managed flow
        // is offered in the app and in a browser, and the owner pairing is the
        // part worth saying.
        (l10n) => l10n.whatsNew020New5,
        (l10n) => l10n.whatsNew020New7,
        (l10n) => l10n.whatsNew020New6,
        (l10n) => l10n.whatsNew020New8,
      ],
    ),
    WhatsNewSection(
      kind: WhatsNewSectionKind.improved,
      bullets: [
        (l10n) => l10n.whatsNew020Improvement1,
        (l10n) => l10n.whatsNew020Improvement2,
        (l10n) => l10n.whatsNew020Improvement3,
        (l10n) => l10n.whatsNew020Improvement4,
        (l10n) => l10n.whatsNew020Improvement5,
        (l10n) => l10n.whatsNew020Improvement6,
        (l10n) => l10n.whatsNew020Improvement7,
        (l10n) => l10n.whatsNew020Improvement8,
        // PC-DEF-064. Added after physical verification of each: the automatic
        // Telegram apply, the PC-E-AI configuration messages, the post-sign-in
        // destination, the logging redaction, and the unclaimed-Dashboard rule.
        (l10n) => l10n.whatsNew020Improvement12,
        (l10n) => l10n.whatsNew020Improvement13,
        (l10n) => l10n.whatsNew020Improvement14,
        (l10n) => l10n.whatsNew020Improvement15,
        (l10n) => l10n.whatsNew020Improvement16,
        // The three one-time effects of upgrading into this release. They are
        // here rather than in Fixes because nothing was broken: they are what
        // the user should expect to see once, and then not again. They stay
        // last: they are what to expect after installing, not what is new.
        (l10n) => l10n.whatsNew020Improvement9,
        (l10n) => l10n.whatsNew020Improvement10,
        (l10n) => l10n.whatsNew020Improvement11,
      ],
    ),
    WhatsNewSection(
      kind: WhatsNewSectionKind.fixed,
      bullets: [(l10n) => l10n.whatsNew020Fix1, (l10n) => l10n.whatsNew020Fix2],
    ),
  ],
);

final WhatsNewRelease whatsNewRelease021 = WhatsNewRelease(
  version: '0.2.1',
  sections: [
    WhatsNewSection(
      kind: WhatsNewSectionKind.fixed,
      bullets: [(l10n) => l10n.whatsNew021Fix1],
    ),
  ],
);

final WhatsNewRelease whatsNewRelease022 = WhatsNewRelease(
  version: '0.2.2',
  sections: [
    WhatsNewSection(
      kind: WhatsNewSectionKind.improved,
      bullets: [
        (l10n) => l10n.whatsNew022Improvement1,
        (l10n) => l10n.whatsNew022Improvement2,
        (l10n) => l10n.whatsNew022Improvement3,
        (l10n) => l10n.whatsNew022Improvement4,
      ],
    ),
    WhatsNewSection(
      kind: WhatsNewSectionKind.fixed,
      bullets: [
        (l10n) => l10n.whatsNew022Fix1,
        (l10n) => l10n.whatsNew022Fix2,
        (l10n) => l10n.whatsNew022Fix3,
      ],
    ),
  ],
);

/// Every release shown in What's New, newest first.
///
/// This is a history, not a slot: a new release is added at the front and the
/// older ones stay, so the page keeps showing what came before. Only the
/// first entry is the current release; tool/release_notes.py and the "new"
/// badge read it from here.
final List<WhatsNewRelease> whatsNewHistory = [
  whatsNewRelease022,
  whatsNewRelease021,
  whatsNewRelease020,
];

/// The release this build introduces: the newest entry of [whatsNewHistory].
WhatsNewRelease get currentWhatsNewRelease => whatsNewHistory.first;

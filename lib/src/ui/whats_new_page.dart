import 'package:flutter/material.dart';

import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';
import 'package:pocketclaw/src/whats_new/whats_new_release.dart';

const String _whatsNewProductName = 'PocketClaw';

/// Full-page release notes, reached from Settings.
///
/// A page rather than a dialog: the notes are read, not acknowledged, and a
/// dialog cannot hold three sections on a phone without scrolling inside a
/// scroll.
///
/// It shows the whole history, newest first. The current release is open;
/// earlier ones are collapsed so the page stays short, and each opens in
/// place. Everything is in the app bundle, so it works offline.
class WhatsNewPage extends StatelessWidget {
  const WhatsNewPage({super.key, this.releases});

  /// Newest first. Defaults to [whatsNewHistory].
  final List<WhatsNewRelease>? releases;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final history = releases ?? whatsNewHistory;
    final current = history.first;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.whatsNewTitle)),
      body: ListView(
        padding: const EdgeInsetsDirectional.fromSTEB(16, 16, 16, 32),
        children: [
          Text(
            '$_whatsNewProductName ${current.version}',
            style: theme.textTheme.titleLarge?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            l10n.whatsNewDescription,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 20),
          for (final section in current.sections)
            _WhatsNewSectionCard(section: section),
          for (final release in history.skip(1))
            _EarlierRelease(release: release),
        ],
      ),
    );
  }
}

/// One earlier release: its title, collapsed until tapped.
class _EarlierRelease extends StatelessWidget {
  const _EarlierRelease({required this.release});

  final WhatsNewRelease release;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsetsDirectional.only(top: 8),
      child: ExpansionTile(
        key: ValueKey('whatsNewRelease-${release.version}'),
        tilePadding: EdgeInsetsDirectional.zero,
        childrenPadding: EdgeInsetsDirectional.zero,
        shape: const Border(),
        collapsedShape: const Border(),
        expandedCrossAxisAlignment: CrossAxisAlignment.start,
        title: Text(
          '$_whatsNewProductName ${release.version}',
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        children: [
          for (final section in release.sections)
            _WhatsNewSectionCard(section: section),
        ],
      ),
    );
  }
}

class _WhatsNewSectionCard extends StatelessWidget {
  const _WhatsNewSectionCard({required this.section});

  final WhatsNewSection section;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    final tokens = context.aperture;
    return Padding(
      padding: const EdgeInsetsDirectional.only(bottom: 12),
      // The bracket is the same device as active navigation and the assistant
      // rail, tinted by what the section is: what was fixed, what is new, what
      // improved. It is drawn on a logical edge, so Arabic mirrors it.
      child: ApertureBracket(
        fill: tokens.surface1,
        bracket: _accentFor(tokens),
        borderColor: tokens.border,
        borderWidth: 1,
        padding: const EdgeInsetsDirectional.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              section.title(l10n),
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 4),
            for (final bullet in section.bullets)
              _WhatsNewBullet(text: bullet(l10n)),
          ],
        ),
      ),
    );
  }

  Color _accentFor(ApertureColors tokens) => switch (section.kind) {
    WhatsNewSectionKind.added => tokens.accent,
    WhatsNewSectionKind.improved => tokens.warning,
    WhatsNewSectionKind.fixed => tokens.success,
  };
}

class _WhatsNewBullet extends StatelessWidget {
  const _WhatsNewBullet({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsetsDirectional.only(top: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            key: const Key('whatsNewBulletMarker'),
            margin: const EdgeInsetsDirectional.only(top: 7, end: 10),
            width: 5,
            height: 5,
            decoration: BoxDecoration(
              color: theme.colorScheme.primary,
              shape: BoxShape.circle,
            ),
          ),
          Expanded(child: Text(text, style: theme.textTheme.bodyMedium)),
        ],
      ),
    );
  }
}

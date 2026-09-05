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
class WhatsNewPage extends StatelessWidget {
  const WhatsNewPage({super.key, this.release});

  final WhatsNewRelease? release;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final resolved = release ?? currentWhatsNewRelease;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.whatsNewTitle)),
      body: ListView(
        padding: const EdgeInsetsDirectional.fromSTEB(16, 16, 16, 32),
        children: [
          Text(
            '$_whatsNewProductName ${resolved.version}',
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
          for (final section in resolved.sections)
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

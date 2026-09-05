import 'package:flutter/material.dart';

import 'package:pocketclaw/src/core/aperture_theme.dart';
import 'package:pocketclaw/src/generated/l10n/app_localizations.dart';

/// The console route that manages models, providers and the default selection.
const String canonicalModelsConsolePath = '/models';

/// A shortcut to the existing model-management page in the Dashboard.
///
/// It navigates and nothing else. Providers, API keys, the model catalog, the
/// default-model choice and model testing all already live in that page, and
/// duplicating any of them in native Settings would create a second writer for
/// state Core owns. This card exists because a beginner should not have to
/// discover the Dashboard navigation to find models.
class ModelsSettingsCard extends StatelessWidget {
  const ModelsSettingsCard({super.key, required this.onManage});

  final Future<void> Function(String path)? onManage;

  @override
  Widget build(BuildContext context) {
    final tokens = context.aperture;
    final l10n = AppLocalizations.of(context)!;
    return Card(
      margin: EdgeInsets.zero,
      child: ListTile(
        key: const Key('models-settings-card'),
        leading: Container(
          width: 40,
          height: 40,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: tokens.surface3,
            borderRadius: BorderRadius.circular(ApertureTheme.radiusSm),
          ),
          child: Icon(
            Icons.auto_awesome_outlined,
            color: tokens.accent,
            size: 20,
          ),
        ),
        title: Text(l10n.manageModelsTitle),
        subtitle: Text(l10n.manageModelsDescription),
        trailing: Icon(Icons.chevron_right_rounded, color: tokens.textFaint),
        onTap: onManage == null
            ? null
            : () => onManage!(canonicalModelsConsolePath),
      ),
    );
  }
}

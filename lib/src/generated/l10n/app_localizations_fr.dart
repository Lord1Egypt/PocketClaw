// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for French (`fr`).
class AppLocalizationsFr extends AppLocalizations {
  AppLocalizationsFr([String locale = 'fr']) : super(locale);

  @override
  String get stop => 'Arrêter';

  @override
  String get webAdmin => 'Admin Web';

  @override
  String get logs => 'Journaux';

  @override
  String get statusRunning => 'En cours';

  @override
  String get statusStopped => 'Arrêté';

  @override
  String get settings => 'Paramètres';

  @override
  String get address => 'Adresse';

  @override
  String get port => 'Port';

  @override
  String get save => 'Enregistrer';

  @override
  String get notStarted => 'Service non démarré';

  @override
  String get startHint =>
      'Veuillez d\'abord démarrer le service depuis le tableau de bord.';

  @override
  String get goToDashboard => 'Aller au tableau de bord';

  @override
  String get back => 'Retour';

  @override
  String get forward => 'Suivant';

  @override
  String get refresh => 'Actualiser';

  @override
  String get publicMode => 'Mode public';

  @override
  String get publicModeHintDesc =>
      'Lorsqu\'il est activé, le service autorise l\'accès externe et le champ d\'adresse sera désactivé';

  @override
  String get publicModeApplying => 'Application du mode réseau...';

  @override
  String get themeSelection => 'Thème';

  @override
  String get launchService => 'DÉMARRER LE SERVICE';

  @override
  String get stopService => 'ARRÊTER LE SERVICE';

  @override
  String get endpoint => 'POINT D\'ACCÈS';

  @override
  String get statusActive => 'ACTIF';

  @override
  String get statusSyncing => 'SYNCHRONISATION';

  @override
  String get statusIdle => 'INACTIF';

  @override
  String get publicModeEnabled => 'Mode public activé';

  @override
  String get localMode => 'Mode local';

  @override
  String get unableToGetDeviceIp =>
      'Impossible d\'obtenir l\'IP de l\'appareil';

  @override
  String get localModeHint =>
      '1. Accédez à la configuration du service\n2. Activez le mode public\n3. Scannez le code QR pour accéder à PocketClaw';

  @override
  String get publicModeHint =>
      '1. Démarrez le service\n2. Scannez le code QR pour accéder à PocketClaw';

  @override
  String get noLogsToExport => 'Aucun journal à exporter';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Journaux enregistrés dans Téléchargements: $path';
  }

  @override
  String get shareLogsText => 'Journaux PocketClaw';

  @override
  String get workspaceDirectory => 'Espace de travail';

  @override
  String get legacyWorkspaceTitle => 'Ancien espace de travail trouvé';

  @override
  String legacyWorkspaceBody(Object path) {
    return 'Une version précédente conservait votre espace de travail dans $path. PocketClaw utilise désormais le stockage propre à l’application et n’a pas touché à ce dossier. Vous pouvez le copier : la copie va dans un dossier distinct, et rien n’est écrasé ni supprimé.';
  }

  @override
  String get legacyWorkspaceImport => 'Copier dans l’espace de travail';

  @override
  String get legacyWorkspaceHide => 'Masquer';

  @override
  String legacyWorkspaceCopied(Object count, Object folder) {
    return '$count fichiers copiés dans $folder.';
  }

  @override
  String legacyWorkspacePartial(Object count, Object folder, Object failed) {
    return '$count fichiers copiés dans $folder. $failed n’ont pas pu être copiés.';
  }

  @override
  String get legacyWorkspaceFailed => 'Le dossier n’a pas pu être copié.';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'Journaux enregistrés dans Téléchargements (bibliothèque multimédia Android): $name';
  }

  @override
  String shareFailed(Object error) {
    return 'Échec de l\'ouverture de la boîte de dialogue de partage: $error';
  }

  @override
  String get exportLogs => 'Exporter les journaux';

  @override
  String logEventsCount(int count) {
    return '$count ÉVÉNEMENTS';
  }

  @override
  String get cancel => 'Annuler';

  @override
  String get language => 'Langue';

  @override
  String get selectLanguage => 'Sélectionner la langue';

  @override
  String get about => 'À propos';

  @override
  String get aboutDescription =>
      'PocketClaw est votre espace de travail privé pour assistant IA.';

  @override
  String get aboutAppVersionLabel => 'Version de PocketClaw';

  @override
  String get aboutCoreVersionLabel => 'Version du runtime';

  @override
  String get aboutVersionUnavailable => 'Indisponible';

  @override
  String get close => 'Fermer';

  @override
  String get contextMemoryTitle => 'Mémoire de contexte Telegram';

  @override
  String get contextMemoryDescription =>
      'Détermine le nombre de messages récents envoyés à l\'IA. Les conversations plus anciennes restent dans Telegram et sont représentées par le résumé évolutif.';

  @override
  String get contextMemoryHelp =>
      'Davantage de messages apportent plus de contexte récent mais consomment plus de jetons. Les messages plus anciens restent dans Telegram et peuvent être conservés via le résumé évolutif.';

  @override
  String get contextMemoryRecommended => 'Recommandé';

  @override
  String get contextMemoryCustom => 'Personnalisé';

  @override
  String get contextMemoryCustomLabel => 'Messages';

  @override
  String contextMemoryRangeError(int min, int max) {
    return 'Saisissez un nombre entier entre $min et $max.';
  }

  @override
  String get contextMemorySaveFailed =>
      'Impossible d\'enregistrer ce paramètre.';

  @override
  String get settingsSave => 'Enregistrer';

  @override
  String get autoStartServiceTitle =>
      'Démarrer le service PocketClaw automatiquement';

  @override
  String get autoStartGatewayTitle => 'Démarrer la passerelle automatiquement';

  @override
  String get autoStartPreferenceOn =>
      'Préférence de démarrage automatique : ACTIVÉE';

  @override
  String get autoStartPreferenceOff =>
      'Préférence de démarrage automatique : DÉSACTIVÉE';

  @override
  String get runtimeRunning => 'Exécution : en cours';

  @override
  String get runtimeStarting => 'Exécution : démarrage';

  @override
  String get runtimeStopped => 'Exécution : arrêté';

  @override
  String get gatewayAutoStartHint =>
      'S\'applique au prochain démarrage du service PocketClaw. L\'exécution de la passerelle se gère dans le tableau de bord.';

  @override
  String get manageTelegramConnection => 'Gérer la connexion Telegram';

  @override
  String get manageModelsTitle => 'Gérer les modèles';

  @override
  String get manageModelsDescription =>
      'Ajoutez, modifiez, testez et choisissez des modèles d\'IA.';

  @override
  String get githubChecking => 'Vérification…';

  @override
  String get githubConnected => 'Connecté';

  @override
  String get githubNotConnected => 'Non connecté';

  @override
  String githubConnectedAs(String login) {
    return 'Connecté en tant que $login';
  }

  @override
  String get githubDescription =>
      'Utilisé par le gh intégré et par Git via HTTPS. Le jeton est chiffré sur cet appareil et n\'est plus jamais affiché.';

  @override
  String get githubTestConnection => 'Tester la connexion';

  @override
  String get githubDisconnect => 'Déconnecter';

  @override
  String get githubConnectAction => 'Connecter GitHub';

  @override
  String get githubConnect => 'Connecter';

  @override
  String get githubTokenLabel => 'Jeton d\'accès personnel';

  @override
  String get githubTokenHint =>
      'Collez un jeton d\'accès personnel GitHub avec les portées nécessaires (repo pour les dépôts privés).';

  @override
  String get githubTokenRejected => 'GitHub n\'a pas accepté ce jeton.';

  @override
  String get githubAuthWorking => 'L\'authentification GitHub fonctionne.';

  @override
  String get githubAuthNotWorking =>
      'L\'authentification GitHub ne fonctionne pas.';

  @override
  String githubAuthenticatedAs(String login) {
    return 'Authentifié en tant que $login.';
  }

  @override
  String get githubCredentialRemoveFailed =>
      'Impossible de supprimer les identifiants.';

  @override
  String get githubYourAccount => 'votre compte GitHub';

  @override
  String githubConnectedReport(String who, String outcome) {
    return 'Connecté en tant que $who. $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return 'Déconnecté. $outcome';
  }

  @override
  String get credentialAppliedNow =>
      'gh et git peuvent l\'utiliser dès maintenant.';

  @override
  String get credentialAppliesNextStart =>
      'Il sera utilisé au prochain démarrage de PocketClaw.';

  @override
  String get credentialAppliesDeferred =>
      'Enregistré. PocketClaw est en cours de démarrage ; le changement s\'appliquera automatiquement dès la fin.';

  @override
  String get whatsNewTitle => 'Nouveautés';

  @override
  String get whatsNewDescription =>
      'Les principaux changements de cette version.';

  @override
  String get whatsNewBadge => 'NOUVEAU';

  @override
  String get whatsNewSectionNew => 'Nouveautés';

  @override
  String get whatsNewSectionImprovements => 'Améliorations';

  @override
  String get whatsNewSectionFixes => 'Corrections';

  @override
  String get whatsNew020New1 =>
      'L\'environnement d\'exécution géré installe les outils fournis avec PocketClaw et les maintient à jour.';

  @override
  String get whatsNew020New2 =>
      'Outils de développement fournis : Git, GitHub CLI, curl, ripgrep, jq et SQLite.';

  @override
  String get whatsNew020New3 =>
      'Prise en charge de l\'environnement Python 3.14 fourni avec PocketClaw.';

  @override
  String get whatsNew020New4 =>
      'Connexion sécurisée à GitHub, partagée par Git et la CLI GitHub fournis.';

  @override
  String get whatsNew020New5 =>
      'Configuration de Telegram en un geste : PocketClaw crée votre propre bot, dans l\'application ou depuis le tableau de bord dans un navigateur. Aucun jeton à copier, et seul votre compte peut lui parler.';

  @override
  String get whatsNew020New6 =>
      'Une vue État sur le tableau de bord : travail en cours, canaux, modèle utilisé et ressources d\'exécution.';

  @override
  String get whatsNew020Improvement1 =>
      'Des fournisseurs plus robustes : une requête en échec ne met plus fin au tour.';

  @override
  String get whatsNew020Improvement2 =>
      'Un catalogue de l\'environnement d\'exécution géré plus précis.';

  @override
  String get whatsNew020Improvement3 =>
      'Une identité PocketClaw plus nette dans l\'interface web et dans l\'espace de travail par défaut.';

  @override
  String get whatsNew020Improvement4 =>
      'Une nouvelle icône d\'application et un écran À propos redessiné, dans le design Aperture de PocketClaw.';

  @override
  String get whatsNew020Improvement5 =>
      'Les réponses de l\'assistant paraissent plus naturelles, sans signature fixe à la fin.';

  @override
  String get whatsNew020Improvement6 =>
      'PocketClaw ne demande plus l\'autorisation Téléphone : rien dans l\'application ne s\'en servait.';

  @override
  String get whatsNew020Improvement7 =>
      'Les journaux de diagnostic sont désormais conservés en privé dans l\'application plutôt que dans le dossier Téléchargements. Votre espace de travail ne bouge pas.';

  @override
  String get whatsNew020Improvement8 =>
      'Une mise à niveau importante de la façon dont PocketClaw conserve son état et exécute ses services, pour une fiabilité plus régulière au quotidien.';

  @override
  String get whatsNew020Improvement9 =>
      'Après cette mise à jour, le tableau de bord vous demande de vous reconnecter une fois.';

  @override
  String get whatsNew020Improvement10 =>
      'Après cette mise à jour, le canal de discussion Web démarre une nouvelle conversation.';

  @override
  String get whatsNew020Improvement11 =>
      'Après cette mise à jour, il vaut la peine de vérifier une fois vos préférences de notification.';

  @override
  String get whatsNew020Fix1 =>
      'La passerelle se rétablit désormais après un enregistrement de processus obsolète laissé par une exécution précédente.';

  @override
  String get whatsNew020Fix2 =>
      'Les listes de canaux comportant plusieurs entrées sont correctement conservées lors de l\'enregistrement.';

  @override
  String get whatsNew021Fix1 =>
      'Le Git intégré ne plante plus lors du clonage d\'un dépôt ou de la mise à jour d\'une branche.';

  @override
  String get settingsGroupConnection => 'Connexion';

  @override
  String get settingsGroupAgent => 'Agent';

  @override
  String get settingsGroupIntegrations => 'Intégrations';

  @override
  String get settingsGroupAppearance => 'Apparence';

  @override
  String get statusTitle => 'État';

  @override
  String get statusSectionSystem => 'Système';

  @override
  String get statusSectionAi => 'IA';

  @override
  String get statusSectionActivity => 'Activité';

  @override
  String get statusSectionChannels => 'Canaux';

  @override
  String get statusSectionResources => 'Ressources';

  @override
  String get statusSinceGatewayStart => 'Depuis le démarrage de la passerelle';

  @override
  String get statusGateway => 'Passerelle';

  @override
  String get statusUptime => 'Temps de fonctionnement';

  @override
  String get statusAppVersion => 'Version de l\'application';

  @override
  String get statusCoreVersion => 'Version du Core';

  @override
  String get statusActiveModel => 'Modèle actif';

  @override
  String get statusConfiguredDefault => 'Valeur par défaut configurée';

  @override
  String get statusProvider => 'Fournisseur';

  @override
  String get statusFallbacks => 'Solutions de repli';

  @override
  String get statusActiveTurns => 'Tours actifs';

  @override
  String get statusActiveSubagents => 'Sous-agents actifs';

  @override
  String get statusWaiting => 'En attente';

  @override
  String get statusCompleted => 'Terminés';

  @override
  String get statusFailed => 'Échoués';

  @override
  String get statusCancelled => 'Annulés';

  @override
  String get statusToolCalls => 'Appels d\'outils';

  @override
  String get statusToolCallsFailed => 'Appels d\'outils échoués';

  @override
  String get statusLastActivity => 'Dernière activité';

  @override
  String get statusCoreMemory => 'Mémoire du Core';

  @override
  String get statusCoreCpuTime => 'Temps CPU du Core';

  @override
  String get statusNoChannels => 'Aucun canal configuré';

  @override
  String get statusDetailUnavailable =>
      'L\'état détaillé n\'est pas disponible';

  @override
  String get statusJustNow => 'À l\'instant';

  @override
  String statusMinutesAgo(int minutes) {
    return 'il y a $minutes min';
  }

  @override
  String statusHoursAgo(int hours) {
    return 'il y a $hours h';
  }

  @override
  String statusDaysAgo(int days) {
    return 'il y a $days j';
  }

  @override
  String statusDurationSeconds(int seconds) {
    return '$seconds s';
  }

  @override
  String statusDurationMinutes(int minutes, int seconds) {
    return '$minutes min $seconds s';
  }

  @override
  String statusDurationHours(int hours, int minutes) {
    return '$hours h $minutes min';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '$days j $hours h';
  }

  @override
  String get notificationPermissionTitle => 'Notifications';

  @override
  String get notificationPermissionGranted =>
      'PocketClaw peut afficher sa notification d\'exécution.';

  @override
  String get notificationPermissionBlocked =>
      'Les notifications sont désactivées, la notification d\'exécution de PocketClaw n\'apparaîtra donc pas.';

  @override
  String get notificationPermissionOpenSettings =>
      'Ouvrir les paramètres de notification';

  @override
  String get whatsNew020New7 =>
      'Gestion de Telegram depuis le tableau de bord : connecter un bot, le remplacer ou le déconnecter.';

  @override
  String get whatsNew020New8 =>
      'Gestion des fournisseurs et des modèles dans le tableau de bord : ajouter un fournisseur, changer une clé d\'API ou supprimer un modèle.';

  @override
  String get whatsNew020Improvement12 =>
      'Les réglages Telegram s\'appliquent dès l\'enregistrement, sans redémarrage manuel.';

  @override
  String get whatsNew020Improvement13 =>
      'Lorsqu\'aucun modèle d\'IA n\'est configuré, PocketClaw indique précisément ce qui manque et vous donne un code à consulter.';

  @override
  String get whatsNew020Improvement14 =>
      'La connexion au tableau de bord vous ramène à l\'écran que vous avez demandé.';

  @override
  String get whatsNew020Improvement15 =>
      'Les journaux de diagnostic restent détaillés sans jamais contenir vos clés, vos jetons ni le texte de vos messages.';

  @override
  String get whatsNew020Improvement16 =>
      'Un tableau de bord que personne n\'a encore revendiqué n\'est jamais joignable depuis le réseau, même avec le mode public activé.';
}

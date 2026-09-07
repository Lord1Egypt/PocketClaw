// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for French (`fr`).
class AppLocalizationsFr extends AppLocalizations {
  AppLocalizationsFr([String locale = 'fr']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => 'Exécuter';

  @override
  String get stop => 'Arrêter';

  @override
  String get config => 'Config';

  @override
  String get webAdmin => 'Admin Web';

  @override
  String get logs => 'Journaux';

  @override
  String get viewLogs => 'Voir les journaux';

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
  String get showWindow => 'Afficher la fenêtre';

  @override
  String get exit => 'Quitter';

  @override
  String get binaryPath => 'Chemin binaire';

  @override
  String get browse => 'Parcourir';

  @override
  String get pathError => 'Chemin invalide';

  @override
  String get arguments => 'Arguments';

  @override
  String get argumentsHint => 'ex. config.json';

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
  String get coreBinaryMissing =>
      'Binaire principal introuvable. Placez le binaire de la plateforme dans app/bin/ ou définissez le chemin dans Paramètres.';

  @override
  String get coreStartFailed => 'Échec du démarrage du service principal.';

  @override
  String get coreStopFailed => 'Échec de l\'arrêt du service principal.';

  @override
  String get coreInvalidBinary => 'Fichier binaire principal invalide.';

  @override
  String coreUnknownError(Object code) {
    return 'Erreur principale inconnue: $code';
  }

  @override
  String get coreValid => 'Le binaire principal est valide.';

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
  String get check => 'Vérifier';

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
  String get deviceReportingTitle =>
      'Commentaires de compatibilité de l\'appareil';

  @override
  String get deviceReportingSubtitle =>
      'Utilisé uniquement pour vérifier la compatibilité de la version du système d\'exploitation et de la version de l\'application. Aucune implication dans les messages de chat, les détails de compte ou le contenu personnel';

  @override
  String get deviceReportingConsentTitle =>
      'Aidez à améliorer la compatibilité des appareils';

  @override
  String get deviceReportingConsentDescription =>
      'Lorsqu\'il est activé, seul un ID d\'installation anonyme, la version du système d\'exploitation et la version de l\'application sont envoyés pour comprendre la compatibilité. La langue et la région peuvent être collectées séparément par Firebase Analytics. Aucun message de chat, contenu saisi, détails de compte, fichiers ou paramètres personnalisés ne sont téléchargés';

  @override
  String get deviceReportingBannerDescription =>
      'Seuls un ID d\'installation anonyme, la version du système d\'exploitation et la version de l\'application sont synchronisés pour améliorer la compatibilité. La langue et la région peuvent être collectées séparément par Firebase Analytics. Aucun message de chat, détail de compte, fichier ou contenu personnel n\'est envoyé';

  @override
  String get deviceReportingWhatWillBeSent =>
      'Seuls ces détails de l\'appareil sont inclus';

  @override
  String get deviceReportingDeviceLabel => 'Modèle de l\'appareil';

  @override
  String get deviceReportingPlatformLabel => 'Catégorie de l\'appareil';

  @override
  String get deviceReportingSystemLabel => 'Version du système d\'exploitation';

  @override
  String get deviceReportingTimingNote =>
      'Une synchronisation s\'exécute une fois à l\'activation, et à nouveau uniquement après la détection d\'une mise à jour du système';

  @override
  String get deviceReportingDeny => 'Pas maintenant';

  @override
  String get deviceReportingAllow => 'Activer';

  @override
  String get deviceReportingUploadSucceeded =>
      'Commentaires de compatibilité de l\'appareil activés';

  @override
  String get deviceReportingUploadFailed =>
      'Commentaires de compatibilité de l\'appareil activés, mais la synchronisation actuelle des informations de l\'appareil n\'est pas terminée';

  @override
  String get deviceReportingDisabled =>
      'Commentaires de compatibilité de l\'appareil désactivés';

  @override
  String get localModeHint =>
      '1. Accédez à la configuration du service\n2. Activez le mode public\n3. Scannez le code QR pour accéder à PocketClaw';

  @override
  String get publicModeHint =>
      '1. Démarrez le service\n2. Scannez le code QR pour accéder à PocketClaw';

  @override
  String get noLogsToExport => 'Aucun journal à exporter';

  @override
  String get logsSavedToMediaLibrary =>
      'Journaux enregistrés dans Téléchargements (bibliothèque multimédia Android)';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Journaux enregistrés dans Téléchargements: $path';
  }

  @override
  String get shareLogsText => 'Journaux PocketClaw';

  @override
  String get workspaceDirectory => 'Espace de travail';

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
  String get unsavedChanges => 'Modifications non enregistrées';

  @override
  String get unsavedChangesHint =>
      'Vous avez des modifications non enregistrées. Voulez-vous les abandonner ?';

  @override
  String get cancel => 'Annuler';

  @override
  String get discard => 'Abandonner';

  @override
  String get saved => 'Enregistré';

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
      'Intégration de Telegram, à configurer depuis les Paramètres.';

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
  String get whatsNew020Fix1 =>
      'La passerelle se rétablit désormais après un enregistrement de processus obsolète laissé par une exécution précédente.';

  @override
  String get whatsNew020Fix2 =>
      'Les listes de canaux comportant plusieurs entrées sont correctement conservées lors de l\'enregistrement.';

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
}

// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for German (`de`).
class AppLocalizationsDe extends AppLocalizations {
  AppLocalizationsDe([String locale = 'de']) : super(locale);

  @override
  String get stop => 'Stoppen';

  @override
  String get webAdmin => 'Web-Admin';

  @override
  String get logs => 'Protokolle';

  @override
  String get statusRunning => 'Läuft';

  @override
  String get statusStopped => 'Gestoppt';

  @override
  String get settings => 'Einstellungen';

  @override
  String get address => 'Adresse';

  @override
  String get port => 'Port';

  @override
  String get save => 'Speichern';

  @override
  String get notStarted => 'Dienst nicht gestartet';

  @override
  String get startHint =>
      'Bitte starten Sie den Dienst zuerst über das Dashboard.';

  @override
  String get goToDashboard => 'Zum Dashboard';

  @override
  String get back => 'Zurück';

  @override
  String get forward => 'Vorwärts';

  @override
  String get refresh => 'Aktualisieren';

  @override
  String get publicMode => 'Öffentlicher Modus';

  @override
  String get publicModeHintDesc =>
      'Wenn aktiviert, erlaubt der Dienst externen Zugriff und das Adressfeld wird deaktiviert';

  @override
  String get publicModeApplying => 'Netzwerkmodus wird angewendet...';

  @override
  String get themeSelection => 'Thema';

  @override
  String get launchService => 'DIENST STARTEN';

  @override
  String get stopService => 'DIENST STOPPEN';

  @override
  String get endpoint => 'ENDPUNKT';

  @override
  String get statusActive => 'AKTIV';

  @override
  String get statusSyncing => 'SYNCHRONISIERT';

  @override
  String get statusIdle => 'INAKTIV';

  @override
  String get publicModeEnabled => 'Öffentlicher Modus aktiviert';

  @override
  String get localMode => 'Lokaler Modus';

  @override
  String get unableToGetDeviceIp =>
      'IP-Adresse des Geräts kann nicht abgerufen werden';

  @override
  String get localModeHint =>
      '1. Gehen Sie zur Dienstkonfiguration\n2. Aktivieren Sie den öffentlichen Modus\n3. Scannen Sie den QR-Code für den Zugriff auf PocketClaw';

  @override
  String get publicModeHint =>
      '1. Starten Sie den Dienst\n2. Scannen Sie den QR-Code für den Zugriff auf PocketClaw';

  @override
  String get noLogsToExport => 'Keine Protokolle zum Exportieren';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Protokolle in Downloads gespeichert: $path';
  }

  @override
  String get shareLogsText => 'PocketClaw-Protokolle';

  @override
  String get workspaceDirectory => 'Arbeitsbereich';

  @override
  String get legacyWorkspaceTitle => 'Früherer Arbeitsbereich gefunden';

  @override
  String legacyWorkspaceBody(Object path) {
    return 'Eine ältere Version hat deinen Arbeitsbereich in $path gespeichert. PocketClaw nutzt jetzt seinen eigenen App-Speicher und hat diesen Ordner unverändert gelassen. Du kannst ihn hineinkopieren: Die Kopie landet in einem eigenen Ordner, und nichts wird überschrieben oder gelöscht.';
  }

  @override
  String get legacyWorkspaceImport => 'In den Arbeitsbereich kopieren';

  @override
  String get legacyWorkspaceHide => 'Ausblenden';

  @override
  String legacyWorkspaceCopied(Object count, Object folder) {
    return '$count Dateien nach $folder kopiert.';
  }

  @override
  String legacyWorkspacePartial(Object count, Object folder, Object failed) {
    return '$count Dateien nach $folder kopiert. $failed konnten nicht kopiert werden.';
  }

  @override
  String get legacyWorkspaceFailed => 'Der Ordner konnte nicht kopiert werden.';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'Protokolle in Downloads gespeichert (Android-Medienbibliothek): $name';
  }

  @override
  String shareFailed(Object error) {
    return 'Freigabedialog konnte nicht geöffnet werden: $error';
  }

  @override
  String get exportLogs => 'Protokolle exportieren';

  @override
  String logEventsCount(int count) {
    return '$count EREIGNISSE';
  }

  @override
  String get cancel => 'Abbrechen';

  @override
  String get language => 'Sprache';

  @override
  String get selectLanguage => 'Sprache auswählen';

  @override
  String get about => 'Über';

  @override
  String get aboutDescription =>
      'PocketClaw ist Ihr privater Arbeitsbereich für KI-Assistenten.';

  @override
  String get aboutAppVersionLabel => 'PocketClaw-Version';

  @override
  String get aboutCoreVersionLabel => 'Laufzeitversion';

  @override
  String get aboutVersionUnavailable => 'Nicht verfügbar';

  @override
  String get close => 'Schließen';

  @override
  String get contextMemoryTitle => 'Telegram-Kontextspeicher';

  @override
  String get contextMemoryDescription =>
      'Legt fest, wie viele der letzten Nachrichten an die KI gesendet werden. Ältere Unterhaltungen bleiben in Telegram und werden durch die fortlaufende Zusammenfassung abgebildet.';

  @override
  String get contextMemoryHelp =>
      'Mehr Nachrichten liefern mehr aktuellen Kontext, verbrauchen aber mehr Tokens. Ältere Nachrichten bleiben in Telegram und können über die fortlaufende Zusammenfassung erhalten bleiben.';

  @override
  String get contextMemoryRecommended => 'Empfohlen';

  @override
  String get contextMemoryCustom => 'Benutzerdefiniert';

  @override
  String get contextMemoryCustomLabel => 'Nachrichten';

  @override
  String contextMemoryRangeError(int min, int max) {
    return 'Gib eine ganze Zahl zwischen $min und $max ein.';
  }

  @override
  String get contextMemorySaveFailed =>
      'Diese Einstellung konnte nicht gespeichert werden.';

  @override
  String get settingsSave => 'Speichern';

  @override
  String get autoStartServiceTitle => 'PocketClaw-Dienst automatisch starten';

  @override
  String get autoStartGatewayTitle => 'Gateway automatisch starten';

  @override
  String get autoStartPreferenceOn => 'Autostart-Einstellung: EIN';

  @override
  String get autoStartPreferenceOff => 'Autostart-Einstellung: AUS';

  @override
  String get runtimeRunning => 'Laufzeit: Läuft';

  @override
  String get runtimeStarting => 'Laufzeit: Wird gestartet';

  @override
  String get runtimeStopped => 'Laufzeit: Gestoppt';

  @override
  String get gatewayAutoStartHint =>
      'Wird beim nächsten Start des PocketClaw-Dienstes wirksam. Die Gateway-Laufzeit wird im Dashboard verwaltet.';

  @override
  String get manageTelegramConnection => 'Telegram-Verbindung verwalten';

  @override
  String get manageModelsTitle => 'Modelle verwalten';

  @override
  String get manageModelsDescription =>
      'KI-Modelle hinzufügen, bearbeiten, testen und auswählen.';

  @override
  String get githubChecking => 'Wird geprüft …';

  @override
  String get githubConnected => 'Verbunden';

  @override
  String get githubNotConnected => 'Nicht verbunden';

  @override
  String githubConnectedAs(String login) {
    return 'Verbunden als $login';
  }

  @override
  String get githubDescription =>
      'Wird vom mitgelieferten gh und von Git über HTTPS verwendet. Das Token wird auf diesem Gerät verschlüsselt und nie wieder angezeigt.';

  @override
  String get githubTestConnection => 'Verbindung testen';

  @override
  String get githubDisconnect => 'Trennen';

  @override
  String get githubConnectAction => 'GitHub verbinden';

  @override
  String get githubConnect => 'Verbinden';

  @override
  String get githubTokenLabel => 'Persönliches Zugriffstoken';

  @override
  String get githubTokenHint =>
      'Fügen Sie ein persönliches GitHub-Zugriffstoken mit den benötigten Berechtigungen ein (repo für private Repositories).';

  @override
  String get githubTokenRejected => 'GitHub hat dieses Token nicht akzeptiert.';

  @override
  String get githubAuthWorking => 'Die GitHub-Authentifizierung funktioniert.';

  @override
  String get githubAuthNotWorking =>
      'Die GitHub-Authentifizierung funktioniert nicht.';

  @override
  String githubAuthenticatedAs(String login) {
    return 'Authentifiziert als $login.';
  }

  @override
  String get githubCredentialRemoveFailed =>
      'Die Anmeldedaten konnten nicht entfernt werden.';

  @override
  String get githubYourAccount => 'Ihr GitHub-Konto';

  @override
  String githubConnectedReport(String who, String outcome) {
    return 'Verbunden als $who. $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return 'Getrennt. $outcome';
  }

  @override
  String get credentialAppliedNow => 'gh und git können es jetzt verwenden.';

  @override
  String get credentialAppliesNextStart =>
      'Es wird beim nächsten Start von PocketClaw verwendet.';

  @override
  String get credentialAppliesDeferred =>
      'Gespeichert. PocketClaw startet gerade und wendet es automatisch an, sobald das abgeschlossen ist.';

  @override
  String get whatsNewTitle => 'Neuerungen';

  @override
  String get whatsNewDescription =>
      'Die wichtigsten Änderungen in dieser Version.';

  @override
  String get whatsNewBadge => 'NEU';

  @override
  String get whatsNewSectionNew => 'Neu';

  @override
  String get whatsNewSectionImprovements => 'Verbesserungen';

  @override
  String get whatsNewSectionFixes => 'Fehlerbehebungen';

  @override
  String get whatsNew020New1 =>
      'Die verwaltete Laufzeitumgebung installiert die mitgelieferten PocketClaw-Werkzeuge und hält sie aktuell.';

  @override
  String get whatsNew020New2 =>
      'Mitgelieferte Entwicklerwerkzeuge: Git, GitHub CLI, curl, ripgrep, jq und SQLite.';

  @override
  String get whatsNew020New3 =>
      'Unterstützung für die mitgelieferte Python-3.14-Laufzeitumgebung von PocketClaw.';

  @override
  String get whatsNew020New4 =>
      'Sichere GitHub-Anmeldung, gemeinsam genutzt von Git und der GitHub CLI.';

  @override
  String get whatsNew020New5 =>
      'Telegram-Einrichtung mit einem Tipp: PocketClaw erstellt Ihren eigenen Bot, in der App oder über das Dashboard im Browser. Es gibt kein Token zum Kopieren, und nur Ihr eigenes Konto kann mit ihm sprechen.';

  @override
  String get whatsNew020New6 =>
      'Eine Statusansicht im Dashboard: laufende Arbeit, Kanäle, das genutzte Modell und die Laufzeitressourcen.';

  @override
  String get whatsNew020Improvement1 =>
      'Robustere Anbieter: Eine fehlgeschlagene Anfrage beendet den Durchgang nicht mehr.';

  @override
  String get whatsNew020Improvement2 =>
      'Ein genauerer Katalog der verwalteten Laufzeitumgebung.';

  @override
  String get whatsNew020Improvement3 =>
      'Eine klarere PocketClaw-Identität in der Weboberfläche und im Standard-Arbeitsbereich.';

  @override
  String get whatsNew020Improvement4 =>
      'Ein neues App-Symbol und ein überarbeiteter Info-Bildschirm im Aperture-Design von PocketClaw.';

  @override
  String get whatsNew020Improvement5 =>
      'Antworten des Assistenten wirken natürlicher, ohne feste Signatur am Ende.';

  @override
  String get whatsNew020Improvement6 =>
      'PocketClaw fordert die Telefonberechtigung nicht mehr an — sie wurde nie genutzt.';

  @override
  String get whatsNew020Improvement7 =>
      'Diagnoseprotokolle liegen jetzt privat in der App statt im Downloads-Ordner. Ihr Arbeitsbereich bleibt, wo er war.';

  @override
  String get whatsNew020Improvement8 =>
      'Ein umfassendes Upgrade dafür, wie PocketClaw seinen Zustand speichert und seine Dienste ausführt — für mehr Zuverlässigkeit im Alltag.';

  @override
  String get whatsNew020Improvement9 =>
      'Nach diesem Update meldet sich das Dashboard einmalig neu an.';

  @override
  String get whatsNew020Improvement10 =>
      'Nach diesem Update beginnt der Web-Chat eine neue Unterhaltung.';

  @override
  String get whatsNew020Improvement11 =>
      'Nach diesem Update lohnt es sich, Ihre Benachrichtigungseinstellungen einmal zu prüfen.';

  @override
  String get whatsNew020Fix1 =>
      'Das Gateway erholt sich jetzt von einem veralteten Prozesseintrag eines früheren Laufs.';

  @override
  String get whatsNew020Fix2 =>
      'Kanallisten mit mehreren Einträgen bleiben beim Speichern korrekt erhalten.';

  @override
  String get whatsNew021Fix1 =>
      'Das mitgelieferte Git stürzt beim Klonen eines Repositorys oder beim Aktualisieren eines Branch nicht mehr ab.';

  @override
  String get whatsNew022Improvement1 =>
      'Nach einer langen Aufgabe liefert Telegram die Antwort als neue Nachricht, sodass du benachrichtigt wirst und sie unter allem erscheint, was du in der Zwischenzeit gesendet hast.';

  @override
  String get whatsNew022Improvement2 =>
      'Telegram zeigt an, wenn deine Nachricht in der Warteschlange steht und wie viele vor ihr warten.';

  @override
  String get whatsNew022Improvement3 =>
      'Der Arbeitsbereich liegt jetzt im eigenen Speicher von PocketClaw, und die App fragt nach keiner Speicherberechtigung. Ein Arbeitsbereich, den eine ältere Version in Download/pocketclaw hinterlassen hat, bleibt unverändert und kann in den Einstellungen hineinkopiert werden.';

  @override
  String get whatsNew022Fix1 =>
      'Sehr große Werkzeugausgaben lassen das Gespräch nicht mehr überlaufen.';

  @override
  String get whatsNew022Fix2 =>
      'Einstellungen ohne Wirkung auf Android entfernt: Geräte, Beim Anmelden starten und Dienstport.';

  @override
  String get settingsGroupConnection => 'Verbindung';

  @override
  String get settingsGroupAgent => 'Agent';

  @override
  String get settingsGroupIntegrations => 'Integrationen';

  @override
  String get settingsGroupAppearance => 'Darstellung';

  @override
  String get statusTitle => 'Status';

  @override
  String get statusSectionSystem => 'System';

  @override
  String get statusSectionAi => 'KI';

  @override
  String get statusSectionActivity => 'Aktivität';

  @override
  String get statusSectionChannels => 'Kanäle';

  @override
  String get statusSectionResources => 'Ressourcen';

  @override
  String get statusSinceGatewayStart => 'Seit Gateway-Start';

  @override
  String get statusGateway => 'Gateway';

  @override
  String get statusUptime => 'Laufzeit';

  @override
  String get statusAppVersion => 'App-Version';

  @override
  String get statusCoreVersion => 'Core-Version';

  @override
  String get statusActiveModel => 'Aktives Modell';

  @override
  String get statusConfiguredDefault => 'Konfigurierter Standard';

  @override
  String get statusProvider => 'Anbieter';

  @override
  String get statusFallbacks => 'Ausweichmodelle';

  @override
  String get statusActiveTurns => 'Aktive Durchläufe';

  @override
  String get statusActiveSubagents => 'Aktive Subagenten';

  @override
  String get statusWaiting => 'Wartend';

  @override
  String get statusCompleted => 'Abgeschlossen';

  @override
  String get statusFailed => 'Fehlgeschlagen';

  @override
  String get statusCancelled => 'Abgebrochen';

  @override
  String get statusToolCalls => 'Tool-Aufrufe';

  @override
  String get statusToolCallsFailed => 'Fehlgeschlagene Tool-Aufrufe';

  @override
  String get statusLastActivity => 'Letzte Aktivität';

  @override
  String get statusCoreMemory => 'Core-Speicher';

  @override
  String get statusCoreCpuTime => 'Core-CPU-Zeit';

  @override
  String get statusNoChannels => 'Keine Kanäle konfiguriert';

  @override
  String get statusDetailUnavailable => 'Detailstatus nicht verfügbar';

  @override
  String get statusJustNow => 'Gerade eben';

  @override
  String statusMinutesAgo(int minutes) {
    return 'vor $minutes Min.';
  }

  @override
  String statusHoursAgo(int hours) {
    return 'vor $hours Std.';
  }

  @override
  String statusDaysAgo(int days) {
    return 'vor $days Tg.';
  }

  @override
  String statusDurationSeconds(int seconds) {
    return '$seconds S.';
  }

  @override
  String statusDurationMinutes(int minutes, int seconds) {
    return '$minutes Min. $seconds S.';
  }

  @override
  String statusDurationHours(int hours, int minutes) {
    return '$hours Std. $minutes Min.';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '$days T. $hours Std.';
  }

  @override
  String get notificationPermissionTitle => 'Benachrichtigungen';

  @override
  String get notificationPermissionGranted =>
      'PocketClaw kann seine Running-Benachrichtigung anzeigen.';

  @override
  String get notificationPermissionBlocked =>
      'Benachrichtigungen sind aus, daher erscheint die PocketClaw-Running-Benachrichtigung nicht.';

  @override
  String get notificationPermissionOpenSettings =>
      'Benachrichtigungseinstellungen öffnen';

  @override
  String get whatsNew020New7 =>
      'Telegram-Verwaltung im Dashboard: einen Bot verbinden, ersetzen oder trennen.';

  @override
  String get whatsNew020New8 =>
      'Anbieter- und Modellverwaltung im Dashboard: Anbieter hinzufügen, API-Schlüssel wechseln oder ein Modell entfernen.';

  @override
  String get whatsNew020Improvement12 =>
      'Telegram-Einstellungen werden sofort beim Speichern übernommen, ohne manuellen Neustart.';

  @override
  String get whatsNew020Improvement13 =>
      'Wenn kein KI-Modell eingerichtet ist, nennt PocketClaw genau das Fehlende und gibt Ihnen einen Code zum Nachschlagen.';

  @override
  String get whatsNew020Improvement14 =>
      'Die Anmeldung am Dashboard führt Sie zu der Seite, die Sie geöffnet hatten.';

  @override
  String get whatsNew020Improvement15 =>
      'Diagnoseprotokolle bleiben ausführlich, enthalten aber niemals Ihre Schlüssel, Token oder Nachrichtentexte.';

  @override
  String get whatsNew020Improvement16 =>
      'Ein Dashboard, das noch niemandem gehört, ist nie aus dem Netzwerk erreichbar – auch bei aktiviertem Public Mode.';
}

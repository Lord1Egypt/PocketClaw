// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Spanish Castilian (`es`).
class AppLocalizationsEs extends AppLocalizations {
  AppLocalizationsEs([String locale = 'es']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => 'Ejecutar';

  @override
  String get stop => 'Detener';

  @override
  String get config => 'Config';

  @override
  String get webAdmin => 'Admin Web';

  @override
  String get logs => 'Registros';

  @override
  String get viewLogs => 'Ver registros';

  @override
  String get statusRunning => 'Ejecutando';

  @override
  String get statusStopped => 'Detenido';

  @override
  String get settings => 'Ajustes';

  @override
  String get address => 'Dirección';

  @override
  String get port => 'Puerto';

  @override
  String get save => 'Guardar';

  @override
  String get showWindow => 'Mostrar ventana';

  @override
  String get exit => 'Salir';

  @override
  String get binaryPath => 'Ruta binaria';

  @override
  String get browse => 'Examinar';

  @override
  String get pathError => 'Ruta inválida';

  @override
  String get arguments => 'Argumentos';

  @override
  String get argumentsHint => 'ej. config.json';

  @override
  String get notStarted => 'Servicio no iniciado';

  @override
  String get startHint => 'Por favor, inicie el servicio desde el panel.';

  @override
  String get goToDashboard => 'Ir al panel';

  @override
  String get back => 'Atrás';

  @override
  String get forward => 'Adelante';

  @override
  String get refresh => 'Actualizar';

  @override
  String get coreBinaryMissing =>
      'Binario principal no encontrado. Coloque el binario de la plataforma en app/bin/ o configure la ruta en Ajustes.';

  @override
  String get coreStartFailed => 'Error al iniciar el servicio principal.';

  @override
  String get coreStopFailed => 'Error al detener el servicio principal.';

  @override
  String get coreInvalidBinary => 'Archivo binario principal inválido.';

  @override
  String coreUnknownError(Object code) {
    return 'Error principal desconocido: $code';
  }

  @override
  String get coreValid => 'El binario principal es válido.';

  @override
  String get publicMode => 'Modo público';

  @override
  String get publicModeHintDesc =>
      'Cuando está activado, el servicio permite acceso externo y el campo de dirección se desactivará';

  @override
  String get publicModeApplying => 'Aplicando el modo de red...';

  @override
  String get themeSelection => 'Tema';

  @override
  String get check => 'Verificar';

  @override
  String get launchService => 'INICIAR SERVICIO';

  @override
  String get stopService => 'DETENER SERVICIO';

  @override
  String get endpoint => 'PUNTO DE ACCESO';

  @override
  String get statusActive => 'ACTIVO';

  @override
  String get statusSyncing => 'SINCRONIZANDO';

  @override
  String get statusIdle => 'INACTIVO';

  @override
  String get publicModeEnabled => 'Modo público activado';

  @override
  String get localMode => 'Modo local';

  @override
  String get unableToGetDeviceIp => 'No se puede obtener la IP del dispositivo';

  @override
  String get deviceReportingTitle =>
      'Comentarios de compatibilidad del dispositivo';

  @override
  String get deviceReportingSubtitle =>
      'Solo se usa para verificar la compatibilidad de la versión del SO y la versión de la app. No implica mensajes de chat, detalles de cuenta ni contenido personal';

  @override
  String get deviceReportingConsentTitle =>
      'Ayude a mejorar la compatibilidad del dispositivo';

  @override
  String get deviceReportingConsentDescription =>
      'Cuando está activado, solo se envía un ID de instalación anónimo, la versión del SO y la versión de la app para comprender la compatibilidad. El idioma y la región pueden ser recopilados por Firebase Analytics por separado. No se suben mensajes de chat, contenido escrito, detalles de cuenta, archivos ni configuración personalizada';

  @override
  String get deviceReportingBannerDescription =>
      'Solo se sincronizan un ID de instalación anónimo, la versión del SO y la versión de la app para mejorar la compatibilidad. El idioma y la región pueden ser recopilados por separado por Firebase Analytics. No se envían mensajes de chat, detalles de cuenta, archivos ni contenido personal';

  @override
  String get deviceReportingWhatWillBeSent =>
      'Solo se incluyen estos detalles del dispositivo';

  @override
  String get deviceReportingDeviceLabel => 'Modelo del dispositivo';

  @override
  String get deviceReportingPlatformLabel => 'Categoría del dispositivo';

  @override
  String get deviceReportingSystemLabel => 'Versión del SO';

  @override
  String get deviceReportingTimingNote =>
      'Una sincronización se ejecuta una vez cuando está activado, y nuevamente solo después de que se detecta una actualización del sistema';

  @override
  String get deviceReportingDeny => 'Ahora no';

  @override
  String get deviceReportingAllow => 'Activar';

  @override
  String get deviceReportingUploadSucceeded =>
      'Comentarios de compatibilidad del dispositivo activados';

  @override
  String get deviceReportingUploadFailed =>
      'Comentarios de compatibilidad del dispositivo activados, pero la sincronización de información del dispositivo actual no se completó';

  @override
  String get deviceReportingDisabled =>
      'Comentarios de compatibilidad del dispositivo desactivados';

  @override
  String get localModeHint =>
      '1. Vaya a Configuración del servicio\n2. Active el Modo público\n3. Escanee el código QR para acceder a PocketClaw';

  @override
  String get publicModeHint =>
      '1. Inicie el servicio\n2. Escanee el código QR para acceder a PocketClaw';

  @override
  String get noLogsToExport => 'No hay registros para exportar';

  @override
  String get logsSavedToMediaLibrary =>
      'Registros guardados en Descargas (biblioteca multimedia de Android)';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Registros guardados en Descargas: $path';
  }

  @override
  String get shareLogsText => 'Registros de PocketClaw';

  @override
  String get workspaceDirectory => 'Espacio de trabajo';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'Registros guardados en Descargas (biblioteca multimedia de Android): $name';
  }

  @override
  String shareFailed(Object error) {
    return 'Error al abrir el diálogo de compartir: $error';
  }

  @override
  String get exportLogs => 'Exportar registros';

  @override
  String logEventsCount(int count) {
    return '$count EVENTOS';
  }

  @override
  String get unsavedChanges => 'Cambios sin guardar';

  @override
  String get unsavedChangesHint =>
      'Tienes cambios sin guardar. ¿Deseas descartarlos?';

  @override
  String get cancel => 'Cancelar';

  @override
  String get discard => 'Descartar';

  @override
  String get saved => 'Guardado';

  @override
  String get language => 'Idioma';

  @override
  String get selectLanguage => 'Seleccionar idioma';

  @override
  String get about => 'Acerca de';

  @override
  String get aboutDescription =>
      'PocketClaw es tu espacio de trabajo privado para asistentes de IA.';

  @override
  String get aboutAppVersionLabel => 'Versión de PocketClaw';

  @override
  String get aboutCoreVersionLabel => 'Versión del entorno';

  @override
  String get aboutVersionUnavailable => 'No disponible';

  @override
  String get close => 'Cerrar';

  @override
  String get contextMemoryTitle => 'Memoria de contexto de Telegram';

  @override
  String get contextMemoryDescription =>
      'Controla cuántos mensajes recientes de la conversación se envían a la IA. Las conversaciones anteriores permanecen en Telegram y se representan mediante el resumen continuo.';

  @override
  String get contextMemoryHelp =>
      'Más mensajes aportan más contexto reciente, pero consumen más tokens. Los mensajes anteriores permanecen en Telegram y pueden conservarse mediante el resumen continuo.';

  @override
  String get contextMemoryRecommended => 'Recomendado';

  @override
  String get contextMemoryCustom => 'Personalizado';

  @override
  String get contextMemoryCustomLabel => 'Mensajes';

  @override
  String contextMemoryRangeError(int min, int max) {
    return 'Introduce un número entero entre $min y $max.';
  }

  @override
  String get contextMemorySaveFailed =>
      'No se pudo guardar esta configuración.';

  @override
  String get settingsSave => 'Guardar';

  @override
  String get autoStartServiceTitle =>
      'Iniciar el servicio de PocketClaw automáticamente';

  @override
  String get autoStartGatewayTitle =>
      'Iniciar la puerta de enlace automáticamente';

  @override
  String get autoStartPreferenceOn =>
      'Preferencia de inicio automático: ACTIVADA';

  @override
  String get autoStartPreferenceOff =>
      'Preferencia de inicio automático: DESACTIVADA';

  @override
  String get runtimeRunning => 'Estado: En ejecución';

  @override
  String get runtimeStarting => 'Estado: Iniciando';

  @override
  String get runtimeStopped => 'Estado: Detenido';

  @override
  String get gatewayAutoStartHint =>
      'Se aplica la próxima vez que se inicie el servicio de PocketClaw. El estado de la puerta de enlace se gestiona en el panel.';

  @override
  String get manageTelegramConnection => 'Gestionar la conexión de Telegram';

  @override
  String get manageModelsTitle => 'Gestionar modelos';

  @override
  String get manageModelsDescription =>
      'Añade, edita, prueba y elige modelos de IA.';

  @override
  String get githubChecking => 'Comprobando…';

  @override
  String get githubConnected => 'Conectado';

  @override
  String get githubNotConnected => 'No conectado';

  @override
  String githubConnectedAs(String login) {
    return 'Conectado como $login';
  }

  @override
  String get githubDescription =>
      'Lo usan el gh incluido y Git a través de HTTPS. El token se cifra en este dispositivo y no vuelve a mostrarse.';

  @override
  String get githubTestConnection => 'Probar conexión';

  @override
  String get githubDisconnect => 'Desconectar';

  @override
  String get githubConnectAction => 'Conectar GitHub';

  @override
  String get githubConnect => 'Conectar';

  @override
  String get githubTokenLabel => 'Token de acceso personal';

  @override
  String get githubTokenHint =>
      'Pega un token de acceso personal de GitHub con los permisos que necesites (repo para repositorios privados).';

  @override
  String get githubTokenRejected => 'GitHub no aceptó este token.';

  @override
  String get githubAuthWorking => 'La autenticación de GitHub funciona.';

  @override
  String get githubAuthNotWorking => 'La autenticación de GitHub no funciona.';

  @override
  String githubAuthenticatedAs(String login) {
    return 'Autenticado como $login.';
  }

  @override
  String get githubCredentialRemoveFailed =>
      'No se pudo eliminar la credencial.';

  @override
  String get githubYourAccount => 'tu cuenta de GitHub';

  @override
  String githubConnectedReport(String who, String outcome) {
    return 'Conectado como $who. $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return 'Desconectado. $outcome';
  }

  @override
  String get credentialAppliedNow => 'gh y git ya pueden usarlo.';

  @override
  String get credentialAppliesNextStart =>
      'Se usará la próxima vez que se inicie PocketClaw.';

  @override
  String get credentialAppliesDeferred =>
      'Guardado. PocketClaw está iniciándose, así que se aplicará automáticamente en cuanto termine.';

  @override
  String get whatsNewTitle => 'Novedades';

  @override
  String get whatsNewDescription => 'Los cambios principales de esta versión.';

  @override
  String get whatsNewBadge => 'NUEVO';

  @override
  String get whatsNewSectionNew => 'Novedades';

  @override
  String get whatsNewSectionImprovements => 'Mejoras';

  @override
  String get whatsNewSectionFixes => 'Correcciones';

  @override
  String get whatsNew020New1 =>
      'El entorno de ejecución gestionado instala y mantiene actualizadas las herramientas incluidas en PocketClaw.';

  @override
  String get whatsNew020New2 =>
      'Herramientas de desarrollo incluidas: Git, GitHub CLI, curl, ripgrep, jq y SQLite.';

  @override
  String get whatsNew020New3 =>
      'Compatibilidad con el entorno de ejecución de Python 3.14 incluido en PocketClaw.';

  @override
  String get whatsNew020New4 =>
      'Inicio de sesión seguro en GitHub, compartido por Git y la CLI de GitHub incluidos.';

  @override
  String get whatsNew020New5 =>
      'Integración con Telegram, configurable desde Ajustes.';

  @override
  String get whatsNew020New6 =>
      'Una vista de Estado en el panel: trabajo activo, canales, el modelo en uso y los recursos de ejecución.';

  @override
  String get whatsNew020Improvement1 =>
      'Proveedores más resistentes: una solicitud fallida ya no interrumpe el turno.';

  @override
  String get whatsNew020Improvement2 =>
      'Un catálogo del entorno de ejecución gestionado más preciso.';

  @override
  String get whatsNew020Improvement3 =>
      'Una identidad de PocketClaw más clara en la interfaz web y en el espacio de trabajo predeterminado.';

  @override
  String get whatsNew020Improvement4 =>
      'Un nuevo icono de la aplicación y una pantalla Acerca de renovada, con el diseño Aperture de PocketClaw.';

  @override
  String get whatsNew020Improvement5 =>
      'Las respuestas del asistente resultan más naturales, sin una firma fija al final.';

  @override
  String get whatsNew020Improvement6 =>
      'PocketClaw ya no pide el permiso de Teléfono: nada en la aplicación lo usaba.';

  @override
  String get whatsNew020Fix1 =>
      'La puerta de enlace ahora se recupera de un registro de proceso obsoleto dejado por una ejecución anterior.';

  @override
  String get whatsNew020Fix2 =>
      'Las listas de canales con más de una entrada se conservan correctamente al guardar los ajustes.';

  @override
  String get settingsGroupConnection => 'Conexión';

  @override
  String get settingsGroupAgent => 'Agente';

  @override
  String get settingsGroupIntegrations => 'Integraciones';

  @override
  String get settingsGroupAppearance => 'Apariencia';

  @override
  String get statusTitle => 'Estado';

  @override
  String get statusSectionSystem => 'Sistema';

  @override
  String get statusSectionAi => 'IA';

  @override
  String get statusSectionActivity => 'Actividad';

  @override
  String get statusSectionChannels => 'Canales';

  @override
  String get statusSectionResources => 'Recursos';

  @override
  String get statusSinceGatewayStart => 'Desde el inicio del Gateway';

  @override
  String get statusGateway => 'Gateway';

  @override
  String get statusUptime => 'Tiempo activo';

  @override
  String get statusAppVersion => 'Versión de la app';

  @override
  String get statusCoreVersion => 'Versión del Core';

  @override
  String get statusActiveModel => 'Modelo activo';

  @override
  String get statusConfiguredDefault => 'Predeterminado configurado';

  @override
  String get statusProvider => 'Proveedor';

  @override
  String get statusFallbacks => 'Alternativas';

  @override
  String get statusActiveTurns => 'Turnos activos';

  @override
  String get statusActiveSubagents => 'Subagentes activos';

  @override
  String get statusWaiting => 'En espera';

  @override
  String get statusCompleted => 'Completados';

  @override
  String get statusFailed => 'Fallidos';

  @override
  String get statusCancelled => 'Cancelados';

  @override
  String get statusToolCalls => 'Llamadas a herramientas';

  @override
  String get statusToolCallsFailed => 'Llamadas fallidas';

  @override
  String get statusLastActivity => 'Última actividad';

  @override
  String get statusCoreMemory => 'Memoria del Core';

  @override
  String get statusCoreCpuTime => 'Tiempo de CPU del Core';

  @override
  String get statusNoChannels => 'No hay canales configurados';

  @override
  String get statusDetailUnavailable =>
      'El estado detallado no está disponible';

  @override
  String get statusJustNow => 'Justo ahora';

  @override
  String statusMinutesAgo(int minutes) {
    return 'hace $minutes min';
  }

  @override
  String statusHoursAgo(int hours) {
    return 'hace $hours h';
  }

  @override
  String statusDaysAgo(int days) {
    return 'hace $days d';
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
    return '$days d $hours h';
  }
}

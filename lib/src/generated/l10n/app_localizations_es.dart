// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Spanish Castilian (`es`).
class AppLocalizationsEs extends AppLocalizations {
  AppLocalizationsEs([String locale = 'es']) : super(locale);

  @override
  String get stop => 'Detener';

  @override
  String get webAdmin => 'Admin Web';

  @override
  String get logs => 'Registros';

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
  String get publicMode => 'Modo público';

  @override
  String get publicModeHintDesc =>
      'Cuando está activado, el servicio permite acceso externo y el campo de dirección se desactivará';

  @override
  String get publicModeApplying => 'Aplicando el modo de red...';

  @override
  String get themeSelection => 'Tema';

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
  String get localModeHint =>
      '1. Vaya a Configuración del servicio\n2. Active el Modo público\n3. Escanee el código QR para acceder a PocketClaw';

  @override
  String get publicModeHint =>
      '1. Inicie el servicio\n2. Escanee el código QR para acceder a PocketClaw';

  @override
  String get noLogsToExport => 'No hay registros para exportar';

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
  String get cancel => 'Cancelar';

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
      'Configuración de Telegram con un toque: PocketClaw crea tu propio bot, en la aplicación o desde el panel en un navegador. No hay ningún token que copiar y solo tu cuenta puede hablar con él.';

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
  String get whatsNew020Improvement7 =>
      'Los registros de diagnóstico ahora se guardan de forma privada dentro de la aplicación, no en la carpeta de descargas. Tu espacio de trabajo sigue donde estaba.';

  @override
  String get whatsNew020Improvement8 =>
      'Una mejora importante en la forma en que PocketClaw guarda su estado y ejecuta sus servicios, para una fiabilidad más constante en el día a día.';

  @override
  String get whatsNew020Improvement9 =>
      'Tras esta actualización, el panel te pedirá iniciar sesión una vez más.';

  @override
  String get whatsNew020Improvement10 =>
      'Tras esta actualización, el canal de chat Web inicia una conversación nueva.';

  @override
  String get whatsNew020Improvement11 =>
      'Tras esta actualización, conviene revisar una vez tus preferencias de notificaciones.';

  @override
  String get whatsNew020Fix1 =>
      'La puerta de enlace ahora se recupera de un registro de proceso obsoleto dejado por una ejecución anterior.';

  @override
  String get whatsNew020Fix2 =>
      'Las listas de canales con más de una entrada se conservan correctamente al guardar los ajustes.';

  @override
  String get whatsNew021Fix1 =>
      'El Git incluido ya no se bloquea al clonar un repositorio o actualizar una rama.';

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

  @override
  String get notificationPermissionTitle => 'Notificaciones';

  @override
  String get notificationPermissionGranted =>
      'PocketClaw puede mostrar su notificación de ejecución.';

  @override
  String get notificationPermissionBlocked =>
      'Las notificaciones están desactivadas, así que la notificación de PocketClaw en ejecución no aparecerá.';

  @override
  String get notificationPermissionOpenSettings =>
      'Abrir ajustes de notificaciones';

  @override
  String get whatsNew020New7 =>
      'Gestión de Telegram desde el panel: conectar un bot, reemplazarlo o desconectarlo.';

  @override
  String get whatsNew020New8 =>
      'Gestión de proveedores y modelos en el panel: añadir un proveedor, rotar una clave de API o eliminar un modelo.';

  @override
  String get whatsNew020Improvement12 =>
      'Los ajustes de Telegram se aplican al guardarlos, sin reinicios manuales.';

  @override
  String get whatsNew020Improvement13 =>
      'Cuando no hay ningún modelo de IA configurado, PocketClaw dice exactamente qué falta y te da un código para consultarlo.';

  @override
  String get whatsNew020Improvement14 =>
      'Al iniciar sesión en el panel vuelves a la pantalla que pediste.';

  @override
  String get whatsNew020Improvement15 =>
      'Los registros de diagnóstico siguen siendo detallados sin contener nunca tus claves, tokens ni el texto de tus mensajes.';

  @override
  String get whatsNew020Improvement16 =>
      'Un panel que nadie ha reclamado todavía nunca es accesible desde la red, ni siquiera con el modo público activado.';
}

// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Russian (`ru`).
class AppLocalizationsRu extends AppLocalizations {
  AppLocalizationsRu([String locale = 'ru']) : super(locale);

  @override
  String get stop => 'Остановить';

  @override
  String get webAdmin => 'Веб-админ';

  @override
  String get logs => 'Журналы';

  @override
  String get statusRunning => 'Работает';

  @override
  String get statusStopped => 'Остановлен';

  @override
  String get settings => 'Настройки';

  @override
  String get address => 'Адрес';

  @override
  String get port => 'Порт';

  @override
  String get save => 'Сохранить';

  @override
  String get notStarted => 'Служба не запущена';

  @override
  String get startHint =>
      'Пожалуйста, сначала запустите службу с панели управления.';

  @override
  String get goToDashboard => 'Перейти на панель';

  @override
  String get back => 'Назад';

  @override
  String get forward => 'Вперед';

  @override
  String get refresh => 'Обновить';

  @override
  String get publicMode => 'Общий режим';

  @override
  String get publicModeHintDesc =>
      'Когда включено, служба разрешает внешний доступ, а поле адреса будет отключено';

  @override
  String get publicModeApplying => 'Применение сетевого режима...';

  @override
  String get themeSelection => 'Тема';

  @override
  String get launchService => 'ЗАПУСТИТЬ СЛУЖБУ';

  @override
  String get stopService => 'ОСТАНОВИТЬ СЛУЖБУ';

  @override
  String get endpoint => 'КОНЕЧНАЯ ТОЧКА';

  @override
  String get statusActive => 'АКТИВЕН';

  @override
  String get statusSyncing => 'СИНХРОНИЗАЦИЯ';

  @override
  String get statusIdle => 'ОЖИДАНИЕ';

  @override
  String get publicModeEnabled => 'Общий режим включен';

  @override
  String get localMode => 'Локальный режим';

  @override
  String get unableToGetDeviceIp => 'Не удается получить IP-адрес устройства';

  @override
  String get localModeHint =>
      '1. Перейдите в Настройку службы\n2. Включите Общий режим\n3. Отсканируйте QR-код для доступа к PocketClaw';

  @override
  String get publicModeHint =>
      '1. Запустите службу\n2. Отсканируйте QR-код для доступа к PocketClaw';

  @override
  String get noLogsToExport => 'Нет журналов для экспорта';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Журналы сохранены в Загрузки: $path';
  }

  @override
  String get shareLogsText => 'Журналы PocketClaw';

  @override
  String get workspaceDirectory => 'Рабочее пространство';

  @override
  String get legacyWorkspaceTitle => 'Найдено прежнее рабочее пространство';

  @override
  String legacyWorkspaceBody(Object path) {
    return 'Более старая версия хранила ваше рабочее пространство в $path. Теперь PocketClaw использует собственное хранилище приложения и не трогал эту папку. Вы можете скопировать её: копия попадёт в отдельную папку, ничего не будет перезаписано или удалено.';
  }

  @override
  String get legacyWorkspaceImport => 'Скопировать в рабочее пространство';

  @override
  String get legacyWorkspaceHide => 'Скрыть';

  @override
  String legacyWorkspaceCopied(Object count, Object folder) {
    return 'Скопировано файлов: $count в $folder.';
  }

  @override
  String legacyWorkspacePartial(Object count, Object folder, Object failed) {
    return 'Скопировано файлов: $count в $folder. Не удалось скопировать: $failed.';
  }

  @override
  String get legacyWorkspaceFailed => 'Не удалось скопировать папку.';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'Журналы сохранены в Загрузки (медиатека Android): $name';
  }

  @override
  String shareFailed(Object error) {
    return 'Не удалось открыть диалоговое окно общего доступа: $error';
  }

  @override
  String get exportLogs => 'Экспорт журналов';

  @override
  String logEventsCount(int count) {
    return '$count СОБЫТИЙ';
  }

  @override
  String get cancel => 'Отмена';

  @override
  String get language => 'Язык';

  @override
  String get selectLanguage => 'Выбрать язык';

  @override
  String get about => 'О приложении';

  @override
  String get aboutDescription =>
      'PocketClaw — ваше личное рабочее пространство для ИИ-помощника.';

  @override
  String get aboutAppVersionLabel => 'Версия PocketClaw';

  @override
  String get aboutCoreVersionLabel => 'Версия среды выполнения';

  @override
  String get aboutVersionUnavailable => 'Недоступно';

  @override
  String get close => 'Закрыть';

  @override
  String get contextMemoryTitle => 'Память контекста Telegram';

  @override
  String get contextMemoryDescription =>
      'Определяет, сколько последних сообщений передаётся ИИ. Более ранняя переписка остаётся в Telegram и отражается в накопительной сводке.';

  @override
  String get contextMemoryHelp =>
      'Больше сообщений — больше свежего контекста, но и больше расход токенов. Ранние сообщения остаются в Telegram и могут сохраняться в накопительной сводке.';

  @override
  String get contextMemoryRecommended => 'Рекомендуется';

  @override
  String get contextMemoryCustom => 'Вручную';

  @override
  String get contextMemoryCustomLabel => 'Сообщения';

  @override
  String contextMemoryRangeError(int min, int max) {
    return 'Введите целое число от $min до $max.';
  }

  @override
  String get contextMemorySaveFailed => 'Не удалось сохранить эту настройку.';

  @override
  String get settingsSave => 'Сохранить';

  @override
  String get autoStartServiceTitle =>
      'Запускать службу PocketClaw автоматически';

  @override
  String get autoStartGatewayTitle => 'Запускать шлюз автоматически';

  @override
  String get autoStartPreferenceOn => 'Автозапуск: ВКЛ';

  @override
  String get autoStartPreferenceOff => 'Автозапуск: ВЫКЛ';

  @override
  String get runtimeRunning => 'Состояние: Работает';

  @override
  String get runtimeStarting => 'Состояние: Запускается';

  @override
  String get runtimeStopped => 'Состояние: Остановлено';

  @override
  String get gatewayAutoStartHint =>
      'Применится при следующем запуске службы PocketClaw. Состоянием шлюза управляют в панели управления.';

  @override
  String get manageTelegramConnection => 'Управление подключением Telegram';

  @override
  String get manageModelsTitle => 'Управление моделями';

  @override
  String get manageModelsDescription =>
      'Добавляйте, изменяйте, проверяйте и выбирайте модели ИИ.';

  @override
  String get githubChecking => 'Проверка…';

  @override
  String get githubConnected => 'Подключено';

  @override
  String get githubNotConnected => 'Не подключено';

  @override
  String githubConnectedAs(String login) {
    return 'Подключено как $login';
  }

  @override
  String get githubDescription =>
      'Используется встроенным gh и Git по HTTPS. Токен шифруется на этом устройстве и больше не отображается.';

  @override
  String get githubTestConnection => 'Проверить подключение';

  @override
  String get githubDisconnect => 'Отключить';

  @override
  String get githubConnectAction => 'Подключить GitHub';

  @override
  String get githubConnect => 'Подключить';

  @override
  String get githubTokenLabel => 'Персональный токен доступа';

  @override
  String get githubTokenHint =>
      'Вставьте персональный токен доступа GitHub с нужными областями (repo для приватных репозиториев).';

  @override
  String get githubTokenRejected => 'GitHub не принял этот токен.';

  @override
  String get githubAuthWorking => 'Аутентификация GitHub работает.';

  @override
  String get githubAuthNotWorking => 'Аутентификация GitHub не работает.';

  @override
  String githubAuthenticatedAs(String login) {
    return 'Выполнен вход как $login.';
  }

  @override
  String get githubCredentialRemoveFailed =>
      'Не удалось удалить учётные данные.';

  @override
  String get githubYourAccount => 'ваша учётная запись GitHub';

  @override
  String githubConnectedReport(String who, String outcome) {
    return 'Подключено как $who. $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return 'Отключено. $outcome';
  }

  @override
  String get credentialAppliedNow => 'gh и git могут использовать его сейчас.';

  @override
  String get credentialAppliesNextStart =>
      'Он будет использован при следующем запуске PocketClaw.';

  @override
  String get credentialAppliesDeferred =>
      'Сохранено. PocketClaw запускается, изменение применится автоматически сразу после этого.';

  @override
  String get whatsNewTitle => 'Что нового';

  @override
  String get whatsNewDescription => 'Основные изменения в этом выпуске.';

  @override
  String get whatsNewBadge => 'НОВОЕ';

  @override
  String get whatsNewSectionNew => 'Новое';

  @override
  String get whatsNewSectionImprovements => 'Улучшения';

  @override
  String get whatsNewSectionFixes => 'Исправления';

  @override
  String get whatsNew020New1 =>
      'Управляемая среда выполнения устанавливает встроенные инструменты PocketClaw и поддерживает их в актуальном состоянии.';

  @override
  String get whatsNew020New2 =>
      'Встроенные инструменты разработчика: Git, GitHub CLI, curl, ripgrep, jq и SQLite.';

  @override
  String get whatsNew020New3 =>
      'Поддержка встроенной в PocketClaw среды Python 3.14.';

  @override
  String get whatsNew020New4 =>
      'Безопасный вход в GitHub, общий для встроенных Git и GitHub CLI.';

  @override
  String get whatsNew020New5 =>
      'Настройка Telegram одним касанием: PocketClaw создаёт вашего собственного бота — в приложении или из панели в браузере. Никакой токен копировать не нужно, и говорить с ним может только ваша учётная запись.';

  @override
  String get whatsNew020New6 =>
      'Раздел «Состояние» на панели: текущая работа, каналы, используемая модель и ресурсы среды выполнения.';

  @override
  String get whatsNew020Improvement1 =>
      'Более устойчивые провайдеры: неудачный запрос больше не обрывает ход.';

  @override
  String get whatsNew020Improvement2 =>
      'Более точный каталог управляемой среды выполнения.';

  @override
  String get whatsNew020Improvement3 =>
      'Более согласованное оформление PocketClaw в веб-интерфейсе и рабочем пространстве по умолчанию.';

  @override
  String get whatsNew020Improvement4 =>
      'Новый значок приложения и обновлённый экран «О программе» в оформлении Aperture.';

  @override
  String get whatsNew020Improvement5 =>
      'Ответы ассистента звучат естественнее — без фиксированной подписи в конце.';

  @override
  String get whatsNew020Improvement6 =>
      'PocketClaw больше не запрашивает разрешение «Телефон» — оно нигде не использовалось.';

  @override
  String get whatsNew020Improvement7 =>
      'Диагностические журналы теперь хранятся приватно внутри приложения, а не в папке «Загрузки». Рабочее пространство остаётся на прежнем месте.';

  @override
  String get whatsNew020Improvement8 =>
      'Существенное обновление того, как PocketClaw хранит своё состояние и запускает службы, — ради более стабильной работы изо дня в день.';

  @override
  String get whatsNew020Improvement9 =>
      'После этого обновления панель управления попросит войти ещё раз.';

  @override
  String get whatsNew020Improvement10 =>
      'После этого обновления веб-канал чата начинает новый разговор.';

  @override
  String get whatsNew020Improvement11 =>
      'После этого обновления стоит один раз проверить настройки уведомлений.';

  @override
  String get whatsNew020Fix1 =>
      'Шлюз теперь восстанавливается после устаревшей записи процесса, оставшейся от предыдущего запуска.';

  @override
  String get whatsNew020Fix2 =>
      'Списки каналов с несколькими записями корректно сохраняются.';

  @override
  String get whatsNew021Fix1 =>
      'Встроенный Git больше не аварийно завершается при клонировании репозитория или обновлении ветки.';

  @override
  String get settingsGroupConnection => 'Подключение';

  @override
  String get settingsGroupAgent => 'Агент';

  @override
  String get settingsGroupIntegrations => 'Интеграции';

  @override
  String get settingsGroupAppearance => 'Оформление';

  @override
  String get statusTitle => 'Состояние';

  @override
  String get statusSectionSystem => 'Система';

  @override
  String get statusSectionAi => 'ИИ';

  @override
  String get statusSectionActivity => 'Активность';

  @override
  String get statusSectionChannels => 'Каналы';

  @override
  String get statusSectionResources => 'Ресурсы';

  @override
  String get statusSinceGatewayStart => 'С момента запуска шлюза';

  @override
  String get statusGateway => 'Шлюз';

  @override
  String get statusUptime => 'Время работы';

  @override
  String get statusAppVersion => 'Версия приложения';

  @override
  String get statusCoreVersion => 'Версия Core';

  @override
  String get statusActiveModel => 'Активная модель';

  @override
  String get statusConfiguredDefault => 'Модель по умолчанию';

  @override
  String get statusProvider => 'Провайдер';

  @override
  String get statusFallbacks => 'Резервные модели';

  @override
  String get statusActiveTurns => 'Активные ходы';

  @override
  String get statusActiveSubagents => 'Активные субагенты';

  @override
  String get statusWaiting => 'В очереди';

  @override
  String get statusCompleted => 'Завершено';

  @override
  String get statusFailed => 'Ошибок';

  @override
  String get statusCancelled => 'Отменено';

  @override
  String get statusToolCalls => 'Вызовы инструментов';

  @override
  String get statusToolCallsFailed => 'Неудачные вызовы';

  @override
  String get statusLastActivity => 'Последняя активность';

  @override
  String get statusCoreMemory => 'Память Core';

  @override
  String get statusCoreCpuTime => 'Процессорное время Core';

  @override
  String get statusNoChannels => 'Каналы не настроены';

  @override
  String get statusDetailUnavailable => 'Подробный статус недоступен';

  @override
  String get statusJustNow => 'Только что';

  @override
  String statusMinutesAgo(int minutes) {
    return '$minutes мин назад';
  }

  @override
  String statusHoursAgo(int hours) {
    return '$hours ч назад';
  }

  @override
  String statusDaysAgo(int days) {
    return '$days дн назад';
  }

  @override
  String statusDurationSeconds(int seconds) {
    return '$seconds с';
  }

  @override
  String statusDurationMinutes(int minutes, int seconds) {
    return '$minutes мин $seconds с';
  }

  @override
  String statusDurationHours(int hours, int minutes) {
    return '$hours ч $minutes мин';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '$days дн $hours ч';
  }

  @override
  String get notificationPermissionTitle => 'Уведомления';

  @override
  String get notificationPermissionGranted =>
      'PocketClaw может показывать уведомление о работе.';

  @override
  String get notificationPermissionBlocked =>
      'Уведомления отключены, поэтому уведомление о работе PocketClaw не появится.';

  @override
  String get notificationPermissionOpenSettings =>
      'Открыть настройки уведомлений';

  @override
  String get whatsNew020New7 =>
      'Управление Telegram из панели: подключить бота, заменить его или отключить.';

  @override
  String get whatsNew020New8 =>
      'Управление провайдерами и моделями в панели: добавить провайдера, сменить ключ API или удалить модель.';

  @override
  String get whatsNew020Improvement12 =>
      'Настройки Telegram применяются сразу после сохранения, без ручного перезапуска.';

  @override
  String get whatsNew020Improvement13 =>
      'Если модель ИИ не настроена, PocketClaw точно называет, чего не хватает, и даёт код для поиска.';

  @override
  String get whatsNew020Improvement14 =>
      'Вход в панель возвращает вас на тот экран, который вы запрашивали.';

  @override
  String get whatsNew020Improvement15 =>
      'Диагностические журналы остаются подробными и никогда не содержат ваши ключи, токены или текст сообщений.';

  @override
  String get whatsNew020Improvement16 =>
      'Панель, которую ещё никто не занял, недоступна из сети даже при включённом публичном режиме.';
}

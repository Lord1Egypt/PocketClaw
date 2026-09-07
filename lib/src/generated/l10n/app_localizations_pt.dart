// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Portuguese (`pt`).
class AppLocalizationsPt extends AppLocalizations {
  AppLocalizationsPt([String locale = 'pt']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => 'Executar';

  @override
  String get stop => 'Parar';

  @override
  String get config => 'Config';

  @override
  String get webAdmin => 'Admin Web';

  @override
  String get logs => 'Registos';

  @override
  String get viewLogs => 'Ver registos';

  @override
  String get statusRunning => 'Em execução';

  @override
  String get statusStopped => 'Parado';

  @override
  String get settings => 'Definições';

  @override
  String get address => 'Endereço';

  @override
  String get port => 'Porta';

  @override
  String get save => 'Guardar';

  @override
  String get showWindow => 'Mostrar janela';

  @override
  String get exit => 'Sair';

  @override
  String get binaryPath => 'Caminho do binário';

  @override
  String get browse => 'Procurar';

  @override
  String get pathError => 'Caminho inválido';

  @override
  String get arguments => 'Argumentos';

  @override
  String get argumentsHint => 'ex. config.json';

  @override
  String get notStarted => 'Serviço não iniciado';

  @override
  String get startHint =>
      'Por favor, inicie o serviço a partir do painel primeiro.';

  @override
  String get goToDashboard => 'Ir para o painel';

  @override
  String get back => 'Voltar';

  @override
  String get forward => 'Avançar';

  @override
  String get refresh => 'Atualizar';

  @override
  String get coreBinaryMissing =>
      'Binário principal não encontrado. Coloque o binário da plataforma em app/bin/ ou defina o caminho nas Definições.';

  @override
  String get coreStartFailed => 'Falha ao iniciar o serviço principal.';

  @override
  String get coreStopFailed => 'Falha ao parar o serviço principal.';

  @override
  String get coreInvalidBinary => 'Ficheiro binário principal inválido.';

  @override
  String coreUnknownError(Object code) {
    return 'Erro principal desconhecido: $code';
  }

  @override
  String get coreValid => 'O binário principal é válido.';

  @override
  String get publicMode => 'Modo público';

  @override
  String get publicModeHintDesc =>
      'Quando ativado, o serviço permite acesso externo e o campo de endereço será desativado';

  @override
  String get publicModeApplying => 'A aplicar o modo de rede...';

  @override
  String get themeSelection => 'Tema';

  @override
  String get check => 'Verificar';

  @override
  String get launchService => 'INICIAR SERVIÇO';

  @override
  String get stopService => 'PARAR SERVIÇO';

  @override
  String get endpoint => 'PONTO DE ACESSO';

  @override
  String get statusActive => 'ATIVO';

  @override
  String get statusSyncing => 'SINCRONIZANDO';

  @override
  String get statusIdle => 'INATIVO';

  @override
  String get publicModeEnabled => 'Modo público ativado';

  @override
  String get localMode => 'Modo local';

  @override
  String get unableToGetDeviceIp =>
      'Não foi possível obter o IP do dispositivo';

  @override
  String get deviceReportingTitle =>
      'Feedback de compatibilidade do dispositivo';

  @override
  String get deviceReportingSubtitle =>
      'Usado apenas para verificar a compatibilidade da versão do SO e da versão da aplicação. Não envolve mensagens de chat, detalhes de conta ou conteúdo pessoal';

  @override
  String get deviceReportingConsentTitle =>
      'Ajude a melhorar a compatibilidade do dispositivo';

  @override
  String get deviceReportingConsentDescription =>
      'Quando ativado, apenas um ID de instalação anónimo, a versão do SO e a versão da aplicação são enviados para compreender a compatibilidade. O idioma e a região podem ser recolhidos separadamente pelo Firebase Analytics. Nenhuma mensagem de chat, conteúdo digitado, detalhes de conta, ficheiros ou configurações personalizadas são carregados';

  @override
  String get deviceReportingBannerDescription =>
      'Apenas um ID de instalação anónimo, a versão do SO e a versão da aplicação são sincronizados para melhorar a compatibilidade. O idioma e a região podem ser recolhidos separadamente pelo Firebase Analytics. Nenhuma mensagem de chat, detalhes de conta, ficheiros ou conteúdo pessoal são enviados';

  @override
  String get deviceReportingWhatWillBeSent =>
      'Apenas estes detalhes do dispositivo estão incluídos';

  @override
  String get deviceReportingDeviceLabel => 'Modelo do dispositivo';

  @override
  String get deviceReportingPlatformLabel => 'Categoria do dispositivo';

  @override
  String get deviceReportingSystemLabel => 'Versão do SO';

  @override
  String get deviceReportingTimingNote =>
      'Uma sincronização é executada uma vez quando ativada, e novamente apenas após uma atualização do sistema ser detectada';

  @override
  String get deviceReportingDeny => 'Agora não';

  @override
  String get deviceReportingAllow => 'Ativar';

  @override
  String get deviceReportingUploadSucceeded =>
      'Feedback de compatibilidade do dispositivo ativado';

  @override
  String get deviceReportingUploadFailed =>
      'Feedback de compatibilidade do dispositivo ativado, mas a sincronização atual das informações do dispositivo não foi concluída';

  @override
  String get deviceReportingDisabled =>
      'Feedback de compatibilidade do dispositivo desativado';

  @override
  String get localModeHint =>
      '1. Vá para Configuração do serviço\n2. Ative o Modo público\n3. Leia o código QR para aceder ao PocketClaw';

  @override
  String get publicModeHint =>
      '1. Inicie o serviço\n2. Leia o código QR para aceder ao PocketClaw';

  @override
  String get noLogsToExport => 'Não há registos para exportar';

  @override
  String get logsSavedToMediaLibrary =>
      'Registos guardados em Transferências (biblioteca de mídia Android)';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Registos guardados em Transferências: $path';
  }

  @override
  String get shareLogsText => 'Registos do PocketClaw';

  @override
  String get workspaceDirectory => 'Espaço de trabalho';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'Registos guardados em Transferências (biblioteca de mídia Android): $name';
  }

  @override
  String shareFailed(Object error) {
    return 'Falha ao abrir a caixa de diálogo de partilha: $error';
  }

  @override
  String get exportLogs => 'Exportar registos';

  @override
  String logEventsCount(int count) {
    return '$count EVENTOS';
  }

  @override
  String get unsavedChanges => 'Alterações não guardadas';

  @override
  String get unsavedChangesHint =>
      'Tem alterações não guardadas. Deseja descartá-las?';

  @override
  String get cancel => 'Cancelar';

  @override
  String get discard => 'Descartar';

  @override
  String get saved => 'Guardado';

  @override
  String get language => 'Idioma';

  @override
  String get selectLanguage => 'Selecionar idioma';

  @override
  String get about => 'Sobre';

  @override
  String get aboutDescription =>
      'PocketClaw é o seu espaço de trabalho privado para assistente de IA.';

  @override
  String get aboutAppVersionLabel => 'Versão do PocketClaw';

  @override
  String get aboutCoreVersionLabel => 'Versão do runtime';

  @override
  String get aboutVersionUnavailable => 'Indisponível';

  @override
  String get close => 'Fechar';

  @override
  String get contextMemoryTitle => 'Memória de contexto do Telegram';

  @override
  String get contextMemoryDescription =>
      'Controla quantas mensagens recentes da conversa são enviadas à IA. As conversas mais antigas permanecem no Telegram e são representadas pelo resumo contínuo.';

  @override
  String get contextMemoryHelp =>
      'Mais mensagens oferecem mais contexto recente, mas consomem mais tokens. Mensagens antigas permanecem no Telegram e podem ser mantidas pelo resumo contínuo.';

  @override
  String get contextMemoryRecommended => 'Recomendado';

  @override
  String get contextMemoryCustom => 'Personalizado';

  @override
  String get contextMemoryCustomLabel => 'Mensagens';

  @override
  String contextMemoryRangeError(int min, int max) {
    return 'Digite um número inteiro entre $min e $max.';
  }

  @override
  String get contextMemorySaveFailed =>
      'Não foi possível salvar esta configuração.';

  @override
  String get settingsSave => 'Salvar';

  @override
  String get autoStartServiceTitle =>
      'Iniciar o serviço do PocketClaw automaticamente';

  @override
  String get autoStartGatewayTitle => 'Iniciar o gateway automaticamente';

  @override
  String get autoStartPreferenceOn =>
      'Preferência de início automático: ATIVADA';

  @override
  String get autoStartPreferenceOff =>
      'Preferência de início automático: DESATIVADA';

  @override
  String get runtimeRunning => 'Tempo de execução: Em execução';

  @override
  String get runtimeStarting => 'Tempo de execução: Iniciando';

  @override
  String get runtimeStopped => 'Tempo de execução: Parado';

  @override
  String get gatewayAutoStartHint =>
      'Aplica-se na próxima vez que o serviço do PocketClaw iniciar. O tempo de execução do gateway é gerenciado no painel.';

  @override
  String get manageTelegramConnection => 'Gerenciar a conexão do Telegram';

  @override
  String get manageModelsTitle => 'Gerenciar modelos';

  @override
  String get manageModelsDescription =>
      'Adicione, edite, teste e escolha modelos de IA.';

  @override
  String get githubChecking => 'Verificando…';

  @override
  String get githubConnected => 'Conectado';

  @override
  String get githubNotConnected => 'Não conectado';

  @override
  String githubConnectedAs(String login) {
    return 'Conectado como $login';
  }

  @override
  String get githubDescription =>
      'Usado pelo gh incluído e pelo Git via HTTPS. O token é criptografado neste dispositivo e nunca é exibido novamente.';

  @override
  String get githubTestConnection => 'Testar conexão';

  @override
  String get githubDisconnect => 'Desconectar';

  @override
  String get githubConnectAction => 'Conectar o GitHub';

  @override
  String get githubConnect => 'Conectar';

  @override
  String get githubTokenLabel => 'Token de acesso pessoal';

  @override
  String get githubTokenHint =>
      'Cole um token de acesso pessoal do GitHub com os escopos necessários (repo para repositórios privados).';

  @override
  String get githubTokenRejected => 'O GitHub não aceitou este token.';

  @override
  String get githubAuthWorking => 'A autenticação do GitHub está funcionando.';

  @override
  String get githubAuthNotWorking =>
      'A autenticação do GitHub não está funcionando.';

  @override
  String githubAuthenticatedAs(String login) {
    return 'Autenticado como $login.';
  }

  @override
  String get githubCredentialRemoveFailed =>
      'Não foi possível remover a credencial.';

  @override
  String get githubYourAccount => 'sua conta do GitHub';

  @override
  String githubConnectedReport(String who, String outcome) {
    return 'Conectado como $who. $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return 'Desconectado. $outcome';
  }

  @override
  String get credentialAppliedNow => 'O gh e o git já podem usá-lo.';

  @override
  String get credentialAppliesNextStart =>
      'Ele será usado na próxima vez que o PocketClaw iniciar.';

  @override
  String get credentialAppliesDeferred =>
      'Salvo. O PocketClaw está iniciando, então será aplicado automaticamente assim que terminar.';

  @override
  String get whatsNewTitle => 'Novidades';

  @override
  String get whatsNewDescription => 'As principais mudanças desta versão.';

  @override
  String get whatsNewBadge => 'NOVO';

  @override
  String get whatsNewSectionNew => 'Novidades';

  @override
  String get whatsNewSectionImprovements => 'Melhorias';

  @override
  String get whatsNewSectionFixes => 'Correções';

  @override
  String get whatsNew020New1 =>
      'O tempo de execução gerenciado instala e mantém atualizadas as ferramentas que acompanham o PocketClaw.';

  @override
  String get whatsNew020New2 =>
      'Ferramentas de desenvolvimento incluídas: Git, GitHub CLI, curl, ripgrep, jq e SQLite.';

  @override
  String get whatsNew020New3 =>
      'Suporte ao tempo de execução Python 3.14 incluído no PocketClaw.';

  @override
  String get whatsNew020New4 =>
      'Login seguro no GitHub, compartilhado pelo Git e pela CLI do GitHub incluídos.';

  @override
  String get whatsNew020New5 =>
      'Integração com o Telegram, configurada nas Configurações.';

  @override
  String get whatsNew020Improvement1 =>
      'Provedores mais resilientes: uma requisição com falha não encerra mais o turno.';

  @override
  String get whatsNew020Improvement2 =>
      'Um catálogo do tempo de execução gerenciado mais preciso.';

  @override
  String get whatsNew020Improvement3 =>
      'Uma identidade do PocketClaw mais consistente na interface web e no espaço de trabalho padrão.';

  @override
  String get whatsNew020Fix1 =>
      'O gateway agora se recupera de um registro de processo obsoleto deixado por uma execução anterior.';

  @override
  String get whatsNew020Fix2 =>
      'Listas de canais com mais de uma entrada são preservadas corretamente ao salvar.';

  @override
  String get settingsGroupConnection => 'Conexão';

  @override
  String get settingsGroupAgent => 'Agente';

  @override
  String get settingsGroupIntegrations => 'Integrações';

  @override
  String get settingsGroupAppearance => 'Aparência';

  @override
  String get statusTitle => 'Estado';

  @override
  String get statusSectionSystem => 'Sistema';

  @override
  String get statusSectionAi => 'IA';

  @override
  String get statusSectionActivity => 'Atividade';

  @override
  String get statusSectionChannels => 'Canais';

  @override
  String get statusSectionResources => 'Recursos';

  @override
  String get statusSinceGatewayStart => 'Desde o início do Gateway';

  @override
  String get statusGateway => 'Gateway';

  @override
  String get statusUptime => 'Tempo ativo';

  @override
  String get statusAppVersion => 'Versão do app';

  @override
  String get statusCoreVersion => 'Versão do Core';

  @override
  String get statusActiveModel => 'Modelo ativo';

  @override
  String get statusConfiguredDefault => 'Padrão configurado';

  @override
  String get statusProvider => 'Provedor';

  @override
  String get statusFallbacks => 'Alternativas';

  @override
  String get statusActiveTurns => 'Turnos ativos';

  @override
  String get statusActiveSubagents => 'Subagentes ativos';

  @override
  String get statusWaiting => 'Aguardando';

  @override
  String get statusCompleted => 'Concluídos';

  @override
  String get statusFailed => 'Falhados';

  @override
  String get statusCancelled => 'Cancelados';

  @override
  String get statusToolCalls => 'Chamadas de ferramentas';

  @override
  String get statusToolCallsFailed => 'Chamadas com falha';

  @override
  String get statusLastActivity => 'Última atividade';

  @override
  String get statusCoreMemory => 'Memória do Core';

  @override
  String get statusCoreCpuTime => 'Tempo de CPU do Core';

  @override
  String get statusChannelFailedToStart => 'Falha ao iniciar';

  @override
  String get statusNoChannels => 'Nenhum canal configurado';

  @override
  String get statusDetailUnavailable => 'Estado detalhado indisponível';

  @override
  String get statusJustNow => 'Agora mesmo';

  @override
  String statusMinutesAgo(int minutes) {
    return 'há $minutes min';
  }

  @override
  String statusHoursAgo(int hours) {
    return 'há $hours h';
  }

  @override
  String statusDaysAgo(int days) {
    return 'há $days d';
  }
}

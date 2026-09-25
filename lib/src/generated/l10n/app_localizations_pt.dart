// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Portuguese (`pt`).
class AppLocalizationsPt extends AppLocalizations {
  AppLocalizationsPt([String locale = 'pt']) : super(locale);

  @override
  String get stop => 'Parar';

  @override
  String get webAdmin => 'Admin Web';

  @override
  String get logs => 'Registos';

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
  String get publicMode => 'Modo público';

  @override
  String get publicModeHintDesc =>
      'Quando ativado, o serviço permite acesso externo e o campo de endereço será desativado';

  @override
  String get publicModeApplying => 'A aplicar o modo de rede...';

  @override
  String get themeSelection => 'Tema';

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
  String get localModeHint =>
      '1. Vá para Configuração do serviço\n2. Ative o Modo público\n3. Leia o código QR para aceder ao PocketClaw';

  @override
  String get publicModeHint =>
      '1. Inicie o serviço\n2. Leia o código QR para aceder ao PocketClaw';

  @override
  String get noLogsToExport => 'Não há registos para exportar';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Registos guardados em Transferências: $path';
  }

  @override
  String get shareLogsText => 'Registos do PocketClaw';

  @override
  String get workspaceDirectory => 'Espaço de trabalho';

  @override
  String get legacyWorkspaceTitle => 'Espaço de trabalho anterior encontrado';

  @override
  String legacyWorkspaceBody(Object path) {
    return 'Uma versão anterior guardava seu espaço de trabalho em $path. O PocketClaw agora usa o armazenamento próprio do app e não alterou essa pasta. Você pode copiá-la: a cópia vai para uma pasta própria, e nada é sobrescrito nem excluído.';
  }

  @override
  String get legacyWorkspaceImport => 'Copiar para o espaço de trabalho';

  @override
  String get legacyWorkspaceHide => 'Ocultar';

  @override
  String legacyWorkspaceCopied(Object count, Object folder) {
    return '$count arquivos copiados para $folder.';
  }

  @override
  String legacyWorkspacePartial(Object count, Object folder, Object failed) {
    return '$count arquivos copiados para $folder. Não foi possível copiar $failed.';
  }

  @override
  String get legacyWorkspaceFailed => 'Não foi possível copiar a pasta.';

  @override
  String get legacyWorkspaceEmpty =>
      'A pasta selecionada não tinha nada para copiar.';

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
  String get cancel => 'Cancelar';

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
      'Configuração do Telegram com um toque: o PocketClaw cria o seu próprio bot, no aplicativo ou pelo painel em um navegador. Não há token para copiar e apenas a sua conta pode falar com ele.';

  @override
  String get whatsNew020New6 =>
      'Uma visão de Status no painel: trabalho ativo, canais, o modelo em uso e os recursos de execução.';

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
  String get whatsNew020Improvement4 =>
      'Um novo ícone do app e uma tela Sobre renovada, com o design Aperture do PocketClaw.';

  @override
  String get whatsNew020Improvement5 =>
      'As respostas do assistente ficam mais naturais, sem uma assinatura fixa no final.';

  @override
  String get whatsNew020Improvement6 =>
      'O PocketClaw não pede mais a permissão de Telefone — nada no app a usava.';

  @override
  String get whatsNew020Improvement7 =>
      'Os registros de diagnóstico agora ficam guardados de forma privada dentro do app, não na pasta de downloads. Seu espaço de trabalho continua onde estava.';

  @override
  String get whatsNew020Improvement8 =>
      'Uma melhoria significativa na forma como o PocketClaw guarda seu estado e executa seus serviços, para mais estabilidade no dia a dia.';

  @override
  String get whatsNew020Improvement9 =>
      'Após esta atualização, o painel pede que você entre novamente uma vez.';

  @override
  String get whatsNew020Improvement10 =>
      'Após esta atualização, o canal de conversa Web inicia uma nova conversa.';

  @override
  String get whatsNew020Improvement11 =>
      'Após esta atualização, vale a pena conferir uma vez suas preferências de notificação.';

  @override
  String get whatsNew020Fix1 =>
      'O gateway agora se recupera de um registro de processo obsoleto deixado por uma execução anterior.';

  @override
  String get whatsNew020Fix2 =>
      'Listas de canais com mais de uma entrada são preservadas corretamente ao salvar.';

  @override
  String get whatsNew021Fix1 =>
      'O Git incluído não trava mais ao clonar um repositório ou atualizar um branch.';

  @override
  String get whatsNew022Improvement1 =>
      'Depois de uma tarefa longa, o Telegram entrega a resposta como uma nova mensagem, então você é notificado e ela aparece abaixo do que enviou nesse meio-tempo.';

  @override
  String get whatsNew022Improvement2 =>
      'O Telegram avisa quando sua mensagem está na fila e quantas estão à frente dela.';

  @override
  String get whatsNew022Improvement3 =>
      'O espaço de trabalho agora fica no armazenamento próprio do PocketClaw, e o app não pede nenhuma permissão de armazenamento. Um espaço de trabalho deixado por uma versão anterior em Download/pocketclaw não é alterado e pode ser copiado nas Configurações.';

  @override
  String get whatsNew022Fix1 =>
      'Saídas de ferramentas muito grandes não estouram mais a conversa.';

  @override
  String get whatsNew022Fix2 =>
      'Removidos itens das Configurações sem efeito no Android: Dispositivos, Iniciar ao entrar e Porta do serviço.';

  @override
  String get whatsNew022Improvement4 =>
      'O PocketClaw agora requer Android 8.0 ou mais recente.';

  @override
  String get whatsNew022Fix3 =>
      'Corrigida uma falha na inicialização do Android que podia aparecer depois de reiniciar o celular.';

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
  String get notificationPermissionTitle => 'Notificações';

  @override
  String get notificationPermissionGranted =>
      'O PocketClaw pode mostrar a notificação de execução.';

  @override
  String get notificationPermissionBlocked =>
      'As notificações estão desativadas, portanto a notificação de execução do PocketClaw não aparecerá.';

  @override
  String get notificationPermissionOpenSettings =>
      'Abrir configurações de notificação';

  @override
  String get whatsNew020New7 =>
      'Gerenciamento do Telegram pelo painel: conectar um bot, substituí-lo ou desconectá-lo.';

  @override
  String get whatsNew020New8 =>
      'Gerenciamento de provedores e modelos no painel: adicionar um provedor, trocar uma chave de API ou remover um modelo.';

  @override
  String get whatsNew020Improvement12 =>
      'As configurações do Telegram são aplicadas assim que você salva, sem reinício manual.';

  @override
  String get whatsNew020Improvement13 =>
      'Quando nenhum modelo de IA está configurado, o PocketClaw diz exatamente o que falta e fornece um código para consultar.';

  @override
  String get whatsNew020Improvement14 =>
      'Entrar no painel leva você à tela que você pediu.';

  @override
  String get whatsNew020Improvement15 =>
      'Os registros de diagnóstico continuam detalhados sem nunca conter suas chaves, tokens ou o texto das mensagens.';

  @override
  String get whatsNew020Improvement16 =>
      'Um painel que ainda não foi reivindicado nunca fica acessível pela rede, mesmo com o modo público ativado.';
}

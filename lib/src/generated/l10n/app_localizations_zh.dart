// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => '运行';

  @override
  String get stop => '停止';

  @override
  String get config => '配置';

  @override
  String get webAdmin => '后台管理';

  @override
  String get logs => '日志';

  @override
  String get viewLogs => '查看日志';

  @override
  String get statusRunning => '正在运行';

  @override
  String get statusStopped => '已停止';

  @override
  String get settings => '设置';

  @override
  String get address => '地址';

  @override
  String get port => '端口';

  @override
  String get save => '保存';

  @override
  String get showWindow => '显示界面';

  @override
  String get exit => '彻底退出';

  @override
  String get binaryPath => '程序路径';

  @override
  String get browse => '浏览';

  @override
  String get pathError => '无效路径';

  @override
  String get arguments => '运行参数';

  @override
  String get argumentsHint => '例如: config.json';

  @override
  String get notStarted => '服务暂未启动';

  @override
  String get startHint => '请先从 Dashboard 主面板启动服务。';

  @override
  String get goToDashboard => '去主面板';

  @override
  String get back => '后退';

  @override
  String get forward => '前进';

  @override
  String get refresh => '刷新';

  @override
  String get coreBinaryMissing => '未找到核心二进制文件。请将平台二进制放入 app/bin/ 或在设置中指定路径。';

  @override
  String get coreStartFailed => '启动核心服务失败。';

  @override
  String get coreStopFailed => '停止核心服务失败。';

  @override
  String get coreInvalidBinary => '核心二进制文件无效。';

  @override
  String coreUnknownError(Object code) {
    return '未知的核心错误：$code';
  }

  @override
  String get coreValid => '核心二进制文件有效。';

  @override
  String get publicMode => '公共模式';

  @override
  String get publicModeHintDesc => '开启后服务将允许外部访问，地址栏将被禁用';

  @override
  String get publicModeApplying => '正在应用网络模式...';

  @override
  String get themeSelection => '主题';

  @override
  String get check => '检查';

  @override
  String get launchService => '启动服务';

  @override
  String get stopService => '停止服务';

  @override
  String get endpoint => '访问地址';

  @override
  String get statusActive => '运行中';

  @override
  String get statusSyncing => '启动中';

  @override
  String get statusIdle => '待机';

  @override
  String get publicModeEnabled => '公共模式已开启';

  @override
  String get localMode => '本地模式';

  @override
  String get unableToGetDeviceIp => '无法获取设备IP';

  @override
  String get deviceReportingTitle => '设备兼容反馈';

  @override
  String get deviceReportingSubtitle =>
      '仅用于识别操作系统版本与应用版本的兼容性，不涉及聊天消息、账号信息或个人内容';

  @override
  String get deviceReportingConsentTitle => '帮助改进设备兼容性';

  @override
  String get deviceReportingConsentDescription =>
      '开启后仅会发送匿名安装标识、操作系统版本和应用版本，用于判断适配情况。设备语言与区域可能由 Firebase Analytics 单独采集。不会上传聊天消息、输入内容、账号信息、文件或自定义配置';

  @override
  String get deviceReportingBannerDescription =>
      '仅同步匿名安装标识、操作系统版本和应用版本，用于改善适配；设备语言与区域可能由 Firebase Analytics 单独采集。不会发送聊天消息、账号、文件或任何个人内容';

  @override
  String get deviceReportingWhatWillBeSent => '仅包含以下设备信息';

  @override
  String get deviceReportingDeviceLabel => '设备型号';

  @override
  String get deviceReportingPlatformLabel => '设备类别';

  @override
  String get deviceReportingSystemLabel => '系统版本';

  @override
  String get deviceReportingTimingNote => '仅在开启时同步一次，之后只会在检测到系统更新时再次同步';

  @override
  String get deviceReportingDeny => '暂不开启';

  @override
  String get deviceReportingAllow => '开启';

  @override
  String get deviceReportingUploadSucceeded => '已开启设备兼容反馈';

  @override
  String get deviceReportingUploadFailed => '已开启设备兼容反馈，当前设备信息同步未完成';

  @override
  String get deviceReportingDisabled => '已关闭设备兼容反馈';

  @override
  String get localModeHint => '1. 进入服务配置\n2. 打开公共模式\n3. 扫描二维码访问PocketClaw';

  @override
  String get publicModeHint => '1. 启动服务\n2. 扫描二维码访问PocketClaw';

  @override
  String get noLogsToExport => '没有可导出的日志';

  @override
  String get logsSavedToMediaLibrary => '已保存到“下载”目录（Android 媒体库）';

  @override
  String logsSavedToDownloads(Object path) {
    return '已保存到 Downloads：$path';
  }

  @override
  String get shareLogsText => 'PocketClaw 日志';

  @override
  String get workspaceDirectory => '工作目录';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return '已保存到“下载”目录（Android 媒体库）：$name';
  }

  @override
  String shareFailed(Object error) {
    return '打开分享对话框失败：$error';
  }

  @override
  String get exportLogs => '导出日志';

  @override
  String logEventsCount(int count) {
    return '$count 条日志';
  }

  @override
  String get unsavedChanges => '未保存的更改';

  @override
  String get unsavedChangesHint => '您有未保存的更改，确定要放弃吗？';

  @override
  String get cancel => '取消';

  @override
  String get discard => '放弃';

  @override
  String get saved => '已保存';

  @override
  String get language => '语言';

  @override
  String get selectLanguage => '选择语言';

  @override
  String get about => '关于';

  @override
  String get aboutDescription => 'PocketClaw 是您的私有 AI 助手工作区。';

  @override
  String get aboutAppVersionLabel => 'PocketClaw 版本';

  @override
  String get aboutCoreVersionLabel => '运行时版本';

  @override
  String get aboutVersionUnavailable => '不可用';

  @override
  String get close => '关闭';

  @override
  String get contextMemoryTitle => 'Telegram 上下文记忆';

  @override
  String get contextMemoryDescription =>
      '控制发送给 AI 的最近对话消息数量。更早的对话仍保留在 Telegram 中，并由滚动摘要表示。';

  @override
  String get contextMemoryHelp =>
      '消息越多，近期上下文越丰富，但消耗的令牌也越多。更早的消息会保留在 Telegram 中，并可能通过滚动摘要保留下来。';

  @override
  String get contextMemoryRecommended => '推荐';

  @override
  String get contextMemoryCustom => '自定义';

  @override
  String get contextMemoryCustomLabel => '消息数';

  @override
  String contextMemoryRangeError(int min, int max) {
    return '请输入 $min 到 $max 之间的整数。';
  }

  @override
  String get contextMemorySaveFailed => '无法保存此设置。';

  @override
  String get settingsSave => '保存';

  @override
  String get autoStartServiceTitle => '自动启动 PocketClaw 服务';

  @override
  String get autoStartGatewayTitle => '自动启动网关';

  @override
  String get autoStartPreferenceOn => '自动启动偏好：开启';

  @override
  String get autoStartPreferenceOff => '自动启动偏好：关闭';

  @override
  String get runtimeRunning => '运行状态：运行中';

  @override
  String get runtimeStarting => '运行状态：正在启动';

  @override
  String get runtimeStopped => '运行状态：已停止';

  @override
  String get gatewayAutoStartHint => '将在下次启动 PocketClaw 服务时生效。网关运行状态在仪表板中管理。';

  @override
  String get manageTelegramConnection => '管理 Telegram 连接';

  @override
  String get manageModelsTitle => '管理模型';

  @override
  String get manageModelsDescription => '添加、编辑、测试并选择 AI 模型。';

  @override
  String get githubChecking => '正在检查…';

  @override
  String get githubConnected => '已连接';

  @override
  String get githubNotConnected => '未连接';

  @override
  String githubConnectedAs(String login) {
    return '已以 $login 身份连接';
  }

  @override
  String get githubDescription =>
      '由内置的 gh 和通过 HTTPS 的 Git 使用。令牌在本设备上加密存储，不会再次显示。';

  @override
  String get githubTestConnection => '测试连接';

  @override
  String get githubDisconnect => '断开连接';

  @override
  String get githubConnectAction => '连接 GitHub';

  @override
  String get githubConnect => '连接';

  @override
  String get githubTokenLabel => '个人访问令牌';

  @override
  String get githubTokenHint => '粘贴具有所需权限范围的 GitHub 个人访问令牌（私有仓库需要 repo）。';

  @override
  String get githubTokenRejected => 'GitHub 未接受此令牌。';

  @override
  String get githubAuthWorking => 'GitHub 身份验证正常。';

  @override
  String get githubAuthNotWorking => 'GitHub 身份验证未正常工作。';

  @override
  String githubAuthenticatedAs(String login) {
    return '已以 $login 身份通过验证。';
  }

  @override
  String get githubCredentialRemoveFailed => '无法移除凭据。';

  @override
  String get githubYourAccount => '您的 GitHub 账户';

  @override
  String githubConnectedReport(String who, String outcome) {
    return '已以 $who 身份连接。$outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return '已断开连接。$outcome';
  }

  @override
  String get credentialAppliedNow => 'gh 和 git 现在即可使用。';

  @override
  String get credentialAppliesNextStart => '将在下次启动 PocketClaw 时使用。';

  @override
  String get credentialAppliesDeferred => '已保存。PocketClaw 正在启动，完成后将自动生效。';

  @override
  String get whatsNewTitle => '新功能';

  @override
  String get whatsNewDescription => '本次发布的主要变化。';

  @override
  String get whatsNewBadge => '新';

  @override
  String get whatsNewSectionNew => '新增';

  @override
  String get whatsNewSectionImprovements => '改进';

  @override
  String get whatsNewSectionFixes => '修复';

  @override
  String get whatsNew020New1 => '托管运行时会为你安装并持续更新 PocketClaw 自带的工具。';

  @override
  String get whatsNew020New2 =>
      '内置开发工具：Git、GitHub CLI、curl、ripgrep、jq 和 SQLite。';

  @override
  String get whatsNew020New3 => '支持 PocketClaw 内置的 Python 3.14 运行时。';

  @override
  String get whatsNew020New4 => '安全的 GitHub 登录，内置的 Git 与 GitHub CLI 共用同一凭据。';

  @override
  String get whatsNew020New5 => 'Telegram 集成，可在“设置”中配置。';

  @override
  String get whatsNew020Improvement1 => '更稳健的服务商：单次请求失败不再中断本轮对话。';

  @override
  String get whatsNew020Improvement2 => '更准确的托管运行时目录。';

  @override
  String get whatsNew020Improvement3 => '网页界面与默认工作区中的 PocketClaw 标识更加统一。';

  @override
  String get whatsNew020Fix1 => '网关现在可以从上一次运行遗留的过期进程记录中恢复。';

  @override
  String get whatsNew020Fix2 => '包含多个条目的频道列表在保存时能够正确保留。';
}

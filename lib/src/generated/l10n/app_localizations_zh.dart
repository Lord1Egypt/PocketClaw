// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Chinese (`zh`).
class AppLocalizationsZh extends AppLocalizations {
  AppLocalizationsZh([String locale = 'zh']) : super(locale);

  @override
  String get stop => '停止';

  @override
  String get webAdmin => '后台管理';

  @override
  String get logs => '日志';

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
  String get publicMode => '公共模式';

  @override
  String get publicModeHintDesc => '开启后服务将允许外部访问，地址栏将被禁用';

  @override
  String get publicModeApplying => '正在应用网络模式...';

  @override
  String get themeSelection => '主题';

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
  String get localModeHint => '1. 进入服务配置\n2. 打开公共模式\n3. 扫描二维码访问PocketClaw';

  @override
  String get publicModeHint => '1. 启动服务\n2. 扫描二维码访问PocketClaw';

  @override
  String get noLogsToExport => '没有可导出的日志';

  @override
  String logsSavedToDownloads(Object path) {
    return '已保存到 Downloads：$path';
  }

  @override
  String get shareLogsText => 'PocketClaw 日志';

  @override
  String get workspaceDirectory => '工作目录';

  @override
  String get legacyWorkspaceTitle => '发现旧的工作区';

  @override
  String legacyWorkspaceBody(Object path) {
    return '旧版本将你的工作区保存在 $path。PocketClaw 现在使用应用自己的存储空间，并未改动该文件夹。你可以将其复制进来：副本会放入单独的文件夹，不会覆盖或删除任何内容。';
  }

  @override
  String get legacyWorkspaceImport => '复制到工作区';

  @override
  String get legacyWorkspaceHide => '隐藏';

  @override
  String legacyWorkspaceCopied(Object count, Object folder) {
    return '已将 $count 个文件复制到 $folder。';
  }

  @override
  String legacyWorkspacePartial(Object count, Object folder, Object failed) {
    return '已将 $count 个文件复制到 $folder。有 $failed 个未能复制。';
  }

  @override
  String get legacyWorkspaceFailed => '无法复制该文件夹。';

  @override
  String get legacyWorkspaceEmpty => '所选文件夹中没有可复制的内容。';

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
  String get cancel => '取消';

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
  String get whatsNew020New5 =>
      '一键配置 Telegram：PocketClaw 会为你创建自己的机器人，可在应用内或浏览器中的仪表板完成。无需复制任何令牌，而且只有你的账号能与它对话。';

  @override
  String get whatsNew020New6 => '仪表板新增状态视图：进行中的工作、渠道、正在使用的模型和运行资源。';

  @override
  String get whatsNew020Improvement1 => '更稳健的服务商：单次请求失败不再中断本轮对话。';

  @override
  String get whatsNew020Improvement2 => '更准确的托管运行时目录。';

  @override
  String get whatsNew020Improvement3 => '网页界面与默认工作区中的 PocketClaw 标识更加统一。';

  @override
  String get whatsNew020Improvement4 =>
      '采用 PocketClaw Aperture 设计的全新应用图标与“关于”界面。';

  @override
  String get whatsNew020Improvement5 => '助手的回复更自然，末尾不再附带固定签名。';

  @override
  String get whatsNew020Improvement6 => 'PocketClaw 不再申请电话权限——应用中没有任何功能使用它。';

  @override
  String get whatsNew020Improvement7 => '诊断日志现在私密地保存在应用内部，而不是下载文件夹中。工作区位置保持不变。';

  @override
  String get whatsNew020Improvement8 =>
      'PocketClaw 保存状态和运行服务的方式经过了一次重要升级，日常运行更加稳定可靠。';

  @override
  String get whatsNew020Improvement9 => '本次更新后，控制台会要求您重新登录一次。';

  @override
  String get whatsNew020Improvement10 => '本次更新后，网页聊天频道会开始一段新的会话。';

  @override
  String get whatsNew020Improvement11 => '本次更新后，建议检查一次您的通知偏好设置。';

  @override
  String get whatsNew020Fix1 => '网关现在可以从上一次运行遗留的过期进程记录中恢复。';

  @override
  String get whatsNew020Fix2 => '包含多个条目的频道列表在保存时能够正确保留。';

  @override
  String get whatsNew021Fix1 => '内置 Git 在克隆仓库或更新分支时不再崩溃。';

  @override
  String get whatsNew022Improvement1 =>
      '长任务完成后，Telegram 会以新消息发送回答，因此你会收到通知，且回答显示在你期间发送的所有消息下方。';

  @override
  String get whatsNew022Improvement2 => '消息排队时，Telegram 会告诉你前面还有几条。';

  @override
  String get whatsNew022Improvement3 =>
      '工作区现在位于 PocketClaw 自己的存储空间中，应用不再请求任何存储权限。旧版本留在 Download/pocketclaw 中的工作区保持不变，可以在设置中复制进来。';

  @override
  String get whatsNew022Fix1 => '超大的工具输出不再会撑爆对话。';

  @override
  String get whatsNew022Fix2 => '移除了在 Android 上没有作用的设置项：设备、登录时启动和服务端口。';

  @override
  String get whatsNew022Improvement4 => 'PocketClaw 现在需要 Android 8.0 或更高版本。';

  @override
  String get whatsNew022Fix3 => '修复了重启手机后可能出现的 Android 启动崩溃问题。';

  @override
  String get settingsGroupConnection => '连接';

  @override
  String get settingsGroupAgent => '智能体';

  @override
  String get settingsGroupIntegrations => '集成';

  @override
  String get settingsGroupAppearance => '外观';

  @override
  String get statusTitle => '状态';

  @override
  String get statusSectionSystem => '系统';

  @override
  String get statusSectionAi => 'AI';

  @override
  String get statusSectionActivity => '活动';

  @override
  String get statusSectionChannels => '频道';

  @override
  String get statusSectionResources => '资源';

  @override
  String get statusSinceGatewayStart => '自网关启动起';

  @override
  String get statusGateway => '网关';

  @override
  String get statusUptime => '运行时间';

  @override
  String get statusAppVersion => '应用版本';

  @override
  String get statusCoreVersion => 'Core 版本';

  @override
  String get statusActiveModel => '当前模型';

  @override
  String get statusConfiguredDefault => '配置的默认模型';

  @override
  String get statusProvider => '提供方';

  @override
  String get statusFallbacks => '备用模型';

  @override
  String get statusActiveTurns => '进行中的轮次';

  @override
  String get statusActiveSubagents => '活动子代理';

  @override
  String get statusWaiting => '等待中';

  @override
  String get statusCompleted => '已完成';

  @override
  String get statusFailed => '失败';

  @override
  String get statusCancelled => '已取消';

  @override
  String get statusToolCalls => '工具调用';

  @override
  String get statusToolCallsFailed => '失败的工具调用';

  @override
  String get statusLastActivity => '最近活动';

  @override
  String get statusCoreMemory => 'Core 内存';

  @override
  String get statusCoreCpuTime => 'Core CPU 时间';

  @override
  String get statusNoChannels => '未配置频道';

  @override
  String get statusDetailUnavailable => '详细状态不可用';

  @override
  String get statusJustNow => '刚刚';

  @override
  String statusMinutesAgo(int minutes) {
    return '$minutes 分钟前';
  }

  @override
  String statusHoursAgo(int hours) {
    return '$hours 小时前';
  }

  @override
  String statusDaysAgo(int days) {
    return '$days 天前';
  }

  @override
  String statusDurationSeconds(int seconds) {
    return '$seconds 秒';
  }

  @override
  String statusDurationMinutes(int minutes, int seconds) {
    return '$minutes 分 $seconds 秒';
  }

  @override
  String statusDurationHours(int hours, int minutes) {
    return '$hours 小时 $minutes 分';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '$days 天 $hours 小时';
  }

  @override
  String get notificationPermissionTitle => '通知';

  @override
  String get notificationPermissionGranted => 'PocketClaw 可以显示运行中通知。';

  @override
  String get notificationPermissionBlocked => '通知已关闭，因此不会显示 PocketClaw 运行中通知。';

  @override
  String get notificationPermissionOpenSettings => '打开通知设置';

  @override
  String get whatsNew020New7 => '在仪表板中管理 Telegram：连接机器人、更换机器人或断开连接。';

  @override
  String get whatsNew020New8 => '在仪表板中管理提供商与模型：添加提供商、轮换 API 密钥或删除模型。';

  @override
  String get whatsNew020Improvement12 => 'Telegram 设置保存后立即生效，无需手动重启。';

  @override
  String get whatsNew020Improvement13 =>
      '当尚未配置 AI 模型时，PocketClaw 会准确说明缺少什么，并给出可查阅的代码。';

  @override
  String get whatsNew020Improvement14 => '登录仪表板后会回到你原本要去的页面。';

  @override
  String get whatsNew020Improvement15 => '诊断日志保持详尽，但绝不包含你的密钥、令牌或消息内容。';

  @override
  String get whatsNew020Improvement16 => '尚无人认领的仪表板绝不会从网络可达，即使已开启公开模式。';
}

// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Korean (`ko`).
class AppLocalizationsKo extends AppLocalizations {
  AppLocalizationsKo([String locale = 'ko']) : super(locale);

  @override
  String get stop => '중지';

  @override
  String get webAdmin => '웹 관리';

  @override
  String get logs => '로그';

  @override
  String get statusRunning => '실행 중';

  @override
  String get statusStopped => '중지됨';

  @override
  String get settings => '설정';

  @override
  String get address => '주소';

  @override
  String get port => '포트';

  @override
  String get save => '저장';

  @override
  String get notStarted => '서비스가 시작되지 않음';

  @override
  String get startHint => '먼저 대시보드에서 서비스를 시작하세요.';

  @override
  String get goToDashboard => '대시보드로 이동';

  @override
  String get back => '뒤로';

  @override
  String get forward => '앞으로';

  @override
  String get refresh => '새로고침';

  @override
  String get publicMode => '공용 모드';

  @override
  String get publicModeHintDesc => '활성화되면 서비스가 외부 액세스를 허용하고 주소 필드가 비활성화됩니다';

  @override
  String get publicModeApplying => '네트워크 모드 적용 중...';

  @override
  String get themeSelection => '테마';

  @override
  String get launchService => '서비스 시작';

  @override
  String get stopService => '서비스 중지';

  @override
  String get endpoint => '엔드포인트';

  @override
  String get statusActive => '활성';

  @override
  String get statusSyncing => '동기화 중';

  @override
  String get statusIdle => '유휴';

  @override
  String get publicModeEnabled => '공용 모드 활성화됨';

  @override
  String get localMode => '로컬 모드';

  @override
  String get unableToGetDeviceIp => '장치 IP를 가져올 수 없음';

  @override
  String get localModeHint =>
      '1. 서비스 구성으로 이동\n2. 공용 모드 켜기\n3. PocketClaw에 액세스하려면 QR 코드 스캔';

  @override
  String get publicModeHint => '1. 서비스 시작\n2. PocketClaw에 액세스하려면 QR 코드 스캔';

  @override
  String get noLogsToExport => '내보낼 로그 없음';

  @override
  String logsSavedToDownloads(Object path) {
    return '로그가 다운로드에 저장되었습니다: $path';
  }

  @override
  String get shareLogsText => 'PocketClaw 로그';

  @override
  String get workspaceDirectory => '작업 공간';

  @override
  String get legacyWorkspaceTitle => '이전 작업 공간을 찾았습니다';

  @override
  String legacyWorkspaceBody(Object path) {
    return '이전 버전은 작업 공간을 $path에 보관했습니다. 이제 PocketClaw는 앱 전용 저장소를 사용하며 해당 폴더는 그대로 두었습니다. 복사해 올 수 있습니다. 복사본은 별도 폴더에 저장되며, 아무것도 덮어쓰거나 삭제하지 않습니다.';
  }

  @override
  String get legacyWorkspaceImport => '작업 공간으로 복사';

  @override
  String get legacyWorkspaceHide => '숨기기';

  @override
  String legacyWorkspaceCopied(Object count, Object folder) {
    return '파일 $count개를 $folder에 복사했습니다.';
  }

  @override
  String legacyWorkspacePartial(Object count, Object folder, Object failed) {
    return '파일 $count개를 $folder에 복사했습니다. $failed개는 복사하지 못했습니다.';
  }

  @override
  String get legacyWorkspaceFailed => '폴더를 복사하지 못했습니다.';

  @override
  String get legacyWorkspaceEmpty => '선택한 폴더에 복사할 항목이 없었습니다.';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return '로그가 다운로드(Android 미디어 라이브러리)에 저장되었습니다: $name';
  }

  @override
  String shareFailed(Object error) {
    return '공유 대화 상자를 열지 못했습니다: $error';
  }

  @override
  String get exportLogs => '로그 내보내기';

  @override
  String logEventsCount(int count) {
    return '$count 이벤트';
  }

  @override
  String get cancel => '취소';

  @override
  String get language => '언어';

  @override
  String get selectLanguage => '언어 선택';

  @override
  String get about => '정보';

  @override
  String get aboutDescription => 'PocketClaw는 개인 AI 도우미 작업 공간입니다.';

  @override
  String get aboutAppVersionLabel => 'PocketClaw 버전';

  @override
  String get aboutCoreVersionLabel => '런타임 버전';

  @override
  String get aboutVersionUnavailable => '사용할 수 없음';

  @override
  String get close => '닫기';

  @override
  String get contextMemoryTitle => '텔레그램 컨텍스트 메모리';

  @override
  String get contextMemoryDescription =>
      'AI에 전달할 최근 대화 메시지 수를 조절합니다. 이전 대화는 텔레그램에 그대로 남으며 롤링 요약으로 표현됩니다.';

  @override
  String get contextMemoryHelp =>
      '메시지를 늘리면 최근 맥락이 풍부해지지만 토큰을 더 많이 사용합니다. 이전 메시지는 텔레그램에 남아 있으며 롤링 요약을 통해 유지될 수 있습니다.';

  @override
  String get contextMemoryRecommended => '권장';

  @override
  String get contextMemoryCustom => '사용자 지정';

  @override
  String get contextMemoryCustomLabel => '메시지';

  @override
  String contextMemoryRangeError(int min, int max) {
    return '$min에서 $max 사이의 정수를 입력하세요.';
  }

  @override
  String get contextMemorySaveFailed => '이 설정을 저장할 수 없습니다.';

  @override
  String get settingsSave => '저장';

  @override
  String get autoStartServiceTitle => 'PocketClaw 서비스 자동 시작';

  @override
  String get autoStartGatewayTitle => '게이트웨이 자동 시작';

  @override
  String get autoStartPreferenceOn => '자동 시작 설정: 켬';

  @override
  String get autoStartPreferenceOff => '자동 시작 설정: 끔';

  @override
  String get runtimeRunning => '런타임: 실행 중';

  @override
  String get runtimeStarting => '런타임: 시작 중';

  @override
  String get runtimeStopped => '런타임: 중지됨';

  @override
  String get gatewayAutoStartHint =>
      '다음에 PocketClaw 서비스를 시작할 때 적용됩니다. 게이트웨이 런타임은 대시보드에서 관리합니다.';

  @override
  String get manageTelegramConnection => 'Telegram 연결 관리';

  @override
  String get manageModelsTitle => '모델 관리';

  @override
  String get manageModelsDescription => 'AI 모델을 추가, 편집, 테스트하고 선택합니다.';

  @override
  String get githubChecking => '확인 중…';

  @override
  String get githubConnected => '연결됨';

  @override
  String get githubNotConnected => '연결되지 않음';

  @override
  String githubConnectedAs(String login) {
    return '$login 계정으로 연결됨';
  }

  @override
  String get githubDescription =>
      '번들로 제공되는 gh와 HTTPS를 통한 Git이 사용합니다. 토큰은 이 기기에서 암호화되며 다시 표시되지 않습니다.';

  @override
  String get githubTestConnection => '연결 테스트';

  @override
  String get githubDisconnect => '연결 해제';

  @override
  String get githubConnectAction => 'GitHub 연결';

  @override
  String get githubConnect => '연결';

  @override
  String get githubTokenLabel => '개인용 액세스 토큰';

  @override
  String get githubTokenHint =>
      '필요한 범위를 가진 GitHub 개인용 액세스 토큰을 붙여넣으세요(비공개 저장소는 repo).';

  @override
  String get githubTokenRejected => 'GitHub가 이 토큰을 거부했습니다.';

  @override
  String get githubAuthWorking => 'GitHub 인증이 정상 작동합니다.';

  @override
  String get githubAuthNotWorking => 'GitHub 인증이 작동하지 않습니다.';

  @override
  String githubAuthenticatedAs(String login) {
    return '$login 계정으로 인증되었습니다.';
  }

  @override
  String get githubCredentialRemoveFailed => '자격 증명을 제거할 수 없습니다.';

  @override
  String get githubYourAccount => '회원님의 GitHub 계정';

  @override
  String githubConnectedReport(String who, String outcome) {
    return '$who 계정으로 연결되었습니다. $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return '연결이 해제되었습니다. $outcome';
  }

  @override
  String get credentialAppliedNow => '이제 gh와 git이 사용할 수 있습니다.';

  @override
  String get credentialAppliesNextStart => '다음에 PocketClaw를 시작할 때 사용됩니다.';

  @override
  String get credentialAppliesDeferred =>
      '저장되었습니다. PocketClaw가 시작 중이므로 완료되는 즉시 자동으로 적용됩니다.';

  @override
  String get whatsNewTitle => '새로운 기능';

  @override
  String get whatsNewDescription => '이번 릴리스의 주요 변경 사항입니다.';

  @override
  String get whatsNewBadge => '새 기능';

  @override
  String get whatsNewSectionNew => '새로운 기능';

  @override
  String get whatsNewSectionImprovements => '개선 사항';

  @override
  String get whatsNewSectionFixes => '수정 사항';

  @override
  String get whatsNew020New1 =>
      '관리형 런타임이 PocketClaw 기본 제공 도구를 설치하고 최신 상태로 유지합니다.';

  @override
  String get whatsNew020New2 =>
      '기본 제공 개발 도구: Git, GitHub CLI, curl, ripgrep, jq, SQLite.';

  @override
  String get whatsNew020New3 => 'PocketClaw 기본 제공 Python 3.14 런타임을 지원합니다.';

  @override
  String get whatsNew020New4 =>
      '안전한 GitHub 로그인. 기본 제공 Git과 GitHub CLI가 함께 사용합니다.';

  @override
  String get whatsNew020New5 =>
      '원탭 Telegram 설정: PocketClaw가 앱에서든 브라우저의 대시보드에서든 당신만의 봇을 만들어 줍니다. 복사할 토큰이 없고, 당신의 계정만 봇과 대화할 수 있습니다.';

  @override
  String get whatsNew020New6 => '대시보드의 상태 보기: 진행 중인 작업, 채널, 사용 중인 모델, 런타임 리소스.';

  @override
  String get whatsNew020Improvement1 => '더 견고해진 공급자: 요청이 실패해도 대화 턴이 끝나지 않습니다.';

  @override
  String get whatsNew020Improvement2 => '더 정확해진 관리형 런타임 카탈로그.';

  @override
  String get whatsNew020Improvement3 =>
      '웹 화면과 기본 작업 공간 전반에서 더 일관된 PocketClaw 표기.';

  @override
  String get whatsNew020Improvement4 =>
      'PocketClaw의 Aperture 디자인을 적용한 새 앱 아이콘과 정보 화면.';

  @override
  String get whatsNew020Improvement5 =>
      '어시스턴트 답변이 더 자연스러워졌고, 끝에 고정된 서명이 붙지 않습니다.';

  @override
  String get whatsNew020Improvement6 =>
      '이제 전화 권한을 요청하지 않습니다. 앱의 어떤 기능도 사용하지 않았습니다.';

  @override
  String get whatsNew020Improvement7 =>
      '진단 로그를 다운로드 폴더가 아닌 앱 내부에 비공개로 저장합니다. 작업 공간의 위치는 그대로입니다.';

  @override
  String get whatsNew020Improvement8 =>
      'PocketClaw가 상태를 저장하고 서비스를 실행하는 방식을 크게 개선하여 일상적인 안정성이 높아졌습니다.';

  @override
  String get whatsNew020Improvement9 => '이번 업데이트 후 대시보드에서 한 번 다시 로그인해야 합니다.';

  @override
  String get whatsNew020Improvement10 => '이번 업데이트 후 웹 채팅 채널은 새 대화를 시작합니다.';

  @override
  String get whatsNew020Improvement11 => '이번 업데이트 후 알림 환경설정을 한 번 확인해 보시기 바랍니다.';

  @override
  String get whatsNew020Fix1 => '이전 실행이 남긴 오래된 프로세스 기록에서 게이트웨이가 복구됩니다.';

  @override
  String get whatsNew020Fix2 => '항목이 두 개 이상인 채널 목록이 저장 시 올바르게 유지됩니다.';

  @override
  String get whatsNew021Fix1 =>
      '번들로 제공되는 Git이 저장소를 복제하거나 브랜치를 업데이트할 때 더 이상 충돌하지 않습니다.';

  @override
  String get whatsNew022Improvement1 =>
      '긴 작업이 끝나면 Telegram이 답변을 새 메시지로 보내므로, 알림이 오고 그동안 보낸 메시지 아래에 표시됩니다.';

  @override
  String get whatsNew022Improvement2 =>
      '메시지가 대기열에 있을 때 Telegram이 이를 알리고 앞에 몇 개가 있는지 보여 줍니다.';

  @override
  String get whatsNew022Improvement3 =>
      '이제 작업 공간은 PocketClaw 전용 저장소에 있으며, 앱은 저장소 권한을 요청하지 않습니다. 이전 버전이 Download/pocketclaw에 남긴 작업 공간은 그대로 유지되며 설정에서 복사해 올 수 있습니다.';

  @override
  String get whatsNew022Fix1 => '매우 큰 도구 출력이 더 이상 대화를 넘치게 하지 않습니다.';

  @override
  String get whatsNew022Fix2 =>
      'Android에서 효과가 없던 설정 항목(기기, 로그인 시 실행, 서비스 포트)을 제거했습니다.';

  @override
  String get settingsGroupConnection => '연결';

  @override
  String get settingsGroupAgent => '에이전트';

  @override
  String get settingsGroupIntegrations => '연동';

  @override
  String get settingsGroupAppearance => '모양';

  @override
  String get statusTitle => '상태';

  @override
  String get statusSectionSystem => '시스템';

  @override
  String get statusSectionAi => 'AI';

  @override
  String get statusSectionActivity => '활동';

  @override
  String get statusSectionChannels => '채널';

  @override
  String get statusSectionResources => '리소스';

  @override
  String get statusSinceGatewayStart => '게이트웨이 시작 이후';

  @override
  String get statusGateway => '게이트웨이';

  @override
  String get statusUptime => '가동 시간';

  @override
  String get statusAppVersion => '앱 버전';

  @override
  String get statusCoreVersion => 'Core 버전';

  @override
  String get statusActiveModel => '활성 모델';

  @override
  String get statusConfiguredDefault => '구성된 기본값';

  @override
  String get statusProvider => '제공자';

  @override
  String get statusFallbacks => '대체 모델';

  @override
  String get statusActiveTurns => '활성 턴';

  @override
  String get statusActiveSubagents => '활성 서브에이전트';

  @override
  String get statusWaiting => '대기 중';

  @override
  String get statusCompleted => '완료';

  @override
  String get statusFailed => '실패';

  @override
  String get statusCancelled => '취소됨';

  @override
  String get statusToolCalls => '도구 호출';

  @override
  String get statusToolCallsFailed => '실패한 도구 호출';

  @override
  String get statusLastActivity => '마지막 활동';

  @override
  String get statusCoreMemory => 'Core 메모리';

  @override
  String get statusCoreCpuTime => 'Core CPU 시간';

  @override
  String get statusNoChannels => '구성된 채널 없음';

  @override
  String get statusDetailUnavailable => '상세 상태를 사용할 수 없습니다';

  @override
  String get statusJustNow => '방금';

  @override
  String statusMinutesAgo(int minutes) {
    return '$minutes분 전';
  }

  @override
  String statusHoursAgo(int hours) {
    return '$hours시간 전';
  }

  @override
  String statusDaysAgo(int days) {
    return '$days일 전';
  }

  @override
  String statusDurationSeconds(int seconds) {
    return '$seconds초';
  }

  @override
  String statusDurationMinutes(int minutes, int seconds) {
    return '$minutes분 $seconds초';
  }

  @override
  String statusDurationHours(int hours, int minutes) {
    return '$hours시간 $minutes분';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '$days일 $hours시간';
  }

  @override
  String get notificationPermissionTitle => '알림';

  @override
  String get notificationPermissionGranted =>
      'PocketClaw가 실행 중 알림을 표시할 수 있습니다.';

  @override
  String get notificationPermissionBlocked =>
      '알림이 꺼져 있어 PocketClaw 실행 중 알림이 표시되지 않습니다.';

  @override
  String get notificationPermissionOpenSettings => '알림 설정 열기';

  @override
  String get whatsNew020New7 => '대시보드에서 Telegram 관리: 봇 연결, 교체, 연결 해제.';

  @override
  String get whatsNew020New8 => '대시보드에서 공급자와 모델 관리: 공급자 추가, API 키 교체, 모델 삭제.';

  @override
  String get whatsNew020Improvement12 =>
      'Telegram 설정은 저장하는 즉시 적용되며 수동 재시작이 필요하지 않습니다.';

  @override
  String get whatsNew020Improvement13 =>
      '설정된 AI 모델이 없을 때 PocketClaw는 무엇이 빠졌는지 정확히 알려주고 찾아볼 수 있는 코드를 제시합니다.';

  @override
  String get whatsNew020Improvement14 => '대시보드에 로그인하면 원래 열려던 화면으로 이동합니다.';

  @override
  String get whatsNew020Improvement15 =>
      '진단 로그는 상세하게 유지되지만 키, 토큰, 메시지 내용은 절대 담지 않습니다.';

  @override
  String get whatsNew020Improvement16 =>
      '아직 아무도 소유하지 않은 대시보드는 공개 모드가 켜져 있어도 네트워크에서 접근할 수 없습니다.';
}

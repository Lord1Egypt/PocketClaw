// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Korean (`ko`).
class AppLocalizationsKo extends AppLocalizations {
  AppLocalizationsKo([String locale = 'ko']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => '실행';

  @override
  String get stop => '중지';

  @override
  String get config => '구성';

  @override
  String get webAdmin => '웹 관리';

  @override
  String get logs => '로그';

  @override
  String get viewLogs => '로그 보기';

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
  String get showWindow => '창 표시';

  @override
  String get exit => '종료';

  @override
  String get binaryPath => '바이너리 경로';

  @override
  String get browse => '찾아보기';

  @override
  String get pathError => '잘못된 경로';

  @override
  String get arguments => '인수';

  @override
  String get argumentsHint => '예: config.json';

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
  String get coreBinaryMissing =>
      '코어 바이너리를 찾을 수 없습니다. 플랫폼 바이너리를 app/bin/에 배치하거나 설정에서 경로를 지정하세요.';

  @override
  String get coreStartFailed => '코어 서비스 시작에 실패했습니다.';

  @override
  String get coreStopFailed => '코어 서비스 중지에 실패했습니다.';

  @override
  String get coreInvalidBinary => '잘못된 코어 바이너리 파일입니다.';

  @override
  String coreUnknownError(Object code) {
    return '알 수 없는 코어 오류: $code';
  }

  @override
  String get coreValid => '코어 바이너리가 유효합니다.';

  @override
  String get publicMode => '공용 모드';

  @override
  String get publicModeHintDesc => '활성화되면 서비스가 외부 액세스를 허용하고 주소 필드가 비활성화됩니다';

  @override
  String get publicModeApplying => '네트워크 모드 적용 중...';

  @override
  String get themeSelection => '테마';

  @override
  String get check => '확인';

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
  String get deviceReportingTitle => '장치 호환성 피드백';

  @override
  String get deviceReportingSubtitle =>
      'OS 버전 및 앱 버전 호환성 확인에만 사용됩니다. 채팅 메시지, 계정 세부 정보 또는 개인 콘텐츠는 관련되지 않습니다';

  @override
  String get deviceReportingConsentTitle => '장치 호환성 개선에 도움을 주세요';

  @override
  String get deviceReportingConsentDescription =>
      '활성화되면 호환성을 이해하기 위해 익명 설치 ID, OS 버전 및 앱 버전만 전송됩니다. 언어 및 지역은 Firebase Analytics에서 별도로 수집될 수 있습니다. 채팅 메시지, 입력된 콘텐츠, 계정 세부 정보, 파일 또는 사용자 정의 설정이 업로드되지 않습니다';

  @override
  String get deviceReportingBannerDescription =>
      '호환성 개선을 위해 익명 설치 ID, OS 버전 및 앱 버전만 동기화됩니다. 언어 및 지역은 Firebase Analytics에서 별도로 수집될 수 있습니다. 채팅 메시지, 계정 세부 정보, 파일 또는 개인 콘텐츠가 전송되지 않습니다';

  @override
  String get deviceReportingWhatWillBeSent => '이러한 장치 세부 정보만 포함됩니다';

  @override
  String get deviceReportingDeviceLabel => '장치 모델';

  @override
  String get deviceReportingPlatformLabel => '장치 범주';

  @override
  String get deviceReportingSystemLabel => 'OS 버전';

  @override
  String get deviceReportingTimingNote =>
      '활성화 시 한 번 동기화가 실행되고 시스템 업데이트가 감지된 후에만 다시 동기화됩니다';

  @override
  String get deviceReportingDeny => '나중에';

  @override
  String get deviceReportingAllow => '활성화';

  @override
  String get deviceReportingUploadSucceeded => '장치 호환성 피드백이 활성화되었습니다';

  @override
  String get deviceReportingUploadFailed =>
      '장치 호환성 피드백이 활성화되었지만 현재 장치 정보 동기화가 완료되지 않았습니다';

  @override
  String get deviceReportingDisabled => '장치 호환성 피드백이 비활성화되었습니다';

  @override
  String get localModeHint =>
      '1. 서비스 구성으로 이동\n2. 공용 모드 켜기\n3. PocketClaw에 액세스하려면 QR 코드 스캔';

  @override
  String get publicModeHint => '1. 서비스 시작\n2. PocketClaw에 액세스하려면 QR 코드 스캔';

  @override
  String get noLogsToExport => '내보낼 로그 없음';

  @override
  String get logsSavedToMediaLibrary => '로그가 다운로드(Android 미디어 라이브러리)에 저장되었습니다';

  @override
  String logsSavedToDownloads(Object path) {
    return '로그가 다운로드에 저장되었습니다: $path';
  }

  @override
  String get shareLogsText => 'PocketClaw 로그';

  @override
  String get workspaceDirectory => '작업 공간';

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
  String get unsavedChanges => '저장되지 않은 변경 사항';

  @override
  String get unsavedChangesHint => '저장되지 않은 변경 사항이 있습니다. 취소하시겠습니까?';

  @override
  String get cancel => '취소';

  @override
  String get discard => '취소';

  @override
  String get saved => '저장됨';

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
  String get whatsNew020New5 => 'Telegram 연동. 설정에서 구성할 수 있습니다.';

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
  String get whatsNew020Fix1 => '이전 실행이 남긴 오래된 프로세스 기록에서 게이트웨이가 복구됩니다.';

  @override
  String get whatsNew020Fix2 => '항목이 두 개 이상인 채널 목록이 저장 시 올바르게 유지됩니다.';

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
}

// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Japanese (`ja`).
class AppLocalizationsJa extends AppLocalizations {
  AppLocalizationsJa([String locale = 'ja']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => '実行';

  @override
  String get stop => '停止';

  @override
  String get config => '設定';

  @override
  String get webAdmin => 'Web管理';

  @override
  String get logs => 'ログ';

  @override
  String get viewLogs => 'ログを表示';

  @override
  String get statusRunning => '実行中';

  @override
  String get statusStopped => '停止済み';

  @override
  String get settings => '設定';

  @override
  String get address => 'アドレス';

  @override
  String get port => 'ポート';

  @override
  String get save => '保存';

  @override
  String get showWindow => 'ウィンドウを表示';

  @override
  String get exit => '終了';

  @override
  String get binaryPath => 'バイナリパス';

  @override
  String get browse => '参照';

  @override
  String get pathError => '無効なパス';

  @override
  String get arguments => '引数';

  @override
  String get argumentsHint => '例: config.json';

  @override
  String get notStarted => 'サービス未起動';

  @override
  String get startHint => 'まずダッシュボードからサービスを起動してください。';

  @override
  String get goToDashboard => 'ダッシュボードへ';

  @override
  String get back => '戻る';

  @override
  String get forward => '進む';

  @override
  String get refresh => '更新';

  @override
  String get coreBinaryMissing =>
      'コアバイナリが見つかりません。プラットフォームバイナリをapp/bin/に配置するか、設定でパスを指定してください。';

  @override
  String get coreStartFailed => 'コアサービスの起動に失敗しました。';

  @override
  String get coreStopFailed => 'コアサービスの停止に失敗しました。';

  @override
  String get coreInvalidBinary => '無効なコアバイナリファイルです。';

  @override
  String coreUnknownError(Object code) {
    return '不明なコアエラー: $code';
  }

  @override
  String get coreValid => 'コアバイナリは有効です。';

  @override
  String get publicMode => 'パブリックモード';

  @override
  String get publicModeHintDesc => '有効にすると、サービスは外部アクセスを許可し、アドレスフィールドは無効になります';

  @override
  String get publicModeApplying => 'ネットワークモードを適用中...';

  @override
  String get themeSelection => 'テーマ';

  @override
  String get check => '確認';

  @override
  String get launchService => 'サービスを起動';

  @override
  String get stopService => 'サービスを停止';

  @override
  String get endpoint => 'エンドポイント';

  @override
  String get statusActive => 'アクティブ';

  @override
  String get statusSyncing => '同期中';

  @override
  String get statusIdle => 'アイドル';

  @override
  String get publicModeEnabled => 'パブリックモード有効';

  @override
  String get localMode => 'ローカルモード';

  @override
  String get unableToGetDeviceIp => 'デバイスIPを取得できません';

  @override
  String get deviceReportingTitle => 'デバイス互換性フィードバック';

  @override
  String get deviceReportingSubtitle =>
      'OSバージョンとアプリバージョンの互換性確認のみに使用されます。チャットメッセージ、アカウント詳細、個人コンテンツは関与しません';

  @override
  String get deviceReportingConsentTitle => 'デバイス互換性の向上にご協力ください';

  @override
  String get deviceReportingConsentDescription =>
      '有効にすると、互換性を把握するために匿名インストールID、OSバージョン、アプリバージョンのみが送信されます。言語と地域情報はFirebase Analyticsによって個別に収集される場合があります。チャットメッセージ、入力コンテンツ、アカウント詳細、ファイル、カスタム設定はアップロードされません';

  @override
  String get deviceReportingBannerDescription =>
      '互換性向上のため、匿名インストールID、OSバージョン、アプリバージョンのみが同期されます。言語と地域情報はFirebase Analyticsによって個別に収集される場合があります。チャットメッセージ、アカウント詳細、ファイル、個人コンテンツは送信されません';

  @override
  String get deviceReportingWhatWillBeSent => 'これらのデバイス詳細のみが含まれます';

  @override
  String get deviceReportingDeviceLabel => 'デバイスモデル';

  @override
  String get deviceReportingPlatformLabel => 'デバイスカテゴリ';

  @override
  String get deviceReportingSystemLabel => 'OSバージョン';

  @override
  String get deviceReportingTimingNote => '有効にすると1回同期され、システム更新が検出された后再同期されます';

  @override
  String get deviceReportingDeny => '後で';

  @override
  String get deviceReportingAllow => '有効にする';

  @override
  String get deviceReportingUploadSucceeded => 'デバイス互換性フィードバックが有効になりました';

  @override
  String get deviceReportingUploadFailed =>
      'デバイス互換性フィードバックが有効ですが、現在のデバイス情報同期が完了しませんでした';

  @override
  String get deviceReportingDisabled => 'デバイス互換性フィードバックが無効になりました';

  @override
  String get localModeHint =>
      '1. サービス設定に移動\n2. パブリックモードをオン\n3. QRコードをスキャンしてPocketClawにアクセス';

  @override
  String get publicModeHint => '1. サービスを起動\n2. QRコードをスキャンしてPocketClawにアクセス';

  @override
  String get noLogsToExport => 'エクスポートするログがありません';

  @override
  String get logsSavedToMediaLibrary => 'ログがダウンロード（Androidメディアライブラリ）に保存されました';

  @override
  String logsSavedToDownloads(Object path) {
    return 'ログがダウンロードに保存されました: $path';
  }

  @override
  String get shareLogsText => 'PocketClawログ';

  @override
  String get workspaceDirectory => 'ワークスペース';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'ログがダウンロード（Androidメディアライブラリ）に保存されました: $name';
  }

  @override
  String shareFailed(Object error) {
    return '共有ダイアログを開けませんでした: $error';
  }

  @override
  String get exportLogs => 'ログをエクスポート';

  @override
  String logEventsCount(int count) {
    return '$count イベント';
  }

  @override
  String get unsavedChanges => '未保存の変更';

  @override
  String get unsavedChangesHint => '未保存の変更があります。破棄しますか？';

  @override
  String get cancel => 'キャンセル';

  @override
  String get discard => '破棄';

  @override
  String get saved => '保存しました';

  @override
  String get language => '言語';

  @override
  String get selectLanguage => '言語を選択';

  @override
  String get about => '概要';

  @override
  String get aboutDescription => 'PocketClaw は、プライベートな AI アシスタントのワークスペースです。';

  @override
  String get aboutAppVersionLabel => 'PocketClaw バージョン';

  @override
  String get aboutCoreVersionLabel => 'ランタイムのバージョン';

  @override
  String get aboutVersionUnavailable => '利用できません';

  @override
  String get close => '閉じる';

  @override
  String get contextMemoryTitle => 'Telegram コンテキストメモリ';

  @override
  String get contextMemoryDescription =>
      'AI に送信する直近の会話メッセージ数を設定します。それ以前の会話は Telegram に残り、ローリング要約で表現されます。';

  @override
  String get contextMemoryHelp =>
      'メッセージを増やすと直近の文脈は充実しますが、トークン消費も増えます。古いメッセージは Telegram に残り、ローリング要約を通じて保持される場合があります。';

  @override
  String get contextMemoryRecommended => '推奨';

  @override
  String get contextMemoryCustom => 'カスタム';

  @override
  String get contextMemoryCustomLabel => 'メッセージ数';

  @override
  String contextMemoryRangeError(int min, int max) {
    return '$min から $max までの整数を入力してください。';
  }

  @override
  String get contextMemorySaveFailed => 'この設定を保存できませんでした。';

  @override
  String get settingsSave => '保存';

  @override
  String get autoStartServiceTitle => 'PocketClaw サービスを自動的に起動';

  @override
  String get autoStartGatewayTitle => 'ゲートウェイを自動的に起動';

  @override
  String get autoStartPreferenceOn => '自動起動の設定: オン';

  @override
  String get autoStartPreferenceOff => '自動起動の設定: オフ';

  @override
  String get runtimeRunning => '実行状態: 実行中';

  @override
  String get runtimeStarting => '実行状態: 起動中';

  @override
  String get runtimeStopped => '実行状態: 停止';

  @override
  String get gatewayAutoStartHint =>
      '次に PocketClaw サービスを起動したときに適用されます。ゲートウェイの実行状態はダッシュボードで管理します。';

  @override
  String get manageTelegramConnection => 'Telegram 接続を管理';

  @override
  String get manageModelsTitle => 'モデルを管理';

  @override
  String get manageModelsDescription => 'AI モデルの追加・編集・テスト・選択ができます。';

  @override
  String get githubChecking => '確認中…';

  @override
  String get githubConnected => '接続済み';

  @override
  String get githubNotConnected => '未接続';

  @override
  String githubConnectedAs(String login) {
    return '$login として接続済み';
  }

  @override
  String get githubDescription =>
      '同梱の gh と HTTPS 経由の Git が使用します。トークンはこの端末で暗号化され、再表示されることはありません。';

  @override
  String get githubTestConnection => '接続をテスト';

  @override
  String get githubDisconnect => '切断';

  @override
  String get githubConnectAction => 'GitHub に接続';

  @override
  String get githubConnect => '接続';

  @override
  String get githubTokenLabel => '個人用アクセストークン';

  @override
  String get githubTokenHint =>
      '必要なスコープを持つ GitHub の個人用アクセストークンを貼り付けてください（プライベートリポジトリには repo）。';

  @override
  String get githubTokenRejected => 'GitHub はこのトークンを受け付けませんでした。';

  @override
  String get githubAuthWorking => 'GitHub の認証は正常です。';

  @override
  String get githubAuthNotWorking => 'GitHub の認証が機能していません。';

  @override
  String githubAuthenticatedAs(String login) {
    return '$login として認証されました。';
  }

  @override
  String get githubCredentialRemoveFailed => '資格情報を削除できませんでした。';

  @override
  String get githubYourAccount => 'お使いの GitHub アカウント';

  @override
  String githubConnectedReport(String who, String outcome) {
    return '$who として接続しました。$outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return '切断しました。$outcome';
  }

  @override
  String get credentialAppliedNow => 'gh と git がすぐに使用できます。';

  @override
  String get credentialAppliesNextStart => '次に PocketClaw を起動したときに使用されます。';

  @override
  String get credentialAppliesDeferred =>
      '保存しました。PocketClaw が起動中のため、完了次第自動的に適用されます。';

  @override
  String get whatsNewTitle => '新機能';

  @override
  String get whatsNewDescription => 'このリリースの主な変更点です。';

  @override
  String get whatsNewBadge => '新着';

  @override
  String get whatsNewSectionNew => '新機能';

  @override
  String get whatsNewSectionImprovements => '改善';

  @override
  String get whatsNewSectionFixes => '修正';

  @override
  String get whatsNew020New1 => 'マネージドランタイムが PocketClaw 同梱ツールの導入と更新を自動で行います。';

  @override
  String get whatsNew020New2 =>
      '同梱の開発ツール: Git、GitHub CLI、curl、ripgrep、jq、SQLite。';

  @override
  String get whatsNew020New3 => 'PocketClaw 同梱の Python 3.14 ランタイムに対応しました。';

  @override
  String get whatsNew020New4 =>
      '安全な GitHub サインイン。同梱の Git と GitHub CLI が共通で利用します。';

  @override
  String get whatsNew020New5 => 'Telegram 連携。設定から構成できます。';

  @override
  String get whatsNew020New6 =>
      'ダッシュボードのステータス表示。実行中の処理、チャネル、使用中のモデル、実行リソースを確認できます。';

  @override
  String get whatsNew020Improvement1 => 'プロバイダーの耐障害性を向上。リクエストが失敗してもターンは終了しません。';

  @override
  String get whatsNew020Improvement2 => 'マネージドランタイムのカタログをより正確にしました。';

  @override
  String get whatsNew020Improvement3 =>
      'ウェブ画面と既定のワークスペースで PocketClaw の表記を統一しました。';

  @override
  String get whatsNew020Improvement4 =>
      'PocketClaw の Aperture デザインに合わせた新しいアプリアイコンと「情報」画面。';

  @override
  String get whatsNew020Improvement5 => 'アシスタントの返信がより自然になり、末尾の固定の署名がなくなりました。';

  @override
  String get whatsNew020Improvement6 =>
      '電話の権限を要求しなくなりました。アプリのどの機能も使用していませんでした。';

  @override
  String get whatsNew020Improvement7 =>
      '診断ログをダウンロードフォルダーではなくアプリ内に非公開で保存するようになりました。ワークスペースの場所は変わりません。';

  @override
  String get whatsNew020Fix1 => '以前の実行が残した古いプロセス記録からゲートウェイが復帰するようになりました。';

  @override
  String get whatsNew020Fix2 => '複数の項目を持つチャンネル一覧が保存時に正しく保持されるようになりました。';

  @override
  String get settingsGroupConnection => '接続';

  @override
  String get settingsGroupAgent => 'エージェント';

  @override
  String get settingsGroupIntegrations => '連携';

  @override
  String get settingsGroupAppearance => '外観';

  @override
  String get statusTitle => 'ステータス';

  @override
  String get statusSectionSystem => 'システム';

  @override
  String get statusSectionAi => 'AI';

  @override
  String get statusSectionActivity => 'アクティビティ';

  @override
  String get statusSectionChannels => 'チャンネル';

  @override
  String get statusSectionResources => 'リソース';

  @override
  String get statusSinceGatewayStart => 'ゲートウェイ起動以降';

  @override
  String get statusGateway => 'ゲートウェイ';

  @override
  String get statusUptime => '稼働時間';

  @override
  String get statusAppVersion => 'アプリのバージョン';

  @override
  String get statusCoreVersion => 'Core のバージョン';

  @override
  String get statusActiveModel => '使用中のモデル';

  @override
  String get statusConfiguredDefault => '設定されたデフォルト';

  @override
  String get statusProvider => 'プロバイダー';

  @override
  String get statusFallbacks => 'フォールバック';

  @override
  String get statusActiveTurns => '実行中のターン';

  @override
  String get statusActiveSubagents => '実行中のサブエージェント';

  @override
  String get statusWaiting => '待機中';

  @override
  String get statusCompleted => '完了';

  @override
  String get statusFailed => '失敗';

  @override
  String get statusCancelled => 'キャンセル';

  @override
  String get statusToolCalls => 'ツール呼び出し';

  @override
  String get statusToolCallsFailed => '失敗したツール呼び出し';

  @override
  String get statusLastActivity => '最終アクティビティ';

  @override
  String get statusCoreMemory => 'Core のメモリ';

  @override
  String get statusCoreCpuTime => 'Core の CPU 時間';

  @override
  String get statusNoChannels => '設定されたチャンネルはありません';

  @override
  String get statusDetailUnavailable => '詳細ステータスは利用できません';

  @override
  String get statusJustNow => 'たった今';

  @override
  String statusMinutesAgo(int minutes) {
    return '$minutes 分前';
  }

  @override
  String statusHoursAgo(int hours) {
    return '$hours 時間前';
  }

  @override
  String statusDaysAgo(int days) {
    return '$days 日前';
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
    return '$hours 時間 $minutes 分';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '$days 日 $hours 時間';
  }
}

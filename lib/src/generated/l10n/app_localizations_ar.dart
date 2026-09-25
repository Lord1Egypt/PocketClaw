// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Arabic (`ar`).
class AppLocalizationsAr extends AppLocalizations {
  AppLocalizationsAr([String locale = 'ar']) : super(locale);

  @override
  String get stop => 'إيقاف';

  @override
  String get webAdmin => 'إدارة الويب';

  @override
  String get logs => 'السجلات';

  @override
  String get statusRunning => 'قيد التشغيل';

  @override
  String get statusStopped => 'متوقف';

  @override
  String get settings => 'الإعدادات';

  @override
  String get address => 'العنوان';

  @override
  String get port => 'المنفذ';

  @override
  String get save => 'حفظ';

  @override
  String get notStarted => 'الخدمة لم تبدأ بعد';

  @override
  String get startHint => 'يرجى بدء الخدمة من لوحة التحكم أولاً.';

  @override
  String get goToDashboard => 'الذهاب إلى لوحة التحكم';

  @override
  String get back => 'رجوع';

  @override
  String get forward => 'تقديم';

  @override
  String get refresh => 'تحديث';

  @override
  String get publicMode => 'الوضع العام';

  @override
  String get publicModeHintDesc =>
      'عند التمكين، تسمح الخدمة بالوصول الخارجي وسيتم تعطيل حقل العنوان';

  @override
  String get publicModeApplying => 'جارٍ تطبيق وضع الشبكة...';

  @override
  String get themeSelection => 'السمة';

  @override
  String get launchService => 'تشغيل الخدمة';

  @override
  String get stopService => 'إيقاف الخدمة';

  @override
  String get endpoint => 'نقطة الوصول';

  @override
  String get statusActive => 'نشط';

  @override
  String get statusSyncing => 'جارٍ المزامنة';

  @override
  String get statusIdle => 'خامل';

  @override
  String get publicModeEnabled => 'الوضع العام مفعل';

  @override
  String get localMode => 'الوضع المحلي';

  @override
  String get unableToGetDeviceIp => 'تعذر الحصول على عنوان IP للجهاز';

  @override
  String get localModeHint =>
      '1. انتقل إلى تكوين الخدمة\n2. شغّل الوضع العام\n3. امسح رمز QR للوصول إلى PocketClaw';

  @override
  String get publicModeHint =>
      '1. ابدأ الخدمة\n2. امسح رمز QR للوصول إلى PocketClaw';

  @override
  String get noLogsToExport => 'لا توجد سجلات للتصدير';

  @override
  String logsSavedToDownloads(Object path) {
    return 'تم حفظ السجلات في التنزيلات: $path';
  }

  @override
  String get shareLogsText => 'سجلات PocketClaw';

  @override
  String get workspaceDirectory => 'مساحة العمل';

  @override
  String get legacyWorkspaceTitle => 'تم العثور على مساحة عمل سابقة';

  @override
  String legacyWorkspaceBody(Object path) {
    return 'كان إصدار أقدم يحفظ مساحة عملك في $path. يستخدم PocketClaw الآن مساحة التخزين الخاصة بالتطبيق وترك ذلك المجلد كما هو. يمكنك نسخه إلى مساحة العمل: تُوضع النسخة في مجلد مستقل، ولا يُستبدل أو يُحذف أي شيء.';
  }

  @override
  String get legacyWorkspaceImport => 'نسخ إلى مساحة العمل';

  @override
  String get legacyWorkspaceHide => 'إخفاء';

  @override
  String legacyWorkspaceCopied(Object count, Object folder) {
    return 'تم نسخ $count ملفًا إلى $folder.';
  }

  @override
  String legacyWorkspacePartial(Object count, Object folder, Object failed) {
    return 'تم نسخ $count ملفًا إلى $folder. تعذّر نسخ $failed.';
  }

  @override
  String get legacyWorkspaceFailed => 'تعذّر نسخ المجلد.';

  @override
  String get legacyWorkspaceEmpty => 'لم يكن في المجلد المحدد ما يمكن نسخه.';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'تم حفظ السجلات في التنزيلات (مكتبة وسائط Android): $name';
  }

  @override
  String shareFailed(Object error) {
    return 'فشل في فتح حوار المشاركة: $error';
  }

  @override
  String get exportLogs => 'تصدير السجلات';

  @override
  String logEventsCount(int count) {
    return '$count أحداث';
  }

  @override
  String get cancel => 'إلغاء';

  @override
  String get language => 'اللغة';

  @override
  String get selectLanguage => 'اختر اللغة';

  @override
  String get about => 'حول';

  @override
  String get aboutDescription =>
      'PocketClaw هو مساحة عملك الخاصة للمساعد الذكي.';

  @override
  String get aboutAppVersionLabel => 'إصدار PocketClaw';

  @override
  String get aboutCoreVersionLabel => 'إصدار وقت التشغيل';

  @override
  String get aboutVersionUnavailable => 'غير متوفر';

  @override
  String get close => 'إغلاق';

  @override
  String get contextMemoryTitle => 'ذاكرة سياق تيليجرام';

  @override
  String get contextMemoryDescription =>
      'تتحكم في عدد رسائل المحادثة الأخيرة المُرسلة إلى الذكاء الاصطناعي. تبقى المحادثات الأقدم في تيليجرام ويُمثّلها الملخّص المتجدّد.';

  @override
  String get contextMemoryHelp =>
      'عدد أكبر من الرسائل يمنح سياقًا أحدث لكنه يستهلك رموزًا أكثر. تبقى الرسائل الأقدم في تيليجرام وقد يُحتفظ بها عبر الملخّص المتجدّد.';

  @override
  String get contextMemoryRecommended => 'موصى به';

  @override
  String get contextMemoryCustom => 'مخصص';

  @override
  String get contextMemoryCustomLabel => 'الرسائل';

  @override
  String contextMemoryRangeError(int min, int max) {
    return 'أدخل رقمًا صحيحًا بين $min و$max.';
  }

  @override
  String get contextMemorySaveFailed => 'تعذّر حفظ هذا الإعداد.';

  @override
  String get settingsSave => 'حفظ';

  @override
  String get autoStartServiceTitle => 'تشغيل خدمة PocketClaw تلقائيًا';

  @override
  String get autoStartGatewayTitle => 'تشغيل البوابة تلقائيًا';

  @override
  String get autoStartPreferenceOn => 'تفضيل التشغيل التلقائي: مفعّل';

  @override
  String get autoStartPreferenceOff => 'تفضيل التشغيل التلقائي: معطّل';

  @override
  String get runtimeRunning => 'حالة التشغيل: قيد التشغيل';

  @override
  String get runtimeStarting => 'حالة التشغيل: قيد البدء';

  @override
  String get runtimeStopped => 'حالة التشغيل: متوقّف';

  @override
  String get gatewayAutoStartHint =>
      'يُطبَّق عند تشغيل خدمة PocketClaw في المرة القادمة. تُدار حالة تشغيل البوابة من لوحة التحكم.';

  @override
  String get manageTelegramConnection => 'إدارة اتصال تيليجرام';

  @override
  String get manageModelsTitle => 'إدارة الموديلات';

  @override
  String get manageModelsDescription =>
      'إضافة الموديلات وتعديلها واختبارها واختيار الموديل الافتراضي.';

  @override
  String get githubChecking => 'جارٍ التحقق…';

  @override
  String get githubConnected => 'متصل';

  @override
  String get githubNotConnected => 'غير متصل';

  @override
  String githubConnectedAs(String login) {
    return 'متصل باسم $login';
  }

  @override
  String get githubDescription =>
      'يُستخدم بواسطة gh المضمّن وبواسطة Git عبر HTTPS. يُشفَّر الرمز على هذا الجهاز ولا يُعرض مرة أخرى.';

  @override
  String get githubTestConnection => 'اختبار الاتصال';

  @override
  String get githubDisconnect => 'قطع الاتصال';

  @override
  String get githubConnectAction => 'ربط GitHub';

  @override
  String get githubConnect => 'ربط';

  @override
  String get githubTokenLabel => 'رمز وصول شخصي';

  @override
  String get githubTokenHint =>
      'ألصق رمز وصول شخصي من GitHub بالصلاحيات التي تحتاجها (repo للمستودعات الخاصة).';

  @override
  String get githubTokenRejected => 'لم يقبل GitHub هذا الرمز.';

  @override
  String get githubAuthWorking => 'مصادقة GitHub تعمل.';

  @override
  String get githubAuthNotWorking => 'مصادقة GitHub لا تعمل.';

  @override
  String githubAuthenticatedAs(String login) {
    return 'تمت المصادقة باسم $login.';
  }

  @override
  String get githubCredentialRemoveFailed => 'تعذّر إزالة بيانات الاعتماد.';

  @override
  String get githubYourAccount => 'حساب GitHub الخاص بك';

  @override
  String githubConnectedReport(String who, String outcome) {
    return 'تم الاتصال باسم $who. $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return 'تم قطع الاتصال. $outcome';
  }

  @override
  String get credentialAppliedNow => 'يمكن لـ gh و git استخدامه الآن.';

  @override
  String get credentialAppliesNextStart =>
      'سيُستخدم عند تشغيل PocketClaw في المرة القادمة.';

  @override
  String get credentialAppliesDeferred =>
      'تم الحفظ. PocketClaw مشغول بالبدء، لذا سيُطبَّق تلقائيًا فور انتهاء ذلك.';

  @override
  String get whatsNewTitle => 'ما الجديد';

  @override
  String get whatsNewDescription => 'أبرز التغييرات في هذا الإصدار.';

  @override
  String get whatsNewBadge => 'جديد';

  @override
  String get whatsNewSectionNew => 'جديد';

  @override
  String get whatsNewSectionImprovements => 'تحسينات';

  @override
  String get whatsNewSectionFixes => 'إصلاحات';

  @override
  String get whatsNew020New1 =>
      'يقوم وقت التشغيل المُدار بتثبيت أدوات PocketClaw المضمّنة وتحديثها نيابةً عنك.';

  @override
  String get whatsNew020New2 =>
      'أدوات تطوير مضمّنة: Git وGitHub CLI وcurl وripgrep وjq وSQLite.';

  @override
  String get whatsNew020New3 =>
      'دعم وقت تشغيل Python 3.14 المضمّن في PocketClaw.';

  @override
  String get whatsNew020New4 =>
      'تسجيل دخول آمن إلى GitHub، يشترك فيه Git وGitHub CLI المضمّنان.';

  @override
  String get whatsNew020New5 =>
      'إعداد تيليجرام بلمسة واحدة: ينشئ PocketClaw بوتك الخاص، من التطبيق أو من لوحة التحكم في المتصفح. لا يوجد رمز لنسخه، ولا يمكن التحدث إليه إلا من حسابك.';

  @override
  String get whatsNew020New6 =>
      'عرض الحالة في لوحة التحكم: الأعمال الجارية والقنوات والنموذج المستخدم وموارد التشغيل.';

  @override
  String get whatsNew020Improvement1 =>
      'مزوّدون أكثر مرونة: لم يعد فشل الطلب ينهي الدور.';

  @override
  String get whatsNew020Improvement2 => 'فهرس أدق لوقت التشغيل المُدار.';

  @override
  String get whatsNew020Improvement3 =>
      'هوية PocketClaw أوضح في واجهة الويب ومساحة العمل الافتراضية.';

  @override
  String get whatsNew020Improvement4 =>
      'أيقونة تطبيق جديدة وشاشة «حول» محدّثة بتصميم Aperture من PocketClaw.';

  @override
  String get whatsNew020Improvement5 =>
      'ردود المساعد أصبحت أكثر طبيعية، بلا توقيع ثابت في نهايتها.';

  @override
  String get whatsNew020Improvement6 =>
      'لم يعد PocketClaw يطلب إذن الهاتف — لم يكن أي جزء من التطبيق يستخدمه.';

  @override
  String get whatsNew020Improvement7 =>
      'أصبحت سجلات التشخيص تُحفظ داخل التطبيق بشكل خاص بدلاً من مجلد التنزيلات. مساحة عملك تبقى في مكانها.';

  @override
  String get whatsNew020Improvement8 =>
      'ترقية كبيرة لطريقة حفظ PocketClaw لحالته وتشغيله لخدماته، من أجل موثوقية أثبت في الاستخدام اليومي.';

  @override
  String get whatsNew020Improvement9 =>
      'بعد هذا التحديث، ستطلب منك لوحة التحكم تسجيل الدخول مرة أخرى.';

  @override
  String get whatsNew020Improvement10 =>
      'بعد هذا التحديث، تبدأ قناة محادثة الويب محادثة جديدة.';

  @override
  String get whatsNew020Improvement11 =>
      'بعد هذا التحديث، من المفيد مراجعة تفضيلات الإشعارات مرة واحدة.';

  @override
  String get whatsNew020Fix1 =>
      'أصبحت البوابة تتعافى من سجل عملية قديم خلّفه تشغيل سابق.';

  @override
  String get whatsNew020Fix2 =>
      'يتم الآن حفظ قوائم القنوات التي تحتوي أكثر من عنصر واحد بشكل صحيح.';

  @override
  String get whatsNew021Fix1 =>
      'لم يعد Git المضمّن ينهار أثناء استنساخ مستودع أو تحديث فرع.';

  @override
  String get whatsNew022Improvement1 =>
      'بعد مهمة طويلة، يرسل Telegram الإجابة في رسالة جديدة، فيصلك إشعار وتظهر أسفل أي رسالة أرسلتها في أثناء ذلك.';

  @override
  String get whatsNew022Improvement2 =>
      'يخبرك Telegram عندما تكون رسالتك في قائمة الانتظار وكم رسالة قبلها.';

  @override
  String get whatsNew022Improvement3 =>
      'أصبحت مساحة العمل الآن في مساحة التخزين الخاصة بـ PocketClaw، ولا يطلب التطبيق أي إذن تخزين. مساحة العمل التي تركها إصدار أقدم في Download⁠/⁠pocketclaw تبقى كما هي، ويمكن نسخها من الإعدادات.';

  @override
  String get whatsNew022Fix1 =>
      'لم تعد مخرجات الأدوات الكبيرة جدًا تتجاوز سعة المحادثة.';

  @override
  String get whatsNew022Fix2 =>
      'أُزيلت عناصر من الإعدادات لم يكن لها أي أثر على Android: الأجهزة، والتشغيل عند تسجيل الدخول، ومنفذ الخدمة.';

  @override
  String get whatsNew022Improvement4 =>
      'يتطلب PocketClaw الآن Android 8.0 أو أحدث.';

  @override
  String get whatsNew022Fix3 =>
      'إصلاح عطل عند بدء التشغيل على Android كان قد يظهر بعد إعادة تشغيل الهاتف.';

  @override
  String get whatsNew023Improvement1 =>
      'تحديث أمني: يُبنى المحرّك الخلفي لـ PocketClaw الآن بإصدار حديث ومدعوم من Go وبمكتبات محدّثة تعالج ثغرات معروفة.';

  @override
  String get whatsNew023Improvement2 =>
      'حُدِّثت أداة GitHub لسطر الأوامر المضمّنة إلى الإصدار 2.101.';

  @override
  String get whatsNew023Improvement3 =>
      'تعزيز عملية البناء استعدادًا لـ F-Droid: يُبنى التطبيق وأدواته المضمّنة بالكامل من الشيفرة المصدرية في بيئة ثابتة وقابلة للتكرار.';

  @override
  String get settingsGroupConnection => 'الاتصال';

  @override
  String get settingsGroupAgent => 'الوكيل';

  @override
  String get settingsGroupIntegrations => 'التكاملات';

  @override
  String get settingsGroupAppearance => 'المظهر';

  @override
  String get statusTitle => 'الحالة';

  @override
  String get statusSectionSystem => 'النظام';

  @override
  String get statusSectionAi => 'الذكاء الاصطناعي';

  @override
  String get statusSectionActivity => 'النشاط';

  @override
  String get statusSectionChannels => 'القنوات';

  @override
  String get statusSectionResources => 'الموارد';

  @override
  String get statusSinceGatewayStart => 'منذ بدء البوابة';

  @override
  String get statusGateway => 'البوابة';

  @override
  String get statusUptime => 'مدة التشغيل';

  @override
  String get statusAppVersion => 'إصدار التطبيق';

  @override
  String get statusCoreVersion => 'إصدار النواة';

  @override
  String get statusActiveModel => 'النموذج النشط';

  @override
  String get statusConfiguredDefault => 'الافتراضي المهيّأ';

  @override
  String get statusProvider => 'المزوّد';

  @override
  String get statusFallbacks => 'البدائل';

  @override
  String get statusActiveTurns => 'الأدوار النشطة';

  @override
  String get statusActiveSubagents => 'الوكلاء الفرعيون النشطون';

  @override
  String get statusWaiting => 'في الانتظار';

  @override
  String get statusCompleted => 'مكتملة';

  @override
  String get statusFailed => 'فاشلة';

  @override
  String get statusCancelled => 'مُلغاة';

  @override
  String get statusToolCalls => 'استدعاءات الأدوات';

  @override
  String get statusToolCallsFailed => 'استدعاءات أدوات فاشلة';

  @override
  String get statusLastActivity => 'آخر نشاط';

  @override
  String get statusCoreMemory => 'ذاكرة النواة';

  @override
  String get statusCoreCpuTime => 'زمن معالج النواة';

  @override
  String get statusNoChannels => 'لا توجد قنوات مهيّأة';

  @override
  String get statusDetailUnavailable => 'الحالة التفصيلية غير متاحة';

  @override
  String get statusJustNow => 'الآن';

  @override
  String statusMinutesAgo(int minutes) {
    return 'قبل $minutes د';
  }

  @override
  String statusHoursAgo(int hours) {
    return 'قبل $hours س';
  }

  @override
  String statusDaysAgo(int days) {
    return 'قبل $days ي';
  }

  @override
  String statusDurationSeconds(int seconds) {
    return '$seconds ث';
  }

  @override
  String statusDurationMinutes(int minutes, int seconds) {
    return '$minutes د $seconds ث';
  }

  @override
  String statusDurationHours(int hours, int minutes) {
    return '$hours س $minutes د';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '$days ي $hours س';
  }

  @override
  String get notificationPermissionTitle => 'الإشعارات';

  @override
  String get notificationPermissionGranted =>
      'يمكن لـ PocketClaw إظهار إشعار التشغيل.';

  @override
  String get notificationPermissionBlocked =>
      'الإشعارات مُعطَّلة، لذا لن يظهر إشعار تشغيل PocketClaw.';

  @override
  String get notificationPermissionOpenSettings => 'فتح إعدادات الإشعارات';

  @override
  String get whatsNew020New7 =>
      'إدارة تيليجرام من لوحة التحكم: ربط بوت أو استبداله أو قطع اتصاله.';

  @override
  String get whatsNew020New8 =>
      'إدارة المزودين والطُرز في لوحة التحكم: إضافة مزود أو تغيير مفتاح API أو إزالة طراز.';

  @override
  String get whatsNew020Improvement12 =>
      'تُطبَّق إعدادات تيليجرام بمجرد حفظها، بدون إعادة تشغيل يدوية.';

  @override
  String get whatsNew020Improvement13 =>
      'عندما لا يكون هناك طراز ذكاء اصطناعي مُعد، يوضّح PocketClaw ما ينقص بالتحديد ويعطيك رمزًا للرجوع إليه.';

  @override
  String get whatsNew020Improvement14 =>
      'تسجيل الدخول إلى لوحة التحكم ينقلك إلى الشاشة التي طلبتها.';

  @override
  String get whatsNew020Improvement15 =>
      'تبقى سجلات التشخيص مفصّلة دون أن تحتوي أبدًا على مفاتيحك أو رموزك أو نص رسائلك.';

  @override
  String get whatsNew020Improvement16 =>
      'لوحة التحكم التي لم يطالب بها أحد بعد لا يمكن الوصول إليها من الشبكة أبدًا، حتى مع تشغيل الوضع العام.';
}

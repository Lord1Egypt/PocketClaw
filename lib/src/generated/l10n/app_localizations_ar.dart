// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Arabic (`ar`).
class AppLocalizationsAr extends AppLocalizations {
  AppLocalizationsAr([String locale = 'ar']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => 'تشغيل';

  @override
  String get stop => 'إيقاف';

  @override
  String get config => 'إعداد';

  @override
  String get webAdmin => 'إدارة الويب';

  @override
  String get logs => 'السجلات';

  @override
  String get viewLogs => 'عرض السجلات';

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
  String get showWindow => 'إظهار النافذة';

  @override
  String get exit => 'خروج';

  @override
  String get binaryPath => 'مسار الثنائي';

  @override
  String get browse => 'تصفح';

  @override
  String get pathError => 'مسار غير صالح';

  @override
  String get arguments => 'المعاملات';

  @override
  String get argumentsHint => 'مثلاً: config.json';

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
  String get coreBinaryMissing =>
      'لم يتم العثور على الملف الثنائي الأساسي. ضع الثنائي للنظام الأساسي في app/bin/ أو حدد المسار في الإعدادات.';

  @override
  String get coreStartFailed => 'فشل في بدء خدمة النواة.';

  @override
  String get coreStopFailed => 'فشل في إيقاف خدمة النواة.';

  @override
  String get coreInvalidBinary => 'ملف ثنائي أساسي غير صالح.';

  @override
  String coreUnknownError(Object code) {
    return 'خطأ أساسي غير معروف: $code';
  }

  @override
  String get coreValid => 'الملف الثنائي الأساسي صالح.';

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
  String get check => 'فحص';

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
  String get deviceReportingTitle => 'تعليقات توافق الجهاز';

  @override
  String get deviceReportingSubtitle =>
      'تُستخدم فقط للتحقق من توافق إصدار نظام التشغيل وإصدار التطبيق. لا تتضمن رسائل الدردشة أو تفاصيل الحساب أو المحتوى الشخصي';

  @override
  String get deviceReportingConsentTitle => 'ساعد في تحسين توافق الجهاز';

  @override
  String get deviceReportingConsentDescription =>
      'عند التمكين، يتم إرسال معرف تثبيت مجهول فقط وإصدار نظام التشغيل وإصدار التطبيق لفهم التوافق. قد يتم جمع اللغة والمنطقة بشكل منفصل بواسطة Firebase Analytics. لا يتم تحميل رسائل الدردشة أو المحتوى المكتوب أو تفاصيل الحساب أو الملفات أو الإعدادات المخصصة';

  @override
  String get deviceReportingBannerDescription =>
      'يتم مزامنة معرف التثبيت المجهول وإصدار نظام التشغيل وإصدار التطبيق فقط لتحسين التوافق. قد يتم جمع اللغة والمنطقة بشكل منفصل بواسطة Firebase Analytics. لا يتم إرسال رسائل الدردشة أو تفاصيل الحساب أو الملفات أو المحتوى الشخصي';

  @override
  String get deviceReportingWhatWillBeSent => 'تتضمن فقط تفاصيل الجهاز هذه';

  @override
  String get deviceReportingDeviceLabel => 'طراز الجهاز';

  @override
  String get deviceReportingPlatformLabel => 'فئة الجهاز';

  @override
  String get deviceReportingSystemLabel => 'إصدار نظام التشغيل';

  @override
  String get deviceReportingTimingNote =>
      'يتم تشغيل المزامنة مرة واحدة عند التمكين، ومرة أخرى فقط بعد اكتشاف تحديث للنظام';

  @override
  String get deviceReportingDeny => 'ليس الآن';

  @override
  String get deviceReportingAllow => 'تشغيل';

  @override
  String get deviceReportingUploadSucceeded => 'تعليقات توافق الجهاز مفعلة';

  @override
  String get deviceReportingUploadFailed =>
      'تعليقات توافق الجهاز مفعلة، لكن مزامنة معلومات الجهاز الحالية لم تكتمل';

  @override
  String get deviceReportingDisabled => 'تعليقات توافق الجهاز معطلة';

  @override
  String get localModeHint =>
      '1. انتقل إلى تكوين الخدمة\n2. شغّل الوضع العام\n3. امسح رمز QR للوصول إلى PocketClaw';

  @override
  String get publicModeHint =>
      '1. ابدأ الخدمة\n2. امسح رمز QR للوصول إلى PocketClaw';

  @override
  String get noLogsToExport => 'لا توجد سجلات للتصدير';

  @override
  String get logsSavedToMediaLibrary =>
      'تم حفظ السجلات في التنزيلات (مكتبة وسائط Android)';

  @override
  String logsSavedToDownloads(Object path) {
    return 'تم حفظ السجلات في التنزيلات: $path';
  }

  @override
  String get shareLogsText => 'سجلات PocketClaw';

  @override
  String get workspaceDirectory => 'مساحة العمل';

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
  String get unsavedChanges => 'تغييرات غير محفوظة';

  @override
  String get unsavedChangesHint =>
      'لديك تغييرات غير محفوظة. هل تريد التخلص منها؟';

  @override
  String get cancel => 'إلغاء';

  @override
  String get discard => 'تجاهل';

  @override
  String get saved => 'تم الحفظ';

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
  String get whatsNew020New5 => 'تكامل مع تيليجرام، يُضبط من الإعدادات.';

  @override
  String get whatsNew020Improvement1 =>
      'مزوّدون أكثر مرونة: لم يعد فشل الطلب ينهي الدور.';

  @override
  String get whatsNew020Improvement2 => 'فهرس أدق لوقت التشغيل المُدار.';

  @override
  String get whatsNew020Improvement3 =>
      'هوية PocketClaw أوضح في واجهة الويب ومساحة العمل الافتراضية.';

  @override
  String get whatsNew020Fix1 =>
      'أصبحت البوابة تتعافى من سجل عملية قديم خلّفه تشغيل سابق.';

  @override
  String get whatsNew020Fix2 =>
      'يتم الآن حفظ قوائم القنوات التي تحتوي أكثر من عنصر واحد بشكل صحيح.';

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
}

// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Hindi (`hi`).
class AppLocalizationsHi extends AppLocalizations {
  AppLocalizationsHi([String locale = 'hi']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => 'चलाएं';

  @override
  String get stop => 'रोकें';

  @override
  String get config => 'कॉन्फ़िग';

  @override
  String get webAdmin => 'वेब एडमिन';

  @override
  String get logs => 'लॉग';

  @override
  String get viewLogs => 'लॉग देखें';

  @override
  String get statusRunning => 'चल रहा है';

  @override
  String get statusStopped => 'रुका हुआ';

  @override
  String get settings => 'सेटिंग्स';

  @override
  String get address => 'पता';

  @override
  String get port => 'पोर्ट';

  @override
  String get save => 'सहेजें';

  @override
  String get showWindow => 'विंडो दिखाएं';

  @override
  String get exit => 'बाहर निकलें';

  @override
  String get binaryPath => 'बाइनरी पथ';

  @override
  String get browse => 'ब्राउज़ करें';

  @override
  String get pathError => 'अमान्य पथ';

  @override
  String get arguments => 'तर्क';

  @override
  String get argumentsHint => 'उदा. config.json';

  @override
  String get notStarted => 'सेवा शुरू नहीं हुई';

  @override
  String get startHint => 'कृपया पहले डैशबोर्ड से सेवा शुरू करें।';

  @override
  String get goToDashboard => 'डैशबोर्ड पर जाएं';

  @override
  String get back => 'वापस';

  @override
  String get forward => 'आगे';

  @override
  String get refresh => 'रीफ्रेश';

  @override
  String get coreBinaryMissing =>
      'कोर बाइनरी नहीं मिली। प्लेटफ़ॉर्म बाइनरी को app/bin/ में रखें या सेटिंग्स में पथ सेट करें।';

  @override
  String get coreStartFailed => 'कोर सेवा शुरू करने में विफल।';

  @override
  String get coreStopFailed => 'कोर सेवा रोकने में विफल।';

  @override
  String get coreInvalidBinary => 'अमान्य कोर बाइनरी फ़ाइल।';

  @override
  String coreUnknownError(Object code) {
    return 'अज्ञात कोर त्रुटि: $code';
  }

  @override
  String get coreValid => 'कोर बाइनरी वैध है।';

  @override
  String get publicMode => 'पब्लिक मोड';

  @override
  String get publicModeHintDesc =>
      'सक्षम होने पर, सेवा बाहरी एक्सेस की अनुमति देती है और पता फ़ील्ड अक्षम हो जाएगी';

  @override
  String get publicModeApplying => 'नेटवर्क मोड लागू किया जा रहा है...';

  @override
  String get themeSelection => 'थीम';

  @override
  String get check => 'जांचें';

  @override
  String get launchService => 'सेवा शुरू करें';

  @override
  String get stopService => 'सेवा रोकें';

  @override
  String get endpoint => 'एंडपॉइंट';

  @override
  String get statusActive => 'सक्रिय';

  @override
  String get statusSyncing => 'सिंक हो रहा है';

  @override
  String get statusIdle => 'निष्क्रिय';

  @override
  String get publicModeEnabled => 'पब्लिक मोड सक्षम';

  @override
  String get localMode => 'लोकल मोड';

  @override
  String get unableToGetDeviceIp => 'डिवाइस IP प्राप्त करने में असमर्थ';

  @override
  String get deviceReportingTitle => 'डिवाइस संगतता प्रतिक्रिया';

  @override
  String get deviceReportingSubtitle =>
      'केवल OS संस्करण और ऐप संस्करण संगतता की जांच के लिए उपयोग किया जाता है। चैट संदेशों, खाता विवरण या व्यक्तिगत सामग्री का कोई संबंध नहीं है';

  @override
  String get deviceReportingConsentTitle =>
      'डिवाइस संगतता में सुधार करने में मदद करें';

  @override
  String get deviceReportingConsentDescription =>
      'सक्षम होने पर, केवल संगतता को समझने के लिए एक अनाम इंस्टॉलेशन ID, OS संस्करण और ऐप संस्करण भेजे जाते हैं। भाषा और क्षेत्र Firebase Analytics द्वारा अलग से एकत्र किए जा सकते हैं। कोई चैट संदेश, टाइप की गई सामग्री, खाता विवरण, फ़ाइलें या कस्टम सेटिंग्स अपलोड नहीं की जाती हैं';

  @override
  String get deviceReportingBannerDescription =>
      'संगतता में सुधार के लिए केवल एक अनाम इंस्टॉलेशन ID, OS संस्करण और ऐप संस्करण सिंक किए जाते हैं। भाषा और क्षेत्र Firebase Analytics द्वारा अलग से एकत्र किए जा सकते हैं। कोई चैट संदेश, खाता विवरण, फ़ाइलें या व्यक्तिगत सामग्री नहीं भेजी जाती';

  @override
  String get deviceReportingWhatWillBeSent => 'केवल ये डिवाइस विवरण शामिल हैं';

  @override
  String get deviceReportingDeviceLabel => 'डिवाइस मॉडल';

  @override
  String get deviceReportingPlatformLabel => 'डिवाइस श्रेणी';

  @override
  String get deviceReportingSystemLabel => 'OS संस्करण';

  @override
  String get deviceReportingTimingNote =>
      'सक्षम होने पर एक बार सिंक चलता है, और फिर केवल तब जब सिस्टम अपडेट का पता चलता है';

  @override
  String get deviceReportingDeny => 'अभी नहीं';

  @override
  String get deviceReportingAllow => 'सक्षम करें';

  @override
  String get deviceReportingUploadSucceeded =>
      'डिवाइस संगतता प्रतिक्रिया सक्षम है';

  @override
  String get deviceReportingUploadFailed =>
      'डिवाइस संगतता प्रतिक्रिया सक्षम है, लेकिन वर्तमान डिवाइस जानकारी सिंक पूरा नहीं हुआ';

  @override
  String get deviceReportingDisabled => 'डिवाइस संगतता प्रतिक्रिया अक्षम है';

  @override
  String get localModeHint =>
      '1. सेवा कॉन्फ़िगरेशन पर जाएं\n2. पब्लिक मोड चालू करें\n3. PocketClaw तक पहुंचने के लिए QR कोड स्कैन करें';

  @override
  String get publicModeHint =>
      '1. सेवा शुरू करें\n2. PocketClaw तक पहुंचने के लिए QR कोड स्कैन करें';

  @override
  String get noLogsToExport => 'निर्यात करने के लिए कोई लॉग नहीं';

  @override
  String get logsSavedToMediaLibrary =>
      'लॉग डाउनलोड (Android मीडिया लाइब्रेरी) में सहेजे गए';

  @override
  String logsSavedToDownloads(Object path) {
    return 'लॉग डाउनलोड में सहेजे गए: $path';
  }

  @override
  String get shareLogsText => 'PocketClaw लॉग';

  @override
  String get workspaceDirectory => 'कार्यस्थान';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'लॉग डाउनलोड (Android मीडिया लाइब्रेरी) में सहेजे गए: $name';
  }

  @override
  String shareFailed(Object error) {
    return 'शेयर डायलॉग खोलने में विफल: $error';
  }

  @override
  String get exportLogs => 'लॉग निर्यात करें';

  @override
  String logEventsCount(int count) {
    return '$count इवेंट्स';
  }

  @override
  String get unsavedChanges => 'असहेजे गए परिवर्तन';

  @override
  String get unsavedChangesHint =>
      'आपके पास असहेजे गए परिवर्तन हैं। क्या आप उन्हें छोड़ना चाहते हैं?';

  @override
  String get cancel => 'रद्द करें';

  @override
  String get discard => 'छोड़ें';

  @override
  String get saved => 'सहेजे गए';

  @override
  String get language => 'भाषा';

  @override
  String get selectLanguage => 'भाषा चुनें';

  @override
  String get about => 'परिचय';

  @override
  String get aboutDescription =>
      'PocketClaw आपका निजी AI सहायक कार्यक्षेत्र है।';

  @override
  String get aboutAppVersionLabel => 'PocketClaw संस्करण';

  @override
  String get aboutCoreVersionLabel => 'रनटाइम संस्करण';

  @override
  String get aboutVersionUnavailable => 'उपलब्ध नहीं';

  @override
  String get close => 'बंद करें';

  @override
  String get contextMemoryTitle => 'टेलीग्राम संदर्भ स्मृति';

  @override
  String get contextMemoryDescription =>
      'नियंत्रित करता है कि हाल की कितनी बातचीत AI को भेजी जाए। पुरानी बातचीत टेलीग्राम में बनी रहती है और चलते सारांश द्वारा दर्शाई जाती है।';

  @override
  String get contextMemoryHelp =>
      'अधिक संदेश अधिक हालिया संदर्भ देते हैं पर अधिक टोकन खर्च करते हैं। पुराने संदेश टेलीग्राम में बने रहते हैं और चलते सारांश के माध्यम से सुरक्षित रह सकते हैं।';

  @override
  String get contextMemoryRecommended => 'अनुशंसित';

  @override
  String get contextMemoryCustom => 'कस्टम';

  @override
  String get contextMemoryCustomLabel => 'संदेश';

  @override
  String contextMemoryRangeError(int min, int max) {
    return '$min और $max के बीच एक पूर्ण संख्या दर्ज करें।';
  }

  @override
  String get contextMemorySaveFailed => 'यह सेटिंग सहेजी नहीं जा सकी।';

  @override
  String get settingsSave => 'सहेजें';

  @override
  String get autoStartServiceTitle => 'PocketClaw सेवा स्वतः प्रारंभ करें';

  @override
  String get autoStartGatewayTitle => 'गेटवे स्वतः प्रारंभ करें';

  @override
  String get autoStartPreferenceOn => 'स्वतः प्रारंभ वरीयता: चालू';

  @override
  String get autoStartPreferenceOff => 'स्वतः प्रारंभ वरीयता: बंद';

  @override
  String get runtimeRunning => 'रनटाइम: चल रहा है';

  @override
  String get runtimeStarting => 'रनटाइम: प्रारंभ हो रहा है';

  @override
  String get runtimeStopped => 'रनटाइम: रुका हुआ';

  @override
  String get gatewayAutoStartHint =>
      'यह अगली बार PocketClaw सेवा शुरू होने पर लागू होगा। गेटवे रनटाइम डैशबोर्ड से प्रबंधित होता है।';

  @override
  String get manageTelegramConnection => 'Telegram कनेक्शन प्रबंधित करें';

  @override
  String get manageModelsTitle => 'मॉडल प्रबंधित करें';

  @override
  String get manageModelsDescription =>
      'AI मॉडल जोड़ें, संपादित करें, परखें और चुनें।';

  @override
  String get githubChecking => 'जाँच हो रही है…';

  @override
  String get githubConnected => 'कनेक्टेड';

  @override
  String get githubNotConnected => 'कनेक्ट नहीं है';

  @override
  String githubConnectedAs(String login) {
    return '$login के रूप में कनेक्टेड';
  }

  @override
  String get githubDescription =>
      'बंडल किए गए gh और HTTPS पर Git द्वारा उपयोग किया जाता है। टोकन इस डिवाइस पर एन्क्रिप्ट किया जाता है और दोबारा नहीं दिखाया जाता।';

  @override
  String get githubTestConnection => 'कनेक्शन जाँचें';

  @override
  String get githubDisconnect => 'डिस्कनेक्ट करें';

  @override
  String get githubConnectAction => 'GitHub कनेक्ट करें';

  @override
  String get githubConnect => 'कनेक्ट करें';

  @override
  String get githubTokenLabel => 'व्यक्तिगत एक्सेस टोकन';

  @override
  String get githubTokenHint =>
      'आवश्यक स्कोप वाला GitHub व्यक्तिगत एक्सेस टोकन चिपकाएँ (निजी रिपॉजिटरी के लिए repo)।';

  @override
  String get githubTokenRejected => 'GitHub ने यह टोकन स्वीकार नहीं किया।';

  @override
  String get githubAuthWorking => 'GitHub प्रमाणीकरण काम कर रहा है।';

  @override
  String get githubAuthNotWorking => 'GitHub प्रमाणीकरण काम नहीं कर रहा है।';

  @override
  String githubAuthenticatedAs(String login) {
    return '$login के रूप में प्रमाणित।';
  }

  @override
  String get githubCredentialRemoveFailed => 'क्रेडेंशियल हटाया नहीं जा सका।';

  @override
  String get githubYourAccount => 'आपका GitHub खाता';

  @override
  String githubConnectedReport(String who, String outcome) {
    return '$who के रूप में कनेक्टेड। $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return 'डिस्कनेक्ट हो गया। $outcome';
  }

  @override
  String get credentialAppliedNow => 'gh और git अब इसका उपयोग कर सकते हैं।';

  @override
  String get credentialAppliesNextStart =>
      'इसका उपयोग अगली बार PocketClaw शुरू होने पर किया जाएगा।';

  @override
  String get credentialAppliesDeferred =>
      'सहेजा गया। PocketClaw प्रारंभ हो रहा है, इसलिए पूरा होते ही यह स्वतः लागू हो जाएगा।';

  @override
  String get whatsNewTitle => 'नया क्या है';

  @override
  String get whatsNewDescription => 'इस रिलीज़ के मुख्य बदलाव.';

  @override
  String get whatsNewBadge => 'नया';

  @override
  String get whatsNewSectionNew => 'नया';

  @override
  String get whatsNewSectionImprovements => 'सुधार';

  @override
  String get whatsNewSectionFixes => 'समाधान';

  @override
  String get whatsNew020New1 =>
      'प्रबंधित रनटाइम PocketClaw के साथ आने वाले टूल इंस्टॉल करता है और उन्हें अद्यतन रखता है.';

  @override
  String get whatsNew020New2 =>
      'साथ में दिए गए डेवलपर टूल: Git, GitHub CLI, curl, ripgrep, jq और SQLite.';

  @override
  String get whatsNew020New3 =>
      'PocketClaw के साथ दिए गए Python 3.14 रनटाइम का समर्थन.';

  @override
  String get whatsNew020New4 =>
      'सुरक्षित GitHub साइन-इन, जिसे साथ दिए गए Git और GitHub CLI दोनों उपयोग करते हैं.';

  @override
  String get whatsNew020New5 =>
      'एक टैप में Telegram सेटअप: PocketClaw आपका अपना बॉट बनाता है — ऐप में या ब्राउज़र में डैशबोर्ड से। कोई टोकन कॉपी नहीं करना पड़ता, और सिर्फ़ आपका खाता उससे बात कर सकता है।';

  @override
  String get whatsNew020New6 =>
      'डैशबोर्ड पर स्थिति दृश्य: चल रहा काम, चैनल, उपयोग में मॉडल और रनटाइम संसाधन.';

  @override
  String get whatsNew020Improvement1 =>
      'अधिक सुदृढ़ प्रदाता: कोई विफल अनुरोध अब बातचीत को समाप्त नहीं करता.';

  @override
  String get whatsNew020Improvement2 => 'अधिक सटीक प्रबंधित रनटाइम सूची.';

  @override
  String get whatsNew020Improvement3 =>
      'वेब इंटरफ़ेस और डिफ़ॉल्ट कार्यक्षेत्र में अधिक स्पष्ट PocketClaw पहचान.';

  @override
  String get whatsNew020Improvement4 =>
      'PocketClaw के Aperture डिज़ाइन पर आधारित नया ऐप आइकन और नया परिचय स्क्रीन.';

  @override
  String get whatsNew020Improvement5 =>
      'सहायक के उत्तर अधिक स्वाभाविक लगते हैं, अंत में किसी निश्चित हस्ताक्षर के बिना.';

  @override
  String get whatsNew020Improvement6 =>
      'PocketClaw अब फ़ोन अनुमति नहीं मांगता — ऐप में इसका कोई उपयोग नहीं था.';

  @override
  String get whatsNew020Improvement7 =>
      'डायग्नोस्टिक लॉग अब डाउनलोड फ़ोल्डर के बजाय ऐप के अंदर निजी रूप से रखे जाते हैं. आपका कार्यक्षेत्र वहीं रहता है.';

  @override
  String get whatsNew020Improvement8 =>
      'PocketClaw अपनी स्थिति कैसे सहेजता है और अपनी सेवाएँ कैसे चलाता है, इसमें एक बड़ा सुधार — रोज़मर्रा की विश्वसनीयता के लिए.';

  @override
  String get whatsNew020Improvement9 =>
      'इस अपडेट के बाद डैशबोर्ड आपसे एक बार फिर साइन इन करने को कहेगा.';

  @override
  String get whatsNew020Improvement10 =>
      'इस अपडेट के बाद वेब चैट चैनल एक नई बातचीत शुरू करता है.';

  @override
  String get whatsNew020Improvement11 =>
      'इस अपडेट के बाद अपनी सूचना प्राथमिकताएँ एक बार जाँच लेना बेहतर रहेगा.';

  @override
  String get whatsNew020Fix1 =>
      'गेटवे अब पिछली बार छूटे हुए पुराने प्रोसेस रिकॉर्ड से उबर जाता है.';

  @override
  String get whatsNew020Fix2 =>
      'एक से अधिक प्रविष्टि वाली चैनल सूचियाँ सहेजते समय सही ढंग से बनी रहती हैं.';

  @override
  String get settingsGroupConnection => 'कनेक्शन';

  @override
  String get settingsGroupAgent => 'एजेंट';

  @override
  String get settingsGroupIntegrations => 'एकीकरण';

  @override
  String get settingsGroupAppearance => 'रूप';

  @override
  String get statusTitle => 'स्थिति';

  @override
  String get statusSectionSystem => 'सिस्टम';

  @override
  String get statusSectionAi => 'एआई';

  @override
  String get statusSectionActivity => 'गतिविधि';

  @override
  String get statusSectionChannels => 'चैनल';

  @override
  String get statusSectionResources => 'संसाधन';

  @override
  String get statusSinceGatewayStart => 'गेटवे शुरू होने से';

  @override
  String get statusGateway => 'गेटवे';

  @override
  String get statusUptime => 'अपटाइम';

  @override
  String get statusAppVersion => 'ऐप संस्करण';

  @override
  String get statusCoreVersion => 'कोर संस्करण';

  @override
  String get statusActiveModel => 'सक्रिय मॉडल';

  @override
  String get statusConfiguredDefault => 'कॉन्फ़िगर किया गया डिफ़ॉल्ट';

  @override
  String get statusProvider => 'प्रदाता';

  @override
  String get statusFallbacks => 'फ़ॉलबैक';

  @override
  String get statusActiveTurns => 'सक्रिय टर्न';

  @override
  String get statusActiveSubagents => 'सक्रिय सबएजेंट';

  @override
  String get statusWaiting => 'प्रतीक्षारत';

  @override
  String get statusCompleted => 'पूर्ण';

  @override
  String get statusFailed => 'विफल';

  @override
  String get statusCancelled => 'रद्द';

  @override
  String get statusToolCalls => 'टूल कॉल';

  @override
  String get statusToolCallsFailed => 'विफल टूल कॉल';

  @override
  String get statusLastActivity => 'अंतिम गतिविधि';

  @override
  String get statusCoreMemory => 'कोर मेमोरी';

  @override
  String get statusCoreCpuTime => 'कोर सीपीयू समय';

  @override
  String get statusNoChannels => 'कोई चैनल कॉन्फ़िगर नहीं';

  @override
  String get statusDetailUnavailable => 'विस्तृत स्थिति उपलब्ध नहीं है';

  @override
  String get statusJustNow => 'अभी';

  @override
  String statusMinutesAgo(int minutes) {
    return '$minutes मि. पहले';
  }

  @override
  String statusHoursAgo(int hours) {
    return '$hours घं. पहले';
  }

  @override
  String statusDaysAgo(int days) {
    return '$days दि. पहले';
  }

  @override
  String statusDurationSeconds(int seconds) {
    return '$seconds से';
  }

  @override
  String statusDurationMinutes(int minutes, int seconds) {
    return '$minutes मि $seconds से';
  }

  @override
  String statusDurationHours(int hours, int minutes) {
    return '$hours घं $minutes मि';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '$days दि $hours घं';
  }

  @override
  String get notificationPermissionTitle => 'सूचनाएँ';

  @override
  String get notificationPermissionGranted =>
      'PocketClaw अपनी चालू सूचना दिखा सकता है।';

  @override
  String get notificationPermissionBlocked =>
      'सूचनाएँ बंद हैं, इसलिए PocketClaw की चालू सूचना दिखाई नहीं देगी।';

  @override
  String get notificationPermissionOpenSettings => 'सूचना सेटिंग्स खोलें';

  @override
  String get whatsNew020New7 =>
      'डैशबोर्ड से Telegram प्रबंधन: बॉट कनेक्ट करें, बदलें या डिस्कनेक्ट करें।';

  @override
  String get whatsNew020New8 =>
      'डैशबोर्ड में प्रदाता और मॉडल प्रबंधन: प्रदाता जोड़ें, API कुंजी बदलें या मॉडल हटाएँ।';

  @override
  String get whatsNew020Improvement12 =>
      'Telegram सेटिंग्स सहेजते ही लागू हो जाती हैं, किसी मैन्युअल रीस्टार्ट के बिना।';

  @override
  String get whatsNew020Improvement13 =>
      'जब कोई AI मॉडल सेट नहीं होता, PocketClaw ठीक-ठीक बताता है कि क्या कमी है और खोजने के लिए एक कोड देता है।';

  @override
  String get whatsNew020Improvement14 =>
      'डैशबोर्ड में साइन इन करने पर आप उसी स्क्रीन पर पहुँचते हैं जो आपने माँगी थी।';

  @override
  String get whatsNew020Improvement15 =>
      'डायग्नोस्टिक लॉग विस्तृत रहते हैं, लेकिन उनमें आपकी कुंजियाँ, टोकन या संदेश का टेक्स्ट कभी नहीं होता।';

  @override
  String get whatsNew020Improvement16 =>
      'जिस डैशबोर्ड पर अभी किसी का स्वामित्व नहीं है, वह नेटवर्क से कभी पहुँच योग्य नहीं होता — पब्लिक मोड चालू होने पर भी।';
}

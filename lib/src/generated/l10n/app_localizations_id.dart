// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Indonesian (`id`).
class AppLocalizationsId extends AppLocalizations {
  AppLocalizationsId([String locale = 'id']) : super(locale);

  @override
  String get appTitle => 'PocketClaw';

  @override
  String get run => 'Jalankan';

  @override
  String get stop => 'Hentikan';

  @override
  String get config => 'Konfig';

  @override
  String get webAdmin => 'Admin Web';

  @override
  String get logs => 'Log';

  @override
  String get viewLogs => 'Lihat log';

  @override
  String get statusRunning => 'Berjalan';

  @override
  String get statusStopped => 'Dihentikan';

  @override
  String get settings => 'Pengaturan';

  @override
  String get address => 'Alamat';

  @override
  String get port => 'Port';

  @override
  String get save => 'Simpan';

  @override
  String get showWindow => 'Tampilkan jendela';

  @override
  String get exit => 'Keluar';

  @override
  String get binaryPath => 'Jalur biner';

  @override
  String get browse => 'Telusuri';

  @override
  String get pathError => 'Jalur tidak valid';

  @override
  String get arguments => 'Argumen';

  @override
  String get argumentsHint => 'mis. config.json';

  @override
  String get notStarted => 'Layanan belum dimulai';

  @override
  String get startHint => 'Silakan mulai layanan dari panel terlebih dahulu.';

  @override
  String get goToDashboard => 'Ke panel';

  @override
  String get back => 'Kembali';

  @override
  String get forward => 'Maju';

  @override
  String get refresh => 'Segarkan';

  @override
  String get coreBinaryMissing =>
      'Biner inti tidak ditemukan. Tempatkan biner platform di app/bin/ atau atur jalur di Pengaturan.';

  @override
  String get coreStartFailed => 'Gagal memulai layanan inti.';

  @override
  String get coreStopFailed => 'Gagal menghentikan layanan inti.';

  @override
  String get coreInvalidBinary => 'File biner inti tidak valid.';

  @override
  String coreUnknownError(Object code) {
    return 'Kesalahan inti tidak dikenal: $code';
  }

  @override
  String get coreValid => 'Biner inti valid.';

  @override
  String get publicMode => 'Mode publik';

  @override
  String get publicModeHintDesc =>
      'Bila diaktifkan, layanan mengizinkan akses eksternal dan kolom alamat akan dinonaktifkan';

  @override
  String get publicModeApplying => 'Menerapkan mode jaringan...';

  @override
  String get themeSelection => 'Tema';

  @override
  String get check => 'Periksa';

  @override
  String get launchService => 'MULAI LAYANAN';

  @override
  String get stopService => 'HENTIKAN LAYANAN';

  @override
  String get endpoint => 'TITIK AKHIR';

  @override
  String get statusActive => 'AKTIF';

  @override
  String get statusSyncing => 'MENYINKRONKAN';

  @override
  String get statusIdle => 'MENGANGGU';

  @override
  String get publicModeEnabled => 'Mode publik diaktifkan';

  @override
  String get localMode => 'Mode lokal';

  @override
  String get unableToGetDeviceIp => 'Tidak dapat mendapatkan IP perangkat';

  @override
  String get deviceReportingTitle => 'Umpan balik kompatibilitas perangkat';

  @override
  String get deviceReportingSubtitle =>
      'Hanya digunakan untuk memverifikasi kompatibilitas versi OS dan versi aplikasi. Tidak melibatkan pesan obrolan, detail akun, atau konten pribadi';

  @override
  String get deviceReportingConsentTitle =>
      'Bantu meningkatkan kompatibilitas perangkat';

  @override
  String get deviceReportingConsentDescription =>
      'Bila diaktifkan, hanya ID instalasi anonim, versi OS, dan versi aplikasi yang dikirim untuk memahami kompatibilitas. Bahasa dan kawasan dapat dikumpulkan secara terpisah oleh Firebase Analytics. Tidak ada pesan obrolan, konten yang diketik, detail akun, file, atau pengaturan kustom yang diunggah';

  @override
  String get deviceReportingBannerDescription =>
      'Hanya ID instalasi anonim, versi OS, dan versi aplikasi yang disinkronkan untuk meningkatkan kompatibilitas. Bahasa dan kawasan dapat dikumpulkan secara terpisah oleh Firebase Analytics. Tidak ada pesan obrolan, detail akun, file, atau konten pribadi yang dikirim';

  @override
  String get deviceReportingWhatWillBeSent =>
      'Hanya detail perangkat ini yang disertakan';

  @override
  String get deviceReportingDeviceLabel => 'Model Perangkat';

  @override
  String get deviceReportingPlatformLabel => 'Kategori Perangkat';

  @override
  String get deviceReportingSystemLabel => 'Versi OS';

  @override
  String get deviceReportingTimingNote =>
      'Sinkronisasi berjalan sekali saat diaktifkan, dan lagi hanya setelah pembaruan sistem terdeteksi';

  @override
  String get deviceReportingDeny => 'Belum sekarang';

  @override
  String get deviceReportingAllow => 'Aktifkan';

  @override
  String get deviceReportingUploadSucceeded =>
      'Umpan balik kompatibilitas perangkat aktif';

  @override
  String get deviceReportingUploadFailed =>
      'Umpan balik kompatibilitas perangkat aktif, tetapi sinkronisasi info perangkat saat ini tidak selesai';

  @override
  String get deviceReportingDisabled =>
      'Umpan balik kompatibilitas perangkat nonaktif';

  @override
  String get localModeHint =>
      '1. Buka Konfigurasi layanan\n2. Aktifkan Mode publik\n3. Pindai kode QR untuk mengakses PocketClaw';

  @override
  String get publicModeHint =>
      '1. Mulai layanan\n2. Pindai kode QR untuk mengakses PocketClaw';

  @override
  String get noLogsToExport => 'Tidak ada log untuk diekspor';

  @override
  String get logsSavedToMediaLibrary =>
      'Log disimpan ke Unduhan (perpustakaan media Android)';

  @override
  String logsSavedToDownloads(Object path) {
    return 'Log disimpan ke Unduhan: $path';
  }

  @override
  String get shareLogsText => 'Log PocketClaw';

  @override
  String get workspaceDirectory => 'Ruang kerja';

  @override
  String logsSavedToMediaLibraryWithName(Object name) {
    return 'Log disimpan ke Unduhan (perpustakaan media Android): $name';
  }

  @override
  String shareFailed(Object error) {
    return 'Gagal membuka dialog berbagi: $error';
  }

  @override
  String get exportLogs => 'Ekspor log';

  @override
  String logEventsCount(int count) {
    return '$count PERISTIWA';
  }

  @override
  String get unsavedChanges => 'Perubahan belum disimpan';

  @override
  String get unsavedChangesHint =>
      'Anda memiliki perubahan yang belum disimpan. Apakah Anda ingin membuangnya?';

  @override
  String get cancel => 'Batal';

  @override
  String get discard => 'Buang';

  @override
  String get saved => 'Disimpan';

  @override
  String get language => 'Bahasa';

  @override
  String get selectLanguage => 'Pilih bahasa';

  @override
  String get about => 'Tentang';

  @override
  String get aboutDescription =>
      'PocketClaw adalah ruang kerja pribadi untuk asisten AI Anda.';

  @override
  String get aboutAppVersionLabel => 'Versi PocketClaw';

  @override
  String get aboutCoreVersionLabel => 'Versi runtime';

  @override
  String get aboutVersionUnavailable => 'Tidak tersedia';

  @override
  String get close => 'Tutup';

  @override
  String get contextMemoryTitle => 'Memori Konteks Telegram';

  @override
  String get contextMemoryDescription =>
      'Mengatur berapa banyak pesan percakapan terbaru yang dikirim ke AI. Percakapan lama tetap ada di Telegram dan diwakili oleh ringkasan berjalan.';

  @override
  String get contextMemoryHelp =>
      'Lebih banyak pesan memberi konteks terbaru yang lebih kaya tetapi memakai lebih banyak token. Pesan lama tetap ada di Telegram dan dapat dipertahankan melalui ringkasan berjalan.';

  @override
  String get contextMemoryRecommended => 'Disarankan';

  @override
  String get contextMemoryCustom => 'Kustom';

  @override
  String get contextMemoryCustomLabel => 'Pesan';

  @override
  String contextMemoryRangeError(int min, int max) {
    return 'Masukkan bilangan bulat antara $min dan $max.';
  }

  @override
  String get contextMemorySaveFailed => 'Pengaturan ini tidak dapat disimpan.';

  @override
  String get settingsSave => 'Simpan';

  @override
  String get autoStartServiceTitle =>
      'Jalankan layanan PocketClaw secara otomatis';

  @override
  String get autoStartGatewayTitle => 'Jalankan Gateway secara otomatis';

  @override
  String get autoStartPreferenceOn => 'Preferensi mulai otomatis: AKTIF';

  @override
  String get autoStartPreferenceOff => 'Preferensi mulai otomatis: NONAKTIF';

  @override
  String get runtimeRunning => 'Runtime: Berjalan';

  @override
  String get runtimeStarting => 'Runtime: Memulai';

  @override
  String get runtimeStopped => 'Runtime: Berhenti';

  @override
  String get gatewayAutoStartHint =>
      'Berlaku saat layanan PocketClaw dijalankan berikutnya. Runtime Gateway dikelola di Dasbor.';

  @override
  String get manageTelegramConnection => 'Kelola koneksi Telegram';

  @override
  String get manageModelsTitle => 'Kelola Model';

  @override
  String get manageModelsDescription =>
      'Tambah, ubah, uji, dan pilih model AI.';

  @override
  String get githubChecking => 'Memeriksa…';

  @override
  String get githubConnected => 'Terhubung';

  @override
  String get githubNotConnected => 'Tidak terhubung';

  @override
  String githubConnectedAs(String login) {
    return 'Terhubung sebagai $login';
  }

  @override
  String get githubDescription =>
      'Digunakan oleh gh bawaan dan oleh Git melalui HTTPS. Token dienkripsi di perangkat ini dan tidak ditampilkan lagi.';

  @override
  String get githubTestConnection => 'Uji koneksi';

  @override
  String get githubDisconnect => 'Putuskan';

  @override
  String get githubConnectAction => 'Hubungkan GitHub';

  @override
  String get githubConnect => 'Hubungkan';

  @override
  String get githubTokenLabel => 'Token akses pribadi';

  @override
  String get githubTokenHint =>
      'Tempel token akses pribadi GitHub dengan cakupan yang Anda butuhkan (repo untuk repositori privat).';

  @override
  String get githubTokenRejected => 'GitHub tidak menerima token ini.';

  @override
  String get githubAuthWorking => 'Autentikasi GitHub berfungsi.';

  @override
  String get githubAuthNotWorking => 'Autentikasi GitHub tidak berfungsi.';

  @override
  String githubAuthenticatedAs(String login) {
    return 'Terautentikasi sebagai $login.';
  }

  @override
  String get githubCredentialRemoveFailed => 'Kredensial tidak dapat dihapus.';

  @override
  String get githubYourAccount => 'akun GitHub Anda';

  @override
  String githubConnectedReport(String who, String outcome) {
    return 'Terhubung sebagai $who. $outcome';
  }

  @override
  String githubDisconnectedReport(String outcome) {
    return 'Terputus. $outcome';
  }

  @override
  String get credentialAppliedNow =>
      'gh dan git dapat menggunakannya sekarang.';

  @override
  String get credentialAppliesNextStart =>
      'Ini akan digunakan saat PocketClaw dijalankan berikutnya.';

  @override
  String get credentialAppliesDeferred =>
      'Tersimpan. PocketClaw sedang memulai, jadi ini akan diterapkan otomatis setelah selesai.';

  @override
  String get whatsNewTitle => 'Apa yang Baru';

  @override
  String get whatsNewDescription => 'Perubahan utama pada rilis ini.';

  @override
  String get whatsNewBadge => 'BARU';

  @override
  String get whatsNewSectionNew => 'Baru';

  @override
  String get whatsNewSectionImprovements => 'Peningkatan';

  @override
  String get whatsNewSectionFixes => 'Perbaikan';

  @override
  String get whatsNew020New1 =>
      'Managed Runtime memasang dan menjaga agar peralatan bawaan PocketClaw tetap mutakhir.';

  @override
  String get whatsNew020New2 =>
      'Peralatan pengembang bawaan: Git, GitHub CLI, curl, ripgrep, jq, dan SQLite.';

  @override
  String get whatsNew020New3 =>
      'Dukungan untuk runtime Python 3.14 bawaan PocketClaw.';

  @override
  String get whatsNew020New4 =>
      'Masuk ke GitHub secara aman, dipakai bersama oleh Git dan GitHub CLI bawaan.';

  @override
  String get whatsNew020New5 =>
      'Integrasi Telegram, disiapkan dari Pengaturan.';

  @override
  String get whatsNew020New6 =>
      'Tampilan Status di Dasbor: pekerjaan aktif, saluran, model yang dipakai, dan sumber daya runtime.';

  @override
  String get whatsNew020Improvement1 =>
      'Penyedia yang lebih tangguh: permintaan yang gagal tidak lagi mengakhiri giliran.';

  @override
  String get whatsNew020Improvement2 =>
      'Katalog Managed Runtime yang lebih akurat.';

  @override
  String get whatsNew020Improvement3 =>
      'Identitas PocketClaw yang lebih rapi di antarmuka web dan ruang kerja bawaan.';

  @override
  String get whatsNew020Improvement4 =>
      'Ikon aplikasi baru dan layar Tentang yang disegarkan, mengikuti desain Aperture PocketClaw.';

  @override
  String get whatsNew020Improvement5 =>
      'Balasan asisten terasa lebih alami, tanpa tanda tangan tetap di akhir.';

  @override
  String get whatsNew020Improvement6 =>
      'PocketClaw tidak lagi meminta izin Telepon — tidak ada bagian aplikasi yang memakainya.';

  @override
  String get whatsNew020Fix1 =>
      'Gateway kini pulih dari catatan proses usang yang ditinggalkan oleh proses sebelumnya.';

  @override
  String get whatsNew020Fix2 =>
      'Daftar saluran dengan lebih dari satu entri kini tersimpan dengan benar.';

  @override
  String get settingsGroupConnection => 'Koneksi';

  @override
  String get settingsGroupAgent => 'Agen';

  @override
  String get settingsGroupIntegrations => 'Integrasi';

  @override
  String get settingsGroupAppearance => 'Tampilan';

  @override
  String get statusTitle => 'Status';

  @override
  String get statusSectionSystem => 'Sistem';

  @override
  String get statusSectionAi => 'AI';

  @override
  String get statusSectionActivity => 'Aktivitas';

  @override
  String get statusSectionChannels => 'Kanal';

  @override
  String get statusSectionResources => 'Sumber Daya';

  @override
  String get statusSinceGatewayStart => 'Sejak Gateway dimulai';

  @override
  String get statusGateway => 'Gateway';

  @override
  String get statusUptime => 'Waktu aktif';

  @override
  String get statusAppVersion => 'Versi aplikasi';

  @override
  String get statusCoreVersion => 'Versi Core';

  @override
  String get statusActiveModel => 'Model aktif';

  @override
  String get statusConfiguredDefault => 'Default terkonfigurasi';

  @override
  String get statusProvider => 'Penyedia';

  @override
  String get statusFallbacks => 'Cadangan';

  @override
  String get statusActiveTurns => 'Giliran aktif';

  @override
  String get statusActiveSubagents => 'Subagen aktif';

  @override
  String get statusWaiting => 'Menunggu';

  @override
  String get statusCompleted => 'Selesai';

  @override
  String get statusFailed => 'Gagal';

  @override
  String get statusCancelled => 'Dibatalkan';

  @override
  String get statusToolCalls => 'Panggilan alat';

  @override
  String get statusToolCallsFailed => 'Panggilan alat gagal';

  @override
  String get statusLastActivity => 'Aktivitas terakhir';

  @override
  String get statusCoreMemory => 'Memori Core';

  @override
  String get statusCoreCpuTime => 'Waktu CPU Core';

  @override
  String get statusNoChannels => 'Tidak ada kanal terkonfigurasi';

  @override
  String get statusDetailUnavailable => 'Status rinci tidak tersedia';

  @override
  String get statusJustNow => 'Baru saja';

  @override
  String statusMinutesAgo(int minutes) {
    return '$minutes mnt lalu';
  }

  @override
  String statusHoursAgo(int hours) {
    return '$hours jam lalu';
  }

  @override
  String statusDaysAgo(int days) {
    return '$days hr lalu';
  }

  @override
  String statusDurationSeconds(int seconds) {
    return '$seconds dtk';
  }

  @override
  String statusDurationMinutes(int minutes, int seconds) {
    return '$minutes mnt $seconds dtk';
  }

  @override
  String statusDurationHours(int hours, int minutes) {
    return '$hours jam $minutes mnt';
  }

  @override
  String statusDurationDays(int days, int hours) {
    return '$days hr $hours jam';
  }
}

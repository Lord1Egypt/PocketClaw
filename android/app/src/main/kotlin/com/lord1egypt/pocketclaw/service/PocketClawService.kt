package com.lord1egypt.pocketclaw.service

import com.lord1egypt.pocketclaw.BuildConfig
import com.lord1egypt.pocketclaw.PocketClawCoreState
import com.lord1egypt.pocketclaw.security.GitHubCredentialStore
import android.app.Notification
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.os.IBinder
import android.os.PowerManager
import android.util.Base64
import android.util.Log
import androidx.core.app.NotificationCompat
import com.lord1egypt.pocketclaw.PocketClawApp
import com.lord1egypt.pocketclaw.MainActivity
import java.io.BufferedReader
import java.io.File
import java.io.FileOutputStream
import java.io.InputStreamReader
import java.security.SecureRandom
import java.util.zip.ZipFile

class PocketClawService : Service() {

    companion object {
        private const val TAG = "PocketClawService"
        private const val NOTIFICATION_ID = 1
        private const val GATEWAY_BINARY_NAME = "libpocketclaw.so"
        private const val WEB_BINARY_NAME = "libpocketclaw-web.so"
        private const val GATEWAY_PORT = 18790
        private const val WEB_PORT = 18800
        private val ANSI_ESCAPE_REGEX = Regex("\\u001B(?:[@-Z\\\\-_]|\\[[0-?]*[ -/]*[@-~])")
        private val SEMANTIC_VERSION_REGEX = Regex(
            "(?<!\\d)v?(\\d+\\.\\d+\\.\\d+(?:-[0-9A-Za-z.-]+)?(?:\\+[0-9A-Za-z.-]+)?)(?!\\d)"
        )
        private const val REALTIME_AUTH_FILE = "realtime_auth"

        /**
         * How long a stop waits for the owning worker to exit before it reports
         * that it did not.
         *
         * PC-DEF-072. It is a bound on *waiting*, never on ownership: when it
         * expires the worker keeps ownership and a start queues behind it. The
         * previous code treated the same expiry as proof the thread was gone.
         */
        private const val SERVICE_THREAD_JOIN_MS = 5_000L

        /** Backoff between Core crash restarts. Interruptible by a stop. */
        private const val RESTART_BACKOFF_MS = 5_000L

        /**
         * Where Core writes the gateway bearer credential.
         *
         * It used to live inside `.picoclaw.pid` in POCKETCLAW_HOME, which on
         * this platform is `Download/pocketclaw` — user-visible shared storage,
         * where the 0600 Core writes with is synthesised by the filesystem
         * rather than enforced. Any app holding storage access could read it,
         * and Android does not isolate loopback sockets between apps, so that
         * credential was one file read away from authenticating to the gateway.
         *
         * no-backup app-private storage is the same boundary the realtime
         * credential already uses. The PID record keeps its discovery fields
         * and is written without a token.
         */
        private const val GATEWAY_AUTH_FILE = "gateway_auth"

        /** Private directory for gateway.log and the panic log. */
        private const val PRIVATE_LOG_DIR = "logs"

        /**
         * Private directory for the Dashboard credential database.
         *
         * launcher-auth.db holds a bcrypt verifier — no plaintext, no session
         * token — so reading it buys an attacker little. Writing it is the
         * problem: under POCKETCLAW_HOME it sits on shared external storage,
         * where an app with storage write access can replace the verifier with
         * one for a password it chose and then log in normally over loopback,
         * which Android does not isolate between apps. That is an
         * authentication bypass that never has to break bcrypt at all.
         */
        private const val PRIVATE_AUTH_DIR = "auth"

        /**
         * Legacy log files this app wrote to shared storage before the logs
         * moved. Matched by exact name: only files PocketClaw is known to have
         * produced are removed, and nothing is matched by pattern or extension.
         */
        private val LEGACY_SHARED_LOG_FILES = listOf(
            "gateway.log",
            "gateway_panic.log",
            "launcher_panic.log",
        )
        private val realtimeAuthLock = Any()

        const val ACTION_START = "com.lord1egypt.pocketclaw.action.START"
        const val ACTION_STOP = "com.lord1egypt.pocketclaw.action.STOP"

        /**
         * Restart Core in one intent.
         *
         * PC-DEF-030. A configuration Core reads only at launch used to be
         * applied by sending ACTION_STOP and then ACTION_START from Flutter.
         * Two intents cannot express "restart": ACTION_STOP ends in an
         * unconditional [stopSelf], which Android honours even though a later
         * start request has already arrived, so the freshly started service is
         * destroyed again and PocketClaw is left stopped. That is what made
         * Telegram onboarding need a manual Service and Gateway restart.
         *
         * Handled here instead, where stopService() and startService() already
         * run in order on one thread and the service is never asked to stop
         * itself at all.
         */
        const val ACTION_RESTART = "com.lord1egypt.pocketclaw.action.RESTART"
        const val EXTRA_PUBLIC_MODE = "public_mode"

        // 共享状态供 UI 读取
        @Volatile
        var isRunning = false
            private set

        @Volatile
        var lastLog = ""
            private set

        // Lines emitted since the UI last collected them.
        //
        // lastLog is a sticky "most recent line" snapshot that never clears.
        // The Flutter side polls status every three seconds and used to append
        // lastLog on each poll, so one backend line was re-added forever until
        // it filled the 500-entry Logs screen and evicted the real history.
        // Handing each line out exactly once makes the producer match what the
        // consumer actually needs, and keeps genuinely repeated lines distinct
        // instead of collapsing them with a dedup heuristic.
        private val pendingLogs = ArrayDeque<String>()
        private const val MAX_PENDING_LOGS = 2000

        // Authenticates the loopback-only Android/Core bridge. This random
        // value lives for one app process, is passed only to the bundled Core
        // child process, and is never persisted or logged.
        private val androidBridgeToken: String by lazy {
            ByteArray(32).also(SecureRandom()::nextBytes).let {
                Base64.encodeToString(it, Base64.NO_WRAP or Base64.URL_SAFE)
            }
        }

        fun bridgeTokenForHost(): String = androidBridgeToken

        /**
         * Removes a service notification left behind by a process that is gone.
         *
         * PC-DEF-070. Called from `Application.onCreate`, which the platform
         * runs before any component of a newly created process: [isRunning] is
         * the default there, so the policy's answer is unconditional and it is
         * asked rather than assumed. Nothing is re-posted -- a process that has
         * only just started knows nothing about a runtime yet, and inventing a
         * replacement claim is the same mistake in the other direction.
         */
        /**
         * The service instance hosted by this process, or null when none is.
         *
         * PC-DEF-070 (reopened). Needed so a re-render goes through the live
         * instance's own derivation rather than re-deriving Running from
         * statics, which is the mistake this defect is made of. Set in
         * onCreate and cleared in onDestroy, and only ever by the instance that
         * owns the slot, so it cannot outlive its service.
         */
        @Volatile
        private var hosted: PocketClawService? = null

        /**
         * Renders the runtime notification again after the notification
         * permission may have changed.
         *
         * A post Android suppressed because `POST_NOTIFICATIONS` was not granted
         * is never retried on its own, and granting afterwards shows nothing
         * retroactively. On a fresh install the service auto-starts before the
         * dialog is answered, so the runtime ends up genuinely running with
         * nothing on screen. This is the event-driven retry: it is called when
         * the permission answer arrives, never on a timer.
         *
         * Nothing is asserted here. The instance re-derives the state from its
         * own live process, so a stopped runtime still cannot produce a Running
         * notification.
         */
        fun refreshRuntimeNotification(context: Context) {
            val service = hosted ?: return
            val enabled = try {
                context.applicationContext
                    .getSystemService(android.app.NotificationManager::class.java)
                    ?.areNotificationsEnabled() ?: false
            } catch (e: Exception) {
                Log.w(TAG, "Could not read notification enablement", e)
                return
            }
            if (!RuntimeNotificationPolicy.shouldRenderOnPermissionChange(
                    serviceHostedInThisProcess = true,
                    notificationsEnabled = enabled,
                )
            ) {
                return
            }
            service.publishRuntimeNotification(starting = !isRunning)
            Log.i(TAG, "Runtime notification re-rendered after a permission change")
        }

        fun cancelStaleRuntimeNotification(context: Context) {
            if (!RuntimeNotificationPolicy.isStaleOnProcessStart(isRunning)) return
            try {
                context.applicationContext
                    .getSystemService(android.app.NotificationManager::class.java)
                    ?.cancel(NOTIFICATION_ID)
            } catch (e: Exception) {
                Log.w(TAG, "Failed to clear a stale runtime notification", e)
            }
        }

        /**
         * Returns the installation-scoped Core realtime credential. It is
         * generated with Android's CSPRNG and stored only in app-private
         * no-backup storage; it is never copied into the public workspace,
         * included in Android backup, or written to logs.
         */
        fun pocketClawTokenForHost(context: Context): String {
            synchronized(realtimeAuthLock) {
                val tokenFile = File(
                    context.applicationContext.noBackupFilesDir,
                    REALTIME_AUTH_FILE,
                )
                if (tokenFile.isFile) {
                    tokenFile.readText(Charsets.UTF_8).trim().takeIf {
                        it.length >= 32
                    }?.let { return it }
                }

                val token = ByteArray(32).also(SecureRandom()::nextBytes).let {
                    Base64.encodeToString(
                        it,
                        Base64.NO_WRAP or Base64.NO_PADDING or Base64.URL_SAFE,
                    )
                }
                FileOutputStream(tokenFile, false).use {
                    it.write(token.toByteArray(Charsets.UTF_8))
                    it.fd.sync()
                }
                tokenFile.setReadable(false, false)
                tokenFile.setWritable(false, false)
                tokenFile.setReadable(true, true)
                tokenFile.setWritable(true, true)
                return token
            }
        }

        /** Records a line for the UI exactly once and updates the snapshot. */
        fun publishLog(line: String) {
            if (line.isEmpty()) return
            lastLog = line
            synchronized(pendingLogs) {
                pendingLogs.addLast(line)
                while (pendingLogs.size > MAX_PENDING_LOGS) {
                    pendingLogs.removeFirst()
                }
            }
        }

        /** Drains every line recorded since the previous call. */
        fun takeNewLogs(): List<String> = synchronized(pendingLogs) {
            if (pendingLogs.isEmpty()) {
                emptyList()
            } else {
                val drained = ArrayList<String>(pendingLogs)
                pendingLogs.clear()
                drained
            }
        }

        @Volatile
        var processId: Int = -1
            private set

        fun start(context: Context, publicMode: Boolean = false) {
            val intent = Intent(context, PocketClawService::class.java).apply {
                action = ACTION_START
                putExtra(EXTRA_PUBLIC_MODE, publicMode)
            }
            context.startForegroundService(intent)
        }

        fun stop(context: Context) {
            val intent = Intent(context, PocketClawService::class.java).apply {
                action = ACTION_STOP
            }
            context.startService(intent)
        }

        /**
         * Stops and starts Core without the service ever leaving the foreground.
         *
         * startForegroundService, not startService: a restart may be requested
         * while the service is already running, and the foreground form is the
         * one that is allowed either way.
         */
        fun restart(context: Context, publicMode: Boolean = false) {
            val intent = Intent(context, PocketClawService::class.java).apply {
                action = ACTION_RESTART
                putExtra(EXTRA_PUBLIC_MODE, publicMode)
            }
            context.startForegroundService(intent)
        }

        /**
         * 返回 workspace 目录路径。
         * Android 11+ 使用 MANAGE_EXTERNAL_STORAGE 权限写入 Downloads；
         * 权限未授予时回退到应用专属外部目录（无需权限）。
         */
        fun getWorkspacePath(context: Context): String {
            val downloadsDir = File(
                android.os.Environment.getExternalStoragePublicDirectory(
                    android.os.Environment.DIRECTORY_DOWNLOADS
                ),
                "pocketclaw"
            )
            // Android 11+ 需要 MANAGE_EXTERNAL_STORAGE；低版本 requestLegacyExternalStorage 已可写
            val canWrite = if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.R) {
                android.os.Environment.isExternalStorageManager()
            } else {
                true
            }
            return if (canWrite) {
                downloadsDir.mkdirs()
                downloadsDir.absolutePath
            } else {
                // 权限未授予，回退到应用专属目录，避免崩溃
                val fallback = context.getExternalFilesDir(null)?.resolve("pocketclaw")
                    ?: File(context.filesDir, "pocketclaw")
                fallback.mkdirs()
                fallback.absolutePath
            }
        }

        fun getGatewayBinaryFile(context: Context): File {
            return resolveBinaryFile(context, GATEWAY_BINARY_NAME)
        }

        /**
         * Private path Core writes the gateway bearer credential to.
         *
         * The directory, not the file, is created here: Core writes the file
         * itself on every gateway start, which is what keeps the credential
         * rotating with the process that uses it.
         */
        /**
         * Absolute path of the private gateway credential, for the one caller
         * that legitimately needs to read it.
         *
         * Exposing the path rather than the token keeps this class the only
         * thing that decides where the credential lives, and keeps the value
         * itself out of every signature.
         */
        fun gatewayTokenFilePath(context: Context): String =
            gatewayTokenFile(context).absolutePath

        private fun gatewayTokenFile(context: Context): File {
            val dir = context.applicationContext.noBackupFilesDir
            dir.mkdirs()
            return File(dir, GATEWAY_AUTH_FILE)
        }

        /** Private directory for gateway logs. */
        private fun privateLogDir(context: Context): File {
            val dir = File(context.applicationContext.noBackupFilesDir, PRIVATE_LOG_DIR)
            dir.mkdirs()
            return dir
        }

        /** Private directory for the Dashboard credential database. */
        private fun privateAuthDir(context: Context): File {
            val dir = File(context.applicationContext.noBackupFilesDir, PRIVATE_AUTH_DIR)
            dir.mkdirs()
            return dir
        }

        /**
         * Deletes the log files this app previously wrote to shared storage.
         *
         * Those files are application output, not user content, and older
         * builds wrote full LLM requests and system-prompt previews into them —
         * so an upgraded install can be carrying prompt text in a directory any
         * app with storage access can read. Nothing else under
         * `Download/pocketclaw` is touched: not the workspace, not memory, not
         * a file whose origin cannot be established.
         *
         * Best effort and idempotent. It runs after the private log directory
         * has been prepared, and a failure here must never stop the service
         * starting.
         */
        private fun removeLegacySharedLogs(context: Context) {
            try {
                val legacyDir = File(getWorkspacePath(context), "logs")
                if (!legacyDir.isDirectory) return
                LEGACY_SHARED_LOG_FILES.forEach { name ->
                    val file = File(legacyDir, name)
                    if (file.isFile) {
                        val removed = file.delete()
                        Log.i(TAG, "legacy shared log ${'$'}name removed=${'$'}removed")
                    }
                }
                // Only if our own files were all that was in it. A directory
                // holding anything else is left exactly as it is.
                if (legacyDir.list()?.isEmpty() == true) {
                    legacyDir.delete()
                }
            } catch (e: Exception) {
                Log.w(TAG, "legacy shared log cleanup skipped: ${'$'}{e.message}")
            }
        }

        fun buildEnvironment(context: Context): Map<String, String> {
            val coreState = PocketClawCoreState.directory(context)

            val workspace = File(getWorkspacePath(context))
            workspace.mkdirs()

            val tmpDir = File(context.cacheDir, "tmp")
            tmpDir.mkdirs()

            val gatewayBinaryPath = try {
                getGatewayBinaryFile(context).absolutePath
            } catch (e: Exception) {
                File(context.applicationInfo.nativeLibraryDir, GATEWAY_BINARY_NAME).absolutePath
            }
            val configPath = PocketClawCoreState.configFile(context).absolutePath

            // Managed Runtime storage. Executables live in nativeLibraryDir,
            // which the installer unpacked and the app cannot write; metadata
            // lives app-private and outside the user workspace, so a Skill
            // writing into Download/pocketclaw cannot reach runtime state.
            val runtimeLibDir = context.applicationInfo.nativeLibraryDir
            val runtimeMetadataDir = File(coreState, "runtime")
            runtimeMetadataDir.mkdirs()

            val environment = mutableMapOf(
                "HOME" to context.filesDir.absolutePath,
                "POCKETCLAW_HOME" to workspace.absolutePath,
                // The workspace stays where the user can reach it. The
                // credentials and the diagnostic log do not: all three move to
                // app-private no-backup storage, which is the boundary that
                // separates user data from runtime control state.
                "POCKETCLAW_GATEWAY_TOKEN_FILE" to gatewayTokenFile(context).absolutePath,
                "POCKETCLAW_LOG_DIR" to privateLogDir(context).absolutePath,
                "POCKETCLAW_DASHBOARD_AUTH_DIR" to privateAuthDir(context).absolutePath,
                "POCKETCLAW_CONFIG" to configPath,
                "POCKETCLAW_BINARY" to gatewayBinaryPath,
                "POCKETCLAW_RUNTIME_LIB_DIR" to runtimeLibDir,
                "POCKETCLAW_RUNTIME_DIR" to runtimeMetadataDir.absolutePath,
                "POCKETCLAW_ANDROID_BRIDGE_TOKEN" to androidBridgeToken,
                // The serialized Core channel is still named "pico", so its
                // struct tag reads PICOCLAW_CHANNELS_PICO_TOKEN. The host does
                // not emit that name: Core's canonical-env adapter translates
                // this key onto the tag. The channel itself is renamed in a
                // later phase, and the adapter table moves with it.
                "POCKETCLAW_CHANNELS_POCKETCLAW_TOKEN" to pocketClawTokenForHost(context),
                // Live channel reconciliation is a PocketClaw product behaviour:
                // saving a channel setting in the Dashboard must apply without
                // the user stopping and starting the Gateway by hand. Core
                // leaves gateway.hot_reload off by default for server
                // deployments, so this is enabled here, at the host boundary,
                // rather than by changing that default for everyone. The
                // gateway child inherits this environment from the launcher and
                // LoadConfig applies env last, so a normal install gets live
                // reconciliation with nothing written into config.json. Being
                // applied last also means this wins over the file: on Android
                // hot reload is the product behaviour, not a preference.
                "POCKETCLAW_GATEWAY_HOT_RELOAD" to "true",
                // The host-bus tools cannot work here and are not part of the
                // PocketClaw Android product surface: an unrooted phone exposes
                // no /dev/i2c-*, /dev/spidev* or /dev/tty* to an app UID, and
                // the app declares no USB host support. Core keeps them for the
                // Linux boards it targets, so they are forced off at this
                // boundary rather than by changing that default. Env is applied
                // after the file, so this also holds for a config imported from
                // another machine or hand-edited to enable them.
                "POCKETCLAW_TOOLS_I2C_ENABLED" to "false",
                "POCKETCLAW_TOOLS_SPI_ENABLED" to "false",
                "POCKETCLAW_TOOLS_SERIAL_ENABLED" to "false",
                "TMPDIR" to tmpDir.absolutePath,
                "PATH" to "/system/bin:/system/xbin",
                "LANG" to "en_US.UTF-8",
                // stdout is captured into PocketClaw's plain-text Logs screen,
                // not rendered by a terminal emulator.
                "NO_COLOR" to "1",
                "TERM" to "dumb",
                "SSL_CERT_DIR" to "/system/etc/security/cacerts",
            )
            activeNetworkDnsServers(context).takeIf { it.isNotEmpty() }?.let {
                environment["POCKETCLAW_DNS_SERVER"] = it
            }
            // The GitHub credential is decrypted here and nowhere else: the
            // Keystore key never leaves the Keystore, and the plaintext exists
            // only in this map and in the child's environment, where the runtime
            // hands it to gh and git as GH_TOKEN and an Authorization header. It
            // is read at launch, so connecting or disconnecting restarts Core.
            GitHubCredentialStore.token(context)?.let {
                environment["POCKETCLAW_GITHUB_TOKEN"] = it
            }
            // PC-DEF-060. Managed Telegram pairing from a desktop browser runs inside
            // Core, which otherwise has no idea where the onboarding service is. This
            // is the same value the Dart side compiles in, read from the same
            // dart-define, so there is one place it is configured. A public base URL,
            // never a credential: the manager bot's token is a server secret and is
            // not in the APK at all.
            BuildConfig.POCKETCLAW_ONBOARDING_BASE_URL
                .trim()
                .takeIf { it.startsWith("https://") }
                ?.let { environment["POCKETCLAW_ONBOARDING_BASE_URL"] = it }
            return environment
        }

        /**
         * Go's statically linked Android binaries cannot read Android's DNS
         * resolver state from /etc/resolv.conf. Pass the default network's
         * DNS servers to the Core process without substituting public DNS.
         */
        private fun activeNetworkDnsServers(context: Context): String {
            return try {
                val connectivity = context.getSystemService(ConnectivityManager::class.java)
                    ?: return ""
                val network = connectivity.activeNetwork ?: return ""
                val capabilities = connectivity.getNetworkCapabilities(network)
                if (capabilities != null &&
                    !capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                ) {
                    return ""
                }
                connectivity.getLinkProperties(network)?.dnsServers.orEmpty()
                    .asSequence()
                    .mapNotNull { address ->
                        val host = address.hostAddress?.trim().orEmpty()
                        when {
                            host.isEmpty() -> null
                            host.contains(':') -> "[$host]:53"
                            else -> "$host:53"
                        }
                    }
                    .distinct()
                    .joinToString(";")
            } catch (e: Exception) {
                Log.w(TAG, "Unable to read active Android DNS configuration", e)
                ""
            }
        }

        /**
         * The Core runtime version, or null when it could not be read.
         *
         * PC-DEF-063. This answered "unknown" for every failure, and Dart
         * cached that string as the version -- so one transient failure was
         * displayed as the Core version until something re-probed. Reading the
         * version means running the Core binary, which can fail transiently,
         * so a failure has to be distinguishable from an answer.
         */
        fun readCoreVersion(context: Context): String? {
            return try {
                val binaryFile = getGatewayBinaryFile(context)
                val pb = ProcessBuilder(binaryFile.absolutePath, "version")
                    .directory(context.filesDir)
                    .redirectErrorStream(true)
                pb.environment().putAll(buildEnvironment(context))

                val process = pb.start()
                val output = process.inputStream.bufferedReader().readText().trim()
                val exitCode = process.waitFor()

                if (exitCode == 0 && output.isNotBlank()) {
                    extractSemanticVersion(output)
                } else {
                    null
                }
            } catch (e: Exception) {
                Log.w(TAG, "getCoreVersion failed: ${e.message}", e)
                null
            }
        }

        private fun extractSemanticVersion(output: String): String? {
            val sanitized = output
                .replace("\r", "")
                .replace(ANSI_ESCAPE_REGEX, "")

            sanitized.lineSequence().forEach { line ->
                if (!line.contains("version", ignoreCase = true)) {
                    return@forEach
                }
                val match = SEMANTIC_VERSION_REGEX.find(line)
                if (match != null) {
                    return match.groupValues[1]
                }
            }

            return SEMANTIC_VERSION_REGEX.find(sanitized)?.groupValues?.get(1)
        }

        private fun resolveBinaryFile(context: Context, binaryName: String): File {
            val nativeLibDir = context.applicationInfo.nativeLibraryDir
            val binaryFile = File(nativeLibDir, binaryName)

            if (binaryFile.exists()) {
                Log.i(TAG, "Using ${binaryName.removePrefix("lib")} from nativeLibraryDir: ${binaryFile.absolutePath}")
                return binaryFile
            }

            Log.w(TAG, "$binaryName not found in nativeLibraryDir, trying to extract from APK")
            val extractedFile = extractBinaryFromApk(context, binaryName)
            if (extractedFile != null) {
                Log.i(TAG, "Using extracted $binaryName: ${extractedFile.absolutePath}")
                return extractedFile
            }

            val binaryLabel = if (binaryName == GATEWAY_BINARY_NAME) {
                "PocketClaw runtime"
            } else {
                "PocketClaw web runtime"
            }

            throw RuntimeException(
                "$binaryLabel binary not found. " +
                    "Tried: ${binaryFile.absolutePath} and APK extraction. " +
                    "Ensure $binaryName is placed in jniLibs/arm64-v8a/"
            )
        }

        private fun extractBinaryFromApk(context: Context, binaryName: String): File? {
            try {
                val abi = android.os.Build.SUPPORTED_ABIS.firstOrNull() ?: "arm64-v8a"
                val zipEntryPath = "lib/$abi/$binaryName"
                val outputFile = File(context.filesDir, binaryName)

                if (outputFile.exists() && outputFile.canExecute()) {
                    Log.i(TAG, "Using cached binary: ${outputFile.absolutePath}")
                    return outputFile
                }

                val apkPath = context.applicationInfo.sourceDir
                Log.i(TAG, "Extracting $binaryName from APK: $apkPath (entry: $zipEntryPath)")

                ZipFile(apkPath).use { zipFile ->
                    val entry = zipFile.getEntry(zipEntryPath)
                        ?: zipFile.getEntry("lib/arm64-v8a/$binaryName")
                        ?: zipFile.getEntry("lib/armeabi-v7a/$binaryName")
                        ?: return null

                    zipFile.getInputStream(entry).use { input ->
                        FileOutputStream(outputFile).use { output ->
                            input.copyTo(output)
                        }
                    }
                }

                outputFile.setExecutable(true)
                Log.i(TAG, "Successfully extracted $binaryName to ${outputFile.absolutePath}")
                return outputFile
            } catch (e: Exception) {
                Log.e(TAG, "Failed to extract $binaryName from APK", e)
                return null
            }
        }
    }

    private var process: Process? = null
    private var logThread: Thread? = null
    private var wakeLock: PowerManager.WakeLock? = null
    private val logBuffer = StringBuilder()
    private val maxLogSize = 64 * 1024 // 64KB 日志缓冲
    private val serviceLock = Object() // 保护启动/停止并发

    /**
     * Who owns the Core runtime. PC-DEF-072.
     *
     * Replaces the `serviceThread` field plus the single `stopped` flag. Those
     * two could disagree with reality in the same direction at the same time: a
     * timed-out join cleared the thread reference although the thread was alive,
     * and the next start cleared `stopped` for everybody, including the worker
     * that had just been abandoned. Ownership is an epoch now, and only its
     * holder may run a Core.
     */
    private val ownership = CoreRuntimeOwnership()

    /**
     * The child process the owning worker is currently blocked on — the onboard
     * run, the version probe, or the Core web process itself.
     *
     * Stop used to reach only the web process, so a worker inside `onboard`
     * (the Python payload extraction, the slow one on a fresh install) could not
     * be made to exit at all and the five-second join simply expired. Destroying
     * whichever child is live is what turns the stop from a request into an
     * event the worker cannot miss.
     */
    private var activeChild: Process? = null

    /** The epoch that registered [activeChild], so no other epoch can clear it. */
    private var activeChildEpoch: Long = 0
    @Volatile
    private var publicMode = false // 是否启用公共模式（监听所有接口）
    @Volatile
    private var gatewayAutoStart = true // 由启动偏好决定是否让 Core 自动拉起 gateway
    private var restartCount = 0
    private val maxRestartAttempts = 3 // 最大重启次数

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        hosted = this
        Log.i(TAG, "Service created")
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        // A null intent means Android re-created the Service on its own. Auto-start
        // is an app-launch decision, so an OS-driven restart must not resurrect it.
        if (intent == null) {
            Log.i(TAG, "Ignoring Android service restart with no originating intent")
            stopSelf(startId)
            return START_NOT_STICKY
        }
        when (intent.action) {
            ACTION_STOP -> {
                stopService()
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
                return START_NOT_STICKY
            }
            ACTION_RESTART -> {
                // stopService() is synchronous and bounded: it destroys the Core
                // process, joins it, and sweeps orphaned children, so the start
                // below cannot race a Core that still holds the launcher port or
                // the gateway pid file.
                publicMode = intent.getBooleanExtra(EXTRA_PUBLIC_MODE, publicMode)
                gatewayAutoStart = LaunchAutoStartPreferences.read(this).gatewayEnabled
                startForeground(NOTIFICATION_ID, createNotification("Restarting..."))
                acquireWakeLock()
                stopService()
                startService()
                return START_NOT_STICKY
            }
            else -> {
                // 从 Intent 读取 publicMode 参数
                publicMode = intent.getBooleanExtra(EXTRA_PUBLIC_MODE, false)
                gatewayAutoStart = LaunchAutoStartPreferences.read(this).gatewayEnabled
                startForeground(NOTIFICATION_ID, createNotification("Starting..."))
                acquireWakeLock()
                startService()
                return START_NOT_STICKY
            }
        }
    }

    override fun onDestroy() {
        stopService()
        releaseWakeLock()
        isRunning = false
        // PC-DEF-070. Only ACTION_STOP used to remove the notification, so every
        // other way this service ends -- a stopSelf from elsewhere, the system
        // tearing it down -- relied on the platform cancelling it for us. It is
        // removed here instead, because the one thing the notification must never
        // do is outlive the runtime it describes.
        stopForeground(STOP_FOREGROUND_REMOVE)
        // Only ever released by the instance that owns the slot.
        if (hosted === this) hosted = null
        Log.i(TAG, "Service destroyed")
        super.onDestroy()
    }

    // --- 核心逻辑 ---

    private fun startService() {
        val outcome = ownership.start { epoch ->
            restartCount = 0
            Thread({ runCoreRuntime(epoch) }, "pocketclaw-core-runtime-$epoch")
                .also { it.start() }
        }
        when (outcome) {
            is CoreRuntimeOwnership.StartOutcome.Started ->
                Log.i(TAG, "Core runtime epoch ${outcome.epoch} started")
            CoreRuntimeOwnership.StartOutcome.AlreadyRunning ->
                Log.w(TAG, "Core runtime is already running, ignoring duplicate start request")
            CoreRuntimeOwnership.StartOutcome.Queued -> {
                // PC-DEF-072. The previous owner has been asked to stop and has
                // not released. Starting here would put a second Core on port
                // 18800; dropping the request would make a restart silently do
                // nothing. It runs when that owner releases, and the state is
                // said out loud rather than left looking idle.
                Log.w(TAG, "Previous Core runtime has not released; start queued behind it")
                publishLog("Waiting for the previous PocketClaw runtime to exit before restarting")
                publishRuntimeNotification(starting = true)
            }
        }
    }

    /**
     * One Core runtime, from binary probe to exit, owned by [epoch].
     *
     * Every step that can outlive a stop re-asks whether [epoch] is still the
     * live runtime, and the `finally` is the only place ownership is released —
     * so a worker that took longer than the stop's join still cleans up after
     * itself, and a start queued behind it runs then rather than never.
     */
    private fun runCoreRuntime(epoch: Long) {
        try {
            val gatewayBinary = getGatewayBinaryFile()
            testBinary(epoch, gatewayBinary)
            ensureOnboarded(epoch, gatewayBinary)
            if (!ownership.isCurrent(epoch)) return
            // 启动前先清理可能残留的旧进程
            killPocketClawOrphanProcesses()
            runWebService(epoch)
        } catch (e: InterruptedException) {
            Log.i(TAG, "Core runtime epoch $epoch interrupted during stop")
            Thread.currentThread().interrupt()
        } catch (e: Exception) {
            if (ownership.isCurrent(epoch)) {
                Log.e(TAG, "Failed to start service", e)
                publishLog("Error: ${e.message}")
                updateNotification("Error: ${e.message}")
            } else {
                Log.i(TAG, "Core runtime epoch $epoch failed after it stopped being current", e)
            }
        } finally {
            clearActiveChild(epoch)
            // The stopper never releases ownership: a bounded join that expired
            // is not evidence that this thread is gone. This is.
            val runQueued = ownership.release(epoch)
            Log.i(TAG, "Core runtime epoch $epoch released (queued start: $runQueued)")
            if (runQueued) {
                // Clear the interrupt first: the queued start must not inherit
                // the stop that was aimed at this worker.
                Thread.interrupted()
                startService()
            }
        }
    }

    /**
     * 测试 gateway 二进制是否可执行
     */
    private fun testBinary(epoch: Long, binaryFile: File) {
        Log.i(TAG, "Testing binary at ${binaryFile.absolutePath}...")
        val env = buildEnvironment()

        val pb = ProcessBuilder(binaryFile.absolutePath, "version")
            .directory(filesDir)
            .redirectErrorStream(true)
        pb.environment().putAll(env)

        try {
            val proc = pb.start()
            if (!adoptActiveChild(epoch, proc)) {
                // Not the live runtime any more: do not leave the child behind.
                proc.destroyForcibly()
                throw InterruptedException("Core runtime epoch $epoch stopped during the binary probe")
            }
            val output = proc.inputStream.bufferedReader().readText()
            val exitCode = proc.waitFor()
            clearActiveChild(epoch)
            Log.i(TAG, "Binary test: exit=$exitCode, output=$output")

            if (!ownership.isCurrent(epoch)) {
                throw InterruptedException("Core runtime epoch $epoch stopped during the binary probe")
            }
            if (exitCode != 0) {
                throw RuntimeException(
                    "Core binary test failed (exit $exitCode): $output"
                )
            }
        } catch (e: java.io.IOException) {
            throw RuntimeException(
                "Cannot execute Core binary at ${binaryFile.absolutePath}: ${e.message}", e
            )
        }
    }

    /**
     * 从 app 的 native library 目录获取 gateway 二进制（用于 onboard 初始化和传递给 web 服务）
     * 如果 nativeLibraryDir 中没有，尝试从 APK 中提取
     */
    private fun getGatewayBinaryFile(): File {
        return Companion.getGatewayBinaryFile(this)
    }

    /**
     * 从 app 的 native library 目录获取 web console 二进制
     * 如果 nativeLibraryDir 中没有，尝试从 APK 中提取
     */
    private fun getWebBinaryFile(): File {
        return Companion.resolveBinaryFile(this, WEB_BINARY_NAME)
    }

    /**
     * 运行 Core 的 `onboard` 初始化配置和工作区
     */
    private fun ensureOnboarded(epoch: Long, binaryFile: File) {
        val configFile = PocketClawCoreState.configFile(this)

        if (configFile.exists()) {
            Log.i(TAG, "Config already exists, skipping onboard")
            return
        }

        Log.i(TAG, "Running onboard...")
        updateNotification("Initializing...")

        val env = buildEnvironment()

        val pb = ProcessBuilder(binaryFile.absolutePath, "onboard")
            .directory(filesDir)
            .redirectErrorStream(true)

        pb.environment().putAll(env)

        val proc = pb.start()
        // PC-DEF-072. This is the slow step on a fresh install -- it extracts the
        // Python payload -- and it is the one a stop could not previously reach,
        // so the five-second join simply expired against it. Registering the
        // child is what lets stopService() destroy it.
        if (!adoptActiveChild(epoch, proc)) {
            proc.destroyForcibly()
            throw InterruptedException("Core runtime epoch $epoch stopped before onboard")
        }
        val output = proc.inputStream.bufferedReader().readText()
        val exitCode = proc.waitFor()
        clearActiveChild(epoch)

        Log.i(TAG, "Onboard exit code: $exitCode, output: $output")

        if (!ownership.isCurrent(epoch)) {
            throw InterruptedException("Core runtime epoch $epoch stopped during onboard")
        }
        if (exitCode != 0) {
            throw RuntimeException("Onboard failed (exit $exitCode): $output")
        }
    }

    /**
     * Registers [proc] as the child [epoch] is blocked on, or reports that
     * [epoch] is no longer the live runtime.
     *
     * The currency check and the registration happen under one lock, so a stop
     * either sees this child and destroys it or is seen here and refuses it.
     * There is no order in which a child is both unregistered and kept.
     */
    private fun adoptActiveChild(epoch: Long, proc: Process): Boolean =
        synchronized(serviceLock) {
            if (!ownership.isCurrent(epoch)) return false
            activeChild = proc
            activeChildEpoch = epoch
            true
        }

    /** Drops the registered child, but only the one this epoch registered. */
    private fun clearActiveChild(epoch: Long) {
        synchronized(serviceLock) {
            if (activeChildEpoch == epoch) {
                activeChild = null
                activeChildEpoch = 0
            }
        }
    }

    /**
     * 运行 web 服务进程（libpocketclaw-web.so）
     * web 服务会通过 TryAutoStartGateway() 自动启动并管理 gateway
     */
    private fun runWebService(epoch: Long) {
        // 检查是否已被要求停止
        if (!ownership.isCurrent(epoch)) {
            Log.i(TAG, "Core runtime epoch $epoch is no longer current, aborting web service start")
            return
        }

        val webBinaryFile = getWebBinaryFile()
        val configFile = PocketClawCoreState.configFile(this)
        val env = buildEnvironment()

        val cmdList = mutableListOf(
            webBinaryFile.absolutePath,
            "--console",
            "--no-browser"
        )

        // The native toggle is the authority for the dashboard listener, so the
        // decision is always passed explicitly -- including when it is off.
        //
        // Omitting the flag is not "off". With no -public on the command line
        // the backend falls back to launcher-config.json's `public` field, and
        // the dashboard's own Config page can persist true there. Turning the
        // native toggle off then rebound the live listener to loopback and left
        // that true behind, so the next service start bound the console to
        // every interface while this toggle still read OFF. Passing the value
        // explicitly makes Go's flag.Visit see it, which is what stops the
        // backend consulting the persisted field at all. See PC-DEF-020.
        cmdList.add("-public=" + publicMode)
        Log.i(TAG, "Public mode explicit: $publicMode")

        cmdList.addAll(listOf("-port", WEB_PORT.toString(), configFile.absolutePath))

        val pb = ProcessBuilder(cmdList)
            .directory(filesDir)
            .redirectErrorStream(true)

        pb.environment().putAll(env)

        Log.i(TAG, "Starting web service on port $WEB_PORT...")
        updateNotification("Starting web service...")

        val proc = pb.start()
        // PC-DEF-072. The only place a Core web process becomes this service's
        // runtime. The currency check and the registration are one critical
        // section, so an epoch that lost ownership between pb.start() and here
        // destroys what it spawned instead of publishing a second Core on
        // port 18800.
        synchronized(serviceLock) {
            if (!ownership.isCurrent(epoch)) {
                Log.i(TAG, "Core runtime epoch $epoch stopped during startup, killing new process")
                proc.destroyForcibly()
                return
            }
            process = proc
            activeChild = proc
            activeChildEpoch = epoch
            isRunning = true
        }

        // The runtime is up and its environment already points at the private
        // log directory, so anything it writes from here lands there. Only now
        // is it safe to remove what the old shared location still holds.
        removeLegacySharedLogs(this)

        processId = try {
            val pidField = proc.javaClass.getDeclaredField("pid")
            pidField.isAccessible = true
            pidField.getInt(proc)
        } catch (e: Exception) {
            -1
        }

        publishRuntimeNotification()
        Log.i(TAG, "Web service started with PID: $processId, listening on port $WEB_PORT")

        // 后台线程读取 stdout/stderr
        logThread = Thread({
            try {
                val reader = BufferedReader(InputStreamReader(proc.inputStream))
                var line: String?
                while (reader.readLine().also { line = it } != null) {
                    val logLine = line ?: continue
                    Log.d(TAG, logLine)
                    appendLog(logLine)
                }
            } catch (e: Exception) {
                if (ownership.isCurrent(epoch)) {
                    Log.w(TAG, "Log reader interrupted", e)
                }
            }
        }, "pocketclaw-web-log-reader").apply {
            isDaemon = true
            start()
        }

        // 等待进程退出（阻塞）
        val exitCode = proc.waitFor()
        isRunning = false
        processId = -1
        clearActiveChild(epoch)

        try { logThread?.join(2000) } catch (_: InterruptedException) {
            Thread.currentThread().interrupt()
        }

        // 如果是被主动停止的，不需要重启
        if (!ownership.isCurrent(epoch)) {
            Log.i(TAG, "Web service for epoch $epoch exited after it stopped being current (code $exitCode)")
            return
        }

        val lastOutput = logBuffer.toString().takeLast(500)
        Log.w(TAG, "Web service exited with code: $exitCode, last output: $lastOutput")
        publishLog("Process exited (code $exitCode)\n$lastOutput")
        publishRuntimeNotification(stoppedDetail = "Stopped (exit code $exitCode)")

        // 非正常退出时自动重启（限制重试次数）
        if (exitCode != 0) {
            restartCount++
            if (restartCount > maxRestartAttempts) {
                Log.e(TAG, "Web service has failed $restartCount times, giving up restart")
                publishLog("Service crashed $restartCount times, stopped retrying")
                updateNotification("Error: too many restarts")
                return
            }
            Log.i(TAG, "Scheduling restart in 5 seconds... (attempt $restartCount/$maxRestartAttempts)")
            // 清理可能残留的占用端口的进程
            killPocketClawOrphanProcesses()
            // Interruptible on purpose: a sleeping worker owns no child process,
            // so the interrupt from stopService() is the only thing that can
            // reach it. Letting it propagate is what ends the epoch promptly
            // instead of after the full backoff.
            Thread.sleep(RESTART_BACKOFF_MS)
            // 再次检查是否被要求停止
            if (!ownership.isCurrent(epoch)) {
                Log.i(TAG, "Core runtime epoch $epoch stopped during the restart wait, aborting")
                return
            }
            runWebService(epoch)
        }
    }

    /**
     * 从 APK 中提取二进制文件到 filesDir
     * 用于某些设备（特别是 TV）so 文件没有被自动解压到 nativeLibraryDir 的情况
     */
    /**
     * 杀掉属于当前应用的所有 PocketClaw Core 残留子进程。
     *
     * 通过 UID 匹配（而非 ppid），因为 force-stop 后 app 重启 PID 会变，
     * 旧的孤儿进程的 ppid 可能已变为 1（被 init 收养），无法通过 ppid 找到。
     */
    /**
     * Whether a /proc/<pid>/cmdline belongs to one of PocketClaw's own Core
     * executables.
     *
     * Matched on the basename of argv[0] against [GATEWAY_BINARY_NAME] and
     * [WEB_BINARY_NAME] — the same constants used to spawn them, so orphan
     * cleanup and the names it is cleaning up cannot drift apart.
     *
     * Exact comparison, deliberately. `contains("picoclaw")` is what this
     * replaced and it stopped matching anything once the binaries were renamed
     * in N3. `contains("pocketclaw")` or a "libpocketclaw" prefix would swing
     * the other way and sweep in every Managed Runtime payload — gh, git,
     * python, curl, rg, jq, sqlite3, git-remote-http are all libpocketclaw-*
     * and are ours, but they are not orphaned Core processes and killing one
     * mid-operation would look like a random tool failure. The application
     * process is excluded for the same reason.
     */
    private fun isCoreExecutableCommandLine(cmdline: String): Boolean {
        val argv0 = cmdline.substringBefore('\u0000').trim()
        if (argv0.isEmpty()) return false
        val basename = argv0.substringAfterLast('/')
        return basename == GATEWAY_BINARY_NAME || basename == WEB_BINARY_NAME
    }

    private fun killPocketClawOrphanProcesses() {
        try {
            val myPid = android.os.Process.myPid()
            val myUid = android.os.Process.myUid()
            val procDir = File("/proc")
            procDir.listFiles()?.forEach { pidDir ->
                val pid = pidDir.name.toIntOrNull() ?: return@forEach
                if (pid == myPid) return@forEach // 不杀自己
                try {
                    // 通过 /proc/<pid>/status 读取进程的 UID
                    val statusFile = File(pidDir, "status")
                    if (!statusFile.canRead()) return@forEach
                    val statusContent = statusFile.readText()

                    // 解析 Uid 行：Uid:\t<real>\t<effective>\t<saved>\t<filesystem>
                    val uidLine = statusContent.lineSequence()
                        .firstOrNull { it.startsWith("Uid:") } ?: return@forEach
                    val uidFields = uidLine.substringAfter("Uid:").trim().split(Regex("\\s+"))
                    val processUid = uidFields.firstOrNull()?.toIntOrNull() ?: return@forEach

                    // 只处理属于同一 UID（同一应用）的进程
                    if (processUid != myUid) return@forEach

                    // 只匹配 Core 可执行文件本身，按 argv[0] 的 basename 精确比对。
                    val cmdlineFile = File(pidDir, "cmdline")
                    if (!cmdlineFile.canRead()) return@forEach
                    val cmdline = cmdlineFile.readText()
                    if (!isCoreExecutableCommandLine(cmdline)) return@forEach

                    Log.i(TAG, "Killing orphan PocketClaw Core process: PID=$pid, UID=$processUid, cmd=$cmdline")
                    android.os.Process.killProcess(pid)
                } catch (e: Exception) {
                    // 忽略无权限的进程
                }
            }
            Log.i(TAG, "Cleaned up orphan PocketClaw Core processes")
        } catch (e: Exception) {
            Log.w(TAG, "Failed to cleanup orphan processes: ${e.message}")
        }
    }

    /**
     * 停止服务（web 进程会在退出时自动停止其管理的 gateway）
     */
    private fun stopService() {
        Log.i(TAG, "Stopping service...")

        // Name the exact owner this stop is aimed at, and mark it stopping, before
        // touching anything it owns.
        val target = ownership.requestStop()

        val child = synchronized(serviceLock) {
            val live = process ?: activeChild
            process = null
            activeChild = null
            activeChildEpoch = 0
            isRunning = false
            processId = -1
            logThread?.interrupt()
            logThread = null
            live
        }

        // Destroy whichever child the worker is blocked on -- the web process,
        // or the onboard/version run that stop could not previously reach at all.
        child?.let { proc ->
            try {
                proc.destroy()

                val waiter = Thread {
                    try {
                        proc.waitFor()
                    } catch (_: InterruptedException) {
                    }
                }
                waiter.start()
                waiter.join(10_000)

                if (proc.isAlive) {
                    Log.w(TAG, "Force killing Core child process")
                    proc.destroyForcibly()
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error stopping Core child process", e)
            }
        }

        // 等待服务线程退出
        if (target != null && target.thread !== Thread.currentThread()) {
            // Interrupt as well as destroy: the restart backoff is a sleep, and a
            // sleeping worker has no child to kill.
            target.thread.interrupt()
            try {
                target.thread.join(SERVICE_THREAD_JOIN_MS)
            } catch (_: InterruptedException) {
                Thread.currentThread().interrupt()
            }
            if (target.thread.isAlive) {
                // PC-DEF-072. The join expired. Ownership is NOT cleared here:
                // clearing it is what let the next start put a second Core on
                // port 18800. The worker releases from its own finally, and a
                // start requested in the meantime is queued behind it.
                Log.w(TAG, "Core runtime epoch ${target.epoch} did not exit within the join; " +
                    "ownership retained until it releases")
                publishLog("PocketClaw runtime is still shutting down")
            }
        }

        // 清理可能残留的孤儿进程（包括 web 服务自己启动的 gateway）
        killPocketClawOrphanProcesses()

        // 重置重启计数
        restartCount = 0

        Log.i(TAG, "Service stopped and cleaned up")
    }

    // --- 环境变量 ---

    /**
     * 构建子进程环境变量
     * 关键：设置 POCKETCLAW_BINARY 指向 gateway 二进制，让 web 服务能找到并启动 gateway
     */
    private fun buildEnvironment(): Map<String, String> {
        return Companion.buildEnvironment(this).toMutableMap().apply {
            put("POCKETCLAW_GATEWAY_AUTOSTART", gatewayAutoStart.toString())
        }
    }

    // --- 通知 ---

    private fun createNotification(status: String): Notification {
        val launchIntent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_SINGLE_TOP
        }
        val pendingIntent = PendingIntent.getActivity(
            this, 0, launchIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val stopIntent = Intent(this, PocketClawService::class.java).apply {
            action = ACTION_STOP
        }
        val stopPendingIntent = PendingIntent.getService(
            this, 1, stopIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        return NotificationCompat.Builder(this, PocketClawApp.CHANNEL_ID)
            .setContentTitle("PocketClaw")
            .setContentText(status)
            .setSmallIcon(com.lord1egypt.pocketclaw.R.drawable.ic_stat_pocketclaw)
            .setContentIntent(pendingIntent)
            .addAction(android.R.drawable.ic_media_pause, "Stop", stopPendingIntent)
            .setOngoing(true)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .setForegroundServiceBehavior(NotificationCompat.FOREGROUND_SERVICE_IMMEDIATE)
            .build()
    }

    private fun updateNotification(status: String) {
        try {
            val notification = createNotification(status)
            val manager = getSystemService(android.app.NotificationManager::class.java)
            manager.notify(NOTIFICATION_ID, notification)
        } catch (e: Exception) {
            Log.w(TAG, "Failed to update notification", e)
        }
    }

    /**
     * Posts the runtime line, deriving Running rather than asserting it.
     *
     * PC-DEF-070. Every caller that wants to say the runtime is up goes through
     * here, and here the claim is checked against the live process under the
     * same lock that owns it. The old call sites wrote "Running (PID: n)"
     * straight out at the moment the process was spawned, so a stop that landed
     * immediately afterwards -- or a service thread orphaned by a stop whose
     * join timed out -- left that sentence on screen with nothing behind it.
     *
     * [starting] says what to show when no process is up yet: the service is
     * still coming up, or it is genuinely down. Neither can produce Running.
     */
    private fun publishRuntimeNotification(starting: Boolean = false, stoppedDetail: String? = null) {
        val (alive, pid) = synchronized(serviceLock) {
            (process?.isAlive == true) to processId
        }
        when (RuntimeNotificationPolicy.resolve(processAlive = alive, starting = starting)) {
            RuntimeNotificationState.RUNNING -> updateNotification("Running (PID: $pid)")
            RuntimeNotificationState.STARTING -> updateNotification("Starting...")
            RuntimeNotificationState.STOPPED ->
                updateNotification(stoppedDetail ?: "Stopped")
        }
    }

    // --- Wake Lock ---

    private fun acquireWakeLock() {
        val pm = getSystemService(Context.POWER_SERVICE) as PowerManager
        wakeLock = pm.newWakeLock(
            PowerManager.PARTIAL_WAKE_LOCK,
            "PocketClaw::ServiceWakeLock"
        ).apply {
            acquire(24 * 60 * 60 * 1000L) // 24 小时上限
        }
        Log.i(TAG, "Wake lock acquired")
    }

    private fun releaseWakeLock() {
        wakeLock?.let {
            if (it.isHeld) {
                it.release()
                Log.i(TAG, "Wake lock released")
            }
        }
        wakeLock = null
    }

    // --- 日志缓冲 ---

    @Synchronized
    private fun appendLog(line: String) {
        logBuffer.appendLine(line)
        if (logBuffer.length > maxLogSize) {
            logBuffer.delete(0, logBuffer.length - maxLogSize)
        }
        publishLog(line)
    }

    @Synchronized
    fun getFullLog(): String = logBuffer.toString()
}

package com.lord1egypt.pocketclaw

import com.lord1egypt.pocketclaw.media.ChatImagePicker
import com.lord1egypt.pocketclaw.security.GitHubCredentialStore
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.os.Build
import android.os.Environment
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import android.util.Log
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import com.lord1egypt.pocketclaw.service.PocketClawService
import com.lord1egypt.pocketclaw.service.LaunchAutoStartPreferences
import com.lord1egypt.pocketclaw.util.HealthChecker
import org.json.JSONObject
import java.io.File
import java.io.IOException
import java.net.HttpURLConnection
import java.net.Inet4Address
import java.net.URL
import java.util.concurrent.Executor

/**
 * Flutter MethodChannel 桥接层，将 Kotlin 原生功能暴露给 Dart 端。
 *
 * 支持的方法：
 * - startService: 启动 PocketClaw 前台服务
 * - stopService: 停止 PocketClaw 前台服务
 * - getServiceStatus: 获取服务状态（isRunning, pid, lastLog）
 * - checkHealth: 检查 /health 端点
 * - getConfig: 读取 config.json 内容
 * - saveConfig: 保存 config.json 内容
 * - getFullLog: 获取完整日志
 * - setAutoStart: 设置开机自启
 * - getAutoStart: 获取开机自启设置
 * - getWebPort: 获取 Web Console 端口号
 */
class PocketClawMethodChannel(
    private val context: Context,
    flutterEngine: FlutterEngine,
    /** Absent on hosts with no Activity to receive a picker result. */
    private val chatImagePicker: ChatImagePicker? = null
) {
    companion object {
        private const val TAG = "PocketClawMethodChannel"
        private const val CHANNEL_NAME = "com.lord1egypt.pocketclaw/pocketclaw"
        private const val KEY_AUTO_START = "auto_start"
        private const val TELEGRAM_BRIDGE_URL =
            "http://127.0.0.1:18800/api/pocketclaw/android/telegram"
        private const val NETWORK_MODE_BRIDGE_URL =
            "http://127.0.0.1:18800/api/pocketclaw/android/network-mode"
        private const val CONTEXT_MEMORY_BRIDGE_URL =
            "http://127.0.0.1:18800/api/pocketclaw/android/context-memory"
        private const val GITHUB_VALIDATE_BRIDGE_URL =
            "http://127.0.0.1:18800/api/pocketclaw/android/github/validate"
        private const val GITHUB_STATUS_BRIDGE_URL =
            "http://127.0.0.1:18800/api/pocketclaw/android/github/status"
        private const val GATEWAY_START_BRIDGE_URL =
            "http://127.0.0.1:18800/api/pocketclaw/android/gateway/start"
        private const val DASHBOARD_AUTH_STATUS_URL =
            "http://127.0.0.1:18800/api/auth/status"
    }

    // Copy a content:// URI to the app cache and return the absolute file path.
    private fun copyContentUriToCache(uriStr: String, fileName: String): String? {
        try {
            if (uriStr.isBlank()) return null
            val uri = android.net.Uri.parse(uriStr)
            val resolver = context.contentResolver
            resolver.openInputStream(uri).use { input ->
                if (input == null) return null
                val cacheDir = context.cacheDir
                val outFile = java.io.File(cacheDir, fileName)
                input.use { inp ->
                    outFile.outputStream().use { out ->
                        inp.copyTo(out)
                        out.flush()
                    }
                }
                return outFile.absolutePath
            }
        } catch (e: Exception) {
            Log.w(TAG, "copyContentUriToCache failed: ${e.message}")
            return null
        }
    }

    private val channel = MethodChannel(flutterEngine.dartExecutor.binaryMessenger, CHANNEL_NAME)
    private val healthChecker = HealthChecker.forHost(context)

    private fun getMainExecutor(): Executor {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            context.mainExecutor
        } else {
            val handler = Handler(Looper.getMainLooper())
            Executor { r -> handler.post(r) }
        }
    }

    init {
        channel.setMethodCallHandler { call, result ->
            when (call.method) {
                "startService" -> {
                    try {
                        // 从参数中读取 publicMode，默认为 false
                        val args = call.argument<String>("args") ?: ""
                        val publicMode = args.contains("-public")
                        // 保存 publicMode 到 SharedPreferences
                        val prefs = PocketClawPreferences.open(context)
                        prefs.edit().putBoolean("public_mode", publicMode).apply()
                        Log.d(TAG, "Starting service with publicMode=$publicMode (args: $args)")
                        PocketClawService.start(context, publicMode)
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("START_FAILED", e.message, null)
                    }
                }
                "dashboardAuthInitialized" -> {
                    // PC-DEF-040. Read once, at a host lifecycle transition --
                    // never polled. /api/auth/status is unauthenticated and
                    // carries no secret: it answers whether an owner exists,
                    // which is exactly what reconciliation needs to know.
                    try {
                        result.success(readDashboardAuthInitialized())
                    } catch (e: Exception) {
                        result.error(
                            "DASHBOARD_AUTH_STATUS_FAILED",
                            e.message ?: "Could not read dashboard auth status",
                            null,
                        )
                    }
                }
                "getPublicMode" -> {
                    try {
                        val prefs = PocketClawPreferences.open(context)
                        result.success(prefs.getBoolean("public_mode", false))
                    } catch (e: Exception) {
                        result.error("GET_PUBLIC_MODE_FAILED", e.message, null)
                    }
                }
                "applyPublicMode" -> {
                    val publicMode = call.argument<Boolean>("public") ?: false
                    Thread {
                        val mainExecutor = getMainExecutor()
                        val previousMode = PocketClawPreferences.open(context)
                            .getBoolean("public_mode", false)
                        try {
                            val accepted = callNetworkModeBridge(
                                "PUT",
                                JSONObject().put("public", publicMode),
                                HttpURLConnection.HTTP_ACCEPTED,
                            )
                            if (accepted.optString("status") != "applying") {
                                throw IllegalStateException("Dashboard did not accept the network mode change")
                            }

                            val deadline = System.currentTimeMillis() + 10_000
                            var finalState: JSONObject? = null
                            while (System.currentTimeMillis() < deadline) {
                                Thread.sleep(100)
                                try {
                                    val state = callNetworkModeBridge(
                                        "GET",
                                        null,
                                        HttpURLConnection.HTTP_OK,
                                    )
                                    when (state.optString("status")) {
                                        "succeeded", "failed" -> {
                                            finalState = state
                                            break
                                        }
                                    }
                                } catch (_: IOException) {
                                    // Connection resets/timeouts are expected briefly while
                                    // port 18800 is being rebound.
                                }
                            }

                            val state = finalState
                                ?: throw IllegalStateException("Dashboard network mode change timed out")
                            val actualMode = state.optBoolean("public", previousMode)
                            val success = state.optString("status") == "succeeded" &&
                                actualMode == publicMode
                            if (success) {
                                PocketClawPreferences.open(context)
                                    .edit()
                                    .putBoolean("public_mode", actualMode)
                                    .apply()
                            }
                            mainExecutor.execute {
                                result.success(mapOf(
                                    "success" to success,
                                    "public" to actualMode,
                                    "message" to state.optString("error", ""),
                                ))
                            }
                        } catch (e: Exception) {
                            Log.w(TAG, "Dashboard network mode change failed: ${e.message}")
                            mainExecutor.execute {
                                result.success(mapOf(
                                    "success" to false,
                                    "public" to previousMode,
                                    "message" to if (publicMode) {
                                        "Could not enable LAN access. PocketClaw remains available locally."
                                    } else {
                                        "Could not disable LAN access. PocketClaw remains in its previous network mode."
                                    },
                                ))
                            }
                        }
                    }.start()
                }
                "stopService" -> {
                    try {
                        PocketClawService.stop(context)
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("STOP_FAILED", e.message, null)
                    }
                }
                "restartService" -> {
                    // One intent, because two cannot express a restart. See
                    // PocketClawService.ACTION_RESTART. PC-DEF-030.
                    try {
                        val args = call.argument<String>("args") ?: ""
                        val publicMode = args.contains("-public")
                        val prefs = PocketClawPreferences.open(context)
                        prefs.edit().putBoolean("public_mode", publicMode).apply()
                        PocketClawService.restart(context, publicMode)
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("RESTART_FAILED", e.message, null)
                    }
                }
                "getServiceStatus" -> {
                    result.success(mapOf(
                        "isRunning" to PocketClawService.isRunning,
                        "pid" to PocketClawService.processId,
                        "lastLog" to PocketClawService.lastLog
                    ))
                }
                "checkHealth" -> {
                    // `detail` is opt-in and only the Status screen asks for
                    // it, so an unwatched Status tab costs exactly what this
                    // poll has always cost.
                    val wantDetail = call.argument<Boolean>("detail") ?: false
                    Thread {
                        val mainExecutor = getMainExecutor()

                        try {
                            val status = healthChecker.check(wantDetail)
                            val resultMap = mapOf(
                                "isHealthy" to status.isHealthy,
                                "status" to status.status,
                                "uptime" to status.uptime,
                                "pid" to status.pid,
                                "error" to (status.error ?: ""),
                                // Raw Status JSON, or null when detail was not
                                // requested or not available. The bearer token
                                // used to fetch it never crosses this boundary.
                                "detail" to status.detailJson
                            )
                            mainExecutor.execute {
                                result.success(resultMap)
                            }
                        } catch (e: Exception) {
                            mainExecutor.execute {
                                result.error("HEALTH_CHECK_FAILED", e.message, null)
                            }
                        }
                    }.start()
                }
                "getConfig" -> {
                    try {
                        val configFile = PocketClawCoreState.configFile(context)
                        if (configFile.exists()) {
                            result.success(configFile.readText())
                        } else {
                            result.success("")
                        }
                    } catch (e: Exception) {
                        result.error("READ_CONFIG_FAILED", e.message, null)
                    }
                }
                "saveConfig" -> {
                    try {
                        val content = call.argument<String>("content") ?: ""
                        val configFile = PocketClawCoreState.configFile(context)
                        configFile.writeText(content)
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("SAVE_CONFIG_FAILED", e.message, null)
                    }
                }
                "getFullLog" -> {
                    result.success(PocketClawService.lastLog)
                }
                "takeNewLogs" -> {
                    // Each line is delivered once. See PocketClawService.publishLog.
                    result.success(PocketClawService.takeNewLogs())
                }
                "configureTelegram" -> {
                    val token = call.argument<String>("token") ?: ""
                    val ownerUserId = call.argument<Number>("ownerUserId")?.toLong() ?: 0L
                    Thread {
                        val mainExecutor = getMainExecutor()
                        try {
                            val body = JSONObject()
                                .put("token", token)
                                .put("owner_user_id", ownerUserId)
                            callTelegramBridge("PUT", body)
                            mainExecutor.execute { result.success(true) }
                        } catch (e: Exception) {
                            mainExecutor.execute {
                                result.error(
                                    "TELEGRAM_CONFIG_FAILED",
                                    "Core rejected the Telegram configuration",
                                    null
                                )
                            }
                        }
                    }.start()
                }
                "pickChatImage" -> {
                    val picker = chatImagePicker
                    if (picker == null) {
                        result.success(null)
                        return@setMethodCallHandler
                    }
                    val acceptTypes = call.argument<List<String>>("acceptTypes").orEmpty()
                    picker.pick(acceptTypes) { outcome ->
                        outcome.fold(
                            onSuccess = { uri -> result.success(uri) },
                            onFailure = { failure ->
                                result.error(
                                    failure.message ?: ChatImagePicker.ERROR_UNREADABLE,
                                    "That image could not be attached",
                                    null,
                                )
                            },
                        )
                    }
                }
                "getGitHubStatus" -> {
                    try {
                        result.success(GitHubCredentialStore.status(context).asMap())
                    } catch (e: Exception) {
                        result.error(
                            "GITHUB_STATUS_FAILED",
                            "Could not read the GitHub connection state",
                            null,
                        )
                    }
                }
                "connectGitHub" -> {
                    // The candidate is checked by Core, which is the only part of
                    // PocketClaw allowed to run the bundled gh, and is stored only
                    // if GitHub accepts it. Nothing is written on failure, and the
                    // token is never returned to Flutter.
                    val candidate = call.argument<String>("token").orEmpty().trim()
                    if (candidate.isEmpty()) {
                        result.error("GITHUB_TOKEN_REQUIRED", "A GitHub token is required", null)
                        return@setMethodCallHandler
                    }
                    Thread {
                        val mainExecutor = getMainExecutor()
                        try {
                            val login = callGitHubBridge(
                                url = GITHUB_VALIDATE_BRIDGE_URL,
                                method = "POST",
                                body = JSONObject().put("token", candidate),
                            ).optString("login")
                            GitHubCredentialStore.connect(context, candidate, login)
                            mainExecutor.execute {
                                result.success(mapOf("connected" to true, "login" to login))
                            }
                        } catch (e: Exception) {
                            // e.message comes from Core, which scrubs the candidate
                            // out of anything gh said before replying.
                            val reason = e.message ?: "GitHub did not accept this token"
                            mainExecutor.execute {
                                result.error("GITHUB_CONNECT_FAILED", reason, null)
                            }
                        }
                    }.start()
                }
                "getTelegramContextMemory" -> {
                    // Core owns config.json, so Settings asks Core what the
                    // effective value is rather than parsing the file here.
                    Thread {
                        val mainExecutor = getMainExecutor()
                        try {
                            val reply = callGitHubBridge(
                                url = CONTEXT_MEMORY_BRIDGE_URL,
                                method = "GET",
                                body = null,
                            )
                            mainExecutor.execute {
                                result.success(
                                    mapOf(
                                        "recentMessages" to reply.optInt("recent_messages"),
                                        "min" to reply.optInt("min"),
                                        "max" to reply.optInt("max"),
                                        "default" to reply.optInt("default"),
                                    )
                                )
                            }
                        } catch (e: Exception) {
                            mainExecutor.execute {
                                result.error(
                                    "CONTEXT_MEMORY_READ_FAILED",
                                    e.message ?: "Could not read the context memory setting",
                                    null,
                                )
                            }
                        }
                    }.start()
                }
                "setTelegramContextMemory" -> {
                    val recentMessages = call.argument<Number>("recentMessages")?.toInt() ?: 0
                    Thread {
                        val mainExecutor = getMainExecutor()
                        try {
                            // Core validates the range and rejects anything
                            // outside it, so the host never has to be the
                            // authority on what is acceptable.
                            val reply = callGitHubBridge(
                                url = CONTEXT_MEMORY_BRIDGE_URL,
                                method = "PUT",
                                body = JSONObject().put("recent_messages", recentMessages),
                            )
                            mainExecutor.execute {
                                result.success(
                                    mapOf(
                                        "recentMessages" to reply.optInt("recent_messages"),
                                        "min" to reply.optInt("min"),
                                        "max" to reply.optInt("max"),
                                        "default" to reply.optInt("default"),
                                    )
                                )
                            }
                        } catch (e: Exception) {
                            mainExecutor.execute {
                                result.error(
                                    "CONTEXT_MEMORY_WRITE_FAILED",
                                    e.message ?: "Core rejected the context memory setting",
                                    null,
                                )
                            }
                        }
                    }.start()
                }
                "testGitHubConnection" -> {
                    // Deliberately tests the credential as gh will actually see
                    // it, rather than re-checking what is in storage: the useful
                    // question is whether the running Core is authenticated.
                    Thread {
                        val mainExecutor = getMainExecutor()
                        try {
                            val login = callGitHubBridge(
                                url = GITHUB_STATUS_BRIDGE_URL,
                                method = "GET",
                                body = null,
                            ).optString("login")
                            mainExecutor.execute {
                                result.success(mapOf("authenticated" to true, "login" to login))
                            }
                        } catch (e: Exception) {
                            val reason = e.message ?: "GitHub authentication is not working"
                            mainExecutor.execute {
                                result.error("GITHUB_TEST_FAILED", reason, null)
                            }
                        }
                    }.start()
                }
                "disconnectGitHub" -> {
                    try {
                        GitHubCredentialStore.disconnect(context)
                        result.success(true)
                    } catch (e: Exception) {
                        result.error(
                            "GITHUB_DISCONNECT_FAILED",
                            "Could not remove the GitHub credential",
                            null,
                        )
                    }
                }
                "getLaunchAutoStartPreferences" -> {
                    try {
                        result.success(LaunchAutoStartPreferences.read(context).asMap())
                    } catch (e: Exception) {
                        result.error(
                            "GET_LAUNCH_AUTOSTART_FAILED",
                            "Could not read launch auto-start preferences",
                            null,
                        )
                    }
                }
                "setLaunchAutoStartPreferences" -> {
                    try {
                        val snapshot = LaunchAutoStartPreferences.update(
                            context = context,
                            serviceEnabled = call.argument<Boolean>("serviceEnabled"),
                            gatewayEnabled = call.argument<Boolean>("gatewayEnabled"),
                        )
                        result.success(snapshot.asMap())
                    } catch (e: Exception) {
                        result.error(
                            "SET_LAUNCH_AUTOSTART_FAILED",
                            "Could not persist launch auto-start preferences",
                            null,
                        )
                    }
                }
                "startGatewayNow" -> {
                    // PC-DEF-034. Turning on "start Gateway automatically"
                    // while the service is already running has to act now;
                    // waiting for the next service start is the behaviour the
                    // user just told us they did not want. The bridge token
                    // stays in the host: Flutter asks for the operation and
                    // never sees the credential.
                    try {
                        result.success(callGatewayStartBridge())
                    } catch (e: Exception) {
                        result.error(
                            "GATEWAY_START_FAILED",
                            e.message ?: "Could not start the Gateway",
                            null,
                        )
                    }
                }
                "setAutoStart" -> {
                    try {
                        val enabled = call.argument<Boolean>("enabled") ?: false
                        val prefs = PocketClawPreferences.open(context)
                        prefs.edit().putBoolean(KEY_AUTO_START, enabled).apply()
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("SET_AUTO_START_FAILED", e.message, null)
                    }
                }
                "getAutoStart" -> {
                    try {
                        val prefs = PocketClawPreferences.open(context)
                        result.success(prefs.getBoolean(KEY_AUTO_START, false))
                    } catch (e: Exception) {
                        result.error("GET_AUTO_START_FAILED", e.message, null)
                    }
                }
                "getCoreVersion" -> {
                    Thread {
                        val mainExecutor = getMainExecutor()
                        try {
                            val version = PocketClawService.readCoreVersion(context)
                            mainExecutor.execute {
                                result.success(version)
                            }
                        } catch (e: Exception) {
                            Log.w(TAG, "getCoreVersion failed: ${e.message}", e)
                            mainExecutor.execute {
                                result.success("unknown")
                            }
                        }
                    }.start()
                }
                "getConfigPath" -> {
                    try {
                        result.success(PocketClawCoreState.configFile(context).absolutePath)
                    } catch (e: Exception) {
                        result.error("CORE_STATE_UNAVAILABLE", e.message, null)
                    }
                }
                "getHomePath" -> {
                    result.success(PocketClawService.getWorkspacePath(context))
                }
                "isStorageManagerGranted" -> {
                    val granted = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
                        Environment.isExternalStorageManager()
                    } else {
                        true
                    }
                    result.success(granted)
                }
                "requestStorageManager" -> {
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
                        try {
                            val intent = Intent(
                                Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION,
                                Uri.parse("package:${context.packageName}")
                            )
                            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                            context.startActivity(intent)
                            result.success(true)
                        } catch (e: Exception) {
                            // 部分设备不支持精确跳转，回退到通用页
                            val intent = Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION)
                            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                            context.startActivity(intent)
                            result.success(true)
                        }
                    } else {
                        result.success(true) // 低版本无需此权限
                    }
                }
                "getPocketClawToken" -> {
                    result.success(PocketClawService.pocketClawTokenForHost(context))
                }
                "getSafeDeviceInfo" -> {
                    val packageInfo = context.packageManager.getPackageInfo(context.packageName, 0)
                    val deviceCategory = if (context.resources.configuration.smallestScreenWidthDp >= 600) {
                        "Tablet"
                    } else {
                        "Mobile"
                    }
                    result.success(mapOf(
                        "deviceModel" to listOf(Build.MANUFACTURER, Build.MODEL)
                            .filter { it.isNotBlank() }
                            .joinToString(" ")
                            .trim(),
                        "osVersion" to "Android ${Build.VERSION.RELEASE}",
                        "deviceCategory" to deviceCategory,
                        "appVersion" to (packageInfo.versionName ?: "unknown")
                    ))
                }
                "setUmengAnalyticsConsent" -> {
                    try {
                        val enabled = call.argument<Boolean>("enabled") ?: false
                        AnalyticsReporter.submitConsent(context, enabled)
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("SET_UMENG_CONSENT_FAILED", e.message, null)
                    }
                }
                "uploadUmengDeviceReport" -> {
                    android.util.Log.d("PocketClawChannel", "=== uploadUmengDeviceReport called ===")
                    try {
                        val payload = call.arguments<Map<String, Any?>>() ?: emptyMap()
                        android.util.Log.d("PocketClawChannel", "Payload received with ${payload.size} fields")
                        val reportResult = AnalyticsReporter.uploadDeviceReport(context, payload)
                        android.util.Log.d("PocketClawChannel", "AnalyticsReporter returned: success=${reportResult["success"]}, message=${reportResult["message"]}")
                        result.success(reportResult)
                        android.util.Log.d("PocketClawChannel", "=== uploadUmengDeviceReport completed ===")
                    } catch (e: Exception) {
                        android.util.Log.e("PocketClawChannel", "uploadUmengDeviceReport failed: ${e.message}", e)
                        result.error("UPLOAD_UMENG_REPORT_FAILED", e.message, null)
                    }
                }
                "getWebPort" -> {
                    result.success(18800)
                }
                "getLanIpv4Address" -> {
                    result.success(activeLanIpv4Address())
                }
                "saveToDownloads" -> {
                    // args: filename: String, bytes: Uint8List
                    try {
                        val filename = call.argument<String>("filename") ?: "pocketclaw_logs.txt"
                        val bytes = call.argument<ByteArray>("bytes")
                        if (bytes == null) {
                            result.error("NO_BYTES", "No bytes provided", null)
                            return@setMethodCallHandler
                        }

                        val savedUriStr = saveToDownloads(filename, bytes)
                        result.success(savedUriStr)
                    } catch (e: Exception) {
                        result.error("SAVE_FAILED", e.message, null)
                    }
                }
                "copyContentUriToCache" -> {
                    try {
                        val uriStr = call.argument<String>("uri") ?: ""
                        val name = call.argument<String>("filename") ?: "pocketclaw_logs.txt"
                        val path = copyContentUriToCache(uriStr, name)
                        result.success(path)
                    } catch (e: Exception) {
                        result.error("COPY_FAILED", e.message, null)
                    }
                }
                else -> {
                    result.notImplemented()
                }
            }
        }
    }

    fun dispose() {
        channel.setMethodCallHandler(null)
    }

    /**
     * Finds a connectable address on an active Wi-Fi/Ethernet network. The
     * wildcard listen address and cellular/VPN-only destinations are never
     * advertised as LAN URLs.
     */
    private fun activeLanIpv4Address(): String? {
        return try {
            val connectivity = context.getSystemService(ConnectivityManager::class.java)
                ?: return null
            val active = connectivity.activeNetwork ?: return null
            lanIpv4Address(connectivity, active)
        } catch (e: Exception) {
            Log.w(TAG, "Unable to determine active LAN IPv4 address", e)
            null
        }
    }

    private fun lanIpv4Address(
        connectivity: ConnectivityManager,
        network: Network,
    ): String? {
        val capabilities = connectivity.getNetworkCapabilities(network) ?: return null
        if (!capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) &&
            !capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET)
        ) {
            return null
        }

        val addresses = connectivity.getLinkProperties(network)?.linkAddresses
            ?.asSequence()
            ?.map { it.address }
            ?.filterIsInstance<Inet4Address>()
            ?.filterNot { address ->
                address.isAnyLocalAddress ||
                    address.isLoopbackAddress ||
                    address.isLinkLocalAddress ||
                    address.isMulticastAddress
            }
            ?.toList()
            .orEmpty()

        return addresses.firstOrNull { isPrivateLanIpv4(it.address) }
            ?.hostAddress
            ?: addresses.firstOrNull()?.hostAddress
    }

    private fun isPrivateLanIpv4(bytes: ByteArray): Boolean {
        if (bytes.size != 4) return false
        val first = bytes[0].toInt() and 0xff
        val second = bytes[1].toInt() and 0xff
        return first == 10 ||
            (first == 172 && second in 16..31) ||
            (first == 192 && second == 168)
    }

    /**
     * Asks Core to start the Gateway over the loopback Android bridge.
     *
     * Returns the status Core reports -- "ok" or "already_running" -- so the
     * caller can distinguish a start from a no-op. The token is read here and
     * never returned, logged or handed to Flutter.
     */
    private fun callGatewayStartBridge(): String {
        val connection = (URL(GATEWAY_START_BRIDGE_URL).openConnection() as HttpURLConnection).apply {
            requestMethod = "POST"
            connectTimeout = 3_000
            readTimeout = 15_000
            setRequestProperty(
                "X-PocketClaw-Android-Bridge",
                PocketClawService.bridgeTokenForHost(),
            )
            setRequestProperty("Accept", "application/json")
        }
        try {
            val code = connection.responseCode
            val payload = if (code in 200..299) {
                connection.inputStream.bufferedReader().use { it.readText() }
            } else {
                connection.errorStream?.bufferedReader()?.use { it.readText() } ?: ""
            }
            if (code !in 200..299) {
                // Core's message, not the token or the URL.
                throw IllegalStateException(
                    JSONObject(payload.ifBlank { "{}" }).optString("message")
                        .ifBlank { "Gateway start failed (HTTP $code)" }
                )
            }
            return JSONObject(payload.ifBlank { "{}" }).optString("status").ifBlank { "ok" }
        } finally {
            connection.disconnect()
        }
    }

    /**
     * Whether the Dashboard already has an owner.
     *
     * No bridge token: this endpoint is deliberately unauthenticated and
     * returns only two booleans. The privileged call that may follow -- the
     * network-mode re-apply -- keeps the token, on loopback, as it always has.
     */
    private fun readDashboardAuthInitialized(): Boolean {
        val connection = (URL(DASHBOARD_AUTH_STATUS_URL).openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 1_000
            readTimeout = 2_000
            setRequestProperty("Accept", "application/json")
        }
        try {
            if (connection.responseCode !in 200..299) return false
            val payload = connection.inputStream.bufferedReader().use { it.readText() }
            return JSONObject(payload.ifBlank { "{}" }).optBoolean("initialized", false)
        } finally {
            connection.disconnect()
        }
    }

    /** Writes paired credentials through Core's own config/security store. */
    private fun callTelegramBridge(
        method: String,
        body: JSONObject? = null
    ) {
        val connection = (URL(TELEGRAM_BRIDGE_URL).openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 3_000
            readTimeout = 5_000
            setRequestProperty(
                "X-PocketClaw-Android-Bridge",
                PocketClawService.bridgeTokenForHost()
            )
            setRequestProperty("Accept", "application/json")
            if (body != null) {
                doOutput = true
                setRequestProperty("Content-Type", "application/json")
            }
        }
        try {
            if (body != null) {
                connection.outputStream.use { output ->
                    output.write(body.toString().toByteArray(Charsets.UTF_8))
                }
            }
            if (connection.responseCode != HttpURLConnection.HTTP_OK) {
                throw IllegalStateException("Core Telegram bridge request failed")
            }
            connection.inputStream.close()
        } finally {
            connection.disconnect()
        }
    }

    /**
     * One request to Core's loopback GitHub bridge.
     *
     * Core's reply carries an account name or a reason, never the credential:
     * it scrubs the candidate out of gh's own output before answering, so an
     * error message is safe to show and safe to log.
     */
    private fun callGitHubBridge(url: String, method: String, body: JSONObject?): JSONObject {
        val connection = (URL(url).openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 3_000
            // A credential check reaches GitHub over the network, so it is
            // allowed to take noticeably longer than a local bridge call.
            readTimeout = 30_000
            setRequestProperty(
                "X-PocketClaw-Android-Bridge",
                PocketClawService.bridgeTokenForHost(),
            )
            setRequestProperty("Accept", "application/json")
            if (body != null) {
                doOutput = true
                setRequestProperty("Content-Type", "application/json")
            }
        }
        try {
            if (body != null) {
                connection.outputStream.use { output ->
                    output.write(body.toString().toByteArray(Charsets.UTF_8))
                }
            }
            if (connection.responseCode != HttpURLConnection.HTTP_OK) {
                val body = connection.errorStream
                    ?.bufferedReader(Charsets.UTF_8)
                    ?.use { it.readText() }
                    ?.trim()
                    .orEmpty()
                // Core answers failures as a structured category plus a message
                // written for the user. A connectivity fault and a rejected
                // credential are different things, and the message already says
                // which; the raw body is only a fallback for a reply that is not
                // Core's own, such as one from a listener that is not up yet.
                val reason = try {
                    JSONObject(body).optString("message").ifBlank { body }
                } catch (malformed: Exception) {
                    body
                }
                throw IllegalStateException(
                    reason.ifEmpty {
                        "PocketClaw could not reach GitHub. Is the service running?"
                    }
                )
            }
            val payload = connection.inputStream.bufferedReader(Charsets.UTF_8).use { it.readText() }
            return if (payload.isBlank()) JSONObject() else JSONObject(payload)
        } finally {
            connection.disconnect()
        }
    }

    private fun callNetworkModeBridge(
        method: String,
        body: JSONObject?,
        expectedStatus: Int,
    ): JSONObject {
        val connection = (URL(NETWORK_MODE_BRIDGE_URL).openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 1_000
            readTimeout = 2_000
            setRequestProperty(
                "X-PocketClaw-Android-Bridge",
                PocketClawService.bridgeTokenForHost(),
            )
            setRequestProperty("Accept", "application/json")
            if (body != null) {
                doOutput = true
                setRequestProperty("Content-Type", "application/json")
            }
        }
        try {
            if (body != null) {
                connection.outputStream.use { output ->
                    output.write(body.toString().toByteArray(Charsets.UTF_8))
                }
            }
            if (connection.responseCode != expectedStatus) {
                throw IllegalStateException("Dashboard network mode bridge rejected the request")
            }
            val response = connection.inputStream.bufferedReader(Charsets.UTF_8).use {
                it.readText()
            }
            return JSONObject(response)
        } finally {
            connection.disconnect()
        }
    }

    // Save bytes to Downloads using MediaStore (preferred for Android Q+).
    private fun saveToDownloads(fileName: String, data: ByteArray): String? {
        try {
            val resolver = context.contentResolver
            val values = android.content.ContentValues().apply {
                put(android.provider.MediaStore.MediaColumns.DISPLAY_NAME, fileName)
                put(android.provider.MediaStore.MediaColumns.MIME_TYPE, "text/plain")
                // Place in Downloads directory under Pictures (app-specific folder not required)
                put(android.provider.MediaStore.MediaColumns.RELATIVE_PATH, android.os.Environment.DIRECTORY_DOWNLOADS)
            }

            val uri = resolver.insert(android.provider.MediaStore.Downloads.getContentUri(android.provider.MediaStore.VOLUME_EXTERNAL_PRIMARY), values)
                ?: return null

            resolver.openOutputStream(uri).use { out ->
                out?.write(data)
                out?.flush()
            }

            return uri.toString()
        } catch (e: Exception) {
            // For older devices, attempt fallback to legacy external storage path
            try {
                val downloads = android.os.Environment.getExternalStoragePublicDirectory(android.os.Environment.DIRECTORY_DOWNLOADS)
                val dir = java.io.File(downloads, "pocketclaw")
                if (!dir.exists()) dir.mkdirs()
                val f = java.io.File(dir, fileName)
                f.writeBytes(data)
                return f.absolutePath
            } catch (ex: Exception) {
                return null
            }
        }
    }
}

package com.lord1egypt.pocketclaw

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
import com.lord1egypt.pocketclaw.service.PicoClawService
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
 * - startService: 启动 PicoClaw 前台服务
 * - stopService: 停止 PicoClaw 前台服务
 * - getServiceStatus: 获取服务状态（isRunning, pid, lastLog）
 * - checkHealth: 检查 /health 端点
 * - getConfig: 读取 config.json 内容
 * - saveConfig: 保存 config.json 内容
 * - getFullLog: 获取完整日志
 * - setAutoStart: 设置开机自启
 * - getAutoStart: 获取开机自启设置
 * - getWebPort: 获取 Web Console 端口号
 */
class PicoClawMethodChannel(
    private val context: Context,
    flutterEngine: FlutterEngine
) {
    companion object {
        private const val TAG = "PicoClawMethodChannel"
        private const val CHANNEL_NAME = "com.lord1egypt.pocketclaw/picoclaw"
        private const val PREF_NAME = "picoclaw_prefs"
        private const val KEY_AUTO_START = "auto_start"
        private const val TELEGRAM_BRIDGE_URL =
            "http://127.0.0.1:18800/api/pocketclaw/android/telegram"
        private const val NETWORK_MODE_BRIDGE_URL =
            "http://127.0.0.1:18800/api/pocketclaw/android/network-mode"
        private const val GATEWAY_BRIDGE_URL =
            "http://127.0.0.1:18800/api/pocketclaw/android/gateway"
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
    private val healthChecker = HealthChecker()

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
                        val tokens = args.split(Regex("\\s+")).filter { it.isNotBlank() }
                        val publicMode = tokens.contains("-public")
                        val launchPreferences = LaunchAutoStartPreferences.read(context)
                        // 保存 publicMode 到 SharedPreferences
                        val prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
                        prefs.edit().putBoolean("public_mode", publicMode).apply()
                        Log.d(
                            TAG,
                            "Starting service with publicMode=$publicMode, " +
                                "serviceAutoStart=${launchPreferences.serviceEnabled}, " +
                                "gatewayAutoStart=${launchPreferences.gatewayEnabled}, " +
                                "preferenceSource=${launchPreferences.source}"
                        )
                        PicoClawService.start(context, publicMode)
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("START_FAILED", e.message, null)
                    }
                }
                "getPublicMode" -> {
                    try {
                        val prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
                        result.success(prefs.getBoolean("public_mode", false))
                    } catch (e: Exception) {
                        result.error("GET_PUBLIC_MODE_FAILED", e.message, null)
                    }
                }
                "applyPublicMode" -> {
                    val publicMode = call.argument<Boolean>("public") ?: false
                    Thread {
                        val mainExecutor = getMainExecutor()
                        val previousMode = context
                            .getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
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
                                context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
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
                        PicoClawService.stop(context)
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("STOP_FAILED", e.message, null)
                    }
                }
                "getServiceStatus" -> {
                    result.success(mapOf(
                        "isRunning" to PicoClawService.isRunning,
                        "isStarting" to PicoClawService.isStarting,
                        "pid" to PicoClawService.processId,
                        "lastLog" to PicoClawService.lastLog
                    ))
                }
                "checkHealth" -> {
                    Thread {
                        val mainExecutor = getMainExecutor()

                        try {
                            val status = healthChecker.check()
                            val resultMap = mapOf(
                                "isHealthy" to status.isHealthy,
                                "status" to status.status,
                                "uptime" to status.uptime,
                                "pid" to status.pid,
                                "error" to (status.error ?: "")
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
                "getGatewayStatus" -> {
                    Thread {
                        val mainExecutor = getMainExecutor()
                        try {
                            val response = callGatewayBridge("GET")
                            mainExecutor.execute { result.success(jsonObjectMap(response)) }
                        } catch (e: Exception) {
                            mainExecutor.execute {
                                result.error(
                                    "GATEWAY_STATUS_FAILED",
                                    "Managed Gateway status is unavailable",
                                    null
                                )
                            }
                        }
                    }.start()
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
                "startGateway" -> {
                    Thread {
                        val mainExecutor = getMainExecutor()
                        try {
                            val response = callGatewayBridge("POST")
                            mainExecutor.execute { result.success(jsonObjectMap(response)) }
                        } catch (e: Exception) {
                            mainExecutor.execute {
                                result.error(
                                    "GATEWAY_START_FAILED",
                                    "Managed Gateway start was rejected",
                                    null
                                )
                            }
                        }
                    }.start()
                }
                "getConfig" -> {
                    try {
                        val configFile = File(context.filesDir, "picoclaw/config.json")
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
                        val configFile = File(context.filesDir, "picoclaw/config.json")
                        configFile.parentFile?.mkdirs()
                        configFile.writeText(content)
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("SAVE_CONFIG_FAILED", e.message, null)
                    }
                }
                "getFullLog" -> {
                    result.success(PicoClawService.lastLog)
                }
                "takeNewLogs" -> {
                    // Each line is delivered once. See PicoClawService.publishLog.
                    result.success(PicoClawService.takeNewLogs())
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
                "setAutoStart" -> {
                    try {
                        val enabled = call.argument<Boolean>("enabled") ?: false
                        val prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
                        prefs.edit().putBoolean(KEY_AUTO_START, enabled).apply()
                        result.success(true)
                    } catch (e: Exception) {
                        result.error("SET_AUTO_START_FAILED", e.message, null)
                    }
                }
                "getAutoStart" -> {
                    try {
                        val prefs = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
                        result.success(prefs.getBoolean(KEY_AUTO_START, false))
                    } catch (e: Exception) {
                        result.error("GET_AUTO_START_FAILED", e.message, null)
                    }
                }
                "getCoreVersion" -> {
                    Thread {
                        val mainExecutor = getMainExecutor()
                        try {
                            val version = PicoClawService.readCoreVersion(context)
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
                    val configFile = File(context.filesDir, "picoclaw/config.json")
                    result.success(configFile.absolutePath)
                }
                "getHomePath" -> {
                    result.success(PicoClawService.getWorkspacePath(context))
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
                "getPicoToken" -> {
                    result.success(PicoClawService.picoTokenForHost(context))
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
                    android.util.Log.d("PicoClawChannel", "=== uploadUmengDeviceReport called ===")
                    try {
                        val payload = call.arguments<Map<String, Any?>>() ?: emptyMap()
                        android.util.Log.d("PicoClawChannel", "Payload received with ${payload.size} fields")
                        val reportResult = AnalyticsReporter.uploadDeviceReport(context, payload)
                        android.util.Log.d("PicoClawChannel", "AnalyticsReporter returned: success=${reportResult["success"]}, message=${reportResult["message"]}")
                        result.success(reportResult)
                        android.util.Log.d("PicoClawChannel", "=== uploadUmengDeviceReport completed ===")
                    } catch (e: Exception) {
                        android.util.Log.e("PicoClawChannel", "uploadUmengDeviceReport failed: ${e.message}", e)
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
                        val filename = call.argument<String>("filename") ?: "picoclaw_logs.txt"
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
                        val name = call.argument<String>("filename") ?: "picoclaw_logs.txt"
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
                PicoClawService.bridgeTokenForHost()
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
                PicoClawService.bridgeTokenForHost(),
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

    private fun callGatewayBridge(method: String): JSONObject {
        val connection = (URL(GATEWAY_BRIDGE_URL).openConnection() as HttpURLConnection).apply {
            requestMethod = method
            connectTimeout = 1_000
            readTimeout = 3_000
            setRequestProperty(
                "X-PocketClaw-Android-Bridge",
                PicoClawService.bridgeTokenForHost(),
            )
            setRequestProperty("Accept", "application/json")
        }
        try {
            val status = connection.responseCode
            if (status != HttpURLConnection.HTTP_OK &&
                status != HttpURLConnection.HTTP_BAD_REQUEST
            ) {
                throw IllegalStateException("Managed Gateway bridge rejected the request")
            }
            val stream = if (status >= 400) connection.errorStream else connection.inputStream
            val response = stream.bufferedReader(Charsets.UTF_8).use { it.readText() }
            return JSONObject(response)
        } finally {
            connection.disconnect()
        }
    }

    private fun jsonObjectMap(json: JSONObject): Map<String, Any?> =
        json.keys().asSequence().associateWith { key ->
            val value = json.opt(key)
            if (value == JSONObject.NULL) null else value
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

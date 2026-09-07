package com.lord1egypt.pocketclaw.util

import com.lord1egypt.pocketclaw.service.PicoClawService
import org.json.JSONObject
import java.io.File
import java.net.HttpURLConnection
import java.net.URL

/**
 * 轮询 PicoClaw /health 端点检查服务状态。
 * 使用 HttpURLConnection 避免额外依赖。
 *
 * Basic checks stay anonymous, exactly as they always have been: the launcher
 * and this host both poll /health for liveness and nothing about that request
 * changed.
 *
 * The Status screen's richer payload is requested with `?detail=1` and carries
 * the gateway's own bearer credential, read from the PID file the gateway
 * writes at startup. That credential is deliberately confined to this class —
 * it is never returned across the method channel, put in a URL, logged, or
 * persisted anywhere the Dart side can reach.
 */
class HealthChecker(
    private val host: String = "127.0.0.1",
    private val port: Int = 18790,
    private val workspacePathProvider: () -> String? = { null },
) {
    data class HealthStatus(
        val isHealthy: Boolean,
        val status: String = "unknown",
        val uptime: String = "",
        val pid: Int = -1,
        val error: String? = null,
        /**
         * The Status snapshot as raw JSON, or null when detail was not asked
         * for, was refused, or the gateway did not supply one. Null means
         * "unavailable" and must never be shown as though it were empty data.
         */
        val detailJson: String? = null,
    )

    fun check(detailed: Boolean = false): HealthStatus {
        if (!detailed) {
            return request(withDetail = false)
        }

        val detailedResult = request(withDetail = true)
        if (detailedResult.isHealthy) {
            return detailedResult
        }
        // The gateway refuses an unauthorized detail request outright rather
        // than quietly downgrading it. Falling back here is a client-side
        // decision, not a server-side downgrade: liveness still has to be
        // reported so Start/Stop keeps working even when detail is
        // unavailable.
        return request(withDetail = false)
    }

    private fun request(withDetail: Boolean): HealthStatus {
        return try {
            val path = if (withDetail) "/health?detail=1" else "/health"
            val conn = URL("http://$host:$port$path").openConnection() as HttpURLConnection
            conn.connectTimeout = 2000
            conn.readTimeout = 2000
            conn.requestMethod = "GET"
            if (withDetail) {
                val token = gatewayToken()
                    ?: return HealthStatus(isHealthy = false, error = "no gateway credential")
                conn.setRequestProperty("Authorization", "Bearer $token")
            }

            try {
                val code = conn.responseCode
                if (code == 200) {
                    val json = JSONObject(conn.inputStream.bufferedReader().readText())
                    HealthStatus(
                        isHealthy = true,
                        status = json.optString("status", "ok"),
                        uptime = json.optString("uptime", ""),
                        pid = json.optInt("pid", -1),
                        detailJson = json.optJSONObject("detail")?.toString(),
                    )
                } else {
                    // The response code is reported; the request URL is not,
                    // so a credential can never reach an error string.
                    HealthStatus(isHealthy = false, error = "HTTP $code")
                }
            } finally {
                conn.disconnect()
            }
        } catch (e: Exception) {
            HealthStatus(isHealthy = false, error = e.message)
        }
    }

    /**
     * Reads the gateway's bearer token from the PID file it writes at startup.
     *
     * This is the credential that already guards /reload — detail mode reuses
     * it rather than introducing a second one. Returns null when the gateway
     * is not running or the file is unreadable, which closes detail mode
     * instead of opening it.
     */
    private fun gatewayToken(): String? {
        return try {
            val workspace = workspacePathProvider() ?: return null
            val pidFile = File(workspace, PID_FILE_NAME)
            if (!pidFile.isFile) return null
            JSONObject(pidFile.readText(Charsets.UTF_8))
                .optString("token", "")
                .takeIf { it.isNotBlank() }
        } catch (_: Exception) {
            null
        }
    }

    companion object {
        private const val PID_FILE_NAME = ".picoclaw.pid"

        /** Builds a checker bound to this installation's workspace. */
        fun forHost(context: android.content.Context): HealthChecker =
            HealthChecker(
                workspacePathProvider = { PicoClawService.getWorkspacePath(context) },
            )
    }
}

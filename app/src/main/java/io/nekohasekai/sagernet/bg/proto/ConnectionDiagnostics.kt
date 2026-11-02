package io.nekohasekai.sagernet.bg.proto

import android.net.ConnectivityManager
import android.content.Context
import android.net.NetworkCapabilities
import android.os.Build
import android.util.Log
import io.nekohasekai.sagernet.SagerNet
import io.nekohasekai.sagernet.ktx.app
import libcore.BoxInstance
import libcore.Libcore
import java.io.File

/** Bounded, endpoint-free evidence that remains available with core logging disabled. */
object ConnectionDiagnostics {
    private const val LIMIT = 32 * 1024
    private const val TAG = "NekoConnectionTest"

    fun preparing() = record("mode=profile state=preparing ${networkState()}")

    fun preparationFailed() = record("mode=profile outcome=fail kind=preparation")

    fun test(box: BoxInstance, link: String, timeout: Int, mode: String): Int {
        record("mode=$mode state=start timeout_ms=$timeout ${networkState()}")
        val result = Libcore.runConnectionTest(box, link, timeout)
        record("mode=$mode ${result.diagnostics}")
        if (result.failure.isNotEmpty()) throw IllegalStateException(result.failure)
        return result.latency.toInt()
    }

    private fun networkState(): String = try {
        val manager = app.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
        val network = if (Build.VERSION.SDK_INT >= 23) manager.activeNetwork else SagerNet.underlyingNetwork
        val caps = manager.getNetworkCapabilities(network)
        "sdk=${Build.VERSION.SDK_INT} wifi=${caps?.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) == true} " +
            "vpn=${caps?.hasTransport(NetworkCapabilities.TRANSPORT_VPN) == true} " +
            "validated=${caps?.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED) == true} " +
            "underlying=${SagerNet.underlyingNetwork != null}"
    } catch (_: Exception) { "network_state=unavailable" }

    @Synchronized
    private fun record(value: String) {
        // Values are fixed tokens, numbers and booleans. Never pass an exception,
        // URL, profile name, config JSON, IP address or TLS name here.
        val line = "${System.currentTimeMillis()} $value\n"
        Log.i(TAG, line.trimEnd())
        try {
            val suffix = if (SagerNet.application.isBgProcess) "bg" else "main"
            val file = File(app.cacheDir, "connection-test-$suffix.log")
            if (file.length() > LIMIT) file.writeText(file.readText().takeLast(LIMIT / 2))
            file.appendText(line)
        } catch (_: Exception) {
            Log.w(TAG, "diagnostic_file_unavailable")
        }
    }

    fun export(): String = buildString {
        append("\nConnection test diagnostics (no endpoints or credentials):\n")
        for (suffix in listOf("main", "bg")) {
            append("process=$suffix\n")
            try {
                val file = File(app.cacheDir, "connection-test-$suffix.log")
                if (file.exists()) append(file.readText().takeLast(LIMIT))
            } catch (_: Exception) { append("unavailable\n") }
        }
    }

    fun clear() {
        for (suffix in listOf("main", "bg")) {
            File(app.cacheDir, "connection-test-$suffix.log").delete()
        }
    }
}

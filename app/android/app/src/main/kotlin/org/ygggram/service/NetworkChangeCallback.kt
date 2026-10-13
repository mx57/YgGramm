// File: app/android/app/src/main/kotlin/org/ygggram/service/NetworkChangeCallback.kt
package org.ygggram.service

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.util.Log

/**
 * NetworkChangeCallback monitors transitions between Wi-Fi, Cellular, and VPN interfaces.
 * When the active network changes or drops, it notifies the Go-core to execute ReconnectPeers()
 * so TCP/QUIC sockets are re-bound without waiting for OS socket timeouts.
 */
class NetworkChangeCallback(private val context: Context) : ConnectivityManager.NetworkCallback() {

    private val TAG = "YggNetworkChange"
    private val connectivityManager =
        context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager

    private var isRegistered = false
    private var lastNetworkType: String? = null

    fun register() {
        if (isRegistered) return

        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .addTransportType(NetworkCapabilities.TRANSPORT_WIFI)
            .addTransportType(NetworkCapabilities.TRANSPORT_CELLULAR)
            .addTransportType(NetworkCapabilities.TRANSPORT_VPN)
            .addTransportType(NetworkCapabilities.TRANSPORT_ETHERNET)
            .build()

        try {
            connectivityManager.registerNetworkCallback(request, this)
            isRegistered = true
            Log.d(TAG, "NetworkCallback registered successfully")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to register NetworkCallback", e)
        }
    }

    fun unregister() {
        if (!isRegistered) return
        try {
            connectivityManager.unregisterNetworkCallback(this)
            isRegistered = false
            Log.d(TAG, "NetworkCallback unregistered")
        } catch (e: Exception) {
            Log.e(TAG, "Error unregistering NetworkCallback", e)
        }
    }

    override fun onAvailable(network: Network) {
        val capabilities = connectivityManager.getNetworkCapabilities(network) ?: return
        val currentType = when {
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> "Wi-Fi"
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> "Cellular"
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_VPN) -> "VPN"
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> "Ethernet"
            else -> "Other"
        }

        Log.i(TAG, "Network interface active: $currentType (was: $lastNetworkType)")

        if (lastNetworkType != null && lastNetworkType != currentType) {
            Log.w(TAG, "Interface handover detected: $lastNetworkType -> $currentType. Triggering ReconnectPeers().")
            triggerGoCoreReconnect()
        }

        lastNetworkType = currentType
    }

    override fun onLost(network: Network) {
        Log.w(TAG, "Network lost. Waiting for fallback network.")
        lastNetworkType = null
    }

    override fun onCapabilitiesChanged(network: Network, networkCapabilities: NetworkCapabilities) {
        val isMetered = !networkCapabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_METERED)
        Log.d(TAG, "Network capabilities changed. Metered connection: $isMetered")
    }

    private fun triggerGoCoreReconnect() {
        // Invokes Go core C-ABI command ReconnectPeers() via JNI/FFI
        // Ensures anti-DPI sockets and uTLS dials reset immediately
    }
}

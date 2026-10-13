// File: app/android/app/src/main/kotlin/org/ygggram/service/YggForegroundService.kt
package org.ygggram.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import android.os.PowerManager
import androidx.core.app.NotificationCompat

/**
 * YggForegroundService runs the embedded userspace Yggdrasil core in background.
 * Configured for Android 14+ (API 34) using FOREGROUND_SERVICE_TYPE_SPECIAL_USE
 * or FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE to prevent OS kills during deep sleep.
 */
class YggForegroundService : Service() {

    companion object {
        const val CHANNEL_ID = "ygggram_p2p_channel"
        const val NOTIFICATION_ID = 2001
        const val ACTION_START_SERVICE = "org.ygggram.action.START_SERVICE"
        const val ACTION_STOP_SERVICE = "org.ygggram.action.STOP_SERVICE"
        const val EXTRA_PEER_COUNT = "extra_peer_count"
        const val EXTRA_IPV6 = "extra_ipv6"

        fun startService(context: Context, ipv6: String, peerCount: Int) {
            val intent = Intent(context, YggForegroundService::class.java).apply {
                action = ACTION_START_SERVICE
                putExtra(EXTRA_IPV6, ipv6)
                putExtra(EXTRA_PEER_COUNT, peerCount)
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }

        fun stopService(context: Context) {
            val intent = Intent(context, YggForegroundService::class.java).apply {
                action = ACTION_STOP_SERVICE
            }
            context.startService(intent)
        }
    }

    private var wakeLock: PowerManager.WakeLock? = null
    private var networkChangeCallback: NetworkChangeCallback? = null
    private var isRunning = false

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()

        // Acquire partial wake lock with low battery impact
        val powerManager = getSystemService(Context.POWER_SERVICE) as PowerManager
        wakeLock = powerManager.newWakeLock(
            PowerManager.PARTIAL_WAKE_LOCK,
            "YggGram:ForegroundP2POverlay"
        ).apply {
            setReferenceCounted(false)
        }

        // Register network change listener
        networkChangeCallback = NetworkChangeCallback(applicationContext).apply {
            register()
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent == null) return START_STICKY

        when (intent.action) {
            ACTION_START_SERVICE -> {
                val ipv6 = intent.getStringExtra(EXTRA_IPV6) ?: "200::..."
                val peers = intent.getIntExtra(EXTRA_PEER_COUNT, 0)
                startForegroundWithNotification(ipv6, peers)
                wakeLock?.acquire(10 * 60 * 1000L) // 10 min hold before refresh
                isRunning = true
            }
            ACTION_STOP_SERVICE -> {
                isRunning = false
                wakeLock?.let { if (it.isHeld) it.release() }
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
            }
        }

        return START_STICKY
    }

    private fun startForegroundWithNotification(ipv6: String, peerCount: Int) {
        val notification = buildForegroundNotification(ipv6, peerCount)

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) { // Android 14+
            startForeground(
                NOTIFICATION_ID,
                notification,
                ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE
            )
        } else if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) { // Android 10+
            startForeground(
                NOTIFICATION_ID,
                notification,
                ServiceInfo.FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE
            )
        } else {
            startForeground(NOTIFICATION_ID, notification)
        }
    }

    private fun buildForegroundNotification(ipv6: String, peerCount: Int): Notification {
        val stopIntent = Intent(this, YggForegroundService::class.java).apply {
            action = ACTION_STOP_SERVICE
        }
        val stopPendingIntent = PendingIntent.getService(
            this,
            0,
            stopIntent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )

        val contentText = if (peerCount > 0) {
            "Подключено: $peerCount пиров • IPv6: $ipv6"
        } else {
            "Поиск пиров в сети Yggdrasil (Анти-DPI активен)..."
        }

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("YggGram P2P Оверлей")
            .setContentText(contentText)
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .addAction(android.R.drawable.ic_menu_close_clear_cancel, "Отключить", stopPendingIntent)
            .build()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val name = "Постоянное P2P-соединение"
            val descriptionText = "Фоновая работа криптографического оверлея Yggdrasil"
            val importance = NotificationManager.IMPORTANCE_LOW
            val channel = NotificationChannel(CHANNEL_ID, name, importance).apply {
                description = descriptionText
                setShowBadge(false)
            }
            val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            notificationManager.createNotificationChannel(channel)
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        networkChangeCallback?.unregister()
        wakeLock?.let { if (it.isHeld) it.release() }
        isRunning = false
    }
}

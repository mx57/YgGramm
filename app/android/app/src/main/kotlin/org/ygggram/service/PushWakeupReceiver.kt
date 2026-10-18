// File: app/android/app/src/main/kotlin/org/ygggram/service/PushWakeupReceiver.kt
package org.ygggram.service

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.os.PowerManager

/**
 * Будильник пробуждения: UnifiedPush/FCM-пин лишь поднимает процесс и держит
 * ядро живым через [YggForegroundService]. Уведомление о «входящем сообщении»
 * здесь намеренно не показывается: ядро пока не умеет читать из сети
 * (приёмный pump — Шаг 3), поэтому любое уведомление было бы выдумкой.
 */
class PushWakeupReceiver : BroadcastReceiver() {

    companion object {
        const val ACTION_UNIFIED_PUSH_MESSAGE = "org.unifiedpush.android.connector.MESSAGE"
        const val ACTION_FCM_WAKEUP = "org.ygggram.action.FCM_WAKEUP"
    }

    override fun onReceive(context: Context, intent: Intent) {
        when (intent.action) {
            ACTION_UNIFIED_PUSH_MESSAGE, ACTION_FCM_WAKEUP -> wakeCore(context)
        }
    }

    private fun wakeCore(context: Context) {
        val powerManager = context.getSystemService(Context.POWER_SERVICE) as PowerManager
        val wakeLock = powerManager.newWakeLock(
            PowerManager.PARTIAL_WAKE_LOCK,
            "YggGram:PushWakeupProcessing"
        ).apply {
            setReferenceCounted(false)
            acquire(5000L)
        }

        try {
            YggForegroundService.startService(context, ipv6 = "", peerCount = 0)
        } finally {
            if (wakeLock.isHeld) wakeLock.release()
        }
    }
}

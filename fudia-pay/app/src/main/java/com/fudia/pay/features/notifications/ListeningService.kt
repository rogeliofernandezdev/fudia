package com.fudia.pay.features.notifications

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.ServiceCompat
import androidx.core.content.ContextCompat
import com.fudia.pay.BuildConfig
import com.fudia.pay.MainActivity
import com.fudia.pay.R
import com.fudia.pay.core.security.DeviceCredentials
import dagger.hilt.android.AndroidEntryPoint
import javax.inject.Inject

/**
 * Servicio en primer plano que mantiene vivo el proceso del listener.
 *
 * Justificación (AGENTS regla 9): HONOR/MagicOS congela el proceso con su
 * propio congelador (`HnFreezerController`, cgroup `cgroup.freeze = 1`)
 * minutos después de dejar la app, aunque esté exenta de optimización de
 * batería. Congelado, Android le entrega las notificaciones pero no se
 * ejecutan hasta que alguien abre la app: el pago llega a Caja tarde o nunca.
 * Un servicio en primer plano de un tipo que PowerGenie reconoce
 * (`connectedDevice`, ver [foregroundTypes]) lo saca de la hibernación. No hace trabajo propio: la notificación fija es la señal
 * honesta de que la app está escuchando.
 */
@AndroidEntryPoint
class ListeningService : Service() {
    @Inject lateinit var credentials: DeviceCredentials

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (!credentials.paired) {
            stopSelf()
            return START_NOT_STICKY
        }
        val type = foregroundTypes()
        try {
            ServiceCompat.startForeground(this, NOTIFICATION_ID, notification(), type)
        } catch (e: RuntimeException) {
            // Arranque en segundo plano no permitido: el siguiente intento
            // (abrir la app, reinicio, reconexión del listener) lo repara.
            if (BuildConfig.DEBUG) Log.d(TAG, "no se pudo pasar a primer plano: ${e.javaClass.simpleName}")
            stopSelf()
            return START_NOT_STICKY
        }
        return START_STICKY
    }

    /** Tipos que PowerGenie (HONOR) reconoce como trabajo legítimo en segundo
     *  plano: con solo specialUse marcaba «FgService:» vacío y congelaba el
     *  proceso. Sin dataSync: Android 15 lo limita a 6 h por día. */
    private fun foregroundTypes(): Int = when {
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE ->
            ServiceInfo.FOREGROUND_SERVICE_TYPE_REMOTE_MESSAGING or
                ServiceInfo.FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE or
                ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE
        Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q -> ServiceInfo.FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE
        else -> 0
    }

    private fun notification(): Notification {
        val manager = getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(CHANNEL_ID, "Detección de pagos", NotificationManager.IMPORTANCE_LOW).apply {
                    description = "Aviso fijo mientras Fudia Pay detecta pagos de Yape y Plin."
                    setShowBadge(false)
                },
            )
        }
        val open = PendingIntent.getActivity(
            this, 0,
            Intent(this, MainActivity::class.java).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP),
            PendingIntent.FLAG_IMMUTABLE,
        )
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_stat_listening)
            .setContentTitle("Detectando pagos")
            .setContentText("Fudia Pay envía a Caja los pagos de Yape y Plin.")
            .setContentIntent(open)
            .setOngoing(true)
            .setShowWhen(false)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .setForegroundServiceBehavior(NotificationCompat.FOREGROUND_SERVICE_IMMEDIATE)
            .build()
    }

    companion object {
        private const val TAG = "FudiaPay"
        private const val CHANNEL_ID = "fudia-pay-listening"
        private const val NOTIFICATION_ID = 1001

        /** Arranque idempotente desde cualquier punto; si Android no lo
         *  permite en ese momento, no rompe al llamador. */
        fun start(context: Context) {
            try {
                ContextCompat.startForegroundService(context, Intent(context, ListeningService::class.java))
            } catch (e: RuntimeException) {
                if (BuildConfig.DEBUG) Log.d(TAG, "arranque del servicio fijo rechazado: ${e.javaClass.simpleName}")
            }
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, ListeningService::class.java))
        }
    }
}

/** Tras reiniciar el teléfono o actualizar la app, el servicio fijo vuelve
 *  sin que nadie abra Fudia Pay. */
class ListeningStartReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (intent.action) {
            Intent.ACTION_BOOT_COMPLETED, Intent.ACTION_MY_PACKAGE_REPLACED -> ListeningService.start(context)
        }
    }
}

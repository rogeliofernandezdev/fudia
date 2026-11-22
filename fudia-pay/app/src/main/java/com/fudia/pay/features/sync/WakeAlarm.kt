package com.fudia.pay.features.sync

import android.app.AlarmManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.os.SystemClock
import android.util.Log
import com.fudia.pay.BuildConfig
import dagger.hilt.android.EntryPointAccessors
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull

/** Cada cuánto se revisa sin push. HONOR alinea las alarmas de las apps
 *  congeladas a ventanas de 5 minutos: pedir menos no acelera nada. */
internal const val WAKE_ALARM_INTERVAL_MS = 5 * 60_000L

/**
 * Respaldo del push: una alarma inexacta cada 5 minutos despierta la app aunque
 * no haya Caja abierta, no haya red para el push o el teléfono no tenga Google
 * Play Services (Huawei). Sin permiso de alarmas exactas ni servicio fijo.
 */
object WakeAlarm {
    private fun pending(context: Context, flags: Int): PendingIntent? = PendingIntent.getBroadcast(
        context, 0,
        Intent(context, WakeAlarmReceiver::class.java),
        flags or PendingIntent.FLAG_IMMUTABLE,
    )

    /** Programa (o reprograma) la próxima revisión. */
    fun schedule(context: Context) {
        val alarms = context.getSystemService(AlarmManager::class.java) ?: return
        val intent = pending(context, PendingIntent.FLAG_UPDATE_CURRENT) ?: return
        // AndAllowWhileIdle: también en reposo profundo (Doze); la app está
        // exenta de la optimización de batería, así que la cuota alcanza.
        alarms.setAndAllowWhileIdle(
            AlarmManager.ELAPSED_REALTIME_WAKEUP,
            SystemClock.elapsedRealtime() + WAKE_ALARM_INTERVAL_MS,
            intent,
        )
    }

    fun cancel(context: Context) {
        val intent = pending(context, PendingIntent.FLAG_NO_CREATE) ?: return
        context.getSystemService(AlarmManager::class.java)?.cancel(intent)
        intent.cancel()
    }
}

/** Revisión de la alarma: reprograma la siguiente y repasa listener y cola. */
class WakeAlarmReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val entryPoint = EntryPointAccessors.fromApplication(context.applicationContext, WakeEntryPoint::class.java)
        if (!entryPoint.credentials().paired) return
        WakeAlarm.schedule(context)
        if (BuildConfig.DEBUG) Log.d("FudiaPay", "alarma de respaldo: revisando")
        val result = goAsync()
        CoroutineScope(SupervisorJob() + Dispatchers.IO).launch {
            try {
                withTimeoutOrNull(BUDGET_MS) {
                    entryPoint.sweep().run()
                    entryPoint.pushTokens().sync()
                }
            } finally {
                result.finish()
            }
        }
    }

    private companion object {
        // goAsync concede unos 10 s antes de que Android considere colgado al receptor.
        const val BUDGET_MS = 9_000L
    }
}

/** Tras reiniciar el teléfono o actualizar la app, la alarma vuelve sin que
 *  nadie abra Fudia Pay. */
class WakeRestartReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        when (intent.action) {
            Intent.ACTION_BOOT_COMPLETED, Intent.ACTION_MY_PACKAGE_REPLACED -> {
                val entryPoint = EntryPointAccessors.fromApplication(context.applicationContext, WakeEntryPoint::class.java)
                if (entryPoint.credentials().paired) WakeAlarm.schedule(context)
            }
        }
    }
}

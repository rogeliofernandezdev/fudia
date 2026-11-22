package com.fudia.pay.features.sync

import android.content.Context
import androidx.hilt.work.HiltWorker
import androidx.work.BackoffPolicy
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone
import java.util.concurrent.TimeUnit
import javax.inject.Inject

/** La marca de tiempo del evento como ISO 8601 UTC: el contrato OpenAPI
 *  (`PayReportInput.occurredAt`) la define `string, date-time`; java.time no
 *  está disponible en minSdk 24 sin desugaring, así que se formatea a mano. */
internal fun isoUtc(millis: Long): String =
    SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
        timeZone = TimeZone.getTimeZone("UTC")
    }.format(Date(millis))

/** La decisión de auto-reparación: con el permiso concedido y el servicio
 *  sin señal en este proceso, el barrido pide el reenlace. La señal
 *  (connectionEpoch) es en memoria: si el sistema mató el proceso, el
 *  proceso nuevo nace sin ella y el worker repara aunque la persistencia
 *  diga «conectado». requestRebind es estático: no requiere servicio vivo. */
internal fun shouldRequestRebind(listenerEnabled: Boolean, boundInProcess: Boolean): Boolean =
    listenerEnabled && !boundInProcess

@HiltWorker
class SyncWorker @AssistedInject constructor(
    @Assisted context: Context,
    @Assisted params: WorkerParameters,
    private val sweep: ListenerSweep,
    private val pushTokens: PushTokens,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val drained = sweep.run()
        // El token de push se reintenta hasta que el backend lo acepta, y la
        // alarma de respaldo se rearma si el sistema la descartó.
        pushTokens.sync()
        WakeAlarm.schedule(applicationContext)
        return if (drained) Result.success() else Result.retry()
    }
}

/** Respaldo del envío directo del listener: un trabajo inmediato cuando el
 * envío directo falla (o al pulsar «Sincronizar ahora») y uno periódico
 * (recuperación tras conectividad o reinicio), con backoff exponencial. */
class SyncScheduler @Inject constructor(@dagger.hilt.android.qualifiers.ApplicationContext private val context: Context) {
    private val constraints = Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build()

    fun enqueueNow() {
        // REPLACE y no APPEND: encadenar el envío nuevo detrás de un trabajo
        // en backoff lo dejaba esperando minutos u horas. La cola en Room
        // conserva los eventos y el backend deduplica si un envío se corta.
        WorkManager.getInstance(context).enqueueUniqueWork(
            SYNC_WORK,
            ExistingWorkPolicy.REPLACE,
            OneTimeWorkRequestBuilder<SyncWorker>()
                .setConstraints(constraints)
                .setBackoffCriteria(BackoffPolicy.EXPONENTIAL, 15, TimeUnit.SECONDS)
                .build(),
        )
    }

    fun schedulePeriodic() {
        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
            PERIODIC_WORK,
            ExistingPeriodicWorkPolicy.KEEP,
            PeriodicWorkRequestBuilder<SyncWorker>(15, TimeUnit.MINUTES)
                .setConstraints(constraints)
                .build(),
        )
    }

    private companion object {
        const val SYNC_WORK = "fudia-pay-sync"
        const val PERIODIC_WORK = "fudia-pay-sync-periodic"
    }
}

package com.fudia.pay.features.sync

import android.content.ComponentName
import android.content.Context
import android.service.notification.NotificationListenerService
import android.util.Log
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
import com.fudia.pay.BuildConfig
import com.fudia.pay.features.notifications.ListeningService
import com.fudia.pay.features.notifications.PayNotificationListener
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
    private val syncer: PaymentSyncer,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        rebindListenerIfDead()
        // Enlazado no basta: el sistema puede dejar de entregarle avisos.
        PayNotificationListener.instance?.verifyDelivery()
        // Si Android detuvo el servicio fijo, se vuelve a pedir.
        ListeningService.start(applicationContext)
        return if (syncer.flush()) Result.success() else Result.retry()
    }

    /** Auto-reparación del listener: si el permiso está concedido pero el
     *  servicio no está enlazado en este proceso (tras un proceso muerto), el
     *  barrido del worker pide el reenlace sin abrir la app. */
    private fun rebindListenerIfDead() {
        val enabled = androidx.core.app.NotificationManagerCompat.getEnabledListenerPackages(applicationContext)
            .contains(applicationContext.packageName)
        if (!shouldRequestRebind(enabled, PayNotificationListener.connectionEpoch != null)) return
        if (BuildConfig.DEBUG) Log.d(TAG, "listener sin enlazar con permiso activo: solicitando reenlace")
        NotificationListenerService.requestRebind(ComponentName(applicationContext, PayNotificationListener::class.java))
    }

    private companion object {
        const val TAG = "FudiaPay"
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

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
import com.fudia.pay.core.network.FudiaPayApi
import com.fudia.pay.core.network.PaymentEvent
import com.fudia.pay.core.network.ReportRequest
import com.fudia.pay.features.payments.PendingEventDao
import dagger.assisted.Assisted
import dagger.assisted.AssistedInject
import java.util.concurrent.TimeUnit
import javax.inject.Inject

/** Sincronización: envía los eventos pendientes en lotes de 20; lo registrado
 * y lo duplicado se retiran de la cola (la idempotencia del backend los
 * conserva) y un fallo deja la cola para el reintento con backoff. */
@HiltWorker
class SyncWorker @AssistedInject constructor(
    @Assisted context: Context,
    @Assisted params: WorkerParameters,
    private val dao: PendingEventDao,
    private val api: FudiaPayApi,
) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        if (!apiPaired()) return Result.retry()
        val batch = dao.all().take(20)
        if (batch.isEmpty()) return Result.success()
        return try {
            val response = api.report(ReportRequest(events = batch.map { event ->
                PaymentEvent(
                    provider = event.provider,
                    eventKey = event.providerEventKey,
                    senderName = event.senderName,
                    amount = event.amount,
                    occurredAt = event.occurredAtMillis,
                )
            }))
            response.events.forEach { outcome -> dao.delete(outcome.eventKey) }
            if (dao.all().isNotEmpty()) Result.retry() else Result.success()
        } catch (error: Exception) {
            batch.forEach { event -> dao.markAttempt(event.providerEventKey) }
            Result.retry()
        }
    }

    private suspend fun apiPaired(): Boolean = runCatching {
        api.session().device.id.isNotBlank()
    }.getOrDefault(false)
}

/** Encola el trabajo: inmediato al detectar un pago y periódico de respaldo
 * (recuperación tras conectividad o reinicio), con backoff exponencial. */
class SyncScheduler @Inject constructor(@dagger.hilt.android.qualifiers.ApplicationContext private val context: Context) {
    private val constraints = Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build()

    fun enqueueNow() {
        WorkManager.getInstance(context).enqueueUniqueWork(
            SYNC_WORK,
            ExistingWorkPolicy.APPEND_OR_REPLACE,
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

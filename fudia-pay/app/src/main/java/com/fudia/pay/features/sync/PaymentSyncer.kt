package com.fudia.pay.features.sync

import android.util.Log
import com.fudia.pay.BuildConfig
import com.fudia.pay.core.network.FudiaPayApi
import com.fudia.pay.core.network.PaymentEvent
import com.fudia.pay.core.network.ReportRequest
import com.fudia.pay.features.payments.PendingEventDao
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Envío de la cola al backend, compartido por el listener y el worker.
 *
 * El listener lo llama apenas encola un pago: su proceso está enlazado por el
 * sistema (prioridad de servicio en primer plano), así que el envío sale al
 * instante aunque la app esté cerrada. WorkManager queda solo como respaldo:
 * en segundo plano Android y las capas de marca (MagicOS, HyperOS) aplazan
 * sus trabajos hasta que el usuario abre la app.
 *
 * Un solo envío a la vez (Mutex): listener y worker viven en el mismo proceso
 * y la idempotencia del backend cubre cualquier carrera restante.
 */
@Singleton
class PaymentSyncer @Inject constructor(
    private val dao: PendingEventDao,
    private val api: FudiaPayApi,
) {
    private val mutex = Mutex()

    /** Envía la cola en lotes de 20. Devuelve true si quedó vacía; ante un
     *  fallo marca el intento y devuelve false para que el llamador reintente. */
    suspend fun flush(): Boolean = mutex.withLock { drain() }

    private suspend fun drain(): Boolean {
        var batch = dao.all().take(BATCH_SIZE)
        while (batch.isNotEmpty()) {
            if (BuildConfig.DEBUG) Log.d(TAG, "sincronización: enviando ${batch.size} eventos")
            val response = try {
                api.report(ReportRequest(events = batch.map { event ->
                    PaymentEvent(
                        provider = event.provider,
                        eventKey = event.providerEventKey,
                        senderName = event.senderName,
                        amount = event.amount,
                        occurredAt = isoUtc(event.occurredAtMillis),
                        operationNumber = event.operationNumber,
                    )
                }))
            } catch (error: CancellationException) {
                throw error
            } catch (error: Exception) {
                if (BuildConfig.DEBUG) Log.w(TAG, "sincronización: fallo, queda en cola", error)
                batch.forEach { event -> dao.markAttempt(event.providerEventKey) }
                return false
            }
            if (BuildConfig.DEBUG) {
                Log.d(TAG, "sincronización: ${response.events.size} confirmados (${response.registered} registrados, ${response.duplicates} duplicados)")
            }
            // Lo registrado y lo duplicado sale de la cola; si el backend no
            // confirmó nada del lote, se corta para no repetirlo en bucle.
            response.events.forEach { outcome -> dao.delete(outcome.eventKey) }
            if (response.events.isEmpty()) return false
            batch = dao.all().take(BATCH_SIZE)
        }
        return true
    }

    private companion object {
        const val TAG = "FudiaPay"
        const val BATCH_SIZE = 20
    }
}

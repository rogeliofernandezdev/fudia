package com.fudia.pay.features.notifications

import android.content.ComponentName
import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import android.util.Log
import com.fudia.pay.BuildConfig
import com.fudia.pay.core.security.DeviceCredentials
import com.fudia.pay.features.payments.PendingEvent
import com.fudia.pay.features.payments.PendingEventDao
import com.fudia.pay.features.sync.PaymentSyncer
import com.fudia.pay.features.sync.SyncScheduler
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * Único mecanismo de captura: el permiso lo concede el usuario en la
 * configuración de Android. Solo procesa los paquetes verificados de Yape y
 * Plin; el parser descarta lo ambiguo. Nunca registra el contenido en logs.
 *
 * Los rastros de depuración (solo debug) reportan decisiones, nunca contenido:
 * paquete de origen, veredicto del parser y resultado del encolado.
 *
 * El pago se guarda en la cola (Room) y se envía al instante desde este mismo
 * servicio; WorkManager queda de respaldo si no hay red o el backend falla.
 */
@AndroidEntryPoint
class PayNotificationListener : NotificationListenerService() {
    @Inject lateinit var credentials: DeviceCredentials
    @Inject lateinit var dao: PendingEventDao
    @Inject lateinit var scheduler: SyncScheduler
    @Inject lateinit var syncer: PaymentSyncer
    @Inject lateinit var listenerStatus: ListenerStatus

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    companion object {
        const val TAG = "FudiaPay"

        /** Señal en memoria: no nula solo mientras el servicio está enlazado
         *  en ESTE proceso. El worker (mismo proceso) la lee para decidir el
         *  reenlace: si el sistema mató el proceso, el proceso nuevo nace con
         *  null y el barrido repara sin abrir la app. */
        @Volatile
        var connectionEpoch: Long? = null
    }

    override fun onListenerConnected() {
        super.onListenerConnected()
        connectionEpoch = System.currentTimeMillis()
        scope.launch { listenerStatus.markConnected() }
        if (BuildConfig.DEBUG) Log.d(TAG, "servicio de escucha conectado")
        // Lo que quedó en cola (sin red, proceso muerto) sale al reenlazar,
        // sin esperar a que el usuario abra la app.
        if (credentials.paired) scope.launch { if (dao.all().isNotEmpty()) sendNow() }
    }

    override fun onListenerDisconnected() {
        // Android desconectó el servicio (ahorro de batería, límites del
        // fabricante): se solicita el reenlace para recuperarlo sin abrir la
        // app, y queda registrado en la salud que muestra la pantalla.
        if (BuildConfig.DEBUG) Log.d(TAG, "desconectado por el sistema: solicitando reenlace")
        connectionEpoch = null
        scope.launch { listenerStatus.markDisconnected() }
        requestRebind(ComponentName(this, PayNotificationListener::class.java))
    }

    override fun onDestroy() {
        connectionEpoch = null
        scope.cancel()
        super.onDestroy()
    }

    /** Envío inmediato desde este proceso, que el sistema mantiene enlazado
     *  con prioridad de servicio en primer plano: WorkManager en segundo plano
     *  lo aplazaría hasta abrir la app. Si falla, el worker reintenta. */
    private suspend fun sendNow() {
        if (!syncer.flush()) scheduler.enqueueNow()
    }

    override fun onNotificationPosted(sbn: StatusBarNotification) {
        if (BuildConfig.DEBUG) Log.d(TAG, "notificación de ${sbn.packageName}")
        if (!credentials.paired) {
            if (BuildConfig.DEBUG) Log.d(TAG, "sin vinculación: ignorada")
            return
        }
        val provider = KnownApps.providerOf(sbn.packageName)
        if (provider == null) {
            if (BuildConfig.DEBUG) Log.d(TAG, "paquete no verificado: ignorada")
            return
        }
        val extras = sbn.notification.extras
        val title = extras.getCharSequence("android.title")?.toString()
        // El texto expandido puede incluir la referencia que no cabe en el resumen.
        val text = extras.getCharSequence("android.text")?.toString()
        val expandedText = extras.getCharSequence("android.bigText")?.toString()
        val whenMillis = sbn.notification.`when`
        val key = sbn.key
        // Fire-and-forget con su propio scope de E/S: el servicio permanece
        // vivo como servicio del sistema mientras encola y envía. Sin
        // runBlocking para no bloquear el hilo del listener.
        scope.launch {
            val payment = NotificationParser.parse(
                provider = provider,
                packageName = sbn.packageName,
                title = title,
                text = text,
                whenMillis = whenMillis,
                key = key,
                expandedText = expandedText,
            )
            if (payment == null) {
                if (BuildConfig.DEBUG) Log.d(TAG, "parser sin confianza: descartada")
                return@launch
            }
            if (payment.senderName.isBlank() || payment.amount <= 0.0) {
                if (BuildConfig.DEBUG) Log.d(TAG, "datos incompletos: descartada")
                return@launch
            }
            val event = PendingEvent(
                providerEventKey = payment.eventKey,
                provider = payment.provider,
                senderName = payment.senderName,
                amount = payment.amount,
                occurredAtMillis = payment.occurredAtMillis,
                createdAtMillis = System.currentTimeMillis(),
                operationNumber = payment.operationNumber,
            )
            // El insert idempotente decide: si la clave ya está en la cola,
            // la notificación repetida no se encola otra vez.
            val inserted = dao.insert(event)
            if (inserted != -1L) {
                if (BuildConfig.DEBUG) Log.d(TAG, "pago encolado para sincronizar")
                sendNow()
            } else if (BuildConfig.DEBUG) {
                Log.d(TAG, "pago duplicado en cola: ignorado")
            }
        }
    }
}

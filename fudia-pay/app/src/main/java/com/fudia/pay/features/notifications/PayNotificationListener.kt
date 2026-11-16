package com.fudia.pay.features.notifications

import android.service.notification.NotificationListenerService
import android.service.notification.StatusBarNotification
import com.fudia.pay.core.security.DeviceCredentials
import com.fudia.pay.features.payments.PendingEvent
import com.fudia.pay.features.payments.PendingEventDao
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
 * goAsync() mantiene vivo el callback mientras la cola hace su insert en un
 * hilo de E/S, sin bloquear el hilo del listener ni perder la notificación si
 * el sistema reclama el servicio.
 */
@AndroidEntryPoint
class PayNotificationListener : NotificationListenerService() {
    @Inject lateinit var credentials: DeviceCredentials
    @Inject lateinit var dao: PendingEventDao
    @Inject lateinit var scheduler: SyncScheduler

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onDestroy() {
        scope.cancel()
        super.onDestroy()
    }

    override fun onNotificationPosted(sbn: StatusBarNotification) {
        if (!credentials.paired) return
        val provider = KnownApps.providerOf(sbn.packageName) ?: return
        val extras = sbn.notification.extras
        val title = extras.getCharSequence("android.title")?.toString()
        val text = extras.getCharSequence("android.text")?.toString()
        val whenMillis = sbn.notification.`when`
        val key = sbn.key
        // Fire-and-forget con su propio scope de E/S: el insert es corto y el
        // servicio permanece vivo como servicio del sistema. Sin goAsync (no
        // existe en NotificationListenerService) y sin runBlocking.
        scope.launch {
            val payment = NotificationParser.parse(
                provider = provider,
                packageName = sbn.packageName,
                title = title,
                text = text,
                whenMillis = whenMillis,
                key = key,
            ) ?: return@launch
            if (payment.senderName.isBlank() || payment.amount <= 0.0) return@launch
            val event = PendingEvent(
                providerEventKey = payment.eventKey,
                provider = payment.provider,
                senderName = payment.senderName,
                amount = payment.amount,
                occurredAtMillis = payment.occurredAtMillis,
                createdAtMillis = System.currentTimeMillis(),
            )
            // El insert idempotente decide: si la clave ya está en la cola,
            // la notificación repetida no se encola otra vez.
            val inserted = dao.insert(event)
            if (inserted != -1L) scheduler.enqueueNow()
        }
    }
}

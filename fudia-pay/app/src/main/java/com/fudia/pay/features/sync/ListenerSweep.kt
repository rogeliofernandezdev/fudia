package com.fudia.pay.features.sync

import android.content.ComponentName
import android.content.Context
import android.service.notification.NotificationListenerService
import android.util.Log
import androidx.core.app.NotificationManagerCompat
import com.fudia.pay.BuildConfig
import com.fudia.pay.core.security.DeviceCredentials
import com.fudia.pay.features.notifications.PayNotificationListener
import dagger.hilt.EntryPoint
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Revisión completa que comparten el worker, el push y la alarma: repara el
 * listener si quedó sin enlazar o sin entregas y envía la cola. Basta con que
 * el proceso despierte para que Android le entregue las notificaciones que
 * retuvo mientras estaba congelado; esta revisión cubre lo que falte.
 */
@Singleton
class ListenerSweep @Inject constructor(
    @ApplicationContext private val context: Context,
    private val credentials: DeviceCredentials,
    private val syncer: PaymentSyncer,
) {
    /** true si la cola quedó vacía. */
    suspend fun run(): Boolean {
        if (!credentials.paired) return true
        rebindListenerIfDead(context)
        // Enlazado no basta: el sistema puede dejar de entregarle avisos.
        PayNotificationListener.instance?.verifyDelivery()
        return syncer.flush()
    }
}

/** Auto-reparación del listener: si el permiso está concedido pero el
 *  servicio no está enlazado en este proceso (tras un proceso muerto), se pide
 *  el reenlace sin abrir la app. requestRebind es estático. */
internal fun rebindListenerIfDead(context: Context) {
    val enabled = NotificationManagerCompat.getEnabledListenerPackages(context).contains(context.packageName)
    if (!shouldRequestRebind(enabled, PayNotificationListener.connectionEpoch != null)) return
    if (BuildConfig.DEBUG) Log.d("FudiaPay", "listener sin enlazar con permiso activo: solicitando reenlace")
    NotificationListenerService.requestRebind(ComponentName(context, PayNotificationListener::class.java))
}

/** Acceso desde componentes que el sistema crea fuera de Hilt (push, alarma). */
@EntryPoint
@InstallIn(SingletonComponent::class)
interface WakeEntryPoint {
    fun sweep(): ListenerSweep
    fun pushTokens(): PushTokens
    fun credentials(): DeviceCredentials
}

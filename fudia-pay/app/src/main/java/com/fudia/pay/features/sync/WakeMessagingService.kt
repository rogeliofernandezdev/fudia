package com.fudia.pay.features.sync

import android.util.Log
import com.fudia.pay.BuildConfig
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import dagger.hilt.android.EntryPointAccessors
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeoutOrNull

/** El único mensaje que el backend envía: «procesa tu bandeja». */
internal fun isWakeMessage(data: Map<String, String>): Boolean = data["type"] == WAKE_TYPE

internal const val WAKE_TYPE = "wake"

/** Espera breve para que Android entregue al listener las notificaciones que
 *  retuvo mientras el proceso estaba congelado, antes de revisar la cola. */
internal const val WAKE_SETTLE_MS = 1_500L
private const val WAKE_BUDGET_MS = 15_000L

/**
 * Push silencioso del backend (Caja abierta o cobrando). HONOR/MagicOS
 * congela la app aunque tenga permisos, pero la descongela al llegar un push
 * de Google: las notificaciones de Yape y Plin retenidas se procesan y el
 * pago llega a Caja en segundos. No muestra nada al usuario.
 */
class WakeMessagingService : FirebaseMessagingService() {
    private val entryPoint by lazy { EntryPointAccessors.fromApplication(applicationContext, WakeEntryPoint::class.java) }

    override fun onMessageReceived(message: RemoteMessage) {
        if (!isWakeMessage(message.data)) return
        if (BuildConfig.DEBUG) Log.d(TAG, "push de despertar recibido")
        // Hilo de trabajo del SDK con unos 20 s de margen: se bloquea aquí para
        // que el proceso siga activo mientras envía la cola.
        runBlocking {
            withTimeoutOrNull(WAKE_BUDGET_MS) {
                delay(WAKE_SETTLE_MS)
                entryPoint.sweep().run()
            }
        }
    }

    override fun onNewToken(token: String) {
        runBlocking { withTimeoutOrNull(WAKE_BUDGET_MS) { entryPoint.pushTokens().sync(token) } }
    }

    private companion object {
        const val TAG = "FudiaPay"
    }
}

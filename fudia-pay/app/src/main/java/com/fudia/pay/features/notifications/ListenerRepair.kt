package com.fudia.pay.features.notifications

import android.content.ComponentName
import android.content.Context
import android.content.pm.PackageManager

/** Silencio mínimo antes de sospechar: sin ninguna notificación entregada
 *  (de cualquier app) ni conexión reciente durante este tiempo. */
internal const val DELIVERY_SILENCE_MS = 10 * 60_000L

/** Separación mínima entre reparaciones forzadas: en un teléfono sano con la
 *  bandeja vacía y sin avisos, la reparación se repite como mucho a este ritmo. */
internal const val FORCED_REBIND_INTERVAL_MS = 30 * 60_000L

/**
 * Listener «zombi» (visto en HONOR/MagicOS con Android 15): el sistema
 * mantiene el servicio enlazado, pero su lista interna de listeners
 * habilitados queda vacía («All notification listeners (0) enabled») y deja
 * de entregarle notificaciones. La app no recibe ningún callback; solo se
 * nota porque `activeNotifications` vuelve vacío y no llega nada.
 *
 * Se sospecha cuando el listener no ve ninguna notificación activa y lleva un
 * rato sin entregas ni reconexiones. Una bandeja vacía en un teléfono sano
 * también cumple la regla: la reparación es inocua (reconexión de un segundo y
 * barrido de la bandeja al reconectar), por eso solo se limita su frecuencia.
 */
internal fun shouldForceRebind(
    visibleCount: Int,
    lastDeliveryAtMillis: Long,
    connectedAtMillis: Long,
    lastForcedAtMillis: Long,
    now: Long,
): Boolean {
    if (visibleCount > 0) return false
    val lastSignal = maxOf(lastDeliveryAtMillis, connectedAtMillis)
    if (now - lastSignal < DELIVERY_SILENCE_MS) return false
    return now - lastForcedAtMillis >= FORCED_REBIND_INTERVAL_MS
}

/** Reparación sin tocar ajustes del sistema: desactivar y reactivar el propio
 *  componente del listener provoca PACKAGE_CHANGED, y Android reconstruye
 *  desde el permiso concedido la lista completa de listeners habilitados (lo
 *  mismo que ocurre al reinstalar el APK). DONT_KILL_APP mantiene el proceso. */
internal object ListenerRepair {
    private fun component(context: Context) = ComponentName(context, PayNotificationListener::class.java)

    fun forceFullRebind(context: Context) {
        val pm = context.packageManager
        val component = component(context)
        pm.setComponentEnabledSetting(component, PackageManager.COMPONENT_ENABLED_STATE_DISABLED, PackageManager.DONT_KILL_APP)
        pm.setComponentEnabledSetting(component, PackageManager.COMPONENT_ENABLED_STATE_DEFAULT, PackageManager.DONT_KILL_APP)
    }

    /** Red de seguridad: si el proceso murió entre los dos pasos de la
     *  reparación, el componente quedaría desactivado para siempre. */
    fun ensureEnabled(context: Context) {
        val pm = context.packageManager
        val component = component(context)
        if (pm.getComponentEnabledSetting(component) == PackageManager.COMPONENT_ENABLED_STATE_DISABLED) {
            pm.setComponentEnabledSetting(component, PackageManager.COMPONENT_ENABLED_STATE_DEFAULT, PackageManager.DONT_KILL_APP)
        }
    }
}

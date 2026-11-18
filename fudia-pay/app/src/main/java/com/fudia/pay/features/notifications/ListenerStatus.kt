package com.fudia.pay.features.notifications

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.longPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import javax.inject.Inject
import javax.inject.Singleton

private val Context.listenerStore by preferencesDataStore(name = "fudia-pay-listener")

/** Salud real del servicio de escucha: conecta o desconecta lo que el
 *  sistema reporta, no solo el permiso concedido. */
data class ListenerHealth(val connectedAtMillis: Long, val disconnectedAtMillis: Long) {
    val connected: Boolean get() = connectedAtMillis > disconnectedAtMillis
}

/** Persistencia de la vida del NotificationListenerService. Los callbacks
 *  (onListenerConnected/onListenerDisconnected) los emite el propio servicio,
 *  así la pantalla muestra la verdad del sistema: en teléfonos con batería
 *  agresiva el permiso puede estar concedido con el servicio muerto. */
@Singleton
class ListenerStatus @Inject constructor(@ApplicationContext private val context: Context) {
    private val store get() = context.listenerStore
    private val connectedKey = longPreferencesKey("connected_at_millis")
    private val disconnectedKey = longPreferencesKey("disconnected_at_millis")

    val health: Flow<ListenerHealth> = store.data.map { prefs ->
        ListenerHealth(
            connectedAtMillis = prefs[connectedKey] ?: 0L,
            disconnectedAtMillis = prefs[disconnectedKey] ?: 0L,
        )
    }

    suspend fun markConnected() {
        store.edit { it[connectedKey] = System.currentTimeMillis() }
    }

    suspend fun markDisconnected() {
        store.edit { it[disconnectedKey] = System.currentTimeMillis() }
    }
}

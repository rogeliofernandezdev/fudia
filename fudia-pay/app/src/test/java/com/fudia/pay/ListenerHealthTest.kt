package com.fudia.pay

import com.fudia.pay.features.notifications.ListenerHealth
import com.fudia.pay.features.notifications.shouldForceRebind
import com.fudia.pay.features.settings.DetectionState
import com.fudia.pay.features.settings.detectionState
import com.fudia.pay.features.sync.shouldRequestRebind
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/** La salud del servicio muestra la verdad del sistema: sin permiso,
 *  conectando o activa. */
class ListenerHealthTest {

    @Test
    fun `la conexion mas reciente define el estado`() {
        assertTrue(ListenerHealth(connectedAtMillis = 10, disconnectedAtMillis = 0).connected)
        assertTrue(ListenerHealth(connectedAtMillis = 10, disconnectedAtMillis = 5).connected)
        assertFalse(ListenerHealth(connectedAtMillis = 5, disconnectedAtMillis = 10).connected)
    }

    @Test
    fun `inicio muestra la deteccion segun permiso y servicio`() {
        val connected = ListenerHealth(connectedAtMillis = 10, disconnectedAtMillis = 0)
        val disconnected = ListenerHealth(connectedAtMillis = 5, disconnectedAtMillis = 10)
        assertEquals(DetectionState.Off, detectionState(granted = false, health = connected))
        assertEquals(DetectionState.Connecting, detectionState(granted = true, health = ListenerHealth(0, 0)))
        assertEquals(DetectionState.Connecting, detectionState(granted = true, health = disconnected))
        assertEquals(DetectionState.Active, detectionState(granted = true, health = connected))
    }

    @Test
    fun `el barrido pide reenlace solo con permiso activo y servicio sin señal en el proceso`() {
        // Permiso concedido y sin señal (proceso muerto o servicio desconectado): reparar.
        assertTrue(shouldRequestRebind(listenerEnabled = true, boundInProcess = false))
        // Servicio enlazado en este proceso: nada que hacer.
        assertFalse(shouldRequestRebind(listenerEnabled = true, boundInProcess = true))
        // Sin permiso, la app no puede hacer nada: no llamar al sistema.
        assertFalse(shouldRequestRebind(listenerEnabled = false, boundInProcess = false))
    }

    private val minute = 60_000L
    private val now = 1_000 * minute

    @Test
    fun `listener zombi enlazado sin entregas fuerza el reenlace completo`() {
        // Conectado hace una hora, ninguna entrega y la bandeja vacía a sus ojos.
        assertTrue(shouldForceRebind(visibleCount = 0, lastDeliveryAtMillis = 0, connectedAtMillis = now - 60 * minute, lastForcedAtMillis = 0, now = now))
    }

    @Test
    fun `si ve notificaciones activas el sistema le entrega y no se toca`() {
        assertFalse(shouldForceRebind(visibleCount = 3, lastDeliveryAtMillis = 0, connectedAtMillis = now - 60 * minute, lastForcedAtMillis = 0, now = now))
    }

    @Test
    fun `una entrega o conexion reciente evita la reparacion`() {
        assertFalse(shouldForceRebind(0, lastDeliveryAtMillis = now - 2 * minute, connectedAtMillis = now - 60 * minute, lastForcedAtMillis = 0, now = now))
        assertFalse(shouldForceRebind(0, lastDeliveryAtMillis = 0, connectedAtMillis = now - 2 * minute, lastForcedAtMillis = 0, now = now))
    }

    @Test
    fun `la reparacion forzada respeta su intervalo minimo`() {
        assertFalse(shouldForceRebind(0, 0, now - 60 * minute, lastForcedAtMillis = now - 10 * minute, now = now))
        assertTrue(shouldForceRebind(0, 0, now - 60 * minute, lastForcedAtMillis = now - 31 * minute, now = now))
    }
}

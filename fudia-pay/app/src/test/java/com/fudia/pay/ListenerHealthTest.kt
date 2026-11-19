package com.fudia.pay

import com.fudia.pay.features.notifications.ListenerHealth
import com.fudia.pay.features.settings.livenessText
import com.fudia.pay.features.settings.sinceLabel
import com.fudia.pay.features.sync.shouldRequestRebind
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/** La salud del servicio muestra la verdad del sistema: conectado o
 *  desconectado con hace cuánto, en español claro. */
class ListenerHealthTest {

    @Test
    fun `la conexion mas reciente define el estado`() {
        assertTrue(ListenerHealth(connectedAtMillis = 10, disconnectedAtMillis = 0).connected)
        assertTrue(ListenerHealth(connectedAtMillis = 10, disconnectedAtMillis = 5).connected)
        assertFalse(ListenerHealth(connectedAtMillis = 5, disconnectedAtMillis = 10).connected)
    }

    @Test
    fun `el texto de antiguedad usa minutos, horas y dias`() {
        assertEquals("ahora mismo", sinceLabel(atMillis = 0, now = 59_999))
        assertEquals("hace 1 minuto", sinceLabel(atMillis = 0, now = 60_000))
        assertEquals("hace 5 minutos", sinceLabel(atMillis = 0, now = 5 * 60_000))
        assertEquals("hace 2 h", sinceLabel(atMillis = 0, now = 120 * 60_000))
        assertEquals("hace 3 d", sinceLabel(atMillis = 0, now = 3 * 1_440 * 60_000))
    }

    @Test
    fun `sin conexiones registradas el texto lo admite`() {
        val health = ListenerHealth(0, 0)
        assertEquals("El servicio de escucha aún no se ha conectado.", livenessText(health, now = 1_000))
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
}

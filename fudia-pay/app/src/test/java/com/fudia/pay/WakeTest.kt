package com.fudia.pay

import com.fudia.pay.features.sync.WAKE_ALARM_INTERVAL_MS
import com.fudia.pay.features.sync.isWakeMessage
import com.fudia.pay.features.sync.shouldUploadPushToken
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/** El push y la alarma despiertan la app que el fabricante congela; el token
 *  solo viaja vinculado y cuando cambia. */
class WakeTest {

    @Test
    fun `solo el mensaje de despertar dispara la revision`() {
        assertTrue(isWakeMessage(mapOf("type" to "wake")))
        assertFalse(isWakeMessage(mapOf("type" to "promo")))
        assertFalse(isWakeMessage(emptyMap()))
    }

    @Test
    fun `el token se registra vinculado y solo si cambio`() {
        assertTrue(shouldUploadPushToken(paired = true, current = "nuevo", uploaded = ""))
        assertTrue(shouldUploadPushToken(paired = true, current = "nuevo", uploaded = "viejo"))
        assertFalse(shouldUploadPushToken(paired = true, current = "igual", uploaded = "igual"))
        assertFalse(shouldUploadPushToken(paired = false, current = "nuevo", uploaded = ""))
        assertFalse(shouldUploadPushToken(paired = true, current = "", uploaded = "viejo"))
    }

    @Test
    fun `la alarma de respaldo coincide con la ventana de 5 minutos de HONOR`() {
        assertEquals(5 * 60_000L, WAKE_ALARM_INTERVAL_MS)
    }
}
